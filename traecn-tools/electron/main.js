const { app, BrowserWindow, ipcMain, shell, safeStorage, dialog } = require('electron');
const path = require('path');
const fs = require('fs');
const http = require('http');
const crypto = require('crypto');
const os = require('os');
const { spawn, execSync } = require('child_process');

let mainWindow = null;
let proxyProcess = null;

const DATA_DIR = path.join(app.getPath('userData'));
const DATA_FILE = path.join(DATA_DIR, 'traecn-data.json');
const PROXY_CONFIG_FILE = path.join(DATA_DIR, 'proxy-config.json');

// ===== Safe Storage Helpers (Windows DPAPI Protection) =====
// 注意：Windows safeStorage 基于 DPAPI，主要防御跨 Windows 账户与离线数据窃取；
// 在同一 Windows 用户上下文权限内，运行中的进程不构成严格隔离边界。
function encryptSecret(plaintext) {
  if (!plaintext || typeof plaintext !== 'string') return plaintext;
  if (plaintext.startsWith('enc:')) return plaintext;
  if (safeStorage && safeStorage.isEncryptionAvailable()) {
    try {
      const encrypted = safeStorage.encryptString(plaintext);
      return 'enc:' + encrypted.toString('base64');
    } catch (e) {
      throw new Error(`安全存储(safeStorage/DPAPI)加密失败，拒绝写入磁盘: ${e.message}`);
    }
  }

  // 严格 Fail-Closed：若系统安全存储不可用，默认严格拒绝明文落盘，杜绝静默泄露
  if (process.env.ALLOW_INSECURE_PLAINTEXT_STORAGE === 'true') {
    console.warn('警告: 显式开启了 ALLOW_INSECURE_PLAINTEXT_STORAGE，凭据将以不安全的明文形式落盘');
    return plaintext;
  }

  throw new Error('系统安全存储(safeStorage/DPAPI)当前不可用，为防止账号凭据意外泄露已拒绝落盘。若在无密钥环的特殊调试沙盒中运行，请显式声明 ALLOW_INSECURE_PLAINTEXT_STORAGE=true');
}

function decryptSecret(ciphertext) {
  if (!ciphertext || typeof ciphertext !== 'string') return { text: ciphertext, failed: false };
  if (!ciphertext.startsWith('enc:')) return { text: ciphertext, failed: false };
  try {
    if (safeStorage && safeStorage.isEncryptionAvailable()) {
      const base64Str = ciphertext.slice(4);
      const buffer = Buffer.from(base64Str, 'base64');
      return { text: safeStorage.decryptString(buffer), failed: false };
    }
  } catch (e) {
    console.error('安全存储(safeStorage/DPAPI)解密失败:', e.message);
    return { text: '', rawCiphertext: ciphertext, failed: true };
  }
  // safeStorage 不可用但凭据为 enc: 密文：标记解密失败，严禁把密文当明文返回！
  return { text: '', rawCiphertext: ciphertext, failed: true };
}

