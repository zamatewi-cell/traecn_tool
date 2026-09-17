const fs = require('fs');

const code = fs.readFileSync('D:/Trae CN/resources/app/extensions/ai-completion/resource/aiserver/server.js', 'utf8');

console.log(code.substring(9507500, 9511500));

