import React, { useState, useEffect } from 'react';
import { useAppStore } from '../store';
import { TRAE_MODELS, PROVIDER_COLORS } from '../types';
import type { ModelMapping } from '../types';
import {
  Power, PowerOff, Lock, Eye, EyeOff, RefreshCw,
  Copy, Edit2, Plus, Trash2, ChevronDown, ChevronRight,
  ExternalLink, Check, Server, CheckCircle, XCircle,
} from 'lucide-react';
import { apiClient } from '../api/client';
import type { ModelInfo } from '../api/client';

export default function ApiProxy() {
  const {
    proxyConfig, updateProxyConfig,
    proxyRunning, startProxy, stopProxy,
    modelMappings, addModelMapping, removeModelMapping,
    accounts, backendHealth, backendUrl, checkBackendHealth, availableModels, fetchAvailableModels,
  } = useAppStore();

  const [showApiKey, setShowApiKey] = useState(false);
  const [showWebPassword, setShowWebPassword] = useState(false);
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    router: true,
    models: false,
    protocol: true,
  });
  const [newMappingSource, setNewMappingSource] = useState('');
  const [newMappingTarget, setNewMappingTarget] = useState('');
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
        alert('启动失败: ' + (result.error || '未知错误'));      } else {
        // After successful start, check health and fetch models
        setTimeout(() => {
          checkBackendHealth();
          fetchBackendModels();
        }, 1000);      }
    }
  };

  const handleCopy = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopied(key);
    setTimeout(() => setCopied(null), 1500);
  };

  const handleAddMapping = () => {
    if (!newMappingSource || !newMappingTarget) return;
    addModelMapping({
      id: crypto.randomUUID(),
      sourceName: newMappingSource,
      targetModel: newMappingTarget,
    });
    setNewMappingSource('');
    setNewMappingTarget('');
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
          <h1 className="text-lg font-semibold">?? 服务配置</h1>
          <div className="flex items-center gap-2">
            <div className={`w-2 h-2 rounded-full ${proxyRunning ? 'bg-green-400 pulse-dot' : 'bg-dark-500'}`} />
            <span className="text-sm text-dark-400">{proxyRunning ? '服务运行中' : '服务已停止'}</span>
          </div>
        </div>
        <button onClick={handleToggleProxy}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
            proxyRunning
              ? 'bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/30'
              : 'bg-blue-600 text-white hover:bg-blue-500'
          }`}>
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

        {/* Auto Start + LAN */}
        <div className="flex items-center gap-8">
          <div className="flex items-center gap-3">
            <span className="text-sm text-dark-300">跟随应用自动启动</span>
            <ToggleSwitch
              active={proxyConfig.autoStart}
              onChange={(v) => updateProxyConfig({ autoStart: v })}
            />
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
              ? 仅监听 127.0.0.1，仅本机可访问（隐私优先）
            </p>
          </div>

          {/* Auth */}
          <div>
            <div className="flex items-center justify-between mb-1.5">
              <label className="text-sm font-medium text-dark-200">
                访问授权 <HelpTip text="授权模式" />
              </label>
              <div className="flex items-center gap-2">
                <span className="text-xs text-dark-400">已启用</span>
                <ToggleSwitch
                  active={proxyConfig.authEnabled}
                  onChange={(v) => updateProxyConfig({ authEnabled: v })}
                />
              </div>
            </div>
            <div>
              <label className="text-xs text-dark-400 mb-1 block">
                模式 <HelpTip text="选择授权验证方式" />
              </label>
              <select
                value={proxyConfig.authMode}
                onChange={(e) => updateProxyConfig({ authMode: e.target.value as 'auto' | 'bearer' | 'none' })}
                className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200 focus:outline-none focus:border-blue-500"
              >
                <option value="auto">自动（推荐）</option>
                <option value="bearer">Bearer Token</option>
                <option value="none">无授权</option>
              </select>
              <p className="text-xs text-dark-500 mt-1">
                开启后客户端需通过 Authorization: Bearer ... 传入 API 密钥（如选择"除健康检查外"则 /healthz 免鉴权）。
              </p>
            </div>
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
            <button onClick={() => setShowApiKey(!showApiKey)}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
              {showApiKey ? <EyeOff size={16} className="text-dark-400" /> : <Eye size={16} className="text-dark-400" />}
            </button>
            <button onClick={() => updateProxyConfig({ apiKey: 'sk-' + crypto.getRandomValues(new Uint8Array(24)).reduce((s, b) => s + b.toString(16).padStart(2, '0'), '') })}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
              <RefreshCw size={16} className="text-dark-400" />
            </button>
            <button onClick={() => handleCopy(proxyConfig.apiKey, 'apikey')}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
              {copied === 'apikey' ? <Check size={16} className="text-green-400" /> : <Copy size={16} className="text-dark-400" />}
            </button>
          </div>
          <p className="text-xs text-orange-400 mt-1">注意：请妥善保管您的 API 密钥，不要泄露给他人。</p>
        </div>

        {/* Web UI Password */}
        <div>
          <label className="block text-sm font-medium text-dark-200 mb-1.5">
            Web UI 管理后台密码 <HelpTip text="用于管理后台的登录密码" />
          </label>
          <div className="flex items-center gap-2">
            <input
              type={showWebPassword ? 'text' : 'password'}
              value={proxyConfig.webUiPassword || ''}
              onChange={(e) => updateProxyConfig({ webUiPassword: e.target.value })}
              placeholder="〈同 API 密钥〉"
              className="flex-1 px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200 font-mono placeholder:text-dark-500 focus:outline-none focus:border-blue-500"
            />
            <button onClick={() => setShowWebPassword(!showWebPassword)}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
              {showWebPassword ? <EyeOff size={16} className="text-dark-400" /> : <Eye size={16} className="text-dark-400" />}
            </button>
            <button onClick={() => handleCopy(proxyConfig.webUiPassword || proxyConfig.apiKey, 'webpw')}
              className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
              {copied === 'webpw' ? <Check size={16} className="text-green-400" /> : <Copy size={16} className="text-dark-400" />}
            </button>
          </div>
          <p className="text-xs text-dark-500 mt-1">提示：在 Docker/Web 部署中，您可以设置一个独立的登录密码，提高 API 密钥的安全性。</p>
        </div>

        {/* User-Agent Override */}
        <div className="flex items-center justify-between">
          <div>
            <span className="text-sm text-dark-200">User-Agent 覆盖</span>
            <p className="text-xs text-dark-500 mt-0.5">替换发出请求的 User-Agent 头</p>
          </div>
          <ToggleSwitch
            active={proxyConfig.userAgentOverride}
            onChange={(v) => updateProxyConfig({ userAgentOverride: v })}
          />
        </div>
      </div>

      {/* Model Router */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl overflow-hidden">
        <button onClick={() => toggleSection('router')}
          className="flex items-center justify-between w-full px-5 py-4 hover:bg-dark-800/30 transition-colors">
          <div className="flex items-center gap-2">
            <span className="text-base">? 模型路由中心 (Model Router)</span>
          </div>
          {expandedSections.router ? <ChevronDown size={18} className="text-dark-400" /> : <ChevronRight size={18} className="text-dark-400" />}
        </button>

        {expandedSections.router && (
          <div className="px-5 pb-5 space-y-4">
            <p className="text-xs text-dark-400">通过通配符或精确映射自定义模型路由规则</p>

            {/* Preset selector */}
            <div className="flex items-center gap-3">
              <select className="px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200">
                <option>默认预设</option>
              </select>
              <button className="flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm transition-colors">
                ? 应用所选
              </button>
              <button className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
                <Plus size={16} className="text-dark-400" />
              </button>
              <button className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
                <Trash2 size={16} className="text-dark-400" />
              </button>
              <button className="p-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg transition-colors">
                <RefreshCw size={16} className="text-dark-400" />
              </button>
            </div>

            {/* Background task model */}
            <div className="flex items-center justify-between bg-dark-900/50 rounded-lg p-3">
              <div>
                <span className="text-sm text-dark-200">? 后台任务模型</span>
                <p className="text-xs text-dark-500 mt-0.5">用于标题生成、摘要提取等后台自动化任务 (默认: doubao-seed-1.6)</p>
              </div>
              <select className="px-3 py-1.5 bg-dark-800 border border-dark-600 rounded-lg text-xs text-dark-300">
                <option value="">Default (doubao-seed-1.6)</option>
                {TRAE_MODELS.map(m => (
                  <option key={m.configName} value={m.configName}>{m.displayName}</option>
                ))}
              </select>
            </div>

            {/* Custom Mappings */}
            <div>
              <h4 className="text-sm font-medium text-dark-200 mb-2">→ 自定义映射 (CUSTOM MAPPINGS)</h4>
              <p className="text-xs text-dark-500 mb-3">
                ? 支持手动输入任意模型 ID，可使用未发布模型(如 claude-opus-4-6)。
                <span className="text-orange-400">注意:并非所有账号都支持未发布模型。</span>
              </p>

              {/* Current mappings */}
              <div className="bg-dark-900/50 rounded-lg p-3 mb-3 min-h-[60px]">
                <div className="text-xs text-dark-500 uppercase mb-2">当前映射列表 (CUSTOM LIST)</div>
                {modelMappings.length === 0 ? (
                  <p className="text-xs text-dark-500 text-center py-4">暂无自定义精确映射</p>
                ) : (
                  <div className="space-y-2">
                    {modelMappings.map(m => (
                      <div key={m.id} className="flex items-center justify-between bg-dark-800 rounded px-3 py-2">
                        <span className="text-sm text-dark-300">{m.sourceName} → {m.targetModel}</span>
                        <button onClick={() => removeModelMapping(m.id)}
                          className="text-dark-500 hover:text-red-400 transition-colors">
                          <Trash2 size={14} />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Add mapping */}
              <div className="flex items-center gap-2">
                <span className="text-xs text-dark-400 shrink-0">⊕ 添加映射 (ADD MAPPING)</span>
                <input
                  value={newMappingSource}
                  onChange={(e) => setNewMappingSource(e.target.value)}
                  placeholder="原始名 (如 gpt-4 或 gpt-4*)"
                  className="flex-1 px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-xs text-dark-200 placeholder:text-dark-500 focus:outline-none focus:border-blue-500"
                />
                <select
                  value={newMappingTarget}
                  onChange={(e) => setNewMappingTarget(e.target.value)}
                  className="px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-xs text-dark-200 w-48"
                >
                  <option value="">选择目标模型</option>
                  {TRAE_MODELS.map(m => (
                    <option key={m.configName} value={m.configName}>{m.displayName}</option>
                  ))}
                </select>
                <button onClick={handleAddMapping}
                  className="flex items-center gap-1 px-3 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-xs transition-colors">
                  <Plus size={14} /> 添加
                </button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Multi-Protocol Support */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl overflow-hidden">
        <button onClick={() => toggleSection('protocol')}
          className="flex items-center justify-between w-full px-5 py-4 hover:bg-dark-800/30 transition-colors">
          <div>
            <div className="flex items-center gap-2">
              <span className="text-blue-400 text-lg">?</span>
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
                  <button onClick={() => handleCopy(`${baseUrl}/v1`, 'openai-base')}
                    className="flex items-center gap-1 text-xs text-dark-400 hover:text-dark-200 transition-colors">
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
              <div className="bg-dark-900/50 border border-dark-700/50 rounded-xl p-4 opacity-50">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-medium text-dark-300">Anthropic 协议</span>
                  <Copy size={12} className="text-dark-500" />
                </div>
                <div className="space-y-1 text-xs text-dark-500 font-mono">
                  <div>/v1/messages</div>
                </div>
                <div className="text-xs text-dark-500 mt-2">即将支持</div>
              </div>

              {/* Gemini */}
              <div className="bg-dark-900/50 border border-dark-700/50 rounded-xl p-4 opacity-50">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-sm font-medium text-dark-300">Gemini 协议</span>
                  <Copy size={12} className="text-dark-500" />
                </div>
                <div className="space-y-1 text-xs text-dark-500 font-mono">
                  <div>/v1beta/models/...</div>
                </div>
                <div className="text-xs text-dark-500 mt-2">即将支持</div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Supported Models */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl overflow-hidden">
        <button onClick={() => toggleSection('models')}
          className="flex items-center justify-between w-full px-5 py-4 hover:bg-dark-800/30 transition-colors">
          <span className="text-base">〉 支持模型与集成 (Supported Models & Integration)</span>
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
                      <th className="text-left py-2 font-medium">描述</th>
                      <th className="text-right py-2 font-medium">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(backendModels.length > 0 ? backendModels : TRAE_MODELS).map((m: any) => (
                      <tr key={m.configName || m.id} className="border-b border-dark-700/30 hover:bg-dark-800/30">
                        <td className="py-2.5">
                          <div className="flex items-center gap-2">
                            <span>{(m as any).icon || '?'}</span>
                            <span className="font-medium" style={{ color: PROVIDER_COLORS[(m as any).provider || 'other'] }}>
                              {m.display_name || m.name || (m as any).displayName}
                            </span>
                          </div>
                        </td>
                        <td className="py-2.5 text-dark-400 font-mono text-xs">{m.id || m.configName}</td>
                        <td className="py-2.5 text-dark-400 text-xs">{m.description || (m as any).displayName}</td>
                        <td className="py-2.5 text-right">
                          <button onClick={() => handleCopy(m.id || m.configName, m.id || m.configName)}
                            className="text-xs text-blue-400 hover:text-blue-300 transition-colors">
                            {copied === (m.id || m.configName) ? '? 已复制' : '? 复制'}
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
    api_key="${proxyConfig.apiKey}"
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
      ?
    </span>
  );
}
