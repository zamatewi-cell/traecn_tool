// Try passing encrypted_prompt_set back to server and other approaches
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
  console.log(`  [${label}] body size: ${bodyStr.length}`);
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
        console.log(`  [${label}] ${res.statusCode}: ${d.substring(0, 400)}`);
        r(d);
      });
    });
    req.on('error', e => { console.log(`  [${label}] Error:`, e.message); r(''); });
    req.setTimeout(30000, () => { req.destroy(); r(''); });
    req.write(bodyStr);
    req.end();
  });
}

function base(overrides = {}) {
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
    app_id: "6eefa01c-1036-4c7e-9ca5-d891f63bfcd8",
    machine_id: h['x-machine-id'],
    os_version: "Windows 11",
    user_input: { id: crypto.randomUUID(), content: "hello, what is 1+1?", type: "text" },
    prompt_template_map: { "master_agent": "You are a helpful AI assistant. Answer concisely." },
    history_list: [],
    settings: {},
    ...overrides,
  };
}

(async () => {
  // Test 1: Pass encrypted_prompt_set from metadata
  console.log('=== Test 1: encrypted_prompt_set from metadata ===');
  await send(base({
    encrypted_prompt_set: detail.metadata.encrypted_prompt_set,
  }), 'T1');

  // Test 2: Pass full metadata
  console.log('=== Test 2: metadata from get_detail_param ===');
  await send(base({
    metadata: detail.metadata,
  }), 'T2');

  // Test 3: Include summary_config with encrypted_model_params AND encrypted_prompt_set
  console.log('=== Test 3: summary_config + encrypted_prompt_set ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
    },
    encrypted_prompt_set: detail.metadata.encrypted_prompt_set,
  }), 'T3');

  // Test 4: Include ALL config_info items as a map
  console.log('=== Test 4: config_info_map ===');
  const configMap = {};
  detail.config_info_list.forEach(c => { configMap[c.config_name] = c; });
  await send(base({
    config_info_map: configMap,
  }), 'T4');

  // Test 5: Put summary model params directly
  console.log('=== Test 5: summary_encrypted_model_params top-level ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
      prompt_max_tokens: summaryConfig.model_detail_list[0].prompt_max_tokens,
      model_extra_config: summaryConfig.model_detail_list[0].model_extra_config,
    },
    summary_encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
    summary_model_name: "Doubao_1_6",
    summary_prompt_template: "Summarize the conversation.",
  }), 'T5');

  // Test 6: Use title_generation config for summary
  console.log('=== Test 6: summary with title_generation config ===');
  await send(base({
    summary_config: {
      config_name: "title_generation",
      model_name: titleConfig.model_detail_list[0].model_name,
      encrypted_model_params: titleConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: titleConfig.model_detail_list[0].max_tokens,
    },
  }), 'T6');

  // Test 7: Pass everything - config_info_list + metadata + summary_config
  console.log('=== Test 7: KITCHEN SINK ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
      prompt_max_tokens: summaryConfig.model_detail_list[0].prompt_max_tokens || 32000,
    },
    title_config: {
      config_name: "title_generation",
      model_name: titleConfig.model_detail_list[0].model_name,
      encrypted_model_params: titleConfig.model_detail_list[0].encrypted_model_params,
    },
    config_info_list: detail.config_info_list,
    encrypted_prompt_set: detail.metadata.encrypted_prompt_set,
    metadata: detail.metadata,
    prompt_template_map: {
      "master_agent": "You are a helpful AI assistant. Answer concisely.",
      "compact_prompt": "Summarize the conversation briefly.",
      "tools": "You have no tools.",
      "misc": "",
      "summary": "Summarize the conversation briefly in one sentence.",
    },
  }), 'T7');

  setTimeout(() => process.exit(0), 2000);
})();
