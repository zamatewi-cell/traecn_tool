const fs = require('fs');
const https = require('https');
const crypto = require('crypto');
const os = require('os');

const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
const d = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
const a = JSON.parse(d['iCubeAuthInfo://icube.cloudide']);
const token = a.token;
const userId = a.userId;

function post(path, body) {
  return new Promise((resolve) => {
    const bodyStr = JSON.stringify(body);
    const reqId = crypto.randomUUID();
    const req = https.request({
      hostname: 'trae-api-cn.mchost.guru', path, method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(bodyStr),
        'X-IDE-Token': token,
        'X-Request-ID': reqId,
        'X-Trae-Request-ID': reqId,
        'x-app-id': '6eefa01c-1036-4c7e-9ca5-d891f63bfcd8',
        'x-app-version': 'default',
        'x-ide-version-code': '20260212',
        'x-app-version-code': '20260212',
        'x-device-brand': os.hostname(),
        'x-device-cpu': 'Intel',
        'x-device-id': '2262131830954826',
        'x-machine-id': crypto.createHash('sha256').update(os.hostname()).digest('hex'),
        'x-os-version': 'Windows',
        'x-device-type': 'windows',
        'x-ide-version': '3.3.37',
        'x-ide-version-type': 'stable',
        'request-traffic-type': 'prod'
      }
    }, res => {
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks).toString() }));
    });
    req.on('error', e => resolve({ status: 0, error: e.message }));
    req.write(bodyStr);
    req.end();
  });
}

async function main() {
  // Dump full get_detail_param response to file
  const funcs = ['chat', 'solo_coder', 'builder_with_mcp'];
  for (const fn of funcs) {
    const r = await post('/api/ide/v1/get_detail_param', {
      function: fn,
      need_prompt: true,
      poly_prompt: true,
      agent_type: fn
    });
    const body = JSON.parse(r.body);
    const outFile = `scripts/detail_param_${fn}.json`;
    fs.writeFileSync(outFile, JSON.stringify(body, null, 2));
    console.log(`${fn}: ${r.status}, keys: ${Object.keys(body)}, file size: ${JSON.stringify(body).length}, saved to ${outFile}`);
    
    // Show metadata
    if (body.metadata) {
      console.log(`  metadata keys: ${Object.keys(body.metadata)}`);
      const md = JSON.stringify(body.metadata);
      console.log(`  metadata (first 500): ${md.substring(0, 500)}`);
    }
    
    // Show any extra top-level keys
    for (const k of Object.keys(body)) {
      if (k !== 'config_info_list' && k !== 'metadata' && k !== 'allow_tenant_user_add_model') {
        console.log(`  ${k}: ${JSON.stringify(body[k]).substring(0, 200)}`);
      }
    }
  }
}

main().catch(console.error);
