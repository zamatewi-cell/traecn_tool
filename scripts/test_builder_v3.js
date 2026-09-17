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
  // Try different function names for get_detail_param
  const funcs = ['builder_v3', 'builder', 'dev', 'side_chat', 'agentic_chat'];
  for (const fn of funcs) {
    const r = await post('/api/ide/v1/get_detail_param', {
      function: fn,
      need_prompt: true,
      poly_prompt: true
    });
    const body = JSON.parse(r.body);
    const models = body.config_info_list || [];
    const promptCount = body.metadata?.encrypted_prompt_set?.length || 0;
    const promptKeys = (body.metadata?.encrypted_prompt_set || []).map(p => p.prompt_key).join(', ');
    console.log(`function=${fn}: ${models.length} models, ${promptCount} prompts`);
    if (models.length > 0) {
      console.log(`  models: ${models.map(m => m.config_name).join(', ')}`);
    }
    if (promptCount > 0) {
      console.log(`  prompt keys: ${promptKeys.substring(0, 300)}`);
    }
    
    // If we got models, try create_agent_task with builder_v3
    if (models.length > 0 && fn === 'builder_v3') {
      const model = models.find(m => !m.config_name.includes('fast_apply') && !m.config_name.includes('summary') && !m.config_name.includes('context') && !m.config_name.includes('title') && !m.config_name.includes('input_opt') && !m.config_name.includes('custom_model') && m.config_switch) || models[0];
      const enc = model.model_detail_list[0].encrypted_model_params;
      const prompts = body.metadata?.encrypted_prompt_set || [];
      
      console.log(`\n--- Testing builder_v3 with ${model.config_name} ---`);
      const taskBody = {
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
        function: fn,
        agent_type: 'builder_v3',
        ide_version: '3.3.37',
        encrypted_prompt_set: prompts,
        user_input: {
          id: genId(),
          content: 'say hello in one word'
        }
      };
      console.log('Body size:', JSON.stringify(taskBody).length);
      const r2 = await post('/api/agent/v3/create_agent_task', taskBody);
      console.log('Status:', r2.status);
      console.log('Response:', r2.body.substring(0, 2000));
    }
    console.log('');
  }
}

main().catch(console.error);
