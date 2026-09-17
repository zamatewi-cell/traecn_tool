const { ZmqClient } = require('D:/Trae CN/resources/app/node_modules/@byted-icube/trae-network-client');

const ep = process.argv[2];
if (!ep) {
  console.error('missing endpoint');
  process.exit(2);
}

(async () => {
  console.log(`PROBE ${ep}`);
  try {
    const c = new ZmqClient(ep);
    const pong = await Promise.race([
      c.ping(),
      new Promise((_, reject) => setTimeout(() => reject(new Error('ping timeout after 5000ms')), 5000)),
    ]);
    console.log(`OK ${ep} ${JSON.stringify(pong)}`);
    process.exit(0);
  } catch (err) {
    const msg = err && err.message ? err.message : String(err);
    console.log(`FAIL ${ep} ${msg}`);
    process.exit(1);
  }
})();
