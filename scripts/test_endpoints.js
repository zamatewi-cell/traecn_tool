// Systematically discover llm_raw_chat fields
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));

function send(path, body, label) {
  const bodyStr = JSON.stringify(body);
  return new Promise(r => {
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru',
      path,
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
        if (d.trim()) {
          // Try JSON parse
          try {
            const j = JSON.parse(d);
            console.log('  JSON:', JSON.stringify(j).substring(0, 500));
          } catch {
            const lines = d.split('\n');
            for (const line of lines) {
              if (line.trim()) console.log(' ', line.substring(0, 400));
            }
          }
        } else {
          console.log('  (empty body)');
        }
        r(d);
      });
    });
    req.on('error', e => { console.log(`[${label}] Error:`, e.message); r(''); });
    req.write(bodyStr);
    req.end();
  });
}

(async () => {
  // Test 1: llm_raw_chat with empty body
  console.log('=== Test 1: llm_raw_chat empty ===');
  await send('/api/ide/v2/llm_raw_chat', {}, 'empty');

  // Test 2: llm_raw_chat with messages
  console.log('=== Test 2: llm_raw_chat with messages ===');
  await send('/api/ide/v2/llm_raw_chat', {
    messages: [{ role: "user", content: "hello" }],
    model: "minimax-m2.5__dev",
    stream: true,
  }, 'messages');

  // Test 3: llm_raw_chat with model_name field
  console.log('=== Test 3: llm_raw_chat with model_name ===');
  await send('/api/ide/v2/llm_raw_chat', {
    messages: [{ role: "user", content: "hello" }],
    model_name: "minimax-m2.5__dev",
    config_name: "minimax-m2.5",
    stream: true,
    user_id: "4355622541471866",
    device_id: "2262131830954826",
    ide_version: "3.3.37",
    version_code: 20260212,
  }, 'model_name');

  // Test 4: llm_raw_chat with encrypted_model_params
  const detail = JSON.parse(fs.readFileSync(__dirname + '/detail_param_chat_v3.json', 'utf8'));
  const mmConfig = detail.config_info_list.find(c => c.config_name === 'minimax-m2.5');
  await send('/api/ide/v2/llm_raw_chat', {
    messages: [{ role: "user", content: "hello" }],
    model_name: "minimax-m2.5__dev",
    config_name: "minimax-m2.5",
    encrypted_model_params: mmConfig.model_detail_list[0].encrypted_model_params,
    stream: true,
    user_id: "4355622541471866",
    device_id: "2262131830954826",
  }, 'with_encrypted_params');

  // Test 5: Try v1 endpoint
  console.log('=== Test 5: llm_raw_chat v1 ===');
  await send('/api/ide/v1/llm_raw_chat', {
    messages: [{ role: "user", content: "hello" }],
    model_name: "minimax-m2.5__dev",
  }, 'v1');

  // Test 6: Try chat_completion endpoint  
  console.log('=== Test 6: chat_completion v1 ===');
  await send('/api/ide/v1/chat_completion', {
    messages: [{ role: "user", content: "hello" }],
    model_name: "minimax-m2.5__dev",
  }, 'chat_completion');

  // Test 7: Try /api/agent/v3 with different agent_type "chat"
  console.log('=== Test 7: create_agent_task agent_type=chat ===');
  await send('/api/agent/v3/create_agent_task', {
    function: "chat_v3",
    agent_type: "chat",
    model_name: "minimax-m2.5__dev",
    config_name: "minimax-m2.5",
    session_id: crypto.randomBytes(12).toString('hex'),
    conversation_id: crypto.randomBytes(12).toString('hex'),
    task_id: crypto.randomUUID(),
    version_code: 20260212,
    user_id: "4355622541471866",
    device_id: "2262131830954826",
    ide_version: "3.3.37",
    user_input: { id: crypto.randomUUID(), content: "hello", type: "text" },
    prompt_template_map: { "master_agent": "You are a helpful assistant." },
    history_list: [],
    settings: {},
  }, 'agent_chat');

  // Test 8: create_agent_task with function = "single_chat"
  console.log('=== Test 8: function=single_chat ===');
  await send('/api/agent/v3/create_agent_task', {
    function: "single_chat",
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
    user_input: { id: crypto.randomUUID(), content: "hello", type: "text" },
    prompt_template_map: { "master_agent": "You are a helpful assistant." },
    history_list: [],
    settings: {},
  }, 'single_chat_func');

  setTimeout(() => process.exit(0), 2000);
})();
