const fs = require('fs');
const { ZmqClient } = require('D:/Trae CN/resources/app/node_modules/@byted-icube/trae-network-client');

const ep = process.argv[2];
if (!ep) {
  console.error('usage: node probe_zmq_request.js <endpoint>');
  process.exit(2);
}

const headersPath = 'd:/codelearn/vscode/reverse_proxy/scripts/captured/agent_task_req_headers.json';
const capturedHeaders = JSON.parse(fs.readFileSync(headersPath, 'utf8'));

const headers = {
  'accept': 'application/json',
  'content-type': 'application/json',
  'x-ide-token': capturedHeaders['x-ide-token'] || '',
  'x-device-id': capturedHeaders['x-device-id'] || '',
  'x-machine-id': capturedHeaders['x-machine-id'] || '',
  'x-ide-version': capturedHeaders['x-ide-version'] || '3.3.37',
  'x-ide-version-code': capturedHeaders['x-ide-version-code'] || '20260212',
  'x-ide-version-type': capturedHeaders['x-ide-version-type'] || 'stable',
};

async function withTimeout(promise, ms) {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error(`timeout after ${ms}ms`)), ms)),
  ]);
}

(async () => {
  console.log(`REQUEST_PROBE ${ep}`);
  const client = new ZmqClient(ep);

  try {
    const resp = await withTimeout(
      client.request({
        method: 'POST',
        url: 'https://trae-api-cn.mchost.guru/api/ide/v2/features',
        headers,
        body: '{}',
        timeoutMs: 10000,
      }),
      12000,
    );

    console.log('OK request returned');
    console.log('type:', typeof resp);
    if (resp && typeof resp === 'object') {
      console.log('keys:', Object.keys(resp));
      if ('statusCode' in resp) console.log('statusCode:', resp.statusCode);
      if ('status' in resp) console.log('status:', resp.status);
      if ('errorCode' in resp) console.log('errorCode:', resp.errorCode);
      if ('errorMessage' in resp) console.log('errorMessage:', resp.errorMessage);
    } else {
      console.log('resp:', String(resp));
    }
    process.exit(0);
  } catch (err) {
    console.log('FAIL', err && err.message ? err.message : String(err));
    process.exit(1);
  }
})();
