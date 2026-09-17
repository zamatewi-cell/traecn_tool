// Test with full model detail and encrypted prompts
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

function postSSE(path, body) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru', path, method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(bodyStr),
        'Accept': 'text/event-stream',
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
      let data = '';
      let fullData = '';
      res.on('data', c => {
        const str = c.toString();
        fullData += str;
        data += str;
        const lines = data.split('\n');
        for (let i = 0; i < lines.length - 1; i++) {
          const line = lines[i].trim();
          if (line.startsWith('event:') || line.startsWith('data:')) {
            const val = line.substring(line.indexOf(':') + 1).trim();
            if (line.startsWith('event:')) {
              process.stdout.write(`\n[EVENT] ${val} `);
            } else if (val.length > 400) {
              process.stdout.write(`[data: ${val.substring(0, 400)}...]`);
            } else {
              process.stdout.write(`[data: ${val}]`);
            }
          }
        }
        data = lines[lines.length - 1];
      });
      res.on('end', () => {
        resolve({ status: res.statusCode, fullData });
      });
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

async function main() {
  // Load chat_v3 params
  const params = JSON.parse(fs.readFileSync('scripts/detail_param_chat_v3.json', 'utf8'));
  const model = params.config_info_list.find(m => m.config_name === 'qwen-3.5');
  const detail = model.model_detail_list[0];
  const metadata = params.metadata;
  
  // Get the prompt configs for this model
  const modelPromptConfigs = detail.prompt_config_list;
  console.log('Model:', model.config_name, '¡ú model_name:', detail.model_name);
  console.log('Prompt configs:', modelPromptConfigs.map(p => p.prompt_key).join(', '));
  
  // Get matching encrypted prompts
  const relevantPrompts = metadata.encrypted_prompt_set.filter(
    ep => modelPromptConfigs.some(pc => pc.prompt_key === ep.prompt_key)
  );
  console.log('Matching encrypted prompts:', relevantPrompts.length);
  
  // Build prompt_template_map: key -> encrypted template
  const promptTemplateMap = {};
  for (const ep of relevantPrompts) {
    promptTemplateMap[ep.prompt_key] = ep.prompt_template;
  }
  
  // Also build encrypted_prompt_list format
  const encryptedPromptList = relevantPrompts.map(ep => ({
    prompt_key: ep.prompt_key,
    prompt_label: ep.prompt_label,
    prompt_version: ep.prompt_version,
    prompt_template: ep.prompt_template
  }));
  
  const sessionId = genId();
  const convId = genId();
  
  // Test 1: encrypted_prompt_list field (from binary analysis)
  console.log('\n=== Test 1: encrypted_prompt_list + agent_type=builder ===');
  const body1 = {
    conversation_id: convId,
    session_id: sessionId,
    task_id: genId(),
    message_id: genId(),
    user_id: userId,
    device_id: '2262131830954826',
    model_name: detail.model_name,  // qwen-3.5__dev
    config_name: model.config_name,  // qwen-3.5
    encrypted_model_params: detail.encrypted_model_params,
    stream: true,
    function: 'chat_v3',
    agent_type: 'builder',
    ide_version: '3.3.37',
    // NEW: model detail fields
    prompt_config_list: modelPromptConfigs,
    encrypted_prompt_list: encryptedPromptList,
    model_extra_config: detail.model_extra_config,
    prompt_max_tokens: detail.prompt_max_tokens,
    max_turn: detail.max_turn,
    tool_call_history_max_tokens: detail.tool_call_history_max_tokens,
    user_input: {
      id: genId(),
      content: 'say hello in one word'
    }
  };
  console.log('Body size:', JSON.stringify(body1).length, 'bytes');
  const r1 = await postSSE('/api/agent/v3/create_agent_task', body1);
  console.log('\nStatus:', r1.status);

  // Test 2: prompt_template_map field
  console.log('\n=== Test 2: prompt_template_map + agent_type=builder ===');
  const body2 = {
    conversation_id: convId,
    session_id: sessionId,
    task_id: genId(),
    message_id: genId(),
    user_id: userId,
    device_id: '2262131830954826',
    model_name: detail.model_name,
    config_name: model.config_name,
    encrypted_model_params: detail.encrypted_model_params,
    stream: true,
    function: 'chat_v3',
    agent_type: 'builder',
    ide_version: '3.3.37',
    prompt_template_map: promptTemplateMap,
    prompt_config_list: modelPromptConfigs,
    model_extra_config: detail.model_extra_config,
    prompt_max_tokens: detail.prompt_max_tokens,
    max_turn: detail.max_turn,
    user_input: {
      id: genId(),
      content: 'say hello in one word'
    }
  };
  console.log('Body size:', JSON.stringify(body2).length, 'bytes');
  const r2 = await postSSE('/api/agent/v3/create_agent_task', body2);
  console.log('\nStatus:', r2.status);

  // Test 3: ALL encrypted prompts (not just model-specific ones)
  console.log('\n=== Test 3: ALL encrypted_prompt_set + agent_type=builder ===');
  const body3 = {
    conversation_id: convId,
    session_id: sessionId,
    task_id: genId(),
    message_id: genId(),
    user_id: userId,
    device_id: '2262131830954826',
    model_name: detail.model_name,
    config_name: model.config_name,
    encrypted_model_params: detail.encrypted_model_params,
    stream: true,
    function: 'chat_v3',
    agent_type: 'builder',
    ide_version: '3.3.37',
    encrypted_prompt_set: metadata.encrypted_prompt_set,
    encrypted_prompt_list: metadata.encrypted_prompt_set,
    prompt_config_list: modelPromptConfigs,
    model_extra_config: detail.model_extra_config,
    prompt_max_tokens: detail.prompt_max_tokens,
    max_turn: detail.max_turn,
    user_input: {
      id: genId(),
      content: 'say hello in one word'
    }
  };
  console.log('Body size:', JSON.stringify(body3).length, 'bytes');
  const r3 = await postSSE('/api/agent/v3/create_agent_task', body3);
  console.log('\nStatus:', r3.status);

  // Test 4: function_config and model_config wrapping
  console.log('\n=== Test 4: function_config + model_config wrapper ===');
  const body4 = {
    conversation_id: convId,
    session_id: sessionId,
    task_id: genId(),
    message_id: genId(),
    user_id: userId,
    device_id: '2262131830954826',
    model_name: detail.model_name,
    config_name: model.config_name,
    encrypted_model_params: detail.encrypted_model_params,
    stream: true,
    function: 'chat_v3',
    agent_type: 'builder',
    ide_version: '3.3.37',
    function_config: {
      prompt_template_map: promptTemplateMap,
      prompt_config_list: modelPromptConfigs,
    },
    model_config: {
      encrypted_model_params: detail.encrypted_model_params,
      model_extra_config: detail.model_extra_config,
      prompt_max_tokens: detail.prompt_max_tokens,
      max_turn: detail.max_turn,
      encrypted_prompt_list: encryptedPromptList,
    },
    user_input: {
      id: genId(),
      content: 'say hello in one word'
    }
  };
  console.log('Body size:', JSON.stringify(body4).length, 'bytes');
  const r4 = await postSSE('/api/agent/v3/create_agent_task', body4);
  console.log('\nStatus:', r4.status);
}

main().catch(console.error);
