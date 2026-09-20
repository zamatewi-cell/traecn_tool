const test = require('node:test');
const assert = require('node:assert/strict');
const { mergeAccountsSafely } = require('../electron/main.js');

test('Adversarial Race 1: 1000 rounds of high-frequency interleaved token refreshes vs stale UI saves', () => {
  const baseTime = Date.now();
  const numAccounts = 5;
  const numRounds = 1000;

  // 初始化磁盘账号
  let diskAccounts = Array.from({ length: numAccounts }, (_, i) => ({
    id: `acc-${i}`,
    label: `Account ${i}`,
    token: `token_init_${i}`,
    refreshToken: `refresh_init_${i}`,
    expiredAt: new Date(baseTime + 3600000).toISOString(),
    lastUsed: new Date(baseTime).toISOString(),
    disabled: false,
  }));

  // 模拟 UI 端的初始内存快照（持有旧 Token）
  let uiSnapshot = JSON.parse(JSON.stringify(diskAccounts));

  let totalRefreshes = 0;
  let verifiedPreserved = 0;

  for (let round = 0; round < numRounds; round++) {
    const targetIdx = round % numAccounts;
    const targetId = `acc-${targetIdx}`;
    const freshVersion = round + 1;
    const freshToken = `token_v${freshVersion}_refreshed`;
    const freshRefreshToken = `refresh_v${freshVersion}_refreshed`;
    const refreshTimestamp = new Date(baseTime + (round + 1) * 100).toISOString();

    // 1. 模拟主进程后台接收到 Go 核心凭据刷新事件，立即写回磁盘
    const targetDiskAcc = diskAccounts.find(a => a.id === targetId);
    targetDiskAcc.token = freshToken;
    targetDiskAcc.refreshToken = freshRefreshToken;
    targetDiskAcc.lastUsed = refreshTimestamp;
    totalRefreshes++;

    // 2. 模拟 UI 在用户修改 Label、Tags 或切换状态后，发起 save-data
    // 注意：UI 发起保存时，持有的仍然是旧内存快照中的旧 Token
    const incomingUI = JSON.parse(JSON.stringify(uiSnapshot));
    const targetUIAcc = incomingUI.find(a => a.id === targetId);
    targetUIAcc.label = `Account ${targetIdx} (Edited Round ${round})`;
    // UI 快照的 lastUsed 停留在旧时间戳
    targetUIAcc.lastUsed = new Date(baseTime + round * 50).toISOString();

    // 偶尔模拟 UI 新增账号
    if (round % 200 === 0) {
      incomingUI.push({
        id: `acc-new-${round}`,
        label: `New Account ${round}`,
        token: `token_new_${round}`,
        lastUsed: refreshTimestamp,
      });
    }

    // 3. 执行主进程 mergeAccountsSafely 合并守卫
    const merged = mergeAccountsSafely(incomingUI, diskAccounts);

    // 4. 断言守卫：最新 Token 绝不能被 UI 旧快照中的旧值覆盖！
    const mergedTarget = merged.find(a => a.id === targetId);
    assert.strictEqual(
      mergedTarget.token,
      freshToken,
      `[Round ${round}] 磁盘最新 Token 绝不可被旧 UI 快照覆盖 (期望: ${freshToken}, 实际: ${mergedTarget.token})`
    );
    assert.strictEqual(
      mergedTarget.refreshToken,
      freshRefreshToken,
      `[Round ${round}] 磁盘最新 RefreshToken 绝不可被旧 UI 快照覆盖`
    );
    // 同时断言：UI 修改的业务属性（Label）正常合入
    assert.strictEqual(
      mergedTarget.label,
      `Account ${targetIdx} (Edited Round ${round})`,
      `[Round ${round}] UI 修改的业务字段应成功合入`
    );

    verifiedPreserved++;
    // 更新磁盘存储为合并后的最新数据
    diskAccounts = merged;
  }

  assert.strictEqual(verifiedPreserved, numRounds, '全部 1000 轮竞态攻击中最新凭据必须 100% 成功保全');
});

test('Adversarial Race 2: Concurrent multi-account simultaneous refreshes and batch UI saves', () => {
  const now = new Date().toISOString();
  const diskAccounts = [
    { id: 'acc-1', token: 'fresh_tok_1', refreshToken: 'fresh_ref_1', lastUsed: now },
    { id: 'acc-2', token: 'fresh_tok_2', refreshToken: 'fresh_ref_2', lastUsed: now },
    { id: 'acc-3', token: 'fresh_tok_3', refreshToken: 'fresh_ref_3', lastUsed: now },
  ];

  const stalePast = new Date(Date.now() - 5000).toISOString();
  const staleUIAccounts = [
    { id: 'acc-1', token: 'stale_tok_1', refreshToken: 'stale_ref_1', lastUsed: stalePast, label: 'acc 1 renamed' },
    { id: 'acc-2', token: 'stale_tok_2', refreshToken: 'stale_ref_2', lastUsed: stalePast, label: 'acc 2 renamed' },
    { id: 'acc-3', token: 'stale_tok_3', refreshToken: 'stale_ref_3', lastUsed: stalePast, label: 'acc 3 renamed' },
  ];

  const merged = mergeAccountsSafely(staleUIAccounts, diskAccounts);

  assert.strictEqual(merged.length, 3);
  for (let i = 1; i <= 3; i++) {
    const acc = merged.find(a => a.id === `acc-${i}`);
    assert.strictEqual(acc.token, `fresh_tok_${i}`, `账号 acc-${i} 凭据保全失败`);
    assert.strictEqual(acc.refreshToken, `fresh_ref_${i}`, `账号 acc-${i} RefreshToken 保全失败`);
    assert.strictEqual(acc.label, `acc ${i} renamed`, `账号 acc-${i} UI 属性合入失败`);
  }
});

test('Adversarial Race 3: Boundary edge cases (undefined lastUsed, null, empty disk)', () => {
  const diskAccounts = [
    { id: 'acc-1', token: 'disk_tok_1', lastUsed: '2026-09-20T12:00:00Z' },
  ];

  // UI 没有传递 lastUsed (例如新增或精简模型对象)
  const incomingWithoutLastUsed = [
    { id: 'acc-1', token: 'stale_tok_1', label: 'Updated Label' },
  ];

  const merged1 = mergeAccountsSafely(incomingWithoutLastUsed, diskAccounts);
  assert.strictEqual(merged1[0].token, 'disk_tok_1', '当传入 lastUsed 缺失时，磁盘有效 token 仍应保全');

  // 空输入防御
  assert.deepStrictEqual(mergeAccountsSafely(null, diskAccounts), []);
  assert.deepStrictEqual(mergeAccountsSafely(undefined, diskAccounts), []);
  assert.deepStrictEqual(mergeAccountsSafely([], diskAccounts), []);
  assert.deepStrictEqual(mergeAccountsSafely(incomingWithoutLastUsed, null), incomingWithoutLastUsed);
});
