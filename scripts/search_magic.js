const fs = require('fs');
const path = require('path');

const targetHex = "f3c1c037";
const targetHexLE = "37c0c1f3";
const targetDec1 = "4089561143"; // 0xf3c1c037 unsigned
const targetDec2 = "-205406153"; // 0xf3c1c037 signed 32-bit
const targetDecLE = "935379443";  // 0x37c0c1f3

function checkBuffer(buf, file) {
  const hex = buf.toString('hex');
  if (hex.includes(targetHex)) {
    console.log(`[HEX MATCH BE] in ${file} at hex offset ${hex.indexOf(targetHex)/2}`);
  }
  if (hex.includes(targetHexLE)) {
    console.log(`[HEX MATCH LE] in ${file} at hex offset ${hex.indexOf(targetHexLE)/2}`);
  }
  const str = buf.toString('utf8');
  for (const t of [targetDec1, targetDec2, targetDecLE, '0xf3', '0xF3', '408956', 'x-bridge-transport']) {
    if (str.includes(t)) {
      if (t === '0xf3' || t === '0xF3') {
        // filter out trivial matches
      } else {
        console.log(`[STR MATCH: ${t}] in ${file}`);
      }
    }
  }
}

function walk(dir) {
  try {
    const list = fs.readdirSync(dir);
    for (const item of list) {
      const p = path.join(dir, item);
      const s = fs.statSync(p);
      if (s.isDirectory()) {
        walk(p);
      } else {
        const buf = fs.readFileSync(p);
        checkBuffer(buf, p);
      }
    }
  } catch(e) {}
}

console.log("Walking D:/Trae CN/resources/app ...");
walk("D:/Trae CN/resources/app");
console.log("Done walking.");
