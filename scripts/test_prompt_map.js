// Test sending create_agent_task with custom prompt_template_map
// containing actual rendered prompt text instead of encrypted prompts
const https = require('https');
const fs = require('fs');
const crypto = require('crypto');

// Read token
const storage = JSON.parse(fs.readFileSync(
  process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json', 'utf8'
));
const auth = JSON.parse(storage['iCubeAuthInfo://icube.cloudide']);
const token = auth.token;

// Read captured headers to match exact format
const capturedHeaders = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));

const sessionId = crypto.randomBytes(12).toString('hex');
const taskId = crypto.randomUUID();

// Minimal system prompt - try to match what single_chat master_agent would contain
const systemPrompt = `You are a helpful AI coding assistant. Answer the user's questions concisely and helpfully.`;
const userMessage = "hello, what is 1+1?";

// Build the body based on what we know about the structure
// The key insight: prompt_template_map should contain RENDERED (not encrypted) prompts
function makeBody(testName, body) {
  return { testName, body };
}

const tests = [
  // Test 1: prompt_template_map with master_agent key
  makeBody("prompt_template_map with master_agent", {
    function: "chat_v3",
    agent_type: "builder_v3",
    model_name: "minimax-m2.5",
    session_id: sessionId,
    task_id: taskId,
    version_code: 20260212,
    user_input: userMessage,
    prompt_template_map: {
      "master_agent": systemPrompt,
      "user_input": userMessage,
    },
    history_list: [],
    settings: {},
  }),

  // Test 2: prompt_template_map with full prompt key names
  makeBody("prompt_template_map with full keys", {
    function: "chat_v3",
    agent_type: "builder_v3",
    model_name: "minimax-m2.5",
    session_id: crypto.randomBytes(12).toString('hex'),
    task_id: crypto.randomUUID(),
    version_code: 20260212,
    user_input: userMessage,
    prompt_template_map: {
      "trae_agent_solo_coder_dev_chat.default.master_agent": systemPrompt,
      "trae_agent_solo_coder_dev.default.compact_prompt": "",
      "trae_agent_solo_coder_dev.default.tools": "[]",
      "trae_agent_solo_coder_dev.default.misc": "{}",
    },
    history_list: [],
    settings: {},
  }),

  // Test 3: user_input_prompt as separate field
  makeBody("user_input_prompt field", {
    function: "chat_v3",
    agent_type: "builder_v3",
    model_name: "minimax-m2.5",
    session_id: crypto.randomBytes(12).toString('hex'),
    task_id: crypto.randomUUID(),
    version_code: 20260212,
    user_input: userMessage,
    user_input_prompt: userMessage,
    prompt_template_map: {
      "master_agent": systemPrompt,
    },
    history_list: [],
    settings: {},
  }),

  // Test 4: Use single_chat agent_type instead of builder_v3
  makeBody("agent_type=single_chat", {
    function: "chat_v3",
    agent_type: "single_chat",
    model_name: "minimax-m2.5",
    session_id: crypto.randomBytes(12).toString('hex'),
    task_id: crypto.randomUUID(),
    version_code: 20260212,
    user_input: userMessage,
    prompt_template_map: {
      "master_agent": systemPrompt,
      "user_input": userMessage,
    },
    history_list: [],
    settings: {},
  }),

  // Test 5: function_config + model_config wrapper with prompts
  makeBody("function_config wrapper + prompts", {
    function: "chat_v3",
    agent_type: "builder_v3",
    session_id: crypto.randomBytes(12).toString('hex'),
    task_id: crypto.randomUUID(),
    version_code: 20260212,
    user_input: userMessage,
    function_config: {
      function: "chat_v3",
    },
    model_config: {
      model_name: "minimax-m2.5",
      config_name: "minimax-m2.5",
    },
    prompt_template_map: {
      "master_agent": systemPrompt,
      "user_input": userMessage,
      "trae_agent_single_chat.default.master_agent": systemPrompt,
      "trae_agent_solo_coder_dev_chat.default.master_agent": systemPrompt,
    },
    history_list: [],
    settings: {},
  }),

  // Test 6: Try trae_agent_single_chat agent type with matching prompt key
  makeBody("agent_type=trae_agent_single_chat", {
    function: "chat_v3",
    agent_type: "trae_agent_single_chat",
    model_name: "minimax-m2.5",
    session_id: crypto.randomBytes(12).toString('hex'),
    task_id: crypto.randomUUID(),
    version_code: 20260212,
    user_input: userMessage,
    prompt_template_map: {
      "master_agent": systemPrompt,
    },
    history_list: [],
    settings: {},
  }),
];

async function sendTest(test) {
  const bodyStr = JSON.stringify(test.body);
  console.log(`\n=== ${test.testName} ===`);
  console.log(`Body size: ${bodyStr.length} bytes`);

  return new Promise((resolve) => {
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru',
      path: '/api/agent/v3/create_agent_task',
      method: 'POST',
      headers: {
        'content-type': 'application/json',
        'x-app-id': capturedHeaders['x-app-id'],
        'x-app-version': capturedHeaders['x-app-version'],
        'x-app-version-code': capturedHeaders['x-app-version-code'],
        'x-device-id': capturedHeaders['x-device-id'],
        'x-device-type': capturedHeaders['x-device-type'],
        'x-ide-token': capturedHeaders['x-ide-token'],
        'x-ide-version': capturedHeaders['x-ide-version'],
        'x-ide-version-code': capturedHeaders['x-ide-version-code'],
        'x-ide-version-type': capturedHeaders['x-ide-version-type'],
        'x-machine-id': capturedHeaders['x-machine-id'],
        'x-request-id': 'req_' + crypto.randomUUID(),
        'x-trae-request-id': crypto.randomUUID(),
        'x-requested-at': Math.floor(Date.now() / 1000).toString(),
        'package-type': 'stable_cn',
        'app-version': capturedHeaders['app-version'],
        'user-agent': capturedHeaders['user-agent'],
      },
    }, (res) => {
      let data = '';
      res.on('data', (chunk) => {
        data += chunk.toString();
        // Print first error event immediately
        const lines = data.split('\n');
        for (const line of lines) {
          if (line.startsWith('event:error') || line.startsWith('data:{"code":')) {
            // Extract error
          }
        }
      });
      res.on('end', () => {
        // Parse SSE events
        const lines = data.split('\n');
        let events = [];
        let currentEvent = {};
        for (const line of lines) {
          if (line.startsWith('event:')) currentEvent.event = line.substring(6);
          if (line.startsWith('data:')) currentEvent.data = line.substring(5);
          if (line === '' && currentEvent.event) {
            events.push({ ...currentEvent });
            currentEvent = {};
          }
        }
        if (currentEvent.event) events.push(currentEvent);
        
        for (const ev of events) {
          if (ev.event === 'error') {
            console.log(`[ERROR] ${ev.data}`);
          } else if (ev.event === 'task_created') {
            console.log(`[OK] task_created: ${ev.data}`);
          } else if (ev.event === 'text_delta') {
            console.log(`[TEXT] ${ev.data.substring(0, 100)}`);
          } else if (ev.event === 'timing_cost') {
            console.log(`[TIMING] ${ev.data.substring(0, 100)}`);
          } else {
            console.log(`[${ev.event}] ${(ev.data || '').substring(0, 80)}`);
          }
        }
        resolve();
      });
    });
    req.on('error', (e) => {
      console.log(`[NET ERROR] ${e.message}`);
      resolve();
    });
    req.write(bodyStr);
    req.end();
  });
}

(async () => {
  for (const test of tests) {
    await sendTest(test);
  }
})();
