const fs = require('fs');

const code = fs.readFileSync('D:/Trae CN/resources/app/extensions/ai-completion/resource/aiserver/cueMain.js', 'utf8');

['masticate', 'masticateVegetablesToPulp', '6195f24ca4d430f8', 'x-request-pin', 'X-Request-Pin'].forEach(k => {
  let idx = 0;
  while (true) {
    idx = code.indexOf(k, idx);
    if (idx === -1) break;
    console.log(`\n=== Keyword ${k} at ${idx} ===`);
    console.log(code.substring(Math.max(0, idx - 100), Math.min(code.length, idx + 300)));
    idx += k.length;
  }
});

