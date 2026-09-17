// Iteratively discover all required fields for create_agent_task
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));

const sessionId = crypto.randomBytes(12).toString('hex');
const conversationId = crypto.randomBytes(12).toString('hex');

const body = {
  function: "chat_v3",
  agent_type: "builder_v3",
  model_name: "minimax-m2.5",
  config_name: "minimax-m2.5",
  session_id: sessionId,
  conversation_id: conversationId,
  task_id: crypto.randomUUID(),
  version_code: 20260212,
  user_id: "4355622541471866",
  device_id: "2262131830954826",
  ide_version: "3.3.37",
  ide_version_code: "20260212",
  ide_version_type: "stable",
  app_id: "6eefa01c-1036-4c7e-9ca5-d891f63bfcd8",
  machine_id: h['x-machine-id'],
  os_version: "Windows 11",
  user_input: {
    id: crypto.randomUUID(),
    content: "hello, what is 1+1?",
    type: "text",
  },
  prompt_template_map: {
    "master_agent": "You are a helpful AI assistant. Answer concisely.",
  },
  history_list: [],
  settings: {},
};

function send(body) {
  const bodyStr = JSON.stringify(body);
  console.log('Sending body (' + bodyStr.length + ' bytes)...');
  
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
    }, res => {
      let d = '';
      res.on('data', c => { d += c; });
      res.on('end', () => {
        console.log('Status:', res.statusCode);
        // Parse response
        if (res.statusCode === 400) {
          try {
            const err = JSON.parse(d);
            console.log('ERROR:', err.message);
            // Extract the missing field name
            const match = err.message.match(/expr_path=([^,]+)/);
            if (match) {
              console.log('MISSING FIELD:', match[1]);
            }
          } catch {
            console.log(d.substring(0, 500));
          }
        } else {
          // SSE response - parse events
          const lines = d.split('\n');
          for (const line of lines) {
            if (line.trim()) console.log(line.substring(0, 200));
          }
        }
        r(d);
      });
    });
    req.on('error', e => { console.log('Error:', e.message); r(''); });
    req.write(bodyStr);
    req.end();
  });
}

(async () => {
  let resp = await send(body);
  // Keep fixing missing fields
  for (let i = 0; i < 20; i++) {
    try {
      const err = JSON.parse(resp);
      const match = err.message?.match(/expr_path=([^,]+)/);
      if (!match) break;
      const field = match[1];
      console.log(`\n--- Adding field: ${field} ---`);
      
      // Add field based on name/path
      const parts = field.split('.');
      let target = body;
      for (let j = 0; j < parts.length - 1; j++) {
        if (!target[parts[j]]) target[parts[j]] = {};
        target = target[parts[j]];
      }
      const leaf = parts[parts.length - 1];
      
      // Guess value based on field name
      if (leaf.includes('id') && !target[leaf]) {
        target[leaf] = crypto.randomUUID();
      } else if (leaf.includes('version')) {
        target[leaf] = "3.3.37";
      } else if (leaf.includes('type')) {
        target[leaf] = "text";
      } else if (leaf.includes('name')) {
        target[leaf] = "";
      } else if (leaf.includes('content')) {
        target[leaf] = "";
      } else if (leaf.includes('list') || leaf.includes('array')) {
        target[leaf] = [];
      } else if (!target[leaf]) {
        target[leaf] = "";
      }
      
      body.task_id = crypto.randomUUID(); // Fresh task_id each time
      resp = await send(body);
    } catch {
      break;
    }
  }
  
  console.log('\n\n=== FINAL BODY ===');
  console.log(JSON.stringify(body, null, 2));
  
  setTimeout(() => process.exit(0), 1000);
})();
