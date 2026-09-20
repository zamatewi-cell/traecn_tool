const test = require('node:test');
const assert = require('node:assert/strict');
const { EventEmitter } = require('node:events');
const { terminateProxyProcess, setProxyProcess, getProxyProcess } = require('../electron/main.js');

test('AntiCase 6: terminateProxyProcess must return failure on timeout and retain handle', async () => {
  const stubbornProc = new EventEmitter();
  stubbornProc.pid = 99999;
  stubbornProc.kill = () => false; // 拒绝退出且不触发 exit 事件

  setProxyProcess(stubbornProc);

  // 传入 100ms 短超时限制
  const result = await terminateProxyProcess(stubbornProc, 100);

  assert.strictEqual(result.success, false, '超时未退出的进程严禁返回 success: true');
  assert.match(result.error, /超时|timeout/i, '错误信息必须明确提示超时');
  assert.strictEqual(getProxyProcess(), stubbornProc, '未退出的子进程句柄必须保留，严禁置为 null');
});

test('AntiCase 6.1: terminateProxyProcess succeeds when process exits normally', async () => {
  const normalProc = new EventEmitter();
  normalProc.pid = 88888;
  normalProc.kill = () => {
    setTimeout(() => normalProc.emit('exit', 0), 20);
    return true;
  };

  setProxyProcess(normalProc);

  const result = await terminateProxyProcess(normalProc, 1000);

  assert.strictEqual(result.success, true, '正常退出的进程应返回 success: true');
  assert.strictEqual(getProxyProcess(), null, '正常退出的子进程句柄应被清理为 null');
});
