// ===== Account Types =====
export interface DeviceFingerprint {
  machineId: string;
  macMachineId: string;
  devDeviceId: string;
  sqmId: string;
  createdAt: string;
  label: string;
}

export interface Account {
  id: string;
  email: string;
  label: string;
  token: string;
  refreshToken: string;
  userId: string;
  host: string;
  expiredAt: string;
  refreshExpiredAt: string;
  username: string;
  scope: string;
  region: string;
  aiRegion: string;
  isCurrent: boolean;
  isPro: boolean;
  disabled: boolean;
  fingerprint: DeviceFingerprint | null;
  fingerprintHistory: DeviceFingerprint[];
  lastUsed: string;
  createdAt: string;
  tags: string[];
}

// ===== Proxy Config =====
export interface ProxyConfig {
  listenPort: number;
  requestTimeout: number;
  autoStart: boolean;
  allowLan: boolean;
  authEnabled: boolean;
  authMode: 'auto' | 'bearer' | 'none';
  apiKey: string;
  webUiPassword: string;
  userAgentOverride: boolean;
  userAgentValue: string;
}

// ===== Model Types =====
export interface TraeModel {
  configName: string;
  modelName: string;
  displayName: string;
  provider: string;
  icon: string;
}

// ===== Model Router =====
export interface ModelMapping {
  id: string;
  sourceName: string;
  targetModel: string;
}

// ===== App Data =====
export interface AppData {
  accounts: Account[];
  proxyConfig: ProxyConfig;
  settings: AppSettings;
}

export interface AppSettings {
  language: string;
  theme: string;
}

// ===== Electron API =====
export interface ElectronAPI {
  loadData: () => Promise<AppData>;
  saveData: (data: AppData) => Promise<{ success: boolean; error?: string }>;
  addAccountOAuth: (deviceInfo?: Record<string, string>) => Promise<{ success: boolean; data?: Record<string, string>; error?: string }>;
  readTraeStorage: (storagePath?: string) => Promise<{ success: boolean; data?: Record<string, string>; error?: string }>;
  importAccounts: () => Promise<{ success: boolean; data?: Account[]; error?: string; canceled?: boolean; filePath?: string }>;
  exportAccounts: (options: { data: any; defaultFileName?: string } | any, defaultFileName?: string) => Promise<{ success: boolean; error?: string; canceled?: boolean; filePath?: string }>;
  generateFingerprint: () => Promise<DeviceFingerprint>;
  startProxy: (config: ProxyConfig) => Promise<{ success: boolean; error?: string }>;
  stopProxy: () => Promise<{ success: boolean; error?: string }>;
  proxyStatus: () => Promise<{ running: boolean }>;
  onProxyLog: (callback: (log: string) => void) => void;
  onProxyStatus: (callback: (status: { running: boolean; code?: number }) => void) => void;
  getAppVersion: () => Promise<string>;
  getPlatform: () => Promise<{ platform: string; arch: string; hostname: string; osVersion: string }>;
  openExternal: (url: string) => Promise<boolean>;
  openPath: (p: string) => Promise<string>;
  getDataDir: () => Promise<string>;
  windowMinimize: () => void;
  windowMaximize: () => void;
  windowClose: () => void;
}

declare global {
  interface Window {
    electronAPI: ElectronAPI;
  }
}

// ===== Constants =====
export const TRAE_MODELS: TraeModel[] = [
  { configName: 'doubao-seed-1.6', modelName: 'doubao-seed-1.6', displayName: 'Doubao-Seed-1.6', provider: 'ByteDance', icon: '?' },
  { configName: 'doubao-1.5-pro', modelName: 'doubao-1.5-pro', displayName: 'Doubao-1.5-Pro', provider: 'ByteDance', icon: '?' },
  { configName: 'deepseek-v3', modelName: 'deepseek-v3', displayName: 'DeepSeek-V3', provider: 'DeepSeek', icon: '?' },
  { configName: 'deepseek-r1', modelName: 'deepseek-r1', displayName: 'DeepSeek-R1', provider: 'DeepSeek', icon: '?' },
  { configName: 'glm-5', modelName: 'glm-5', displayName: 'GLM-5', provider: 'Zhipu', icon: '?' },
  { configName: 'glm-4-plus', modelName: 'glm-4-plus', displayName: 'GLM-4-Plus', provider: 'Zhipu', icon: '?' },
  { configName: 'kimi-k2', modelName: 'kimi-k2', displayName: 'Kimi-K2', provider: 'Moonshot', icon: '?' },
  { configName: 'minimax-m1', modelName: 'minimax-m1', displayName: 'MiniMax-M1', provider: 'MiniMax', icon: '?' },
  { configName: 'qwen3-coder', modelName: 'qwen3-coder', displayName: 'Qwen3-Coder', provider: 'Alibaba', icon: '??' },
  { configName: 'qwen3', modelName: 'qwen3', displayName: 'Qwen3', provider: 'Alibaba', icon: '??' },
  { configName: 'gemini-2.5-pro', modelName: 'gemini-2.5-pro', displayName: 'Gemini-2.5-Pro', provider: 'Google', icon: '?' },
  { configName: 'claude-sonnet-4', modelName: 'claude-sonnet-4', displayName: 'Claude-Sonnet-4', provider: 'Anthropic', icon: '?' },
  { configName: 'gpt-4.1', modelName: 'gpt-4.1', displayName: 'GPT-4.1', provider: 'OpenAI', icon: '?' },
];

export const PROVIDER_COLORS: Record<string, string> = {
  ByteDance: '#00d4aa',
  DeepSeek: '#6366f1',
  Zhipu: '#f59e0b',
  Moonshot: '#8b5cf6',
  MiniMax: '#ec4899',
  Alibaba: '#f97316',
  Google: '#3b82f6',
  Anthropic: '#d97706',
  OpenAI: '#22c55e',
};
