const fs = require('fs');
const https = require('https');
const crypto = require('crypto');
const os = require('os');

const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
const d = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
const a = JSON.parse(d['iCubeAuthInfo://icube.cloudide']);
const token = a.token;
const userId = a.userId;

function genId() {
  return crypto.randomBytes(12).toString('hex');
}

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
  // Load cached detail params
  const chatParams = JSON.parse(fs.readFileSync('scripts/detail_param_chat.json', 'utf8'));
  const soloParams = JSON.parse(fs.readFileSync('scripts/detail_param_solo_coder.json', 'utf8'));
  
  // Use a real chat model
  const model = chatParams.config_info_list.find(m => m.config_name === 'deepseek-V3.1') 
    || chatParams.config_info_list.find(m => m.config_name === 'doubao-for-auto');
  const enc = model.model_detail_list[0].encrypted_model_params;
  console.log('Using model:', model.config_name);
  
  // The encrypted prompt set from the metadata
  const prompts = chatParams.metadata.encrypted_prompt_set;
  console.log('Number of prompt templates:', prompts.length);
  
  // Build the request with encrypted_prompt_set included
  const body = {
    conversation_id: genId(),
    session_id: genId(),
    task_id: genId(),
    message_id: genId(),
    user_id: userId,
    device_id: '2262131830954826',
    model_name: model.config_name,
    config_name: model.config_name,
    encrypted_model_params: enc,
    stream: true,
    function: 'chat',
    agent_type: 'chat',
    ide_version: '3.3.37',
    user_input: {
      id: genId(),
      content: 'say hello in one word'
    },
    encrypted_prompt_set: prompts
  };
  
  console.log('Body size:', JSON.stringify(body).length, 'bytes');
  console.log('\n--- Test 1: chat with encrypted_prompt_set ---');
  const r1 = await post('/api/agent/v3/create_agent_task', body);
  console.log('Status:', r1.status);
  console.log('Response (first 2000):', r1.body.substring(0, 2000));
  
  // Test 2: Try with prompt_set field name
  delete body.encrypted_prompt_set;
  body.prompt_set = prompts;
  body.conversation_id = genId();
  body.session_id = genId();
  body.task_id = genId();
  body.message_id = genId();
  console.log('\n--- Test 2: chat with prompt_set ---');
  const r2 = await post('/api/agent/v3/create_agent_task', body);
  console.log('Status:', r2.status);
  console.log('Response (first 2000):', r2.body.substring(0, 2000));
  
  // Test 3: builder_with_mcp with chat prompts
  delete body.prompt_set;
  body.encrypted_prompt_set = prompts;
  body.agent_type = 'builder_with_mcp';
  body.conversation_id = genId();
  body.session_id = genId();
  body.task_id = genId();
  body.message_id = genId();
  console.log('\n--- Test 3: builder_with_mcp with encrypted_prompt_set ---');
  const r3 = await post('/api/agent/v3/create_agent_task', body);
  console.log('Status:', r3.status);
  console.log('Response (first 2000):', r3.body.substring(0, 2000));
}

main().catch(console.error);
