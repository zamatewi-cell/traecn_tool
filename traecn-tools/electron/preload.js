const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
  // Data
  loadData: () => ipcRenderer.invoke('load-data'),
  saveData: (data) => ipcRenderer.invoke('save-data', data),

  // Accounts
  addAccountOAuth: (deviceInfo) => ipcRenderer.invoke('add-account-oauth', deviceInfo),
  readTraeStorage: (storagePath) => ipcRenderer.invoke('read-trae-storage', storagePath),
  importAccounts: () => ipcRenderer.invoke('import-accounts'),
  exportAccounts: (data, defaultFileName) => ipcRenderer.invoke('export-accounts', { data, defaultFileName }),

  // Device fingerprint
  generateFingerprint: () => ipcRenderer.invoke('generate-fingerprint'),

  // Proxy
  startProxy: (config) => ipcRenderer.invoke('start-proxy', config),
  stopProxy: () => ipcRenderer.invoke('stop-proxy'),
  proxyStatus: () => ipcRenderer.invoke('proxy-status'),
  onProxyLog: (callback) => ipcRenderer.on('proxy-log', (_, log) => callback(log)),
  onProxyStatus: (callback) => ipcRenderer.on('proxy-status', (_, status) => callback(status)),

  // System
  getAppVersion: () => ipcRenderer.invoke('get-app-version'),
  getPlatform: () => ipcRenderer.invoke('get-platform'),
  openExternal: (url) => ipcRenderer.invoke('open-external', url),
  openPath: (p) => ipcRenderer.invoke('open-path', p),
  getDataDir: () => ipcRenderer.invoke('get-data-dir'),

  // Window
  windowMinimize: () => ipcRenderer.send('window-minimize'),
  windowMaximize: () => ipcRenderer.send('window-maximize'),
  windowClose: () => ipcRenderer.send('window-close'),
});

// Helper for direct HTTP API calls to trae-proxy backend
// This is exposed via window.electronAPI in the renderer
