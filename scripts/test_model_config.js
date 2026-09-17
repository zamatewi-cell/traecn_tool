// Test different approaches to fix "model config is empty" error
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));
const detail = JSON.parse(fs.readFileSync(__dirname + '/detail_param_chat_v3.json', 'utf8'));

// Find minimax-m2.5 config from detail_param
const configList = detail.config_info_list;
const mmConfig = configList.find(c => c.config_name === 'minimax-m2.5');
const modelDetail = mmConfig.model_detail_list[0];

console.log('Config name:', mmConfig.config_name);
console.log('Model name in detail:', modelDetail.model_name);
console.log('Has encrypted_model_params:', !!modelDetail.encrypted_model_params);
console.log('Prompt config list:', modelDetail.prompt_config_list.map(p => p.prompt_key));

function send(body, label) {
  const bodyStr = JSON.stringify(body);
  return new Promise(r => {
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru',
      path: '/api/agent/v3/create_agent_task',
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
      res.on('data', c => { d += c; });
      res.on('end', () => {
        console.log(`\n[${label}] Status: ${res.statusCode}`);
        const lines = d.split('\n');
        for (const line of lines) {
          if (line.trim()) console.log(' ', line.substring(0, 300));
        }
        r(d);
      });
    });
    req.on('error', e => { console.log(`[${label}] Error:`, e.message); r(''); });
    req.write(bodyStr);
    req.end();
  });
}

function makeBody(overrides = {}) {
  return {
    function: "chat_v3",
    agent_type: "builder_v3",
    model_name: "minimax-m2.5",
    config_name: "minimax-m2.5",
    session_id: crypto.randomBytes(12).toString('hex'),
    conversation_id: crypto.randomBytes(12).toString('hex'),
    task_id: crypto.randomUUID(),
    version_code: 20260212,
    user_id: "4355622541471866",
    device_id: "2262131830954826",
    ide_version: "3.3.37",
    ide_version_code: "20260212",
    ide_version_type: "stable",
    app_id: "6eefa01c-1036-4c7e-9ca5-d891f63bfcd8",
    machine_id: h['x-machine-id'],
    os_version: "Windows 11",
    user_input: {
      id: crypto.randomUUID(),
      content: "hello, what is 1+1?",
      type: "text",
    },
    prompt_template_map: {
      "master_agent": "You are a helpful AI assistant. Answer the user's question concisely.\n\nUser: {{user_input}}",
    },
    history_list: [],
    settings: {},
    ...overrides,
  };
}

(async () => {
  // Test 1: Include encrypted_model_params directly
  console.log('\n=== Test 1: encrypted_model_params in body ===');
  await send(makeBody({
    encrypted_model_params: modelDetail.encrypted_model_params,
  }), 'Test1-encrypted_model_params');

  // Test 2: Include model_config with full model detail
  console.log('\n=== Test 2: model_config object ===');
  await send(makeBody({
    model_config: {
      model_name: modelDetail.model_name,
      encrypted_model_params: modelDetail.encrypted_model_params,
      max_tokens: modelDetail.max_tokens,
      prompt_max_tokens: modelDetail.prompt_max_tokens,
    },
  }), 'Test2-model_config');

  // Test 3: Change model_name to the __dev variant
  console.log('\n=== Test 3: model_name = minimax-m2.5__dev ===');
  await send(makeBody({
    model_name: modelDetail.model_name,  // minimax-m2.5__dev
  }), 'Test3-dev_model_name');

  // Test 4: Include function_config with config_info_list entry
  console.log('\n=== Test 4: function_config with full config ===');
  await send(makeBody({
    function_config: {
      config_name: mmConfig.config_name,
      config_source: mmConfig.config_source,
      model_detail_list: mmConfig.model_detail_list,
    },
  }), 'Test4-function_config');

  // Test 5: model_config with the full config_info entry
  console.log('\n=== Test 5: model_config = full config_info entry ===');
  await send(makeBody({
    model_config: mmConfig,
  }), 'Test5-full_config_info');

  setTimeout(() => process.exit(0), 2000);
})();
