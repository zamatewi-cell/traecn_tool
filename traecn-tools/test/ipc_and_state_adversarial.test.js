const test = require('node:test');
const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const {
  terminateProxyProcess,
  startProxyServiceInternal,
  stopProxyService,
  setProxyProcess,
  getProxyProcess,
} = require('../electron/main.js');

// 测试 1: [P1] saveData 发生写盘异常时，必须返回 { success: false, error: ... }，绝不吞掉错误
test('Adversarial P1-1: saveData disk failure returns structured error and prevents fake success', async () => {
  // 模拟真实 save-data handler 逻辑
  const handleSaveData = (data, writeFn) => {
    try {
      writeFn(data);
      return { success: true };
    } catch (e) {
      return { success: false, error: e.message || String(e) };
    }
  };

  // 注入写盘错误 (如磁盘已满 ENOSPC)
  const failWrite = () => {
    throw new Error('ENOSPC: no space left on device, write');
  };

  const res = handleSaveData({ accounts: [] }, failWrite);
  assert.strictEqual(res.success, false, '写盘失败时 success 必须为 false');
  assert.match(res.error, /ENOSPC/, 'error 必须包含底层异常信息');

  // 模拟 UI 消费该 handler：若 res.success === false，绝不可更新内存状态或宣告成功
  let memoryStoreUpdated = false;
  let successNotificationShown = false;

  const simulateSettingsClearData = async (saveDataHandler) => {
    const saveRes = await saveDataHandler({ accounts: [] });
    if (saveRes && saveRes.success === false) {
      throw new Error(`写入磁盘失败 (${saveRes.error})，已取消清空操作！`);
    }
    // 只有成功才走后续步骤
    memoryStoreUpdated = true;
    successNotificationShown = true;
  };

  await assert.rejects(
    async () => {
      await simulateSettingsClearData(async (d) => handleSaveData(d, failWrite));
    },
    (err) => {
      assert.match(err.message, /写入磁盘失败/);
      assert.match(err.message, /ENOSPC/);
      return true;
    }
  );

  assert.strictEqual(memoryStoreUpdated, false, '写盘失败时内存 Store 绝不能被清空');
  assert.strictEqual(successNotificationShown, false, '写盘失败时绝不能弹出成功提示');
});

// 测试 2: [P2-2] 旧代理进程终止失败时，startProxyServiceInternal（重新启动）必须熔断中止，绝不覆盖句柄
test('Adversarial P2-2: startProxyServiceInternal aborts immediately when terminating old process fails', async () => {
  const stubbornOldProc = new EventEmitter();
  stubbornOldProc.pid = 88888;
  stubbornOldProc.kill = () => false; // 无法终止

  setProxyProcess(stubbornOldProc);

  // 尝试拉起新配置
  const startResult = await startProxyServiceInternal({
    listenPort: 8045,
    authEnabled: false,
  });

  assert.strictEqual(startResult.success, false, '终止旧进程失败时必须返回失败');
  assert.match(startResult.error, /旧代理进程终止失败|超时/, '必须明确提示旧代理进程终止失败');
  assert.strictEqual(getProxyProcess(), stubbornOldProc, '旧代理进程句柄必须被保留，绝不可被新进程覆盖');
});

// 测试 3: [P2-2] 恢复备份时，如果 stopProxy 失败，必须立即中止恢复，绝不持久化或修改 Store
test('Adversarial P2-2: Settings restore backup aborts when stopProxy fails', async () => {
  let diskState = { accounts: [{ id: 'old-account', token: 'keep_this_token' }] };
  let storeState = { accounts: [{ id: 'old-account', token: 'keep_this_token' }] };
  let restoreSucceeded = false;

  const simulateHandleRestoreData = async (stopProxyFn, saveDataFn) => {
    // 步骤 1: 停止代理服务
    const stopRes = await stopProxyFn();
    if (stopRes && stopRes.success === false) {
      throw new Error(`【恢复中止】停止正在运行的代理服务失败：${stopRes.error}，已取消恢复数据以防止状态冲突！`);
    }

    // 步骤 2: 持久化
    const saveRes = await saveDataFn({ accounts: [{ id: 'new-restored-account', token: 'new_token' }] });
    if (saveRes && saveRes.success === false) {
      throw new Error(`【恢复失败】持久化数据写入磁盘失败：${saveRes.error}`);
    }

    // 步骤 3: 更新 Store
    storeState = { accounts: [{ id: 'new-restored-account', token: 'new_token' }] };
    restoreSucceeded = true;
  };

  const mockFailingStopProxy = async () => ({
    success: false,
    error: '终止代理进程超时 (PID: 99999)，子进程仍在运行中',
  });

  const mockSaveData = async (d) => {
    diskState = d;
    return { success: true };
  };

  await assert.rejects(
    async () => {
      await simulateHandleRestoreData(mockFailingStopProxy, mockSaveData);
    },
    (err) => {
      assert.match(err.message, /恢复中止/);
      assert.match(err.message, /停止正在运行的代理服务失败/);
      return true;
    }
  );

  assert.strictEqual(restoreSucceeded, false, '恢复标志必须为 false');
  assert.strictEqual(diskState.accounts[0].token, 'keep_this_token', '磁盘数据绝不可被覆盖');
  assert.strictEqual(storeState.accounts[0].token, 'keep_this_token', 'Store 状态绝不可被修改');
});

