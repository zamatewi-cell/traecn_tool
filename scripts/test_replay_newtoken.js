const fs = require('fs');
const https = require('https');
const crypto = require('crypto');

// Load latest token from Trae local storage
const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
const d = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
const a = JSON.parse(d['iCubeAuthInfo://icube.cloudide']);
const latestToken = a.token;

const h = JSON.parse(fs.readFileSync('scripts/captured/agent_task_req_headers.json', 'utf8'));
const encBody = fs.readFileSync('scripts/captured/agent_task_req_body.bin');

const allHeaders = { ...h };
allHeaders['x-ide-token'] = latestToken;
delete allHeaders['accept-encoding'];
delete allHeaders['content-length'];
delete allHeaders['sec-fetch-dest'];
delete allHeaders['sec-fetch-mode'];
delete allHeaders['sec-fetch-site'];

const req = https.request({
  hostname: 'trae-api-cn.mchost.guru',
  path: '/api/agent/v3/create_agent_task',
  method: 'POST',
  headers: allHeaders,
}, res => {
  let d = Buffer.alloc(0);
  res.on('data', c => { d = Buffer.concat([d, c]); });
  res.on('end', () => {
    console.log('Status:', res.statusCode);
    console.log('Response (' + d.length + ' bytes):');
    console.log(d.toString('utf-8').substring(0, 500));
  });
});

req.on('error', e => console.log('Error:', e.message));
req.write(encBody);
req.end();

