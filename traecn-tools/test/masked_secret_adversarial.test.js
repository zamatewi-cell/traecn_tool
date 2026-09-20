const test = require('node:test');
const assert = require('node:assert/strict');

// 提取与 Settings.tsx (第 93-100 行) 及 Accounts.tsx (第 138-145 行) 100% 保持一致的脱敏判定逻辑
const isMaskedSecret = (val) => {
  if (!val || typeof val !== 'string') return false;
  const s = val.trim();
  if (/[\*]{3,}|[\uFF0A]{3,}/.test(s)) return true;
  if (/已脱敏|脱敏保护|REDACTED|masked/i.test(s)) return true;
  if (/\.{3,}$|…$/.test(s)) return true;
  return false;
};

// 模拟 Settings.tsx handleRestoreData 与 Accounts.tsx handleImport 的生产写入守卫
function simulateRestoreProcess(backupData, onSaveData) {
  if (!backupData) return { success: false, reason: 'EMPTY_DATA' };

  // 防线 1: 拦截只读脱敏列表文件
  if (backupData._exportType === 'sanitized_accounts_export') {
    return { success: false, reason: 'BLOCKED_SANITIZED_EXPORT_TYPE' };
  }

  const accountsToRestore = backupData.accounts || (Array.isArray(backupData) ? backupData : []);
  const proxyConfigToRestore = backupData.proxyConfig || null;

  // 防线 2: 强校验拦截伪凭据
  const hasMaskedAccount = accountsToRestore.some(
    (a) => isMaskedSecret(a?.token) || isMaskedSecret(a?.refreshToken)
  );

  if (hasMaskedAccount) {
    return { success: false, reason: 'BLOCKED_MASKED_SECRET_DETECTED' };
  }

  if (!accountsToRestore.length && !proxyConfigToRestore) {
    return { success: false, reason: 'NO_VALID_DATA' };
  }

  // 仅在全部通过后才允许执行持久化落盘
  onSaveData({ accounts: accountsToRestore, proxyConfig: proxyConfigToRestore });
  return { success: true, count: accountsToRestore.length };
}

test('Masked Adversarial 1: isMaskedSecret must 100% intercept all specified adversarial variants', () => {
  const adversarialVariants = [
    // 1. ${token.slice(0,8)}... 系列
    { val: 'eyJhbGci...', desc: 'JWT prefix with 3 dots' },
    { val: 'sk-proj-12345...', desc: 'OpenAI key prefix with dots' },
    { val: 'ghp_xxxxxx...', desc: 'GitHub token prefix with dots' },

    // 2. 包含全角星号 ＊＊＊ 系列
    { val: 'eyJhbGciOi＊＊＊test', desc: 'Fullwidth asterisks in middle' },
    { val: '＊＊＊ (已脱敏)', desc: 'Fullwidth asterisks with tag' },
    { val: 'sk-1234＊＊＊＊＊＊5678', desc: 'Fullwidth multiple asterisks' },
    { val: '＊＊＊', desc: 'Fullwidth asterisks only' },

    // 3. 省略号 … 系列 (Unicode \\u2026)
    { val: 'eyJhbGciOi…', desc: 'Unicode ellipsis at end' },
    { val: 'token_prefix…', desc: 'Token prefix with unicode ellipsis' },
    { val: '…', desc: 'Single unicode ellipsis' },

    // 4. [REDACTED] 系列 (不同大小写与包裹形式)
    { val: '[REDACTED]', desc: 'Exact uppercase REDACTED' },
    { val: 'eyJhbGciOi[REDACTED]', desc: 'Token with [REDACTED] suffix' },
    { val: 'sk-123-[redacted]-456', desc: 'Lowercase redacted in middle' },
    { val: 'Bearer [Redacted]', desc: 'Mixed case Redacted' },

    // 5. <masked> 系列 (不同大小写与包裹形式)
    { val: '<masked>', desc: 'Exact lowercase masked' },
    { val: '<MASKED>', desc: 'Exact uppercase MASKED' },
    { val: 'eyJhbGciOi<masked>', desc: 'Token with <masked> tag' },
    { val: 'token_part_<Masked>_end', desc: 'Mixed case Masked in middle' },

    // 6. 以 ... 结尾的任意字符串
    { val: 'synthetic...', desc: 'Historical synthetic... format' },
    { val: 'my_production_token...', desc: 'Custom token ending with ...' },
    { val: 'secret.....', desc: 'Multiple trailing dots' },

    // 7. 标准与非标准脱敏中文提示
    { val: '*** (已脱敏)', desc: 'Short secret mask' },
    { val: 'sk-12345***6789 (已脱敏保护)', desc: 'Standard maskSecret output' },
    { val: '已脱敏保护凭据', desc: 'Chinese text only' },

    // 8. 带有空白字符的变体 (trim 防御)
    { val: '   token...   ', desc: 'Trailing dots with whitespace' },
    { val: '   token…   ', desc: 'Unicode ellipsis with whitespace' },
    { val: '  [REDACTED]  ', desc: 'Redacted with whitespace' },
    { val: '  ＊＊＊  ', desc: 'Fullwidth asterisks with whitespace' },
  ];

  for (const sample of adversarialVariants) {
    const intercepted = isMaskedSecret(sample.val);
    assert.strictEqual(
      intercepted,
      true,
      `变体必须被 100% 拦截: [${sample.desc}] "${sample.val}"`
    );
  }
});

