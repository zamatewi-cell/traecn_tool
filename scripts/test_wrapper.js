// Test with function_config + model_config wrapper and fresh session
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

function postSSE(path, body, timeout = 30000) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    let timer = setTimeout(() => {
      req.destroy();
      resolve({ status: 0, error: 'timeout' });
    }, timeout);
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
            } else if (val.length > 500) {
              process.stdout.write(`[data: ${val.substring(0, 500)}...]`);
            } else {
              process.stdout.write(`[data: ${val}]`);
            }
          }
        }
        data = lines[lines.length - 1];
      });
      res.on('end', () => {
        clearTimeout(timer);
        resolve({ status: res.statusCode, fullData });
      });
    });
    req.on('error', e => {
      clearTimeout(timer);
      resolve({ status: 0, error: e.message });
    });
    req.write(bodyStr);
    req.end();
  });
}

async function main() {
  const params = JSON.parse(fs.readFileSync('scripts/detail_param_chat_v3.json', 'utf8'));
  const metadata = params.metadata;
  
  // Try multiple models
  const modelNames = ['deepseek-V3.1', 'qwen-3.5', 'doubao-for-auto'];
  
  for (const modelName of modelNames) {
    const model = params.config_info_list.find(m => m.config_name === modelName);
    if (!model) { console.log(`Model ${modelName} not found, skipping`); continue; }
    const detail = model.model_detail_list[0];
    
    // Build prompt template map from encrypted prompts
    const promptTemplateMap = {};
    const promptConfigs = detail.prompt_config_list || [];
    for (const pc of promptConfigs) {
      const ep = metadata.encrypted_prompt_set.find(e => e.prompt_key === pc.prompt_key);
      if (ep) promptTemplateMap[ep.prompt_key] = ep.prompt_template;
    }
    
    const encryptedPromptList = promptConfigs.map(pc => {
      const ep = metadata.encrypted_prompt_set.find(e => e.prompt_key === pc.prompt_key);
      return ep ? { prompt_key: ep.prompt_key, prompt_label: ep.prompt_label, prompt_version: ep.prompt_version, prompt_template: ep.prompt_template } : null;
    }).filter(Boolean);
    
    console.log(`\n=== Model: ${modelName} (model_name: ${detail.model_name}) ===`);
    console.log(`Prompt configs: ${promptConfigs.length}, Mapped prompts: ${encryptedPromptList.length}`);
    
    // Fresh IDs for each test
    const sessionId = genId();
    const body = {
      conversation_id: genId(),
      session_id: sessionId,
      task_id: genId(),
      message_id: genId(),
      user_id: userId,
      device_id: '2262131830954826',
      model_name: modelName,         // use config_name, NOT model_name
      config_name: modelName,
      encrypted_model_params: detail.encrypted_model_params,
      stream: true,
      function: 'chat_v3',
      agent_type: 'builder',
      ide_version: '3.3.37',
      function_config: {
        prompt_template_map: promptTemplateMap,
      },
      model_config: {
        encrypted_model_params: detail.encrypted_model_params,
        model_extra_config: detail.model_extra_config,
        prompt_max_tokens: detail.prompt_max_tokens,
        max_turn: detail.max_turn,
        encrypted_prompt_list: encryptedPromptList,
        prompt_config_list: promptConfigs,
      },
      user_input: {
        id: genId(),
        content: 'say hello in one word'
      }
    };
    console.log('Body size:', JSON.stringify(body).length, 'bytes');
    const r = await postSSE('/api/agent/v3/create_agent_task', body, 60000);
    console.log('\nStatus:', r.status);
    
    // If we got a good response, save it
    if (r.fullData && !r.fullData.includes('"code":')) {
      fs.writeFileSync(`scripts/response_${modelName}.txt`, r.fullData);
      console.log('Saved response!');
    }
  }
}

main().catch(console.error);
