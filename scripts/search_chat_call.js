const fs = require('fs');
const f = 'D:\\Trae CN\\resources\\app\\out\\vs\\workbench\\workbench.desktop.main.js';
const text = fs.readFileSync(f, 'utf8');

// Search for service:"chat" calls
const patterns = ['service:"chat"', "service:'chat'", 'service:"chat",method:"chat"'];
for (const p of patterns) {
  let pos = 0;
  let count = 0;
  while (count < 10) {
    pos = text.indexOf(p, pos);
    if (pos === -1) break;
    const ctx = text.substring(Math.max(0, pos - 200), Math.min(text.length, pos + 500));
    console.log(`\n=== ${p} at ${pos} ===`);
    console.log(ctx.substring(0, 500));
    pos++;
    count++;
  }
}

// Also search for the data passed with chat calls
console.log('\n\n=== Search for chat data construction ===');
const chatDataPatterns = ['executeStreamRequest({service:"chat"', 'executeStreamRequest({service:\"chat\"'];
for (const p of chatDataPatterns) {
  let pos = text.indexOf(p);
  if (pos !== -1) {
    const ctx = text.substring(pos, Math.min(text.length, pos + 800));
    console.log(`\n[${pos}]: ${ctx.substring(0, 600)}`);
  }
}