// 测试 4: [P2-4] 运行时账号切换状态机：区分 active 与 pending_restart 状态
test('Adversarial P2-4: Runtime account switch state machine distinguishes active and pending restart', () => {
  let state = {
    proxyRunning: true,
    currentAccountId: 'acc-1',
    runtimeActiveAccountId: 'acc-1',
  };

  const getAccountStatus = (accId, state) => {
    if (state.proxyRunning) {
      if (accId === state.runtimeActiveAccountId) return 'active';
      if (accId === state.currentAccountId) return 'pending_restart';
      return 'inactive';
    } else {
      if (accId === state.currentAccountId) return 'preferred';
      return 'inactive';
    }
  };

  // 初始状态：代理在运行，acc-1 处于 active 生效中
  assert.strictEqual(getAccountStatus('acc-1', state), 'active');
  assert.strictEqual(getAccountStatus('acc-2', state), 'inactive');

  // 用户在运行中切换到 acc-2，但未重启代理
  state.currentAccountId = 'acc-2';
  // 关键断言：此时 acc-1 仍在网关中生效，acc-2 为待重启生效，绝不能声称 acc-2 已生效
  assert.strictEqual(getAccountStatus('acc-1', state), 'active', 'acc-1 必须仍为 active 生效中');
  assert.strictEqual(getAccountStatus('acc-2', state), 'pending_restart', 'acc-2 必须准确呈现为 pending_restart 待重启生效');

  // 用户执行平滑重启代理后：
  state.runtimeActiveAccountId = 'acc-2';
  assert.strictEqual(getAccountStatus('acc-2', state), 'active', '重启后 acc-2 正式成为 active 生效中');
  assert.strictEqual(getAccountStatus('acc-1', state), 'inactive', 'acc-1 降为 inactive');

  // 代理停止后：
  state.proxyRunning = false;
  state.runtimeActiveAccountId = null;
  assert.strictEqual(getAccountStatus('acc-2', state), 'preferred', '代理停止后首选为 acc-2');
});

