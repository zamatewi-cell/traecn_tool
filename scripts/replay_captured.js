// Replay the EXACT captured encrypted body with aha transport header
const https = require('https');
const fs = require('fs');
const h = JSON.parse(fs.readFileSync(__dirname + '/captured/agent_task_req_headers.json', 'utf8'));
const encBody = fs.readFileSync(__dirname + '/captured/agent_task_req_body.bin');

console.log('Captured body size:', encBody.length, 'bytes');
console.log('Transport header:', h['x-bridge-transport']);

// Replay with ALL original headers - pass them verbatim
const allHeaders = { ...h };
// Remove hop-by-hop headers that shouldn't be forwarded
delete allHeaders['accept-encoding'];
delete allHeaders['content-length']; // will be set automatically
delete allHeaders['sec-fetch-dest'];
delete allHeaders['sec-fetch-mode'];
delete allHeaders['sec-fetch-site'];

console.log('Headers being sent:');
for (const [k, v] of Object.entries(allHeaders)) {
  console.log(`  ${k}: ${String(v).substring(0, 80)}`);
}

const req = https.request({
  hostname: 'trae-api-cn.mchost.guru',
  path: '/api/agent/v3/create_agent_task',
  method: 'POST',
  headers: allHeaders,
}, res => {
  let d = Buffer.alloc(0);
  res.on('data', c => { d = Buffer.concat([d, c]); });
  res.on('end', () => {
    console.log('\nStatus:', res.statusCode);
    console.log('Response headers:');
    for (const [k, v] of Object.entries(res.headers)) {
      console.log(`  ${k}: ${v}`);
    }
    console.log('\nResponse (' + d.length + ' bytes):');
    // Try to decode as text
    const text = d.toString('utf-8');
    const lines = text.split('\n');
    for (const line of lines) {
      if (line.trim()) console.log(line.substring(0, 400));
    }
  });
});

req.on('error', e => console.log('Error:', e.message));
req.write(encBody);
req.end();
