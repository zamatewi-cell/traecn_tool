// Test create_agent_task with full metadata including encrypted prompts
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
      res.on('data', c => {
        data += c.toString();
        // Print SSE events as they come
        const lines = data.split('\n');
        for (let i = 0; i < lines.length - 1; i++) {
          const line = lines[i].trim();
          if (line.startsWith('event:') || line.startsWith('data:')) {
            const val = line.substring(line.indexOf(':') + 1).trim();
            if (line.startsWith('event:')) {
              process.stdout.write(`\n[EVENT] ${val} `);
            } else if (val.length > 200) {
              process.stdout.write(`[data: ${val.substring(0, 200)}...]`);
            } else {
              process.stdout.write(`[data: ${val}]`);
            }
          }
        }
        data = lines[lines.length - 1];
      });
      res.on('end', () => {
        resolve({ status: res.statusCode });
      });
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

async function main() {
  const chatParams = JSON.parse(fs.readFileSync('scripts/detail_param_chat.json', 'utf8'));
  const model = chatParams.config_info_list.find(m => m.config_name === 'deepseek-V3.1')
    || chatParams.config_info_list[0];
  const enc = model.model_detail_list[0].encrypted_model_params;
  const metadata = chatParams.metadata;
  
  console.log('Model:', model.config_name);
  console.log('Prompts:', metadata.encrypted_prompt_set.length);
  console.log('Client config:', metadata.client_config);
  
  const baseBody = {
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
    }
  };

  // Test 1: Include full metadata at top level
  console.log('\n=== Test 1: full metadata at top level ===');
  const body1 = {
    ...baseBody,
    conversation_id: genId(),
    session_id: genId(), 
    task_id: genId(),
    message_id: genId(),
    metadata: metadata,
    encrypted_prompt_set: metadata.encrypted_prompt_set,
    ab_versions: metadata.ab_versions,
    client_config: metadata.client_config
  };
  console.log('Body size:', JSON.stringify(body1).length, 'bytes');
  const r1 = await postSSE('/api/agent/v3/create_agent_task', body1);
  console.log('\nStatus:', r1.status);

  // Test 2: Include metadata + extra fields from binary analysis
  console.log('\n=== Test 2: metadata + extra fields ===');
  const body2 = {
    ...baseBody,
    conversation_id: genId(),
    session_id: genId(),
    task_id: genId(),
    message_id: genId(),
    metadata: metadata,
    encrypted_prompt_set: metadata.encrypted_prompt_set,
    client_config: metadata.client_config,
    locale_language: 'zh-CN',
    has_history: false,
    original_prompt: 'say hello in one word',
    use_inline_chat_prompt: false,
    has_user_interaction_contexts: false,
    relevant_context: '',
    relevant_context_group: ''
  };
  console.log('Body size:', JSON.stringify(body2).length, 'bytes');
  const r2 = await postSSE('/api/agent/v3/create_agent_task', body2);
  console.log('\nStatus:', r2.status);

  // Test 3: Put prompts inside model config
  console.log('\n=== Test 3: prompt_set instead of encrypted_prompt_set ===');
  const body3 = {
    ...baseBody,
    conversation_id: genId(),
    session_id: genId(),
    task_id: genId(),
    message_id: genId(),
    prompt_set: metadata.encrypted_prompt_set,
    client_config: metadata.client_config,
    locale_language: 'zh-CN',
    has_history: false,
    original_prompt: 'say hello in one word'
  };
  console.log('Body size:', JSON.stringify(body3).length, 'bytes');
  const r3 = await postSSE('/api/agent/v3/create_agent_task', body3);
  console.log('\nStatus:', r3.status);
}

main().catch(console.error);