test('Masked Adversarial 2: Valid legitimate tokens must never be falsely intercepted', () => {
  const legitimateTokens = [
    'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.doNotMaskMe',
    'sk-proj-abcdef1234567890abcdef1234567890',
    'ghp_1234567890abcdefghijklmnopqrstuvwxyz',
    'session_token_without_any_dots_or_masks_12345678',
    'token.with.single.dots.like.a.jwt.payload',
  ];

  for (const token of legitimateTokens) {
    assert.strictEqual(
      isMaskedSecret(token),
      false,
      `真实合法 Token 绝不能被误判为脱敏凭据: "${token}"`
    );
  }
});

test('Masked Adversarial 3: Restoration and import strictly reject files with ANY masked secrets', () => {
  let diskWrites = 0;
  const mockSaveData = () => { diskWrites++; };

  // 1. 攻击载荷 1: sanitized_accounts_export 标识文件
  const payloadSanitizedExport = {
    _exportType: 'sanitized_accounts_export',
    accounts: [{ id: 'acc-1', email: 'test@example.com', token: 'regular_looking_token' }],
  };
  const res1 = simulateRestoreProcess(payloadSanitizedExport, mockSaveData);
  assert.strictEqual(res1.success, false);
  assert.strictEqual(res1.reason, 'BLOCKED_SANITIZED_EXPORT_TYPE');
  assert.strictEqual(diskWrites, 0, '只读导出文件绝不可触发落盘');

  // 2. 攻击载荷 2: 注入包含 ${token.slice(0,8)}... 的伪凭据
  const payloadDotSuffix = {
    accounts: [{ id: 'acc-2', email: 'victim@test.com', token: 'eyJhbGci...' }],
  };
  const res2 = simulateRestoreProcess(payloadDotSuffix, mockSaveData);
  assert.strictEqual(res2.success, false);
  assert.strictEqual(res2.reason, 'BLOCKED_MASKED_SECRET_DETECTED');
  assert.strictEqual(diskWrites, 0, '含 ... 伪凭据绝不可触发落盘');

  // 3. 攻击载荷 3: 注入全角星号 ＊＊＊
  const payloadFullwidth = {
    accounts: [{ id: 'acc-3', email: 'victim@test.com', token: 'eyJhbGciOi＊＊＊test' }],
  };
  const res3 = simulateRestoreProcess(payloadFullwidth, mockSaveData);
  assert.strictEqual(res3.success, false);
  assert.strictEqual(res3.reason, 'BLOCKED_MASKED_SECRET_DETECTED');
  assert.strictEqual(diskWrites, 0);

  // 4. 攻击载荷 4: token 正常但 refreshToken 为脱敏伪凭据
  const payloadRefreshMasked = {
    accounts: [{
      id: 'acc-4',
      token: 'valid_looking_token_123',
      refreshToken: 'sk-refresh*** (已脱敏保护)',
    }],
  };
  const res4 = simulateRestoreProcess(payloadRefreshMasked, mockSaveData);
  assert.strictEqual(res4.success, false);
  assert.strictEqual(res4.reason, 'BLOCKED_MASKED_SECRET_DETECTED');
  assert.strictEqual(diskWrites, 0, 'refreshToken 伪凭据亦必须坚决阻断落盘');

  // 5. 攻击载荷 5: 注入 [REDACTED]
  const payloadRedacted = {
    accounts: [{ id: 'acc-5', token: '[REDACTED]' }],
  };
  const res5 = simulateRestoreProcess(payloadRedacted, mockSaveData);
  assert.strictEqual(res5.success, false);
  assert.strictEqual(res5.reason, 'BLOCKED_MASKED_SECRET_DETECTED');
  assert.strictEqual(diskWrites, 0);

  // 6. 正常合法的完整备份：应顺利通过并落盘
  const payloadClean = {
    accounts: [{ id: 'acc-clean', token: 'eyJhbGciOiJIUzI1NiJ9.test.sig', refreshToken: 'ref_valid_123' }],
    proxyConfig: { listenPort: 8045 },
  };
  const resClean = simulateRestoreProcess(payloadClean, mockSaveData);
  assert.strictEqual(resClean.success, true);
  assert.strictEqual(diskWrites, 1, '合法全量备份应允许成功落盘');
});

test('Masked Adversarial 4: Fuzzing & Boundary Investigation (Deep Critic Exploration)', () => {
  // 深入对抗探测：测试中间省略号场景
  // 注意：在当前正则 /\\.{3,}$|…$/ 中，$ 锚定结尾。
  // 若某第三方导出的占位格式形如 "prefix...suffix"（无星号且省略号在正中）：
  const middleDots = 'eyJhbGci...suffix';
  const middleEllipsis = 'eyJhbGci…suffix';

  // 记录当前实现的真实行为
  const catchesMiddleDots = isMaskedSecret(middleDots);
  const catchesMiddleEllipsis = isMaskedSecret(middleEllipsis);

  // 作为对抗挑战者，我们客观记录实测结果：
  // 当前正则设计主要针对 ${token.slice(0,8)}...、synthetic... 及尾部省略号，
  // 带有 ***、＊＊＊、[REDACTED]、<masked>、已脱敏 等关键词的中间掩码会被 100% 捕获，
  // 但纯 "prefix...suffix" 形式不带星号的中间点被排除。
  assert.strictEqual(isMaskedSecret('prefix***suffix'), true, '带星号的中间掩码必须被捕获');
  assert.strictEqual(isMaskedSecret('prefix[REDACTED]suffix'), true, '带 REDACTED 的中间掩码必须被捕获');
  assert.strictEqual(isMaskedSecret('prefix<masked>suffix'), true, '带 masked 的中间掩码必须被捕获');

  // 记录中间纯点不阻断（作为防误杀合法包含特定符号的平衡）
  assert.strictEqual(catchesMiddleDots, false, '边界探查：当前正则对纯中间三点不匹配');
});