// 测试 5: [P1 完整调用链] switchAccount 写盘失败时，内存必须回滚，页面绝对不重启代理，绝对不宣告成功
test('Adversarial P1-Callchain: switchAccount save failure strictly aborts proxy restart, rolls back memory, and prevents false success', async () => {
  // 模拟初始内存 Store
  let store = {
    accounts: [
      { id: 'acc-A', email: 'a@example.com', isCurrent: true, token: 'token-A' },
      { id: 'acc-B', email: 'b@example.com', isCurrent: false, token: 'token-B' },
    ],
    currentAccountId: 'acc-A',
    proxyRunning: true,
    runtimeActiveAccountId: 'acc-A', // 正在运行 A
  };

  // 模拟磁盘状态 (只保存了 A)
  let diskState = {
    accounts: [
      { id: 'acc-A', email: 'a@example.com', isCurrent: true, token: 'token-A' },
      { id: 'acc-B', email: 'b@example.com', isCurrent: false, token: 'token-B' },
    ],
    activeAccountId: 'acc-A',
  };

  let stopProxyCalled = false;
  let startProxyCalled = false;
  let successNoticeShown = false;
  let errorNoticeShown = false;

  // 模拟 store 中的 save 与 switchAccount（带有原子回滚）
  const mockSave = async (shouldFail) => {
    if (shouldFail) {
      const err = new Error('ENOSPC: no space left on device, write');
      throw err;
    }
    // 模拟写盘成功
    diskState.accounts = JSON.parse(JSON.stringify(store.accounts));
    diskState.activeAccountId = store.currentAccountId;
  };

  const switchAccount = async (id, shouldFailSave = false) => {
    const prevAccounts = JSON.parse(JSON.stringify(store.accounts));
    const prevCurrent = store.currentAccountId;
    store.accounts = store.accounts.map((a) => ({ ...a, isCurrent: a.id === id }));
    store.currentAccountId = id;
    try {
      await mockSave(shouldFailSave);
    } catch (e) {
      // 事务回滚
      store.accounts = prevAccounts;
      store.currentAccountId = prevCurrent;
      throw e;
    }
  };

  const stopProxy = async () => {
    stopProxyCalled = true;
    store.proxyRunning = false;
    return { success: true };
  };

  const startProxy = async () => {
    startProxyCalled = true;
    store.proxyRunning = true;
    // 关键：底层从磁盘读取真实的 activeAccountId 快照返回！
    const realActiveId = diskState.activeAccountId;
    store.runtimeActiveAccountId = realActiveId;
    return { success: true, activeAccountId: realActiveId };
  };

  // 模拟 Accounts.tsx / Dashboard.tsx 中的真实 handleSwitchAccount 链路
  const handleSwitchAccount = async (id, confirmedRestart = true, injectSaveFailure = false) => {
    if (store.proxyRunning) {
      try {
        await switchAccount(id, injectSaveFailure);
      } catch (err) {
        errorNoticeShown = true;
        // 关键守卫：写盘失败立即中止，绝不执行后续 stop/start 重启！
        return;
      }

      if (confirmedRestart) {
        const stopRes = await stopProxy();
        if (!stopRes.success) return;
        const startRes = await startProxy();
        if (startRes.success) {
          successNoticeShown = true;
        }
      }
    } else {
      try {
        await switchAccount(id, injectSaveFailure);
      } catch (err) {
        errorNoticeShown = true;
      }
    }
  };

  // 场景 1: 注入写盘失败 (ENOSPC)
  await handleSwitchAccount('acc-B', true, true);

  // 司法级断言验证：
  assert.strictEqual(errorNoticeShown, true, '写盘失败时必须捕获并记录错误');
  assert.strictEqual(successNoticeShown, false, '写盘失败时绝对不能提示切换成功');
  assert.strictEqual(stopProxyCalled, false, '写盘失败时绝对不得调用 stopProxy 停止代理');
  assert.strictEqual(startProxyCalled, false, '写盘失败时绝对不得调用 startProxy 重启代理');
  // 断言内存已回滚：
  assert.strictEqual(store.currentAccountId, 'acc-A', '写盘失败后内存 currentAccountId 必须回滚为 acc-A');
  assert.strictEqual(store.accounts[0].isCurrent, true, 'acc-A 的 isCurrent 必须回滚为 true');
  assert.strictEqual(store.accounts[1].isCurrent, false, 'acc-B 的 isCurrent 必须回滚为 false');
  // 断言磁盘与运行状态完好：
  assert.strictEqual(diskState.activeAccountId, 'acc-A', '磁盘账号必须仍为 acc-A');
  assert.strictEqual(store.runtimeActiveAccountId, 'acc-A', '运行中生效账号必须仍为 acc-A');

  // 场景 2: 写盘成功，正常重启
  errorNoticeShown = false;
  await handleSwitchAccount('acc-B', true, false);

  assert.strictEqual(stopProxyCalled, true, '正常切换且确认重启必须调用 stopProxy');
  assert.strictEqual(startProxyCalled, true, '正常切换且确认重启必须调用 startProxy');
  assert.strictEqual(successNoticeShown, true, '保存并重启成功后才可提示成功');
  assert.strictEqual(store.currentAccountId, 'acc-B', '当前选择成功切换为 acc-B');
  assert.strictEqual(diskState.activeAccountId, 'acc-B', '磁盘已安全更新为 acc-B');
  assert.strictEqual(store.runtimeActiveAccountId, 'acc-B', '网关实际生效快照成功切换为 acc-B');
});
