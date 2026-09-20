import { create } from 'zustand';
import type { Account, ProxyConfig, AppData, DeviceFingerprint, ModelMapping } from '../types';
import { apiClient, type ModelInfo } from '../api/client';

const api = () => window.electronAPI;

interface AppStore {
  // State
  accounts: Account[];
  proxyConfig: ProxyConfig;
  proxyRunning: boolean;
  currentAccountId: string | null;
  modelMappings: ModelMapping[];
  proxyLogs: string[];
  loading: boolean;

  // Backend state
  availableModels: ModelInfo[];
  backendHealth: string; // 'healthy' | 'error' | 'unknown'
  backendUrl: string;

  // Actions
  init: () => Promise<void>;
  save: () => Promise<void>;

  // Account actions
  addAccount: (account: Account) => void;
  updateAccount: (id: string, updates: Partial<Account>) => void;
  removeAccount: (id: string) => void;
  switchAccount: (id: string) => void;
  setAccountDisabled: (id: string, disabled: boolean) => void;

  // Proxy actions
  updateProxyConfig: (updates: Partial<ProxyConfig>) => void;
  startProxy: () => Promise<{ success: boolean; error?: string }>;
  stopProxy: () => Promise<{ success: boolean; error?: string }>;
  addProxyLog: (log: string) => void;

  // Model mapping
  addModelMapping: (mapping: ModelMapping) => void;
  removeModelMapping: (id: string) => void;

  // Backend API actions
  checkBackendHealth: () => Promise<void>;
  fetchAvailableModels: () => Promise<void>;
  setBackendUrl: (url: string) => void;
}

