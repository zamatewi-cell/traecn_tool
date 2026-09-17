const fs = require('fs');

const code = fs.readFileSync('D:/Trae CN/resources/app/extensions/ai-completion/resource/aiserver/cueMain.js', 'utf8');

let idx = code.indexOf('CommonApi=class');
if (idx === -1) idx = code.indexOf('CommonApi = class');
if (idx === -1) idx = code.indexOf('class CommonApi');
if (idx === -1) {
  // search regex
  const m = code.match(/\w+\.CommonApi\s*=\s*class/);
  if (m) idx = m.index;
}

console.log('Index:', idx);
if (idx !== -1) {
  console.log(code.substring(idx, idx + 1000));
}

