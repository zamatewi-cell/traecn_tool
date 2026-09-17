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

function post(path, body) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru', path, method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(bodyStr),
        'X-IDE-Token': token,
        'X-Request-ID': reqId,
        'X-Trae-Request-ID': reqId,
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
      }
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

async function main() {
  const chatParams = JSON.parse(fs.readFileSync('scripts/detail_param_chat.json', 'utf8'));
  const model = chatParams.config_info_list.find(m => m.config_name === 'deepseek-V3.1');
  const enc = model.model_detail_list[0].encrypted_model_params;
  const prompts = chatParams.metadata.encrypted_prompt_set;
  
  // Test 1: llm_raw_chat_prompt endpoint
  console.log('=== Test 1: /api/ide/v1/llm_raw_chat_prompt ===');
  const r1 = await post('/api/ide/v1/llm_raw_chat_prompt', {
    config_name: model.config_name,
    encrypted_model_params: enc,
    function: 'chat',
    messages: [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: 'say hello' }
    ],
    stream: true
  });
  console.log('Status:', r1.status);
  console.log('Response (300):', r1.body.substring(0, 300));
  
  // Test 2: chat_prompt endpoint
  console.log('\n=== Test 2: /api/ide/v1/chat_prompt ===');
  const r2 = await post('/api/ide/v1/chat_prompt', {
    config_name: model.config_name,
    encrypted_model_params: enc,
    function: 'chat',
    messages: [
      { role: 'user', content: 'say hello' }
    ],
    stream: true,
    prompt: 'say hello'
  });
  console.log('Status:', r2.status);
  console.log('Response (300):', r2.body.substring(0, 300));
  
  // Test 3: llm_raw_chat v2 with prompt_set
  console.log('\n=== Test 3: /api/ide/v2/llm_raw_chat with prompt_set ===');
  const r3 = await post('/api/ide/v2/llm_raw_chat', {
    config_name: model.config_name,
    encrypted_model_params: enc,
    function: 'chat',
    prompt_set: prompts,
    messages: [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: 'say hello' }
    ],
    stream: true,
    user_id: userId,
    device_id: '2262131830954826'
  });
  console.log('Status:', r3.status);
  console.log('Response (500):', r3.body.substring(0, 500));
  
  // Test 4: llm_raw_chat v1 with prompt_set and more fields
  console.log('\n=== Test 4: /api/ide/v1/llm_raw_chat with all fields ===');
  const r4 = await post('/api/ide/v1/llm_raw_chat', {
    config_name: model.config_name,
    model_name: model.config_name,
    encrypted_model_params: enc,
    function: 'chat',
    prompt_set: prompts,
    messages: [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: 'say hello' }
    ],
    stream: true,
    user_id: userId,
    device_id: '2262131830954826',
    session_id: genId(),
    conversation_id: genId()
  });
  console.log('Status:', r4.status);
  console.log('Response (500):', r4.body.substring(0, 500));
  
  // Test 5: llm_raw_chat v1 with minimal structured body (matching Go struct pattern)
  console.log('\n=== Test 5: /api/ide/v1/llm_raw_chat minimal ===');
  const r5 = await post('/api/ide/v1/llm_raw_chat', {
    config_name: model.config_name,
    encrypted_model_params: enc,
    messages: [
      { role: 'user', content: 'say hello' }
    ],
    stream: true
  });
  console.log('Status:', r5.status);
  console.log('Response (500):', r5.body.substring(0, 500));
}

main().catch(console.error);
