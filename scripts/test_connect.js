// Test the connect endpoint and then create_agent_task with the session
const fs = require('fs');
const https = require('https');
const crypto = require('crypto');
const os = require('os');

const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
const d = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
const a = JSON.parse(d['iCubeAuthInfo://icube.cloudide']);
const token = a.token;
const userId = a.userId;

function genId() { return crypto.randomBytes(12).toString('hex'); }

const defaultHeaders = {
  'Content-Type': 'application/json',
  'X-IDE-Token': token,
  'x-app-id': '6eefa01c-1036-4c7e-9ca5-d891f63bfcd8',
  'x-app-version': 'default',
  'x-ide-version-code': '20260212',
  'x-app-version-code': '20260212',
  'x-device-brand': os.hostname(),
  'x-device-cpu': 'Intel',
  'x-device-id': '2262131830954826',
  'x-machine-id': crypto.createHash('sha256').update(os.hostname()).digest('hex'),
  'x-os-version': 'Windows',
  'x-device-type': 'windows',
  'x-ide-version': '3.3.37',
  'x-ide-version-type': 'stable',
  'request-traffic-type': 'prod'
};

function post(path, body) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    const headers = {
      ...defaultHeaders,
      'Content-Length': Buffer.byteLength(bodyStr),
      'X-Request-ID': reqId,
      'X-Trae-Request-ID': reqId
    };
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru', path, method: 'POST', headers
    }, res => {
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks).toString() }));
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

function postSSE(path, body) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    const headers = {
      ...defaultHeaders,
      'Content-Length': Buffer.byteLength(bodyStr),
      'Accept': 'text/event-stream',
      'X-Request-ID': reqId,
      'X-Trae-Request-ID': reqId
    };
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru', path, method: 'POST', headers
    }, res => {
      let data = '';
      const events = [];
      res.on('data', c => {
        data += c.toString();
        const lines = data.split('\n');
        let currentEvent = null;
        for (let i = 0; i < lines.length - 1; i++) {
          const line = lines[i].trim();
          if (line.startsWith('event:')) currentEvent = line.substring(6).trim();
          else if (line.startsWith('data:') && currentEvent) {
            events.push({ event: currentEvent, data: line.substring(5).trim() });
          }
        }
        data = lines[lines.length - 1];
      });
      res.on('end', () => resolve({ status: res.statusCode, events }));
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

async function main() {
  // Test 1: Connect
  console.log('=== Test 1: /api/ide/v1/connect ===');
  const sessionId = crypto.randomUUID();
  const r1 = await post('/api/ide/v1/connect', {
    session_id: sessionId,
    user_id: userId,
    device_id: '2262131830954826',
    app_id: '6eefa01c-1036-4c7e-9ca5-d891f63bfcd8',
    version_code: '20260212'
  });
  console.log('Status:', r1.status);
  console.log('Response:', r1.body.substring(0, 500));
  
  // Test 2: Connect with empty body
  console.log('\n=== Test 2: /api/ide/v1/connect (empty) ===');
  const r2 = await post('/api/ide/v1/connect', {});
  console.log('Status:', r2.status);
  console.log('Response:', r2.body.substring(0, 500));
  
  // Test 3: Ping
  console.log('\n=== Test 3: /api/ide/v1/ping ===');
  const r3 = await post('/api/ide/v1/ping', {});
  console.log('Status:', r3.status);
  console.log('Response:', r3.body.substring(0, 500));

  // Test 4: chat_prompt endpoint
  console.log('\n=== Test 4: /api/ide/v1/chat_prompt ===');
  const chatParams = JSON.parse(fs.readFileSync('scripts/detail_param_chat.json', 'utf8'));
  const model = chatParams.config_info_list.find(m => m.config_name === 'deepseek-V3.1') || chatParams.config_info_list[0];
  const r4 = await post('/api/ide/v1/chat_prompt', {
    config_name: model.config_name,
    function: 'chat',
    agent_type: 'chat',
    user_input: 'say hello',
    encrypted_model_params: model.model_detail_list[0].encrypted_model_params
  });
  console.log('Status:', r4.status);
  console.log('Response:', r4.body.substring(0, 500));
  
  // Test 5: model_list
  console.log('\n=== Test 5: /api/ide/v1/model_list ===');
  const r5 = await post('/api/ide/v1/model_list', { function: 'chat' });
  console.log('Status:', r5.status);
  console.log('Response:', r5.body.substring(0, 500));
  
  // Test 6: get_model_list  
  console.log('\n=== Test 6: /api/ide/v1/get_model_list ===');
  const r6 = await post('/api/ide/v1/get_model_list', { function: 'chat' });
  console.log('Status:', r6.status);
  console.log('Response:', r6.body.substring(0, 500));
}

main().catch(console.error);
