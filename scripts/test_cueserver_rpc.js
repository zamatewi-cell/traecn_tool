const path = require('path');

async function main() {
  process.env.NODE_PATH = [
    process.env.NODE_PATH || '',
    'D:/Trae CN/resources/app/node_modules',
  ]
    .filter(Boolean)
    .join(path.delimiter);
  require('module').Module._initPaths();

  const ipc = require('@aha-kit/ipc');
  const { Connection } = require('@aha-kit/rpc');

  const transport = await ipc.connect('cueServer');
  console.log('connected:', !!transport);

  const rpc = new Connection({
    send: (message) => transport.send(message),
    onData: (onMessage) => {
      transport.on('message', onMessage);
      return {
        dispose: () => transport.off('message', onMessage),
      };
    },
  });

  const ping = await rpc.sendRequest('ping', {});
  console.log('ping:', JSON.stringify(ping));

  // Try a known request envelope shape from process manager internals.
  const channelId = `probe-${Date.now()}`;
  const req = {
    packet_type: 'request',
    session_id: '',
    channel_id: channelId,
    params: {
      service: 'healthcheck',
      method: 'ping',
      data: '',
      common_params: {},
      user_info: {
        name: '',
        token: '',
        region: '',
        is_internal: false,
        user_id: '',
        scope: '',
        tenant_id: undefined,
      },
      streamlined_common_params: {},
      client_info: {
        connect_session_id: '',
      },
    },
  };

  const res = await rpc.sendRequest('request', req);
  console.log('request healthcheck.ping:', JSON.stringify(res));

  await transport.disconnect();
}

main().catch((err) => {
  console.error('ERR:', err && err.stack ? err.stack : err);
  process.exit(1);
});
