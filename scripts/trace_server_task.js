const fs = require('fs');

const code = fs.readFileSync('D:/Trae CN/resources/app/extensions/ai-completion/resource/aiserver/server.js', 'utf8');

let idx = 0;
while (true) {
  idx = code.indexOf('create_agent_task', idx);
  if (idx === -1) break;
  console.log(`\n=== Offset ${idx} ===`);
  console.log(code.substring(Math.max(0, idx - 300), Math.min(code.length, idx + 500)));
  idx += 17;
}

