// Deep-dive into summary_config structure
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));
const detail = JSON.parse(fs.readFileSync(__dirname + '/detail_param_chat_v3.json', 'utf8'));

const summaryConfig = detail.config_info_list.find(c => c.config_name === 'summary');
const summaryModelDetail = summaryConfig.model_detail_list[0];

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
      timeout: 30000,
    }, res => {
      let d = '';
      res.on('data', c => d += c);
      res.on('end', () => {
        console.log(`\n[${label}] Status: ${res.statusCode}`);
        const lines = d.split('\n');
        for (const line of lines) {
          if (line.trim()) console.log(' ', line.substring(0, 400));
        }
        r(d);
      });
    });
    req.on('error', e => { console.log(`[${label}] Error:`, e.message); r(''); });
    req.setTimeout(30000, () => { console.log(`[${label}] Timeout`); req.destroy(); r(''); });
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
      "master_agent": "You are a helpful AI coding assistant. Answer concisely.",
    },
    history_list: [],
    settings: {},
    ...overrides,
  };
}

(async () => {
  // Test 1: summary_config with prompt_template + encrypted_model_params
  console.log('=== Test 1: summary_config with prompt_template ===');
  await send(makeBody({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6", 
      encrypted_model_params: summaryModelDetail.encrypted_model_params,
      max_tokens: summaryModelDetail.max_tokens,
      prompt_template: "Please summarize the conversation briefly.",
    },
  }), 'T1');

  // Test 2: summary_config with template field
  console.log('=== Test 2: summary_config with template ===');
  await send(makeBody({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryModelDetail.encrypted_model_params,
      max_tokens: summaryModelDetail.max_tokens,
      template: "Please summarize the conversation briefly.",
    },
  }), 'T2');

  // Test 3: summary_config with summary_template
  console.log('=== Test 3: summary_config with summary_template ===');
  await send(makeBody({
    summary_config: {
      config_name: "summary",  
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryModelDetail.encrypted_model_params,
      max_tokens: summaryModelDetail.max_tokens,
      summary_template: "Please summarize the conversation briefly.",
      prompt_max_tokens: 32000,
    },
  }), 'T3');

  // Test 4: summary_config with full model_detail_list-like structure
  console.log('=== Test 4: summary_config with full model detail ===');
  await send(makeBody({
    summary_config: {
      ...summaryModelDetail,
      prompt_template: "Summarize the conversation.",
    },
  }), 'T4');

  // Test 5: Put summary template in prompt_template_map
  console.log('=== Test 5: summary prompt in prompt_template_map ===');
  await send(makeBody({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryModelDetail.encrypted_model_params,
      max_tokens: summaryModelDetail.max_tokens,
    },
    prompt_template_map: {
      "master_agent": "You are a helpful AI coding assistant. Answer concisely.",
      "summary": "Summarize the conversation briefly.",
      "compact_prompt": "Summarize the conversation briefly.",
    },
  }), 'T5');

  // Test 6: Try with disable flags
  console.log('=== Test 6: disable_summary + skip_summary ===');
  await send(makeBody({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryModelDetail.encrypted_model_params,
      max_tokens: summaryModelDetail.max_tokens,
    },
    disable_summary: true,
    skip_summary: true,
  }), 'T6');

  // Test 7: summary_config with template_data field
  console.log('=== Test 7: summary_config with template_data ===');
  await send(makeBody({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryModelDetail.encrypted_model_params,
      max_tokens: summaryModelDetail.max_tokens,
      template_data: "Summarize the conversation.",
      summary_prompt_template: "Summarize the conversation.",
    },
  }), 'T7');

  setTimeout(() => process.exit(0), 2000);
})();
