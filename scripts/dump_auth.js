// Dump auth info fields
const fs = require('fs');
const storagePath = process.env.APPDATA + '\\Trae CN\\User\\globalStorage\\storage.json';
const d = JSON.parse(fs.readFileSync(storagePath, 'utf8'));
const a = JSON.parse(d['iCubeAuthInfo://icube.cloudide']);
console.log('Auth fields:', Object.keys(a));
for (const k of Object.keys(a)) {
  const v = String(a[k]);
  console.log(`  ${k}: ${v.substring(0, 80)}${v.length > 80 ? '...' : ''}`);
}
