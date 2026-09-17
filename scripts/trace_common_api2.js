const fs = require('fs');

const code = fs.readFileSync('D:/Trae CN/resources/app/extensions/ai-completion/resource/aiserver/cueMain.js', 'utf8');

let idx = 0;
while (true) {
  idx = code.indexOf('.CommonApi', idx);
  if (idx === -1) break;
  console.log(`\n=== Offset ${idx} ===`);
  console.log(code.substring(Math.max(0, idx - 100), Math.min(code.length, idx + 200)));
  idx += 10;
}