export const useAppStore = create<AppStore>((set, get) => ({
  accounts: [],
  proxyConfig: {
    listenPort: 8045,
    requestTimeout: 120,
    autoStart: false,
    allowLan: false,
    authEnabled: true,
    authMode: 'auto',
    apiKey: '',
    webUiPassword: '',
    userAgentOverride: false,
    userAgentValue: '',
  },
  proxyRunning: false,
  currentAccountId: null,
  modelMappings: [],
  proxyLogs: [],
  loading: true,

  // Backend state
  availableModels: [],
  backendHealth: 'unknown',
  backendUrl: 'http://127.0.0.1:9090',

  init: async () => {
    try {
      const data = await api().loadData();
      const current = data.accounts.find((a: Account) => a.isCurrent);
      const mergedConfig = { ...get().proxyConfig, ...data.proxyConfig };
      set({
        accounts: data.accounts || [],
        proxyConfig: mergedConfig,
        currentAccountId: current?.id || null,
        loading: false,
      });

      // 同步配置到单例 apiClient (URL 与 API Key)
      const url = `http://127.0.0.1:${mergedConfig.listenPort}`;
      apiClient.setBaseUrl(url);
      apiClient.setApiKey(mergedConfig.authEnabled && mergedConfig.apiKey ? mergedConfig.apiKey : '');
      set({ backendUrl: url });

      // Check proxy status
      const status = await api().proxyStatus();
      set({ proxyRunning: status.running });

      // Listen for proxy events
      api().onProxyLog((log: string) => get().addProxyLog(log));
      api().onProxyStatus((status: { running: boolean }) => set({ proxyRunning: status.running }));

      // 订阅主进程凭据刷新推送，原子更新渲染层内存快照（严禁调用 save()，防止写盘循环）
      (api() as any).onAccountUpdated?.((updatedAcc: Account) => {
        if (!updatedAcc || !updatedAcc.id) return;
        set((state) => ({
          accounts: state.accounts.map((a) =>
            a.id === updatedAcc.id
              ? {
                  ...a,
                  token: updatedAcc.token,
                  refreshToken: updatedAcc.refreshToken || a.refreshToken,
                  expiredAt: updatedAcc.expiredAt || a.expiredAt,
                  refreshExpiredAt: updatedAcc.refreshExpiredAt || a.refreshExpiredAt,
                  lastUsed: updatedAcc.lastUsed || a.lastUsed,
                }
              : a
          ),
        }));
      });

      // Check backend health and fetch models with auth
      await get().checkBackendHealth();
      await get().fetchAvailableModels();
    } catch (e) {
      console.error('Failed to init:', e);
      set({ loading: false, backendHealth: 'error' });
    }
  },

  save: async () => {
    const { accounts, proxyConfig } = get();
    const data: AppData = {
      accounts,
      proxyConfig,
      settings: { language: 'zh', theme: 'dark' },
    };
    await api().saveData(data);
  },

  addAccount: (account) => {
    set((state) => ({ accounts: [...state.accounts, account] }));
    get().save();
  },

  updateAccount: (id, updates) => {
    set((state) => ({
      accounts: state.accounts.map((a) => (a.id === id ? { ...a, ...updates } : a)),
    }));
    get().save();
  },

  removeAccount: (id) => {
    set((state) => ({
      accounts: state.accounts.filter((a) => a.id !== id),
      currentAccountId: state.currentAccountId === id ? null : state.currentAccountId,
    }));
    get().save();
  },

  switchAccount: (id) => {
    set((state) => ({
      accounts: state.accounts.map((a) => ({ ...a, isCurrent: a.id === id })),
      currentAccountId: id,
    }));
    get().save();
  },

  setAccountDisabled: (id, disabled) => {
    set((state) => ({
      accounts: state.accounts.map((a) => (a.id === id ? { ...a, disabled } : a)),
    }));
    get().save();
  },

  updateProxyConfig: (updates) => {
    set((state) => {
      const updated = { ...state.proxyConfig, ...updates };
      const url = `http://127.0.0.1:${updated.listenPort}`;
      apiClient.setBaseUrl(url);
      apiClient.setApiKey(updated.authEnabled && updated.apiKey ? updated.apiKey : '');
      return {
        proxyConfig: updated,
        backendUrl: url,
      };
    });
    get().save();
  },

  startProxy: async () => {
    const result = await api().startProxy(get().proxyConfig);
    if (result.success) set({ proxyRunning: true });
    return result;
  },

  stopProxy: async () => {
    const result = await api().stopProxy();
    if (result.success) set({ proxyRunning: false });
    return result;
  },

  addProxyLog: (log) => {
    set((state) => ({
      proxyLogs: [...state.proxyLogs.slice(-500), log],
    }));
  },

  addModelMapping: (mapping) => {
    set((state) => ({ modelMappings: [...state.modelMappings, mapping] }));
  },

  removeModelMapping: (id) => {
    set((state) => ({
      modelMappings: state.modelMappings.filter((m) => m.id !== id),
    }));
  },

  // Backend API actions
  checkBackendHealth: async () => {
    const { proxyConfig } = get();
    const url = `http://127.0.0.1:${proxyConfig.listenPort}`;
    apiClient.setBaseUrl(url);
    apiClient.setApiKey(proxyConfig.authEnabled && proxyConfig.apiKey ? proxyConfig.apiKey : '');
    set({ backendUrl: url });
    
    try {
      const result = await apiClient.testConnection();
      set({ backendHealth: result.success ? 'healthy' : 'error' });
    } catch (e) {
      set({ backendHealth: 'error' });
    }
  },

  fetchAvailableModels: async () => {
    const { proxyConfig } = get();
    const url = `http://127.0.0.1:${proxyConfig.listenPort}`;
    apiClient.setBaseUrl(url);
    apiClient.setApiKey(proxyConfig.authEnabled && proxyConfig.apiKey ? proxyConfig.apiKey : '');
    
    try {
      const result = await apiClient.getModels();
      if (result.success && result.data) {
        set({ availableModels: result.data.data || [] });
      }
    } catch (e) {
      console.error('Failed to fetch models:', e);
    }
  },

  setBackendUrl: (url) => {
    set({ backendUrl: url });
    apiClient.setBaseUrl(url);
  },
}));
