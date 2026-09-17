// Test with specific prompt type keys: userInputPrompt, initial, sequential
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
        const lines = data.split('\n');
        for (let i = 0; i < lines.length - 1; i++) {
          const line = lines[i].trim();
          if (line.startsWith('event:') || line.startsWith('data:')) {
            const val = line.substring(line.indexOf(':') + 1).trim();
            if (line.startsWith('event:')) {
              process.stdout.write(`\n[EVENT] ${val} `);
            } else if (val.length > 500) {
              process.stdout.write(`[data: ${val.substring(0, 500)}...]`);
            } else {
              process.stdout.write(`[data: ${val}]`);
            }
          }
        }
        data = lines[lines.length - 1];
      });
      res.on('end', () => resolve({ status: res.statusCode }));
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

async function main() {
  const params = JSON.parse(fs.readFileSync('scripts/detail_param_chat_v3.json', 'utf8'));
  const metadata = params.metadata;
  const model = params.config_info_list.find(m => m.config_name === 'deepseek-V3.1');
  const detail = model.model_detail_list[0];
  
  // Get the master_agent prompt (most likely the user input prompt)
  const masterPrompt = metadata.encrypted_prompt_set.find(e => 
    e.prompt_key.includes('master_agent') && e.prompt_key.includes('default')
  );
  // Also get single_chat prompt  
  const singleChatPrompt = metadata.encrypted_prompt_set.find(e =>
    e.prompt_key.includes('single_chat')
  );
  
  console.log('Master prompt:', masterPrompt?.prompt_key, masterPrompt?.prompt_template.length, 'chars');
  console.log('Single chat prompt:', singleChatPrompt?.prompt_key, singleChatPrompt?.prompt_template.length, 'chars');
  
  const baseFields = {
    user_id: userId,
    device_id: '2262131830954826',
    config_name: 'deepseek-V3.1',
    model_name: 'deepseek-V3.1',
    encrypted_model_params: detail.encrypted_model_params,
    stream: true,
    function: 'chat_v3',
    ide_version: '3.3.37',
  };
  
  // Test 1: prompt_template_map with userInputPrompt/initial/sequential keys
  console.log('\n=== Test 1: prompt_template_map with type keys ===');
  const tpl = masterPrompt.prompt_template;
  const body1 = {
    ...baseFields,
    conversation_id: genId(), session_id: genId(), task_id: genId(), message_id: genId(),
    agent_type: 'builder',
    prompt_template_map: {
      userInputPrompt: tpl,
      initial: tpl,
      sequential: tpl,
    },
    user_input: { id: genId(), content: 'say hello in one word' }
  };
  console.log('Body size:', JSON.stringify(body1).length);
  const r1 = await postSSE('/api/agent/v3/create_agent_task', body1);
  console.log('\nStatus:', r1.status);
  
  // Test 2: user_input_prompt field directly
  console.log('\n=== Test 2: user_input_prompt field ===');
  const body2 = {
    ...baseFields,
    conversation_id: genId(), session_id: genId(), task_id: genId(), message_id: genId(),
    agent_type: 'builder',
    user_input_prompt: tpl,
    initial_prompt: tpl,
    prompt_template_map: { master_agent: tpl },
    user_input: { id: genId(), content: 'say hello in one word' }
  };
  console.log('Body size:', JSON.stringify(body2).length);
  const r2 = await postSSE('/api/agent/v3/create_agent_task', body2);
  console.log('\nStatus:', r2.status);
  
  // Test 3: Try agent_type=solo_coder (original name from prompts)
  console.log('\n=== Test 3: agent_type=solo_coder ===');
  const body3 = {
    ...baseFields,
    conversation_id: genId(), session_id: genId(), task_id: genId(), message_id: genId(),
    agent_type: 'solo_coder',
    prompt_template_map: {
      userInputPrompt: tpl,
      initial: tpl,
      sequential: tpl,
      master_agent: tpl,
    },
    user_input: { id: genId(), content: 'say hello in one word' }
  };
  const r3 = await postSSE('/api/agent/v3/create_agent_task', body3);
  console.log('\nStatus:', r3.status);
  
  // Test 4: Try direct rendered prompt (not encrypted)
  console.log('\n=== Test 4: Plaintext user_input_prompt ===');
  const body4 = {
    ...baseFields,
    conversation_id: genId(), session_id: genId(), task_id: genId(), message_id: genId(),
    agent_type: 'builder',
    user_input_prompt: 'You are a helpful assistant. Please respond to the following user message:\n\nsay hello in one word',
    prompt_template_map: {
      userInputPrompt: 'You are a helpful assistant. Please respond to the following user message:\n\n{{user_input}}',
      initial: 'You are a helpful assistant. Please respond to the following user message:\n\n{{user_input}}',
    },
    user_input: { id: genId(), content: 'say hello in one word' }
  };
  const r4 = await postSSE('/api/agent/v3/create_agent_task', body4);
  console.log('\nStatus:', r4.status);

  // Test 5: agent_type=builder_v3 (the actual agent_id from logs)
  console.log('\n=== Test 5: agent_type=builder_v3 with prompt maps ===');
  const body5 = {
    ...baseFields,
    conversation_id: genId(), session_id: genId(), task_id: genId(), message_id: genId(),
    agent_type: 'builder_v3',
    prompt_template_map: {
      userInputPrompt: tpl,
      initial: tpl, 
      sequential: tpl,
    },
    user_input: { id: genId(), content: 'say hello in one word' }
  };
  const r5 = await postSSE('/api/agent/v3/create_agent_task', body5);
  console.log('\nStatus:', r5.status);
}

main().catch(console.error);
