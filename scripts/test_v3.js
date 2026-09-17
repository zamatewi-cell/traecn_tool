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

function post(path, body) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru', path, method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(bodyStr),
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
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, body: Buffer.concat(chunks).toString() }));
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

async function main() {
  const chatParams = JSON.parse(fs.readFileSync('scripts/detail_param_chat.json', 'utf8'));
  const model = chatParams.config_info_list.find(m => m.config_name === 'deepseek-V3.1');
  const enc = model.model_detail_list[0].encrypted_model_params;
  const prompts = chatParams.metadata.encrypted_prompt_set;
  
  // Try builder_v3 and other potential agent_types from the binary + logs
  const agentTypes = ['builder_v3', 'builder_v2', 'dev', 'trae_agent_dev', 'general_purpose_task', 'single_task', 'multi_task'];
  
  for (const at of agentTypes) {
    const body = {
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
      agent_type: at,
      ide_version: '3.3.37',
      encrypted_prompt_set: prompts,
      user_input: {
        id: genId(),
        content: 'say hello in one word'
      }
    };
    const r = await post('/api/agent/v3/create_agent_task', body);
    const snippet = r.body.substring(0, 400).replace(/\n/g, ' | ');
    console.log(`${at}: ${r.status} -> ${snippet}`);
    console.log('');
  }
}

main().catch(console.error);
