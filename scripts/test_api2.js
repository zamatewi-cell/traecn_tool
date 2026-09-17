const fs = require('fs');
const https = require('https');
const crypto = require('crypto');
const os = require('os');

const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
const d = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
const a = JSON.parse(d['iCubeAuthInfo://icube.cloudide']);
const token = a.token;
const userId = a.userId;

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
        'x-app-version': '1.107.1',
        'x-ide-version-code': '20250325',
        'x-app-version-code': '20250325',
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
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks).toString() }));
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

function genId() {
  return crypto.randomBytes(12).toString('hex');
}

async function main() {
  // Get encrypted model params for chat function
  const paramRes = await post('/api/ide/v1/get_detail_param', {
    function: 'chat',
    need_prompt: true,
    poly_prompt: true
  });
  const params = JSON.parse(paramRes.body);
  const model = params.config_info_list.find(m => m.config_name === 'deepseek-V3.1') 
    || params.config_info_list.find(m => m.config_name === 'doubao-for-auto')
    || params.config_info_list.find(m => !m.config_name.includes('fast_apply') && !m.config_name.includes('summary') && !m.config_name.includes('context') && !m.config_name.includes('title') && !m.config_name.includes('input_opt') && !m.config_name.includes('custom_model') && m.config_switch);
  const enc = model.model_detail_list[0].encrypted_model_params;
  console.log('Using model:', model.config_name, '(', model.model_detail_list[0].model_name, ')');
  console.log('All models:', params.config_info_list.map(m => m.config_name).join(', '));

  // Test different agent_type values  
  const agentTypes = ['chat', 'builder', 'builder_with_mcp', 'inline_chat', 'solo_coder'];
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
      user_input: {
        id: genId(),
        content: 'say hello'
      }
    };
    const r = await post('/api/agent/v3/create_agent_task', body);
    const snippet = r.body.substring(0, 400).replace(/\n/g, ' | ');
    console.log(`${at}: ${r.status} -> ${snippet}`);
    console.log('');
  }
}

main().catch(console.error);
