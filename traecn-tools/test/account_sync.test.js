const test = require('node:test');
const assert = require('node:assert/strict');
const { mergeAccountsSafely } = require('../electron/main.js');

test('AntiCase 2: UI save must not overwrite fresh tokens updated by main process', () => {
  const diskAccounts = [
    {
      id: 'acc-1',
      label: 'Work Account',
      token: 'token_v2_refreshed',
      refreshToken: 'refresh_v2_refreshed',
      expiredAt: '2026-10-01T00:00:00Z',
      lastUsed: '2026-09-20T12:00:00Z',
    },
  ];

  // UI 传递的旧快照（持有旧 Token，但修改了 label）
  const incomingUIAccounts = [
    {
      id: 'acc-1',
      label: 'Work Account (Renamed)',
      token: 'token_v1_stale',
      refreshToken: 'refresh_v1_stale',
      expiredAt: '2026-09-20T10:00:00Z',
      lastUsed: '2026-09-20T11:00:00Z',
    },
  ];

  const merged = mergeAccountsSafely(incomingUIAccounts, diskAccounts);

  assert.strictEqual(merged.length, 1);
  assert.strictEqual(merged[0].label, 'Work Account (Renamed)', 'UI 属性应成功更新');
  assert.strictEqual(merged[0].token, 'token_v2_refreshed', '最新 Token 不应被 UI 旧值覆盖');
  assert.strictEqual(merged[0].refreshToken, 'refresh_v2_refreshed', '最新 RefreshToken 不应被覆盖');
  assert.strictEqual(merged[0].expiredAt, '2026-10-01T00:00:00Z', '最新过期时间应被保留');
});

test('AntiCase 2.1: UI save adds new accounts smoothly without interference', () => {
  const diskAccounts = [
    {
      id: 'acc-1',
      label: 'Existing Account',
      token: 'token_existing',
      lastUsed: '2026-09-20T12:00:00Z',
    },
  ];

  const incomingUIAccounts = [
    {
      id: 'acc-1',
      label: 'Existing Account',
      token: 'token_existing',
      lastUsed: '2026-09-20T12:00:00Z',
    },
    {
      id: 'acc-new',
      label: 'New Account',
      token: 'token_new',
      lastUsed: '2026-09-20T13:00:00Z',
    },
  ];

  const merged = mergeAccountsSafely(incomingUIAccounts, diskAccounts);

  assert.strictEqual(merged.length, 2);
  assert.strictEqual(merged[1].id, 'acc-new');
  assert.strictEqual(merged[1].token, 'token_new');
});
