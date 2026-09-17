const { app, BrowserWindow, ipcMain, shell } = require('electron');
const path = require('path');
const fs = require('fs');
const http = require('http');
const crypto = require('crypto');
const os = require('os');
const { spawn } = require('child_process');

let mainWindow = null;
let proxyProcess = null;

const DATA_DIR = path.join(app.getPath('userData'));
const DATA_FILE = path.join(DATA_DIR, 'traecn-data.json');
const PROXY_CONFIG_FILE = path.join(DATA_DIR, 'proxy-config.json');

// ===== Data Store =====
function loadData() {
  try {
    if (fs.existsSync(DATA_FILE)) {
      return JSON.parse(fs.readFileSync(DATA_FILE, 'utf8'));
    }
  } catch (e) {
    console.error('Failed to load data:', e);
  }
  return {
    accounts: [],
    proxyConfig: {
      listenPort: 8045,
      requestTimeout: 120,
      autoStart: false,
      allowLan: false,
      authEnabled: true,
      authMode: 'auto',
      apiKey: generateApiKey(),
      webUiPassword: '',
      userAgentOverride: false,
      userAgentValue: '',
    },
    settings: {
      language: 'zh',
      theme: 'dark',
    },
  };
}

function saveData(data) {
  try {
    if (!fs.existsSync(DATA_DIR)) {
      fs.mkdirSync(DATA_DIR, { recursive: true });
    }
    fs.writeFileSync(DATA_FILE, JSON.stringify(data, null, 2), 'utf8');
  } catch (e) {
    console.error('Failed to save data:', e);
  }
}

function generateApiKey() {
  return 'sk-' + crypto.randomBytes(24).toString('hex');
}

// ===== Device Fingerprint =====
function generateDeviceId() {
  return String(Math.floor(1e15 + Math.random() * 9e15));
}

function generateMachineId() {
  const hostname = os.hostname();
  return crypto.createHash('sha256').update(hostname + 'traecn-tools-salt').digest('hex');
}

function generateMacMachineId() {
  return crypto.randomUUID();
}

function generateDevDeviceId() {
  return crypto.randomUUID();
}

function generateSqmId() {
  return '{' + crypto.randomUUID().toUpperCase() + '}';
}

function generateFingerprint() {
  return {
    machineId: 'auth0|user_' + crypto.randomBytes(16).toString('base64url'),
    macMachineId: generateMacMachineId(),
    devDeviceId: generateDevDeviceId(),
    sqmId: generateSqmId(),
    createdAt: new Date().toISOString(),
    label: 'auto_generated',
  };
}

// ===== OAuth Login =====
function startOAuthLogin(deviceInfo) {
  return new Promise((resolve, reject) => {
    const server = http.createServer((req, res) => {
      const url = new URL(req.url, `http://127.0.0.1`);
      if (url.pathname === '/authorize') {
        // Extract token from query params or body
        let body = '';
        req.on('data', chunk => { body += chunk; });
        req.on('end', () => {
          const params = url.searchParams;
          const tokenData = {
            token: params.get('token') || '',
            userId: params.get('user_id') || '',
            code: params.get('code') || '',
          };

          // Also try to parse body if present
          if (body) {
            try {
              const bodyData = JSON.parse(body);
              Object.assign(tokenData, bodyData);
            } catch (e) {
              // try URL encoded
              const bodyParams = new URLSearchParams(body);
              for (const [key, value] of bodyParams) {
                tokenData[key] = value;
              }
            }
          }

          res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
          res.end(`<!DOCTYPE html><html><body style="background:#0a0e1a;color:white;display:flex;justify-content:center;align-items:center;height:100vh;font-family:sans-serif">
            <div style="text-align:center">
              <h1>? 登录成功</h1>
              <p>您可以关闭此窗口并返回 TraeCN Tools</p>
            </div>
          </body></html>`);

          server.close();
          resolve(tokenData);
        });
        return;
      }
      res.writeHead(404);
      res.end();
    });

    server.listen(0, '127.0.0.1', () => {
      const port = server.address().port;
      const callbackUrl = `http://127.0.0.1:${port}/authorize`;
      const loginTraceId = crypto.randomUUID();
      const machineId = deviceInfo.machineId || generateMachineId();
      const deviceId = deviceInfo.deviceId || generateDeviceId();

      const authParams = new URLSearchParams({
        login_version: '1',
        auth_from: 'trae',
        login_channel: 'native_ide',
        plugin_version: '2.3.13343',
        auth_type: 'local',
        client_id: 'ono9krqynydwx5',
        redirect: '0',
        login_trace_id: loginTraceId,
        auth_callback_url: callbackUrl,
        machine_id: machineId,
        device_id: deviceId,
        x_device_id: deviceId,
        x_machine_id: machineId,
        x_device_brand: os.hostname(),
        x_device_type: 'windows',
        x_os_version: `Windows ${os.release()}`,
        x_env: '',
        x_app_version: '3.3.37',
        x_app_type: 'stable',
      });

      const redirectUrl = `https://www.trae.cn/authorization?${authParams.toString()}`;
      const loginUrl = `https://www.trae.cn/login?redirect_url=${encodeURIComponent(redirectUrl)}`;

      shell.openExternal(loginUrl);

      // Set timeout for login
      setTimeout(() => {
        server.close();
        reject(new Error('登录超时（5分钟）'));
      }, 5 * 60 * 1000);
    });

    server.on('error', reject);
  });
}

