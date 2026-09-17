// Simple MITM proxy to capture Trae CN API requests
// Usage: 
// 1. Run this script
// 2. Set HTTPS_PROXY=http://127.0.0.1:8888 before starting Trae CN
// 3. Chat in Trae CN  
// 4. Check captured_requests/ folder for the full request bodies

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');

const outDir = path.join(__dirname, 'captured_requests');
if (!fs.existsSync(outDir)) fs.mkdirSync(outDir);

let reqCount = 0;

const proxy = http.createServer();

proxy.on('request', (clientReq, clientRes) => {
  // Regular HTTP proxy request
  const url = new URL(clientReq.url);
  console.log(`[HTTP] ${clientReq.method} ${url.hostname}${url.pathname}`);
  
  const options = {
    hostname: url.hostname,
    port: url.port || 80,
    path: url.pathname + url.search,
    method: clientReq.method,
    headers: clientReq.headers
  };
  delete options.headers['proxy-connection'];
  
  const proxyReq = http.request(options, (proxyRes) => {
    clientRes.writeHead(proxyRes.statusCode, proxyRes.headers);
    proxyRes.pipe(clientRes);
  });
  clientReq.pipe(proxyReq);
});

proxy.on('connect', (req, clientSocket, head) => {
  // HTTPS CONNECT tunnel
  const [hostname, port] = req.url.split(':');
  console.log(`[CONNECT] ${hostname}:${port}`);
  
  const serverSocket = require('net').connect(parseInt(port) || 443, hostname, () => {
    clientSocket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
    
    // For trae-api-cn.mchost.guru, we want to intercept
    // For now, just forward transparently and capture at TCP level
    if (hostname.includes('trae-api')) {
      console.log(`[INTERCEPT] Will capture traffic to ${hostname}`);
      // We can't easily MITM without cert, so let's use a different approach
      // Just note that traffic is flowing
    }
    
    serverSocket.write(head);
    serverSocket.pipe(clientSocket);
    clientSocket.pipe(serverSocket);
  });
  
  serverSocket.on('error', (err) => {
    console.error(`[ERROR] ${hostname}: ${err.message}`);
    clientSocket.end();
  });
});

proxy.listen(8888, '127.0.0.1', () => {
  console.log('Proxy listening on http://127.0.0.1:8888');
  console.log('Set HTTPS_PROXY=http://127.0.0.1:8888 before starting Trae CN');
});
