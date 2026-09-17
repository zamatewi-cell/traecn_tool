const { ZmqClient } = require('D:/Trae CN/resources/app/node_modules/@byted-icube/trae-network-client');

function getCandidates() {
  const pids = [20908, 8624, 8240, 23384, 2564, 24236, 15404, 11140, 23588, 6788, 23696, 1664, 12320, 13360, 15744, 27404, 10104];
  const endpoints = new Set();

  // Observed local listeners from netstat.
  for (const port of [51000, 49941, 51495, 63523, 17788]) {
    endpoints.add(`tcp://127.0.0.1:${port}`);
  }

  // Guessed pipe patterns from native binary strings.
  endpoints.add('ipc://\\\\.\\pipe\\trae-network-service-ipc');
  endpoints.add('ipc://\\\\.\\pipe\\trae-network-service-ipc-main');
  endpoints.add('ipc://\\\\.\\pipe\\trae-network-service-ipc-default');

  for (const pid of pids) {
    endpoints.add(`ipc://\\\\.\\pipe\\trae-network-service-ipc-${pid}`);
    endpoints.add(`ipc://\\\\.\\pipe\\trae-network-service-ipc-${String(pid).slice(-4)}`);
  }

  return [...endpoints];
}

async function testEndpoint(ep) {
  try {
    const client = new ZmqClient(ep);
    await client.ping();
    return { ok: true, ep };
  } catch (err) {
    const msg = err && err.message ? err.message : String(err);
    return { ok: false, ep, msg };
  }
}

(async () => {
  const candidates = getCandidates();
  console.log(`Probing ${candidates.length} endpoints...`);

  const oks = [];
  const fails = [];

  for (const ep of candidates) {
    const r = await testEndpoint(ep);
    if (r.ok) {
      oks.push(r);
      console.log(`OK   ${ep}`);
    } else {
      fails.push(r);
      // Keep output concise but useful.
      if (r.msg.includes('Connect failed') || r.msg.includes('No endpoints') || r.msg.includes('timeout')) {
        console.log(`FAIL ${ep} -> ${r.msg}`);
      } else {
        console.log(`FAIL ${ep}`);
      }
    }
  }

  console.log('\n=== SUMMARY ===');
  console.log(`OK: ${oks.length}`);
  console.log(`FAIL: ${fails.length}`);

  if (oks.length === 0) {
    console.log('\nNo working endpoint found from heuristic candidates.');
  }
})();
