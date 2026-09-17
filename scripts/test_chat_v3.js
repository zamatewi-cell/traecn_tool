// Test chat_v3 function: get_detail_param + create_agent_task
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

function postJSON(path, body) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru', path, method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(bodyStr),
        'Accept': 'application/json',
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
      res.on('data', c => data += c.toString());
      res.on('end', () => {
        try { resolve({ status: res.statusCode, data: JSON.parse(data) }); }
        catch { resolve({ status: res.statusCode, raw: data.substring(0, 500) }); }
      });
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

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
            } else if (val.length > 300) {
              process.stdout.write(`[data: ${val.substring(0, 300)}...]`);
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
  // Step 1: Get detail params for chat_v3
  console.log('=== Step 1: get_detail_param for chat_v3 ===');
  const paramResp = await postJSON('/api/ide/v1/get_detail_param', {
    function: 'chat_v3',
    need_prompt: true,
    poly_prompt: true
  });
  console.log('Status:', paramResp.status);
  
  if (paramResp.data) {
    const configs = paramResp.data.config_info_list || [];
    const metadata = paramResp.data.metadata || {};
    const prompts = metadata.encrypted_prompt_set || [];
    console.log('Models:', configs.length, configs.map(c => c.config_name).join(', '));
    console.log('Prompts:', prompts.length, prompts.map(p => p.prompt_key).join(', '));
    console.log('Client config:', JSON.stringify(metadata.client_config));
    
    // Save full response
    fs.writeFileSync('scripts/detail_param_chat_v3.json', JSON.stringify(paramResp.data, null, 2));
    console.log('Saved to detail_param_chat_v3.json');
    
    // Pick a model
    const model = configs.find(m => m.config_name === 'deepseek-V3.1') || configs[0];
    if (!model) {
      console.log('No models found, aborting');
      return;
    }
    const enc = model.model_detail_list[0].encrypted_model_params;
    console.log('\nUsing model:', model.config_name);

    // Step 2: create_agent_task with function=chat_v3, agent_type=chat
    console.log('\n=== Step 2: create_agent_task function=chat_v3 agent_type=chat ===');
    const body1 = {
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
      function: 'chat_v3',
      agent_type: 'chat',
      ide_version: '3.3.37',
      user_input: {
        id: genId(),
        content: 'say hello in one word'
      }
    };
    console.log('Body size:', JSON.stringify(body1).length, 'bytes');
    const r1 = await postSSE('/api/agent/v3/create_agent_task', body1);
    console.log('\nStatus:', r1.status);

    // Step 3: create_agent_task with function=chat_v3, agent_type=chat_v3
    console.log('\n=== Step 3: create_agent_task function=chat_v3 agent_type=chat_v3 ===');
    const body2 = {
      ...body1,
      conversation_id: genId(),
      session_id: genId(),
      task_id: genId(),
      message_id: genId(),
      agent_type: 'chat_v3',
      user_input: { id: genId(), content: 'say hello in one word' }
    };
    const r2 = await postSSE('/api/agent/v3/create_agent_task', body2);
    console.log('\nStatus:', r2.status);

    // Step 4: create_agent_task with function=chat_v3, agent_type=builder
    console.log('\n=== Step 4: create_agent_task function=chat_v3 agent_type=builder ===');
    const body3 = {
      ...body1,
      conversation_id: genId(),
      session_id: genId(),
      task_id: genId(),
      message_id: genId(),
      agent_type: 'builder',
      user_input: { id: genId(), content: 'say hello in one word' }
    };
    const r3 = await postSSE('/api/agent/v3/create_agent_task', body3);
    console.log('\nStatus:', r3.status);

    // Step 5: solo_builder function
    console.log('\n=== Step 5: create_agent_task function=solo_builder agent_type=builder ===');
    const body4 = {
      ...body1,
      conversation_id: genId(),
      session_id: genId(),
      task_id: genId(),
      message_id: genId(),
      function: 'solo_builder',
      agent_type: 'builder',
      user_input: { id: genId(), content: 'say hello in one word' }
    };
    const r4 = await postSSE('/api/agent/v3/create_agent_task', body4);
    console.log('\nStatus:', r4.status);
  } else {
    console.log('Raw:', paramResp.raw);
  }
}

main().catch(console.error);
