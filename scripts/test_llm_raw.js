// Enumerate llm_raw_chat required fields
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));

function send(body, label) {
  const bodyStr = JSON.stringify(body);
  return new Promise(r => {
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru',
      path: '/api/ide/v2/llm_raw_chat',
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        'x-app-id': h['x-app-id'],
        'x-ide-token': h['x-ide-token'],
        'x-device-id': h['x-device-id'],
        'x-ide-version': h['x-ide-version'],
        'x-ide-version-code': h['x-ide-version-code'],
        'x-machine-id': h['x-machine-id'],
        'package-type': 'stable_cn',
        'user-agent': h['user-agent'],
      },
    }, res => {
      let d = '';
      res.on('data', c => d += c);
      res.on('end', () => {
        console.log(`[${label}] ${res.statusCode}: ${d.substring(0, 500)}`);
        r({ status: res.statusCode, data: d });
      });
    });
    req.on('error', e => { console.log(`[${label}] Error:`, e.message); r({ status: 0, data: '' }); });
    req.write(bodyStr);
    req.end();
  });
}

(async () => {
  // The empty body works (200 + SSE error). Bodies with "messages" fail (400).
  // Let's find which fields cause 400 vs 200.
  
  // Try each field alone to see which causes 400
  const testFields = [
    ['model', 'test'],
    ['model_name', 'test'],
    ['stream', true],
    ['messages', []],
    ['content', 'hello'],
    ['prompt', 'hello'],
    ['input', 'hello'],
    ['text', 'hello'],
    ['query', 'hello'],
    ['user_input', 'hello'],
    ['user_input', { content: 'hello' }],
    ['config_name', 'minimax-m2.5'],
    ['function', 'chat_v3'],
    ['version_code', 20260212],
    ['user_id', '4355622541471866'],
    ['device_id', '2262131830954826'],
    ['max_tokens', 1024],
    ['temperature', 0.7],
    ['encrypted_model_params', 'test'],
  ];

  console.log('--- Testing individual fields ---');
  for (const [key, val] of testFields) {
    const body = {};
    body[key] = val;
    const { status } = await send(body, key + '=' + JSON.stringify(val).substring(0, 30));
  }

  console.log('\n--- Testing known-good fields from create_agent_task ---');
  // Now test with fields we know work for create_agent_task
  const base = {
    function: "chat_v3",
    model_name: "minimax-m2.5__dev",
    config_name: "minimax-m2.5",
    version_code: 20260212,
    user_id: "4355622541471866",
    device_id: "2262131830954826",
    ide_version: "3.3.37",
  };
  
  const r1 = await send(base, 'base_fields');
  
  // Add encrypted_model_params from detail
  const detail = JSON.parse(fs.readFileSync(__dirname + '/detail_param_chat_v3.json', 'utf8'));
  const mmConfig = detail.config_info_list.find(c => c.config_name === 'minimax-m2.5');
  const emp = mmConfig.model_detail_list[0].encrypted_model_params;
  
  const r2 = await send({
    ...base,
    encrypted_model_params: emp,
  }, 'base+encrypted_params');

  // Try with prompt field 
  const r3 = await send({
    ...base,
    encrypted_model_params: emp,
    prompt: "You are a helpful assistant.\n\nUser: hello\n\nAssistant:",
  }, 'base+prompt');

  // Try with request_messages (maybe the field name)
  const r4 = await send({
    ...base,
    encrypted_model_params: emp,
    request_messages: [{ role: "user", content: "hello" }],
  }, 'base+request_messages');

  // Try chat_messages
  const r5 = await send({
    ...base,
    encrypted_model_params: emp,
    chat_messages: [{ role: "user", content: "hello" }],
  }, 'base+chat_messages');

  setTimeout(() => process.exit(0), 2000);
})();
