const fs = require('fs');
const f = 'D:\\Trae CN\\resources\\app\\out\\vs\\workbench\\workbench.desktop.main.js';
const text = fs.readFileSync(f, 'utf8');

// Search for agent_type in the JS
const idx = text.indexOf('agent_type');
let count = 0;
let pos = 0;
while (pos !== -1 && count < 10) {
  pos = text.indexOf('agent_type', pos);
  if (pos === -1) break;
  const ctx = text.substring(Math.max(0, pos - 80), pos + 120);
  console.log(`[${pos}]: ...${ctx}...`);
  console.log();
  pos++;
  count++;
}
