import React, { useEffect, useState } from 'react';
import { useAppStore } from '../store';
import { TRAE_MODELS, PROVIDER_COLORS } from '../types';
import {
  Users, Zap, TrendingUp, Activity, ArrowRight,
  Plus, RefreshCw, Download, UserCheck, Server, CheckCircle, XCircle, ShieldCheck,
} from 'lucide-react';
import { apiClient } from '../api/client';

export default function Dashboard() {
  const {
    accounts, proxyRunning, currentAccountId, runtimeActiveAccountId, startProxy, stopProxy, backendHealth,
    availableModels, backendUrl, checkBackendHealth, fetchAvailableModels, switchAccount,
  } = useAppStore();

  const [refreshing, setRefreshing] = useState(false);
  const [appVersion, setAppVersion] = useState('1.0.2');

  const currentAccount = accounts.find((a) => a.id === currentAccountId);
  const activeAccounts = accounts.filter((a) => !a.disabled);
  const totalAccounts = accounts.length;

  const handleDashboardSwitchAccount = async (id: string) => {
    if (proxyRunning) {
      const confirmed = window.confirm(
        '当前代理服务正在运行中，切换账号需重启代理服务方能使网关生效。\n\n' +
        '点击【确定】将立即保存并自动平滑重启代理；\n' +
        '点击【取消】将仅记录首选项，并在下次启动代理时生效。'
      );
      try {
        await switchAccount(id);
      } catch (err: any) {
        alert('切换账号保存失败: ' + (err?.message || err) + '，已取消切换与重启！');
        return;
      }
      if (confirmed) {
        try {
          const stopRes = await stopProxy();
          if (stopRes && stopRes.success === false) {
            alert('重启代理服务失败 (停止旧进程超时): ' + (stopRes.error || '未知错误'));
            return;
          }
          const startRes = await startProxy();
          if (!startRes.success) {
            alert('重启代理服务失败: ' + (startRes.error || '未知错误'));
          } else {
            alert('账号切换成功，代理服务已重启并立即生效！');
          }
        } catch (err: any) {
          alert('重启代理服务异常: ' + (err?.message || err));
        }
      } else {
        alert('已记录该账号为首选账号。当前代理服务仍在运行旧账号，将在下次重启代理后生效。');
      }
    } else {
      try {
        await switchAccount(id);
      } catch (err: any) {
        alert('切换账号保存失败: ' + (err?.message || err));
      }
    }
  };

  // Check backend health & version on mount
  useEffect(() => {
    checkBackendHealth();
    fetchAvailableModels();
    window.electronAPI?.getAppVersion?.().then((ver) => {
      if (ver) setAppVersion(ver);
    }).catch(() => {});
  }, []);

  // 真实刷新
  const handleRefresh = async () => {
    setRefreshing(true);
    try {
      await checkBackendHealth();
      await fetchAvailableModels();
    } finally {
      setTimeout(() => setRefreshing(false), 500);
    }
  };

  const maskSecret = (secret?: string) => {
    if (!secret) return '';
    if (secret.length <= 10) return '*** (已脱敏)';
    return `${secret.slice(0, 6)}***${secret.slice(-4)} (已脱敏保护)`;
  };

  // 真实导出账号数据
  const handleExportAccounts = async () => {
    if (accounts.length === 0) {
      alert('当前没有账号可导出');
      return;
    }
    try {
      const sanitized = accounts.map((a) => ({
        id: a.id,
        email: a.email,
        username: a.username,
        isPro: a.isPro,
        disabled: a.disabled,
        scope: a.scope,
        region: a.region,
        token: a.token ? maskSecret(a.token) : '',
        refreshToken: a.refreshToken ? maskSecret(a.refreshToken) : '',
        lastUsed: a.lastUsed,
      }));
      const result = await window.electronAPI?.exportAccounts({
        data: {
          _exportType: 'sanitized_accounts_export',
          accounts: sanitized,
          exportTime: new Date().toISOString(),
          version: appVersion,
        },
        defaultFileName: `traecn-accounts-export-${new Date().toISOString().slice(0, 10)}.json`,
      });
      if (result?.success) {
        alert(`账号数据已成功导出至：${result.filePath}`);
      }
    } catch (e: any) {
      alert(`导出遇到错误: ${e?.message || e}`);
    }
  };

  // Group models by provider
  const providerGroups = TRAE_MODELS.reduce((acc, m) => {
    if (!acc[m.provider]) acc[m.provider] = [];
    acc[m.provider].push(m);
    return acc;
  }, {} as Record<string, typeof TRAE_MODELS>);

  return (
    <div className="p-6 space-y-6 max-w-7xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">
            你好{currentAccount ? `, ${currentAccount.username || currentAccount.email}` : ''} 👋
          </h1>
          <p className="text-dark-400 mt-1">TraeCN Tools 管理面板</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => { window.location.hash = '/accounts'; }}
            className="flex items-center gap-2 px-4 py-2 bg-dark-800 hover:bg-dark-700 border border-dark-600 rounded-lg text-sm transition-colors text-dark-200"
          >
            <Plus size={16} />
            添加账号
          </button>
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm font-medium transition-colors text-white disabled:opacity-50"
          >
            <RefreshCw size={16} className={refreshing ? 'animate-spin' : ''} />
            {refreshing ? '刷新中...' : '刷新状态'}
          </button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-5 gap-4">
        <StatCard
          icon={<Users size={20} className="text-blue-400" />}
          value={totalAccounts}
          label="总账号数"
          color="blue"
          onClick={() => { window.location.hash = '/accounts'; }}
        />
        <StatCard
          icon={<Zap size={20} className="text-green-400" />}
          value={`${availableModels.length || TRAE_MODELS.length}`}
          label="可用模型"
          sublabel={availableModels.length > 0 ? '网关已注册' : '全量支持'}
          color="green"
          onClick={() => { window.location.hash = '/proxy'; }}
        />
        <StatCard
          icon={<Activity size={20} className="text-cyan-400" />}
          value={activeAccounts.length}
          label="活跃账号"
          color="cyan"
          onClick={() => { window.location.hash = '/accounts'; }}
        />
        <StatCard
          icon={<TrendingUp size={20} className="text-orange-400" />}
          value={proxyRunning ? '运行中' : '已停止'}
          label="反代服务"
          color={proxyRunning ? 'green' : 'red'}
          onClick={() => { window.location.hash = '/proxy'; }}
        />
        <StatCard
          icon={backendHealth === 'healthy' ? <CheckCircle size={20} className="text-green-400" /> : <XCircle size={20} className="text-red-400" />}
          value={backendHealth === 'healthy' ? '正常' : backendHealth === 'error' ? '异常' : '检查中'}
          label="API 探测"
          sublabel={backendHealth === 'healthy' ? backendUrl : '点击重新检测'}
          color={backendHealth === 'healthy' ? 'green' : backendHealth === 'error' ? 'red' : 'gray'}
          onClick={handleRefresh}
        />
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-2 gap-6">
        {/* Current Account */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-4">
            <div className={`w-2 h-2 rounded-full ${currentAccount ? 'bg-green-400 pulse-dot' : 'bg-dark-500'}`} />
            <h2 className="text-base font-semibold">当前生效账号</h2>
          </div>
          {currentAccount ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center font-bold text-white">
                    {(currentAccount.email[0] || 'T').toUpperCase()}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="text-dark-100 font-medium">{currentAccount.username || currentAccount.email}</span>
                      {currentAccount.isPro ? (
                        <span className="px-2 py-0.5 bg-blue-500/20 text-blue-400 border border-blue-500/30 rounded text-xs font-medium">
                          PRO 会员
                        </span>
                      ) : (
                        <span className="px-2 py-0.5 bg-green-500/20 text-green-400 border border-green-500/30 rounded text-xs font-medium">
                          基础账号
                        </span>
                      )}
                    </div>
                    <div className="text-xs text-dark-400 mt-0.5 font-mono truncate max-w-[280px]">
                      {currentAccount.email}
                    </div>
                  </div>
                </div>
              </div>

              {/* 真实权益与状态说明（取代虚假100%配额条） */}
              <div className="bg-dark-900/60 border border-dark-700/60 rounded-xl p-3.5 space-y-2">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-dark-300 flex items-center gap-1.5">
                    <ShieldCheck size={14} className="text-green-400" />
                    模型调用通道
                  </span>
                  <span className="text-green-400 font-medium">
                    {currentAccount.disabled ? '已禁用' : '正常就绪 (Ready)'}
                  </span>
                </div>
                <div className="text-xs text-dark-400 leading-relaxed">
                  当前账号由 Trae CN 官方原生会话直接驱动，支持 21 个内置大语言模型自由调用。
                </div>
              </div>

              <button
                onClick={() => { window.location.hash = '/accounts'; }}
                className="w-full py-2 bg-dark-700/50 hover:bg-dark-600/50 border border-dark-600/50 rounded-lg text-sm text-dark-200 transition-colors"
              >
                切换账号
              </button>
            </div>
          ) : (
            <div className="text-center py-8 text-dark-400">
              <p>暂未选择生效账号</p>
              <button
                onClick={() => { window.location.hash = '/accounts'; }}
                className="mt-3 px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm text-white transition-colors"
              >
                前往添加或导入账号
              </button>
            </div>
          )}
        </div>

        {/* Best Account Recommendation */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-4">
            <TrendingUp size={16} className="text-cyan-400" />
            <h2 className="text-base font-semibold">首选账号推荐</h2>
          </div>
          {activeAccounts.length > 0 ? (
            <div className="space-y-4">
              <div className="bg-gradient-to-r from-green-500/10 to-cyan-500/10 border border-green-500/20 rounded-xl p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-xs text-dark-400 mb-1">活跃健康度最优</div>
                    <div className="text-lg font-semibold text-white truncate max-w-[260px]">
                      {activeAccounts[0].email}
                    </div>
                    <div className="text-xs text-green-400 mt-1">
                      {activeAccounts[0].id === currentAccountId ? '当前已生效' : '随时可切换'}
                    </div>
                  </div>
                  <div className="w-12 h-12 rounded-full bg-green-500/20 flex items-center justify-center">
                    <UserCheck size={22} className="text-green-400" />
                  </div>
                </div>
              </div>
              <button
                onClick={() => {
                  handleDashboardSwitchAccount(activeAccounts[0].id);
                }}
                disabled={activeAccounts[0].id === currentAccountId && (!proxyRunning || runtimeActiveAccountId === activeAccounts[0].id)}
                className="w-full py-2.5 bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-500 hover:to-cyan-500 rounded-lg text-sm font-medium transition-all text-white disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {(() => {
                  if (activeAccounts[0].id === currentAccountId) {
                    if (proxyRunning && runtimeActiveAccountId !== activeAccounts[0].id) {
                      return '当前首选 (待重启代理后生效)';
                    }
                    return '当前已是该账号';
                  }
                  return '一键切换此账号';
                })()}
              </button>
            </div>
          ) : (
            <div className="text-center py-8 text-dark-400">
              <p>暂无可用活跃账号</p>
              <button
                onClick={() => { window.location.hash = '/accounts'; }}
                className="mt-3 px-4 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm text-dark-200 transition-colors"
              >
                立即添加账号
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Available Models */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-semibold">支持模型清单</h2>
          <span className="text-xs text-dark-400">
            共 {availableModels.length || TRAE_MODELS.length} 个模型已接入
          </span>
        </div>
        <div className="grid grid-cols-3 gap-3">
          {availableModels.length > 0 ? (
            (() => {
              const grouped = availableModels.reduce((acc, m: any) => {
                const provider = m.provider || 'other';
                if (!acc[provider]) acc[provider] = [];
                acc[provider].push(m);
                return acc;
              }, {} as Record<string, typeof availableModels>);

              return Object.entries(grouped).map(([provider, models]) => (
                <div key={provider} className="bg-dark-900/50 rounded-lg p-3">
                  <div className="flex items-center gap-2 mb-2">
                    <div
                      className="w-2 h-2 rounded-full"
                      style={{ backgroundColor: PROVIDER_COLORS[provider] || '#6b7280' }}
                    />
                    <span className="text-xs font-medium text-dark-300 uppercase">{provider}</span>
                  </div>
                  <div className="space-y-1">
                    {models.map((m: any) => (
                      <div key={m.id || m.configName} className="flex items-center gap-2 text-sm text-dark-200">
                        <span>⚡</span>
                        <span className="truncate">{m.display_name || m.name || m.displayName || m.id}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ));
            })()
          ) : (
            Object.entries(providerGroups).map(([provider, models]) => (
              <div key={provider} className="bg-dark-900/50 rounded-lg p-3">
                <div className="flex items-center gap-2 mb-2">
                  <div
                    className="w-2 h-2 rounded-full"
                    style={{ backgroundColor: PROVIDER_COLORS[provider] }}
                  />
                  <span className="text-xs font-medium text-dark-300 uppercase">{provider}</span>
                </div>
                <div className="space-y-1">
                  {models.map((m) => (
                    <div key={m.configName} className="flex items-center gap-2 text-sm text-dark-200">
                      <span>{m.icon}</span>
                      <span className="truncate">{m.displayName}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Quick Links */}
      <div className="grid grid-cols-2 gap-4">
        <a
          href="#/accounts"
          className="flex items-center justify-between bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 hover:border-dark-600 transition-colors group"
        >
          <span className="text-blue-400 font-medium">查看与管理所有账号</span>
          <ArrowRight size={18} className="text-dark-500 group-hover:text-blue-400 transition-colors" />
        </a>
        <button
          onClick={handleExportAccounts}
          className="flex items-center justify-between bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 hover:border-dark-600 transition-colors group text-left"
        >
          <span className="text-blue-400 font-medium">导出账号数据备份</span>
          <Download size={18} className="text-dark-500 group-hover:text-blue-400 transition-colors" />
        </button>
      </div>
    </div>
  );
}

function StatCard({ icon, value, label, sublabel, color, onClick }: {
  icon: React.ReactNode;
  value: string | number;
  label: string;
  sublabel?: string;
  color: string;
  onClick?: () => void;
}) {
  return (
    <div
      className={`bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 card-hover${onClick ? ' cursor-pointer' : ''}`}
      onClick={onClick}
    >
      <div className="mb-3">{icon}</div>
      <div className="text-2xl font-bold text-white">{value}</div>
      <div className="text-sm text-dark-400 mt-1">{label}</div>
      {sublabel && (
        <div className="text-xs text-green-400 mt-0.5 truncate">{sublabel}</div>
      )}
    </div>
  );
}