// ===== Read Trae CN Storage =====
function readTraeStorage(storagePath) {
  try {
    const defaultPath = storagePath || path.join(
      process.env.APPDATA || '',
      'Trae CN', 'User', 'globalStorage', 'storage.json'
    );

    if (!fs.existsSync(defaultPath)) {
      return null;
    }

    const data = JSON.parse(fs.readFileSync(defaultPath, 'utf8'));
    const authKey = 'iCubeAuthInfo://icube.cloudide';
    if (data[authKey]) {
      const authInfo = JSON.parse(data[authKey]);
      return {
        token: authInfo.token,
        refreshToken: authInfo.refreshToken,
        userId: authInfo.userId,
        host: authInfo.host,
        expiredAt: authInfo.expiredAt,
        refreshExpiredAt: authInfo.refreshExpiredAt,
        username: authInfo.account?.username,
        scope: authInfo.account?.scope,
        region: authInfo.userRegion?.region,
        aiRegion: authInfo.userRegion?._aiRegion,
      };
    }
    return null;
  } catch (e) {
    console.error('Failed to read Trae storage:', e);
    return null;
  }
}

// ===== Proxy Service =====
function startProxyService(config) {
  if (proxyProcess) {
    proxyProcess.kill();
    proxyProcess = null;
  }

  // Write proxy config
  const proxyConfig = {
    listen_addr: `:${config.listenPort}`,
    log_level: 'info',
    accounts: [],
  };

  const data = loadData();
  const activeAccounts = data.accounts.filter(a => !a.disabled && a.token);
  activeAccounts.forEach(acc => {
    proxyConfig.accounts.push({
      name: acc.email || acc.label || 'default',
      token: acc.token,
      weight: 1,
    });
  });

  fs.writeFileSync(PROXY_CONFIG_FILE, JSON.stringify(proxyConfig, null, 2));

  // Try to find the Go binary
  const binaryName = process.platform === 'win32' ? 'trae-proxy.exe' : 'trae-proxy';
  const binaryPaths = [
    path.join(app.getAppPath(), '..', binaryName),
    path.join(app.getAppPath(), binaryName),
    path.join(process.cwd(), binaryName),
    path.join(process.cwd(), 'cmd', 'trae-proxy', binaryName),
  ];

  let binaryPath = null;
  for (const p of binaryPaths) {
    if (fs.existsSync(p)) {
      binaryPath = p;
      break;
    }
  }

  if (!binaryPath) {
    return { success: false, error: '未找到代理服务二进制文件 (trae-proxy.exe)' };
  }

  try {
    proxyProcess = spawn(binaryPath, ['-config', PROXY_CONFIG_FILE], {
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    proxyProcess.stdout.on('data', (data) => {
      if (mainWindow) {
        mainWindow.webContents.send('proxy-log', data.toString());
      }
    });

    proxyProcess.stderr.on('data', (data) => {
      if (mainWindow) {
        mainWindow.webContents.send('proxy-log', data.toString());
      }
    });

    proxyProcess.on('exit', (code) => {
      proxyProcess = null;
      if (mainWindow) {
        mainWindow.webContents.send('proxy-status', { running: false, code });
      }
    });

    return { success: true };
  } catch (e) {
    return { success: false, error: e.message };
  }
}

function stopProxyService() {
  if (proxyProcess) {
    proxyProcess.kill();
    proxyProcess = null;
    return { success: true };
  }
  return { success: false, error: '服务未运行' };
}

// ===== Window Management =====
function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1280,
    height: 860,
    minWidth: 1024,
    minHeight: 700,
    title: 'TraeCN Tools',
    backgroundColor: '#060810',
    frame: false,
    titleBarStyle: 'hidden',
    titleBarOverlay: {
      color: '#0a0e1a',
      symbolColor: '#94a3b8',
      height: 40,
    },
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: false,
    },
  });

  if (process.env.NODE_ENV === 'development' || process.argv.includes('--dev')) {
    const devPort = process.env.VITE_PORT || '5173';
    mainWindow.loadURL(`http://localhost:${devPort}`);
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, '..', 'dist', 'index.html'));
  }

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// ===== IPC Handlers =====
function setupIPC() {
  // Data operations
  ipcMain.handle('load-data', () => loadData());
  ipcMain.handle('save-data', (_, data) => { saveData(data); return true; });

  // Account operations
  ipcMain.handle('add-account-oauth', async (_, deviceInfo) => {
    try {
      const result = await startOAuthLogin(deviceInfo || {});
      return { success: true, data: result };
    } catch (e) {
      return { success: false, error: e.message };
    }
  });

  ipcMain.handle('read-trae-storage', (_, storagePath) => {
    const result = readTraeStorage(storagePath);
    return result ? { success: true, data: result } : { success: false, error: '未找到 Trae CN 认证信息' };
  });

  ipcMain.handle('import-accounts', (_, filePath) => {
    try {
      const content = JSON.parse(fs.readFileSync(filePath, 'utf8'));
      return { success: true, data: content };
    } catch (e) {
      return { success: false, error: e.message };
    }
  });

  ipcMain.handle('export-accounts', (_, { filePath, data }) => {
    try {
      fs.writeFileSync(filePath, JSON.stringify(data, null, 2), 'utf8');
      return { success: true };
    } catch (e) {
      return { success: false, error: e.message };
    }
  });

  // Device fingerprint
  ipcMain.handle('generate-fingerprint', () => generateFingerprint());

  // Proxy service
  ipcMain.handle('start-proxy', (_, config) => startProxyService(config));
  ipcMain.handle('stop-proxy', () => stopProxyService());
  ipcMain.handle('proxy-status', () => ({ running: proxyProcess !== null }));

  // System
  ipcMain.handle('get-app-version', () => app.getVersion());
  ipcMain.handle('get-platform', () => ({
    platform: process.platform,
    arch: process.arch,
    hostname: os.hostname(),
    osVersion: os.version?.() || os.release(),
  }));

  ipcMain.handle('open-external', (_, url) => {
    // Only allow specific trusted domains
    const allowed = ['https://www.trae.cn', 'https://trae.cn'];
    try {
      const parsed = new URL(url);
      if (allowed.some(a => url.startsWith(a))) {
        shell.openExternal(url);
        return true;
      }
    } catch (e) {}
    return false;
  });

  ipcMain.handle('open-path', (_, p) => shell.openPath(p));
  ipcMain.handle('get-data-dir', () => DATA_DIR);

  // Window controls
  ipcMain.on('window-minimize', () => mainWindow?.minimize());
  ipcMain.on('window-maximize', () => {
    if (mainWindow?.isMaximized()) mainWindow.unmaximize();
    else mainWindow?.maximize();
  });
  ipcMain.on('window-close', () => mainWindow?.close());
}

// ===== App Lifecycle =====
app.whenReady().then(() => {
  setupIPC();
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (proxyProcess) {
    proxyProcess.kill();
    proxyProcess = null;
  }
  if (process.platform !== 'darwin') app.quit();
});