// ===== Data Store =====
function loadData() {
  try {
    if (fs.existsSync(DATA_FILE)) {
      const parsed = JSON.parse(fs.readFileSync(DATA_FILE, 'utf8'));
      if (parsed.accounts && Array.isArray(parsed.accounts)) {
        parsed.accounts.forEach(acc => {
          if (acc.token) {
            const dec = decryptSecret(acc.token);
            if (dec.failed) {
              acc._rawEncryptedToken = dec.rawCiphertext;
              acc._tokenDecryptFailed = true;
              acc.token = '';
            } else {
              acc.token = dec.text;
            }
          }
          if (acc.refreshToken) {
            const dec = decryptSecret(acc.refreshToken);
            if (dec.failed) {
              acc._rawEncryptedRefreshToken = dec.rawCiphertext;
              acc._refreshTokenDecryptFailed = true;
              acc.refreshToken = '';
            } else {
              acc.refreshToken = dec.text;
            }
          }
        });
      }
      if (parsed.proxyConfig) {
        if (parsed.proxyConfig.apiKey) {
          const dec = decryptSecret(parsed.proxyConfig.apiKey);
          if (dec.failed) {
            parsed.proxyConfig._rawEncryptedApiKey = dec.rawCiphertext;
            parsed.proxyConfig._apiKeyDecryptFailed = true;
            parsed.proxyConfig.apiKey = '';
          } else {
            parsed.proxyConfig.apiKey = dec.text;
          }
        }
        if (parsed.proxyConfig.webUiPassword) {
          const dec = decryptSecret(parsed.proxyConfig.webUiPassword);
          if (dec.failed) {
            parsed.proxyConfig._rawEncryptedPassword = dec.rawCiphertext;
            parsed.proxyConfig._passwordDecryptFailed = true;
            parsed.proxyConfig.webUiPassword = '';
          } else {
            parsed.proxyConfig.webUiPassword = dec.text;
          }
        }
      }
      return parsed;
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
  if (!fs.existsSync(DATA_DIR)) {
    fs.mkdirSync(DATA_DIR, { recursive: true });
  }
  // Deep clone data to avoid mutating memory state
  const dataToSave = JSON.parse(JSON.stringify(data));
  if (dataToSave.accounts && Array.isArray(dataToSave.accounts)) {
    dataToSave.accounts.forEach(acc => {
      if (acc._tokenDecryptFailed && (!acc.token || acc.token === '')) {
        // 凭据解密失败且用户未录入新 token 时，保留磁盘原密文，严禁空串覆盖！
        acc.token = acc._rawEncryptedToken;
      } else if (acc.token) {
        acc.token = encryptSecret(acc.token);
      }
      delete acc._tokenDecryptFailed;
      delete acc._rawEncryptedToken;

      if (acc._refreshTokenDecryptFailed && (!acc.refreshToken || acc.refreshToken === '')) {
        acc.refreshToken = acc._rawEncryptedRefreshToken;
      } else if (acc.refreshToken) {
        acc.refreshToken = encryptSecret(acc.refreshToken);
      }
      delete acc._refreshTokenDecryptFailed;
      delete acc._rawEncryptedRefreshToken;
    });
  }
  if (dataToSave.proxyConfig) {
    if (dataToSave.proxyConfig._apiKeyDecryptFailed && (!dataToSave.proxyConfig.apiKey || dataToSave.proxyConfig.apiKey === '')) {
      dataToSave.proxyConfig.apiKey = dataToSave.proxyConfig._rawEncryptedApiKey;
    } else if (dataToSave.proxyConfig.apiKey) {
      dataToSave.proxyConfig.apiKey = encryptSecret(dataToSave.proxyConfig.apiKey);
    }
    delete dataToSave.proxyConfig._apiKeyDecryptFailed;
    delete dataToSave.proxyConfig._rawEncryptedApiKey;

    if (dataToSave.proxyConfig._passwordDecryptFailed && (!dataToSave.proxyConfig.webUiPassword || dataToSave.proxyConfig.webUiPassword === '')) {
      dataToSave.proxyConfig.webUiPassword = dataToSave.proxyConfig._rawEncryptedPassword;
    } else if (dataToSave.proxyConfig.webUiPassword) {
      dataToSave.proxyConfig.webUiPassword = encryptSecret(dataToSave.proxyConfig.webUiPassword);
    }
    delete dataToSave.proxyConfig._passwordDecryptFailed;
    delete dataToSave.proxyConfig._rawEncryptedPassword;
  }
  fs.writeFileSync(DATA_FILE, JSON.stringify(dataToSave, null, 2), 'utf8');
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

function getTraeStorageCandidates() {
  const candidates = [];
  if (process.env.APPDATA) {
    candidates.push(path.join(process.env.APPDATA, 'Trae CN', 'User', 'globalStorage', 'storage.json'));
    candidates.push(path.join(process.env.APPDATA, 'Trae', 'User', 'globalStorage', 'storage.json'));
  }
  if (process.env.USERPROFILE) {
    candidates.push(path.join(process.env.USERPROFILE, '.config', 'Trae CN', 'User', 'globalStorage', 'storage.json'));
  }
  return candidates;
}

const traeKeyA = Buffer.from([82, 9, 106, 213, 48, 54, 165, 56, 191, 64, 163, 158, 129, 243, 215, 251, 124, 227, 57, 130, 155, 47, 255, 135, 52, 142, 67, 68, 196, 222, 233, 203, 84, 123, 148, 50, 166, 194, 35, 61, 238, 76, 149, 11, 66, 250, 195, 78, 8, 46, 161, 102, 40, 217, 36, 178, 118, 91, 162, 73, 109, 139, 209, 37]);
const traeKeyB = Buffer.from([31, 221, 168, 51, 136, 7, 199, 49, 177, 18, 16, 89, 39, 128, 236, 95, 96, 81, 127, 169, 25, 181, 74, 13, 45, 229, 122, 159, 147, 201, 156, 239, 160, 224, 59, 77, 174, 42, 245, 176, 200, 235, 187, 60, 131, 83, 153, 97, 23, 43, 4, 126, 186, 119, 214, 38, 225, 105, 20, 99, 85, 33, 12, 125]);
const traePw = Buffer.alloc(64);
for (let i = 0; i < 64; i++) traePw[i] = traeKeyA[i] ^ traeKeyB[i];

function decryptTraeBlob(enc) {
  try {
    const blob = Buffer.from(enc.trim(), 'base64');
    if (blob.length < 6 + 32 + 16) return null;
    const salt = blob.slice(6, 38);
    const ciphertext = blob.slice(38);
    
    const hSalt = crypto.createHash('sha512').update(salt).digest();
    const kdf = crypto.createHash('sha512').update(Buffer.concat([hSalt, traePw])).digest();
    const key = kdf.slice(0, 16);
    const iv = kdf.slice(16, 32);

    const decipher = crypto.createDecipheriv('aes-128-cbc', key, iv);
    const decrypted = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
    const jsonBuf = decrypted.slice(64);
    return JSON.parse(jsonBuf.toString('utf8'));
  } catch (e) {
    console.warn('Trae AES-128-CBC blob 解密失败:', e.message);
    return null;
  }
}

// ===== Read Trae CN Storage =====
function readTraeStorage(storagePath) {
  try {
    const candidates = getTraeStorageCandidates();
    let targetPath = storagePath;
    if (!targetPath) {
      targetPath = candidates.find(p => fs.existsSync(p)) || candidates[0];
    } else {
      // 严格白名单：只允许读取预设候选列表中的路径，拒绝任意外部路径遍历
      const resolved = path.resolve(targetPath);
      const isAllowed = candidates.some(c => path.resolve(c) === resolved);
      if (!isAllowed) {
        console.warn('拒绝读取非白名单 Trae Storage 路径:', targetPath);
        return null;
      }
    }

    if (!targetPath || !fs.existsSync(targetPath)) {
      return null;
    }

    const data = JSON.parse(fs.readFileSync(targetPath, 'utf8'));
    const authKey = 'iCubeAuthInfo://icube.cloudide';
    if (data[authKey]) {
      let authInfo = null;
      try {
        authInfo = JSON.parse(data[authKey]);
      } catch (e) {
        authInfo = decryptTraeBlob(data[authKey]);
      }

      if (authInfo) {
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
  const isLan = !!config.allowLan;
  const listenHost = isLan ? '0.0.0.0' : '127.0.0.1';
  const proxyConfig = {
    listen_addr: `${listenHost}:${config.listenPort || 8045}`,
    allow_lan: isLan,
    log_level: 'info',
    request_timeout: config.requestTimeout || 120,
    accounts: [],
  };

  if (config.authEnabled && config.apiKey) {
    proxyConfig.api_keys = [config.apiKey];
  }

  const data = loadData();
  const activeAccounts = data.accounts.filter(a => !a.disabled && a.token);
  activeAccounts.forEach(acc => {
    proxyConfig.accounts.push({
      name: acc.email || acc.label || 'default',
      token: acc.token,
      weight: 1,
    });
  });

  // 安全防线：主动清理任何可能存在的历史明文配置文件，杜绝磁盘凭据残留
  if (fs.existsSync(PROXY_CONFIG_FILE)) {
    try {
      fs.unlinkSync(PROXY_CONFIG_FILE);
    } catch (e) {
      console.warn('清理旧版 proxy-config.json 异常:', e.message);
    }
  }

  // Try to find the Go binary
  const binaryName = process.platform === 'win32' ? 'trae-proxy.exe' : 'trae-proxy';
  const binaryPaths = [
    ...(process.resourcesPath ? [
      path.join(process.resourcesPath, 'bin', binaryName),
      path.join(process.resourcesPath, binaryName),
    ] : []),
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
    // 零磁盘落盘架构：通过 stdin 管道向 Go 核心传递运行时配置
    proxyProcess = spawn(binaryPath, ['-config', 'stdin'], {
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    // 立即通过内存管道写入配置并关闭输入端
    proxyProcess.stdin.write(JSON.stringify(proxyConfig));
    proxyProcess.stdin.end();

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
    killProxyProcess();
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

  ipcMain.handle('import-accounts', async () => {
    try {
      const { canceled, filePaths } = await dialog.showOpenDialog(mainWindow, {
        title: '选择导入的账号配置文件',
        properties: ['openFile'],
        filters: [{ name: 'JSON Files', extensions: ['json'] }],
      });
      if (canceled || filePaths.length === 0) {
        return { success: false, canceled: true };
      }
      const content = JSON.parse(fs.readFileSync(filePaths[0], 'utf8'));
      return { success: true, data: content, filePath: filePaths[0] };
    } catch (e) {
      return { success: false, error: e.message };
    }
  });

  ipcMain.handle('export-accounts', async (_, { data, defaultFileName }) => {
    try {
      const { canceled, filePath } = await dialog.showSaveDialog(mainWindow, {
        title: '导出脱敏信息',
        defaultPath: defaultFileName || `traecn-accounts-sanitized-${new Date().toISOString().slice(0, 10)}.json`,
        filters: [{ name: 'JSON Files', extensions: ['json'] }],
      });
      if (canceled || !filePath) {
        return { success: false, canceled: true };
      }
      fs.writeFileSync(filePath, JSON.stringify(data, null, 2), 'utf8');
      return { success: true, filePath };
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
    try {
      const parsed = new URL(url);
      if (parsed.protocol === 'https:' && ['trae.cn', 'www.trae.cn'].includes(parsed.hostname)) {
        shell.openExternal(url);
        return true;
      }
    } catch (e) {
      console.warn('拒绝打开非受信任或非法 URL:', url);
    }
    return false;
  });

  ipcMain.handle('open-path', (_, p) => {
    if (!p || typeof p !== 'string') return '';
    const allowedDirs = [
      path.resolve(DATA_DIR),
      path.resolve(app.getPath('userData')),
      path.resolve(app.getPath('logs')),
      ...getTraeStorageCandidates().map(c => path.resolve(path.dirname(c))),
    ];
    const resolved = path.resolve(p);
    const isPermitted = allowedDirs.some(dir => resolved === dir || resolved.startsWith(dir + path.sep));
    if (!isPermitted) {
      console.warn('安全拒绝: 尝试打开非受信任目录:', p);
      return '安全拒绝: 路径不在受信任的目录列表中';
    }
    return shell.openPath(resolved);
  });
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

function killProxyProcess() {
  if (proxyProcess && proxyProcess.pid) {
    const pid = proxyProcess.pid;
    try {
      if (process.platform === 'win32') {
        // Windows 环境下利用 taskkill /PID ... /T /F 强杀整棵子进程树，彻底杜绝孤儿进程残留
        execSync(`taskkill /pid ${pid} /T /F`, { stdio: 'ignore' });
      } else {
        proxyProcess.kill('SIGTERM');
        setTimeout(() => {
          try {
            if (proxyProcess && !proxyProcess.killed) {
              proxyProcess.kill('SIGKILL');
            }
          } catch (e) {}
        }, 800);
      }
    } catch (e) {
      try {
        proxyProcess.kill('SIGKILL');
      } catch (err) {}
    }
    proxyProcess = null;
    if (fs.existsSync(PROXY_CONFIG_FILE)) {
      try {
        fs.unlinkSync(PROXY_CONFIG_FILE);
      } catch (e) {}
    }
  }
}

app.on('before-quit', killProxyProcess);
app.on('will-quit', killProxyProcess);
process.on('exit', killProxyProcess);

app.on('window-all-closed', () => {
  killProxyProcess();
  if (process.platform !== 'darwin') app.quit();
});

