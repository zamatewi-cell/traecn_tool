// Test with discovered binary field names for summary
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));
const detail = JSON.parse(fs.readFileSync(__dirname + '/detail_param_chat_v3.json', 'utf8'));
const summaryConfig = detail.config_info_list.find(c => c.config_name === 'summary');

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
        console.log(`[${label}] ${res.statusCode}: ${d.substring(0, 500)}`);
        r(d);
      });
    });
    req.on('error', e => { console.log(`[${label}] Error:`, e.message); r(''); });
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
  // Test 1: summary_base_prompt in prompt_template_map
  console.log('=== T1: summary_base_prompt in prompt_template_map ===');
  await send(base({
    prompt_template_map: {
      "master_agent": "You are a helpful AI assistant. Answer concisely.",
      "summary_base_prompt": "Summarize the following conversation in one sentence.",
      "summary_user_input": "{{conversation}}",
    },
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
    },
  }), 'T1');

  // Test 2: summary_base_prompt as top-level field
  console.log('=== T2: summary_base_prompt top-level ===');
  await send(base({
    summary_base_prompt: "Summarize the following conversation in one sentence.",
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
    },
  }), 'T2');

  // Test 3: summary_config with summary_base_prompt inside
  console.log('=== T3: summary_base_prompt inside summary_config ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
      summary_base_prompt: "Summarize the following conversation in one sentence.",
      prompt_template_summary_user_input: "{{conversation}}",
    },
  }), 'T3');

  // Test 4: Try prompt_config in summary_config  
  console.log('=== T4: prompt_config inside summary_config ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
      prompt_config: {
        summary_base_prompt: "Summarize the conversation briefly.",
        summary_user_input: "{{conversation}}",
      },
    },
  }), 'T4');

  // Test 5: Completely different - nested under model_extra_config
  console.log('=== T5: summary fields in model_extra_config JSON ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
      model_extra_config: JSON.stringify({
        summary_base_prompt: "Summarize the conversation briefly.",
        summary_message_token_limit: 8000,
        summary_look_back_count: 5,
      }),
    },
  }), 'T5');

  // Test 6: Put prompt_template + prompt_config together
  console.log('=== T6: prompt_template field in summary_config ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
      prompt_template: "Summarize the conversation: {{messages}}",
      prompt_config_list: [{
        prompt_key: "summary",
        prompt_label: "default",
        prompt_version: "0.0.1",
      }],
    },
    prompt_template_map: {
      "master_agent": "You are a helpful assistant. Answer concisely.",
      "summary": "Summarize the conversation briefly.",
    },
  }), 'T6');

  // Test 7: Use the exact prompt key patterns from encrypted_prompt_set
  console.log('=== T7: exact prompt keys ===');
  await send(base({
    summary_config: {
      config_name: "summary",
      model_name: "Doubao_1_6",
      encrypted_model_params: summaryConfig.model_detail_list[0].encrypted_model_params,
      max_tokens: summaryConfig.model_detail_list[0].max_tokens,
    },
    prompt_template_map: {
      "trae_agent_solo_coder_dev_chat.default.master_agent": "You are a helpful AI assistant.",
      "trae_agent_solo_coder_dev.default.compact_prompt": "Summarize the conversation briefly.",
      "trae_agent_solo_coder_dev.default.tools": "[]",
      "trae_agent_solo_coder_dev.default.misc": "",
    },
  }), 'T7');

  setTimeout(() => process.exit(0), 2000);
})();
