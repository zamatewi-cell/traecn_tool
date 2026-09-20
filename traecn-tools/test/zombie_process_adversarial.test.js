const test = require('node:test');
const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const { terminateProxyProcess, setProxyProcess, getProxyProcess } = require('../electron/main.js');

test('Zombie Adversarial 1: Stubborn child process ignoring all kill attempts must return failure and retain handle', async () => {
  const zombieProc = new EventEmitter();
  zombieProc.pid = 77777;
  zombieProc.killed = false;
  zombieProc.exitCode = null;
  // 模拟对 SIGTERM/SIGKILL 免疫，永不触发 exit 事件
  zombieProc.kill = (signal) => {
    return false;
  };

  setProxyProcess(zombieProc);

  const res = await terminateProxyProcess(zombieProc, 150);

  assert.strictEqual(res.success, false, '僵尸进程超时未退出必须返回 success: false');
  assert.match(res.error, /超时|timeout/i, '错误信息必须提示超时');
  assert.strictEqual(res.pid, 77777);
  assert.strictEqual(getProxyProcess(), zombieProc, '僵尸进程句柄绝不可被置为 null，必须保留以便后续处置');
});

test('Zombie Adversarial 2: Settings clear data guard strictly aborts reset when terminateProxyProcess fails', async () => {
  const stubbornProc = new EventEmitter();
  stubbornProc.pid = 66666;
  stubbornProc.kill = () => false;

  setProxyProcess(stubbornProc);

  // 模拟生产磁盘数据
  let diskData = {
    accounts: [{ id: 'prod-acc-1', token: 'critical_production_token' }],
    proxyConfig: { listenPort: 8045, authEnabled: true, apiKey: 'critical_key' },
  };
  let clearDataExecuted = false;

  // 模拟 Settings.tsx 中的 handleClearData 完整前置守卫逻辑
  const simulateHandleClearData = async (stopProxyFn, saveDataFn) => {
    // 步骤 1: 强同步前置守卫
    const stopRes = await stopProxyFn();
    if (stopRes && !stopRes.success && stopRes.error !== '服务未运行') {
      throw new Error(`停止正在运行的代理服务失败: ${stopRes.error}，为防止凭据残留已中止重置！`);
    }

    // 步骤 2: 清空本地持久化存储
    await saveDataFn({ accounts: [], proxyConfig: {} });
    clearDataExecuted = true;
  };

  const mockStopProxy = async () => {
    return await terminateProxyProcess(getProxyProcess(), 100);
  };

  const mockSaveData = async (newData) => {
    diskData = newData;
  };

  // 执行清除操作，断言必须被异常中断阻断
  await assert.rejects(
    async () => {
      await simulateHandleClearData(mockStopProxy, mockSaveData);
    },
    (err) => {
      assert.match(err.message, /停止正在运行的代理服务失败/);
      assert.match(err.message, /已中止重置/);
      return true;
    },
    '当子进程卡死无法停止时，必须坚决抛出异常中断清除数据流程'
  );

  // 断言：步骤 2 清空动作绝对未被执行
  assert.strictEqual(clearDataExecuted, false, '清空数据标志必须为 false');
  // 断言：磁盘核心数据 100% 毫发无损
  assert.strictEqual(diskData.accounts.length, 1, '生产账号数据绝不能被清空');
  assert.strictEqual(diskData.accounts[0].token, 'critical_production_token', '生产 Token 绝不能被抹除');
  assert.strictEqual(getProxyProcess(), stubbornProc, '僵尸进程句柄依然保留在全局供恢复');
});

test('Zombie Adversarial 3: Process whose kill method throws unexpected exception', async () => {
  const buggyProc = new EventEmitter();
  buggyProc.pid = 55555;
  buggyProc.kill = () => {
    throw new Error('EPERM: Operation not permitted on zombie process');
  };

  setProxyProcess(buggyProc);

  const res = await terminateProxyProcess(buggyProc, 100);

  assert.strictEqual(res.success, false, 'kill 抛异常且未退出的进程应返回 success: false');
  assert.strictEqual(getProxyProcess(), buggyProc, '句柄不可丢失');
});

test('Zombie Adversarial 4: Healthy process exiting cleanly enables data clear successfully', async () => {
  const healthyProc = new EventEmitter();
  healthyProc.pid = 44444;
  healthyProc.kill = () => {
    setTimeout(() => healthyProc.emit('exit', 0), 20);
    return true;
  };

  setProxyProcess(healthyProc);

  let diskData = { accounts: [{ id: 'acc-1', token: 'old_token' }] };
  let clearSuccess = false;

  const res = await terminateProxyProcess(healthyProc, 1000);
  assert.strictEqual(res.success, true);
  assert.strictEqual(getProxyProcess(), null, '正常退出的进程句柄应置为 null');

  if (res.success) {
    diskData = { accounts: [] };
    clearSuccess = true;
  }

  assert.strictEqual(clearSuccess, true);
  assert.strictEqual(diskData.accounts.length, 0);
});
