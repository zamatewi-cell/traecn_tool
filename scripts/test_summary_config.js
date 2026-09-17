// Test fixing summary config error
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));
const detail = JSON.parse(fs.readFileSync(__dirname + '/detail_param_chat_v3.json', 'utf8'));

const mmConfig = detail.config_info_list.find(c => c.config_name === 'minimax-m2.5');
const summaryConfig = detail.config_info_list.find(c => c.config_name === 'summary');
const titleConfig = detail.config_info_list.find(c => c.config_name === 'title_generation');

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
      res.on('data', c => d += c);
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
    model_name: "minimax-m2.5__dev",
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
  // Test 1: Add summary_config_name and summary_model_name
  console.log('=== Test 1: summary_config_name + summary_model_name ===');
  await send(makeBody({
    summary_config_name: "summary",
    summary_model_name: "Doubao_1_6",
  }), 'summary_fields');

  // Test 2: Include config_info_list with summary
  console.log('=== Test 2: config_info_list with summary ===');
  await send(makeBody({
    config_info_list: [mmConfig, summaryConfig, titleConfig],
  }), 'config_info_list');

  // Test 3: Include summary_config as object
  console.log('=== Test 3: summary_config object ===');
  await send(makeBody({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
    },
  }), 'summary_config_obj');

  // Test 4: Try agent_type = "solo_coder" - maybe simpler flow
  console.log('=== Test 4: agent_type=solo_coder ===');
  await send(makeBody({
    agent_type: "solo_coder",
  }), 'solo_coder');

  // Test 5: Try COMPLETELY different - use llm_raw_chat endpoint instead
  console.log('=== Test 5: llm_raw_chat endpoint ===');
  const rawBody = JSON.stringify({
    model_name: "minimax-m2.5__dev",
    messages: [
      { role: "user", content: "hello, what is 1+1?" }
    ],
    max_tokens: 1024,
    stream: true,
    config_name: "minimax-m2.5",
    user_id: "4355622541471866",
    device_id: "2262131830954826",
    ide_version: "3.3.37",
    version_code: 20260212,
    function: "chat_v3",
  });
  
  await new Promise(r => {
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
        'x-machine-id': h['x-machine-id'],
        'package-type': 'stable_cn',
        'user-agent': h['user-agent'],
      },
    }, res => {
      let d = '';
      res.on('data', c => d += c);
      res.on('end', () => {
        console.log(`\n[Test5-llm_raw_chat] Status: ${res.statusCode}`);
        const lines = d.split('\n');
        for (const line of lines) {
          if (line.trim()) console.log(' ', line.substring(0, 300));
        }
        r();
      });
    });
    req.on('error', e => { console.log('[Test5] Error:', e.message); r(); });
    req.write(rawBody);
    req.end();
  });

  // Test 6: Try the exact model_detail encrypted params as request-level field
  console.log('=== Test 6: encrypted_model_params at body level + __dev model name ===');
  await send(makeBody({
    encrypted_model_params: mmConfig.model_detail_list[0].encrypted_model_params,
  }), 'encrypted_params_dev');

  setTimeout(() => process.exit(0), 2000);
})();
