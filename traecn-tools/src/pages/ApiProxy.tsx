import React, { useState, useEffect } from 'react';
import { useAppStore } from '../store';
import { TRAE_MODELS, PROVIDER_COLORS } from '../types';
import {
  Power, PowerOff, Eye, EyeOff, RefreshCw,
  Copy, ChevronDown, ChevronRight,
  ExternalLink, Check, Server, CheckCircle, XCircle,
} from 'lucide-react';
import { apiClient } from '../api/client';
import type { ModelInfo } from '../api/client';

export default function ApiProxy() {
  const {
    proxyConfig, updateProxyConfig,
    proxyRunning, startProxy, stopProxy,
    backendHealth, checkBackendHealth,
  } = useAppStore();

  const [showApiKey, setShowApiKey] = useState(false);
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    models: false,
    protocol: true,
  });
  const [copied, setCopied] = useState<string | null>(null);
  const [backendModels, setBackendModels] = useState<ModelInfo[]>([]);

  // Fetch backend models on mount and when proxy starts
  useEffect(() => {
    if (proxyRunning) {
      fetchBackendModels();
    }
  }, [proxyRunning]);

  const fetchBackendModels = async () => {
    const result = await apiClient.getModels();
    if (result.success && result.data) {
      setBackendModels(result.data.data || []);
    }
  };

  const handleToggleProxy = async () => {
    if (proxyRunning) {
      await stopProxy();
    } else {
      const result = await startProxy();
      if (!result.success) {
        alert('启动失败: ' + (result.error || '未知错误'));
      } else {
        setTimeout(() => {
          checkBackendHealth();
          fetchBackendModels();
        }, 1000);
      }
    }
  };

  const handleCopy = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopied(key);
    setTimeout(() => setCopied(null), 1500);
  };

  const toggleSection = (key: string) => {
    setExpandedSections((s) => ({ ...s, [key]: !s[key] }));
  };

  const baseUrl = `http://127.0.0.1:${proxyConfig.listenPort}`;

  return (
    <div className="p-6 space-y-6 max-w-4xl">
      {/* Service Config Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold">服务配置</h1>
          <div className="flex items-center gap-2">
            <div className={`w-2 h-2 rounded-full ${proxyRunning ? 'bg-green-400 pulse-dot' : 'bg-dark-500'}`} />
            <span className="text-sm text-dark-400">{proxyRunning ? '服务运行中' : '服务已停止'}</span>
          </div>
        </div>
        <button
          onClick={handleToggleProxy}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            proxyRunning
              ? 'bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30'
              : 'bg-blue-600 text-white hover:bg-blue-500'
          }`}
        >
          {proxyRunning ? <PowerOff size={16} /> : <Power size={16} />}
          {proxyRunning ? '停止服务' : '启动服务'}
        </button>
      </div>

      {/* Basic Config */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-5">
        <div className="grid grid-cols-2 gap-5">
          {/* Listen Port */}
          <div>
            <label className="block text-sm font-medium text-blue-400 mb-1.5">
              监听端口 <HelpTip text="反代服务监听的本地端口" />
            </label>
            <input
              type="number"
              value={proxyConfig.listenPort}
              onChange={(e) => updateProxyConfig({ listenPort: parseInt(e.target.value) || 8045 })}
              className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200 focus:outline-none focus:border-blue-500"
            />
            <p className="text-xs text-dark-500 mt-1">默认 8045，修改端口需重启服务</p>
          </div>

          {/* Request Timeout */}
          <div>
            <label className="block text-sm font-medium text-blue-400 mb-1.5">
              请求超时 <HelpTip text="单位：秒，范围 30-7200" />
            </label>
            <input
              type="number"
              value={proxyConfig.requestTimeout}
              onChange={(e) => updateProxyConfig({ requestTimeout: Math.max(30, Math.min(7200, parseInt(e.target.value) || 120)) })}
              className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200 focus:outline-none focus:border-blue-500"
            />
            <p className="text-xs text-dark-500 mt-1">默认 120 秒，范围 30-7200秒。修改后需重启服务生效。</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-5">
          {/* Allow LAN */}
          <div>
            <label className="block text-sm font-medium text-dark-200 mb-1.5">
              允许局域网访问 <HelpTip text="开启后允许局域网内其他设备访问" />
            </label>
            <ToggleSwitch
              active={proxyConfig.allowLan}
              onChange={(v) => updateProxyConfig({ allowLan: v })}
            />
            <p className="text-xs text-dark-500 mt-1">
              默认仅监听 127.0.0.1，仅本机可访问（隐私优先）
            </p>
          </div>

          {/* Auth */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-sm font-medium text-dark-200">
                访问授权 <HelpTip text="授权模式" />
              </label>
              <div className="flex items-center gap-2">
                <span className="text-xs text-dark-400">{proxyConfig.authEnabled ? '已启用' : '已关闭'}</span>
                <ToggleSwitch
                  active={proxyConfig.authEnabled}
                  onChange={(v) => updateProxyConfig({ authEnabled: v })}
                />
              </div>
            </div>
            <p className="text-xs text-dark-500 mt-1">
              开启后，客户端请求需在 Header 中携带 Authorization: Bearer &lt;API密钥&gt;。若允许局域网访问，系统强制要求开启授权。
            </p>
          </div>
        </div>

        {/* API Key */}
        <div>
          <label className="block text-sm font-medium text-dark-200 mb-1.5">
            API 密钥 <HelpTip text="用于访问授权的密钥" />
          </label>
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <input
                type={showApiKey ? 'text' : 'password'}
                value={proxyConfig.apiKey}
                readOnly
                className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200 font-mono"
              />
            </div>
            <button
              onClick={() => setShowApiKey(!showApiKey)}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors"
              title="查看/隐藏 API Key"
            >
              {showApiKey ? <EyeOff size={16} className="text-dark-400" /> : <Eye size={16} className="text-dark-400" />}
            </button>
            <button
              onClick={() => updateProxyConfig({ apiKey: 'sk-' + crypto.getRandomValues(new Uint8Array(24)).reduce((s, b) => s + b.toString(16).padStart(2, '0'), '') })}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors"
              title="重新生成随机 API Key"
            >
              <RefreshCw size={16} className="text-dark-400" />
            </button>
            <button
              onClick={() => handleCopy(proxyConfig.apiKey, 'apikey')}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors"
              title="复制 API Key"
            >
              {copied === 'apikey' ? <Check size={16} className="text-green-400" /> : <Copy size={16} className="text-dark-400" />}
            </button>
          </div>
          <p className="text-xs text-orange-400 mt-1">注意：请妥善保管您的 API 密钥，不要泄露给他人。</p>
        </div>


      </div>



      {/* Multi-Protocol Support */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl overflow-hidden">
        <button
          onClick={() => toggleSection('protocol')}
          className="flex items-center justify-between w-full px-5 py-4 hover:bg-dark-800/30 transition-colors"
        >
          <div>
            <div className="flex items-center gap-2">
              <span className="text-base font-medium">多协议支持 (Multi-Protocol Support)</span>
            </div>
            <p className="text-xs text-dark-500 mt-0.5">快速同步 API 地址与密钥到本地 AI 工具</p>
          </div>
          {expandedSections.protocol ? <ChevronDown size={18} className="text-dark-400" /> : <ChevronRight size={18} className="text-dark-400" />}
        </button>

        {expandedSections.protocol && (
          <div className="px-5 pb-5 space-y-4">
            <p className="text-sm text-dark-300">反代服务支持 OpenAI 协议，满足不同工具的集成需求</p>

            <div className="grid grid-cols-3 gap-4">
              {/* OpenAI */}
              <div className="bg-dark-900/50 border-2 border-blue-500/30 rounded-xl p-4">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-medium text-blue-400">OpenAI 协议</span>
                  <button
                    onClick={() => handleCopy(`${baseUrl}/v1`, 'openai-base')}
                    className="flex items-center gap-1 text-xs text-dark-400 hover:text-dark-200 transition-colors"
                  >
                    {copied === 'openai-base' ? <Check size={12} className="text-green-400" /> : <Copy size={12} />}
                    复制 BASE
                  </button>
                </div>
                <div className="space-y-1 text-xs text-dark-300 font-mono">
                  <div>/v1/chat/completions</div>
                  <div>/v1/completions</div>
                  <div className="text-blue-400">/v1/responses (Codex)</div>
                </div>
              </div>

              {/* Anthropic */}
              <div className="bg-dark-900/50 border-2 border-purple-500/30 rounded-xl p-4">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-medium text-purple-400">Anthropic 协议</span>
                  <button
                    onClick={() => handleCopy(baseUrl, 'anthropic-base')}
                    className="flex items-center gap-1 text-xs text-dark-400 hover:text-dark-200 transition-colors"
                  >
                    {copied === 'anthropic-base' ? <Check size={12} className="text-green-400" /> : <Copy size={12} />}
                    复制 BASE
                  </button>
                </div>
                <div className="space-y-1 text-xs text-dark-300 font-mono">
                  <div>/v1/messages</div>
                </div>
                <div className="text-xs text-purple-400/80 mt-2">支持 Claude Code / Continue</div>
              </div>

              {/* Gemini (P6: 明确标为规划中，去除假复制图标) */}
              <div className="bg-dark-900/50 border border-dark-700/50 rounded-xl p-4 opacity-60">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-medium text-dark-300">Gemini 协议</span>
                  <span className="text-[10px] text-dark-400 px-1.5 py-0.5 bg-dark-800 rounded">Roadmap</span>
                </div>
                <div className="space-y-1 text-xs text-dark-500 font-mono">
                  <div>/v1beta/models/...</div>
                </div>
                <div className="text-xs text-dark-400 mt-2">待后续版本开放</div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Supported Models */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl overflow-hidden">
        <button
          onClick={() => toggleSection('models')}
          className="flex items-center justify-between w-full px-5 py-4 hover:bg-dark-800/30 transition-colors"
        >
          <span className="text-base font-medium">支持模型与集成 (Supported Models & Integration)</span>
          {expandedSections.models ? <ChevronDown size={18} className="text-dark-400" /> : <ChevronRight size={18} className="text-dark-400" />}
        </button>

        {expandedSections.models && (
          <div className="px-5 pb-5">
            <div className="grid grid-cols-[1fr_1fr] gap-6">
              {/* Model table */}
              <div>
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-xs text-dark-400 border-b border-dark-700/50">
                      <th className="text-left py-2 font-medium">模型名称</th>
                      <th className="text-left py-2 font-medium">模型 ID</th>
                      <th className="text-right py-2 font-medium">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(backendModels.length > 0 ? backendModels : TRAE_MODELS).map((m: any) => (
                      <tr key={m.configName || m.id} className="border-b border-dark-700/30 hover:bg-dark-800/30">
                        <td className="py-2.5">
                          <div className="flex items-center gap-2">
                            <span>⚡</span>
                            <span className="font-medium truncate max-w-[140px]" style={{ color: PROVIDER_COLORS[(m as any).provider || 'other'] || '#94a3b8' }}>
                              {m.display_name || m.name || (m as any).displayName || m.id}
                            </span>
                          </div>
                        </td>
                        <td className="py-2.5 text-dark-400 font-mono text-xs truncate max-w-[160px]">{m.id || m.configName}</td>
                        <td className="py-2.5 text-right">
                          <button
                            onClick={() => handleCopy(m.id || m.configName, m.id || m.configName)}
                            className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                          >
                            {copied === (m.id || m.configName) ? '已复制' : '复制 ID'}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {/* Quick Integration */}
              <div className="bg-dark-900/50 rounded-xl p-4">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-semibold text-dark-200">快速集成 (QUICK INTEGRATION)</span>
                  <span className="px-2 py-0.5 bg-dark-700 text-dark-300 rounded text-xs">Python (OpenAI SDK)</span>
                </div>
                <pre className="text-xs text-dark-300 font-mono bg-dark-950 rounded-lg p-4 overflow-x-auto leading-relaxed">
{`from openai import OpenAI

client = OpenAI(
    base_url="http://127.0.0.1:${proxyConfig.listenPort}/v1",
    api_key="${proxyConfig.apiKey || 'YOUR_API_KEY'}"
)

response = client.chat.completions.create(
    model="deepseek-r1",
    messages=[{"role": "user", "content": "Hello"}]
)

print(response.choices[0].message.content)`}
                </pre>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function ToggleSwitch({ active, onChange }: { active: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      onClick={() => onChange(!active)}
      className={`toggle-switch ${active ? 'active' : 'inactive'}`}
      role="switch"
      aria-checked={active}
    />
  );
}

function HelpTip({ text }: { text: string }) {
  return (
    <span className="inline-block ml-1 text-dark-500 cursor-help" title={text}>
      ⓘ
    </span>
  );
}
