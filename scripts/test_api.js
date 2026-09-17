const fs = require('fs');
const https = require('https');
const crypto = require('crypto');
const os = require('os');

const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
const d = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
const a = JSON.parse(d['iCubeAuthInfo://icube.cloudide']);
const token = a.token;
const userId = a.userId;

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

function genId() {
  return crypto.randomBytes(12).toString('hex');
}

async function main() {
  // Step 1: Dump the FULL get_detail_param response for chat function
  const chatRes = await post('/api/ide/v1/get_detail_param', {
    function: 'chat',
    need_prompt: true,
    poly_prompt: true
  });
  const chatParams = JSON.parse(chatRes.body);
  
  // Dump top-level keys
  console.log('=== get_detail_param (chat) top-level keys ===');
  console.log(Object.keys(chatParams));
  
  // Dump poly_prompt_config
  if (chatParams.poly_prompt_config) {
    const ppc = JSON.stringify(chatParams.poly_prompt_config);
    console.log('\npoly_prompt_config length:', ppc.length);
    console.log('poly_prompt_config keys:', Object.keys(chatParams.poly_prompt_config));
    // Dump first 2000 chars
    console.log(ppc.substring(0, 2000));
  }
  
  // Dump prompt_config
  if (chatParams.prompt_config) {
    console.log('\nprompt_config:', JSON.stringify(chatParams.prompt_config).substring(0, 1000));
  }
  
  // Show models for chat
  console.log('\n=== Models for chat ===');
  for (const m of chatParams.config_info_list) {
    const mdl = m.model_detail_list?.[0];
    console.log(`  ${m.config_name} source=${m.config_source} switch=${m.config_switch} model=${mdl?.model_name}`);
  }
  
  // Dump model_detail_list for first model
  const firstModel = chatParams.config_info_list[0];
  console.log('\n=== First model detail ===');
  console.log(JSON.stringify(firstModel, null, 2).substring(0, 2000));
  
  // Step 2: Try builder with chat function's model (doubao-for-auto)
  // builder passed model config check before
  const model = chatParams.config_info_list.find(m => m.config_name === 'doubao-for-auto') || chatParams.config_info_list[0];
  const enc = model.model_detail_list[0].encrypted_model_params;
  
  // Try builder with additional prompt/template fields
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
      content: 'say hello in one word',
      content_type: 1,
      prompt: 'say hello in one word',
      type: 'user_input'
    },
    system_prompt: 'You are a helpful assistant.',
    prompt: 'say hello in one word',
    messages: [{ role: 'user', content: 'say hello in one word' }],
    history_ids: [],
    tools: [],
    config_source: model.config_source,
    prompt_template: chatParams.poly_prompt_config || {}
  };
  
  console.log('\n--- Test 1: chat/chat with prompt_template ---');
  const r1 = await post('/api/agent/v3/create_agent_task', body);
  console.log('Status:', r1.status);
  console.log('Response:', r1.body.substring(0, 2000));

  // Test 2: builder/builder (which reached task_created before)
  body.function = 'builder';
  body.agent_type = 'builder';
  body.conversation_id = genId();
  body.session_id = genId();
  body.task_id = genId();
  body.message_id = genId();
  console.log('\n--- Test 2: builder/builder with prompt fields ---');
  const r2 = await post('/api/agent/v3/create_agent_task', body);
  console.log('Status:', r2.status);
  console.log('Response:', r2.body.substring(0, 2000));
  
  // Test 3: Try llm_raw_chat v2 with more complete body
  const rawBody = {
    model: model.config_name,
    config_name: model.config_name,
    encrypted_model_params: enc,
    messages: [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: 'say hello in one word' }
    ],
    stream: true,
    function: 'chat',
    user_id: userId,
    device_id: '2262131830954826',
    max_tokens: 1024,
    temperature: 0.7
  };
  console.log('\n--- Test 3: llm_raw_chat v2 ---');
  const r3 = await post('/api/ide/v2/llm_raw_chat', rawBody);
  console.log('Status:', r3.status);
  console.log('Headers:', JSON.stringify(r3.headers || {}));
  console.log('Response:', r3.body.substring(0, 2000));
  
  // Test 4: llm_raw_chat v1 with same body
  console.log('\n--- Test 4: llm_raw_chat v1 ---');
  const r4 = await post('/api/ide/v1/llm_raw_chat', rawBody);
  console.log('Status:', r4.status);
  console.log('Response:', r4.body.substring(0, 2000));
}

main().catch(console.error);
