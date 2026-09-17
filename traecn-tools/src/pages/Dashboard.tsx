import React, { useEffect } from 'react';
import { useAppStore } from '../store';
import { TRAE_MODELS, PROVIDER_COLORS } from '../types';
import {
  Users, Zap, TrendingUp, Activity, ArrowRight,
  Plus, RefreshCw, Download, UserCheck, Server, CheckCircle, XCircle,
} from 'lucide-react';
import { apiClient } from '../api/client';

export default function Dashboard() {
  const { accounts, proxyRunning, currentAccountId, backendHealth, availableModels, backendUrl, checkBackendHealth, fetchAvailableModels } = useAppStore();
  const currentAccount = accounts.find((a) => a.id === currentAccountId);
  const activeAccounts = accounts.filter((a) => !a.disabled);
  const totalAccounts = accounts.length;
  const proAccounts = accounts.filter((a) => a.isPro).length;

  // Check backend health on mount
  useEffect(() => {
    checkBackendHealth();
    fetchAvailableModels();
  }, []);

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
            你好{currentAccount ? `, ${currentAccount.username || currentAccount.email}` : ''} ?
          </h1>
          <p className="text-dark-400 mt-1">TraeCN Tools 管理面板</p>
        </div>
        <div className="flex items-center gap-3">
          <button className="flex items-center gap-2 px-4 py-2 bg-dark-800 hover:bg-dark-700 border border-dark-600 rounded-lg text-sm transition-colors">
            <Plus size={16} />
            添加账号
          </button>
          <button className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm font-medium transition-colors">
            <RefreshCw size={16} />
            刷新配额
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
        />
        <StatCard
          icon={<Zap size={20} className="text-green-400" />}
          value={`${availableModels.length || TRAE_MODELS.length}`}
          label="可用模型"
          sublabel={availableModels.length > 0 ? '后端提供' : '免费使用'}
          color="green"
        />
        <StatCard
          icon={<Activity size={20} className="text-cyan-400" />}
          value={activeAccounts.length}
          label="活跃账号"
          color="cyan"
        />
        <StatCard
          icon={<TrendingUp size={20} className="text-orange-400" />}
          value={proxyRunning ? '运行中' : '已停止'}
          label="反代服务"
          color={proxyRunning ? 'green' : 'red'}
        />
        <StatCard
          icon={backendHealth === 'healthy' ? <CheckCircle size={20} className="text-green-400" /> : <XCircle size={20} className="text-red-400" />}
          value={backendHealth === 'healthy' ? '正常' : backendHealth === 'error' ? '异常' : '未知'}
          label="API 服务"
          sublabel={backendHealth === 'healthy' ? backendUrl : '点击检查'}
          color={backendHealth === 'healthy' ? 'green' : backendHealth === 'error' ? 'red' : 'gray'}
          onClick={checkBackendHealth}
        />
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-2 gap-6">
        {/* Current Account */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-4">
            <div className="w-2 h-2 rounded-full bg-green-400 pulse-dot" />
            <h2 className="text-base font-semibold">当前账号</h2>
          </div>
          {currentAccount ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="text-dark-200">{currentAccount.email}</span>
                  {currentAccount.isPro && (
                    <span className="px-2 py-0.5 bg-blue-500/20 text-blue-400 rounded text-xs font-medium">
                      ◆ PRO
                    </span>
                  )}
                </div>
              </div>
              <div className="space-y-3">
                <QuotaBar label="模型配额" value={100} sublabel="免费 · 无限" color="green" />
                <div className="text-xs text-dark-400 mt-2">
                  TraeCN 所有模型完全免费，无配额限制
                </div>
              </div>
              <button className="w-full py-2 bg-dark-700/50 hover:bg-dark-600/50 border border-dark-600/50 rounded-lg text-sm text-dark-300 transition-colors">
                切换账号
              </button>
            </div>
          ) : (
            <div className="text-center py-8 text-dark-400">
              <p>未选择当前账号</p>
              <button className="mt-3 px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm text-white transition-colors">
                添加账号
              </button>
            </div>
          )}
        </div>

        {/* Best Account Recommendation */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-4">
            <TrendingUp size={16} className="text-cyan-400" />
            <h2 className="text-base font-semibold">最佳账号推荐</h2>
          </div>
          {activeAccounts.length > 0 ? (
            <div className="space-y-4">
              <div className="bg-gradient-to-r from-green-500/10 to-cyan-500/10 border border-green-500/20 rounded-xl p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="text-xs text-dark-400 mb-1">推荐用于所有模型</div>
                    <div className="text-lg font-semibold text-white">
                      {activeAccounts[0].email}
                    </div>
                  </div>
                  <div className="w-12 h-12 rounded-full bg-green-500/20 flex items-center justify-center">
                    <span className="text-green-400 font-bold text-lg">?</span>
                  </div>
                </div>
              </div>
              <button className="w-full py-2.5 bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-500 hover:to-cyan-500 rounded-lg text-sm font-medium transition-all">
                一键切换最佳
              </button>
            </div>
          ) : (
            <div className="text-center py-8 text-dark-400">
              <p>暂无可用账号</p>
            </div>
          )}
        </div>
      </div>

      {/* Available Models */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
        <h2 className="text-base font-semibold mb-4">可用模型 · 全部免费</h2>
        <div className="grid grid-cols-3 gap-3">
          {availableModels.length > 0 ? (
            // Show backend models
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
                    <span className="text-xs font-medium text-dark-300">{provider}</span>
                  </div>
                  <div className="space-y-1">
                    {models.map((m: any) => (
                      <div key={m.id || m.configName} className="flex items-center gap-2 text-sm text-dark-200">
                        <span>{(m as any).icon || '?'}</span>
                        <span>{m.display_name || m.name || (m as any).displayName}</span>
                      </div>
                    ))}
                  </div>
                </div>
              ));
            })()
          ) : (
            // Show default TRAE_MODELS
            Object.entries(providerGroups).map(([provider, models]) => (
              <div key={provider} className="bg-dark-900/50 rounded-lg p-3">
                <div className="flex items-center gap-2 mb-2">
                  <div
                    className="w-2 h-2 rounded-full"
                    style={{ backgroundColor: PROVIDER_COLORS[provider] }}
                  />
                  <span className="text-xs font-medium text-dark-300">{provider}</span>
                </div>
                <div className="space-y-1">
                  {models.map((m) => (
                    <div key={m.configName} className="flex items-center gap-2 text-sm text-dark-200">
                      <span>{m.icon}</span>
                      <span>{m.displayName}</span>
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
        <a href="#/accounts" className="flex items-center justify-between bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 hover:border-dark-600 transition-colors group">
          <span className="text-blue-400 font-medium">查看所有账号</span>
          <ArrowRight size={18} className="text-dark-500 group-hover:text-blue-400 transition-colors" />
        </a>
        <button className="flex items-center justify-between bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 hover:border-dark-600 transition-colors group text-left">
          <span className="text-blue-400 font-medium">导出账号数据</span>
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
        <div className="text-xs text-green-400 mt-0.5">? {sublabel}</div>
      )}
    </div>
  );
}

function QuotaBar({ label, value, sublabel, color }: {
  label: string;
  value: number;
  sublabel: string;
  color: string;
}) {
  const colorMap: Record<string, string> = {
    green: 'bg-green-500',
    blue: 'bg-blue-500',
    orange: 'bg-orange-500',
    red: 'bg-red-500',
  };

  return (
    <div>
      <div className="flex items-center justify-between text-sm mb-1.5">
        <span className="text-dark-300">{label}</span>
        <span className="text-dark-200">{sublabel} <span className="font-semibold text-green-400">{value}%</span></span>
      </div>
      <div className="h-2 bg-dark-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full ${colorMap[color] || colorMap.green} relative`}
          style={{ width: `${value}%` }}
        >
          <div className="absolute inset-0 progress-bar-shine" />
        </div>
      </div>
    </div>
  );
}
