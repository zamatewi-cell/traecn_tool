import React, { useState } from 'react';
import { Key, Plus, Copy, Check, Trash2, Shield, Clock, Globe } from 'lucide-react';

interface UserToken {
  id: string;
  name: string;
  token: string;
  createdAt: string;
  lastUsed?: string;
  expiresAt?: string;
  requests: number;
  enabled: boolean;
  allowedIPs?: string[];
}

export default function Tokens() {
  const [tokens, setTokens] = useState<UserToken[]>([
    {
      id: '1',
      name: '开发环境 Token',
      token: 'sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx',
      createdAt: '2024-01-15',
      lastUsed: '2024-01-20 14:30',
      requests: 1250,
      enabled: true,
    },
    {
      id: '2',
      name: '生产环境 Token',
      token: 'sk-yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy',
      createdAt: '2024-01-10',
      lastUsed: '2024-01-20 15:45',
      requests: 3420,
      enabled: true,
    },
    {
      id: '3',
      name: '测试 Token',
      token: 'sk-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz',
      createdAt: '2024-01-05',
      expiresAt: '2024-02-05',
      requests: 580,
      enabled: false,
    },
  ]);

  const [showCreateModal, setShowCreateModal] = useState(false);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const handleCopy = (token: string, id: string) => {
    navigator.clipboard.writeText(token);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const handleDelete = (id: string) => {
    if (confirm('确定要删除这个 Token 吗？此操作不可恢复。')) {
      setTokens(tokens.filter(t => t.id !== id));
    }
  };

  const handleToggle = (id: string) => {
    setTokens(tokens.map(t => 
      t.id === id ? { ...t, enabled: !t.enabled } : t
    ));
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Key size={20} className="text-blue-400" />
          <h1 className="text-lg font-semibold">用户 Token</h1>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm text-white font-medium transition-colors"
        >
          <Plus size={16} />
          创建 Token
        </button>
      </div>

      {/* Info Banner */}
      <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-4 mb-6">
        <div className="flex items-start gap-3">
          <Shield size={20} className="text-blue-400 mt-0.5" />
          <div>
            <h3 className="text-sm font-medium text-blue-300 mb-1">Token 安全提示</h3>
            <p className="text-sm text-blue-400/80">
              Token 是您访问 API 的凭证，请妥善保管，不要分享给他人。Token 丢失请立即删除并重新创建。
            </p>
          </div>
        </div>
      </div>

      {/* Token List */}
      <div className="space-y-3">
        {tokens.map((token) => (
          <div
            key={token.id}
            className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 hover:border-dark-600 transition-colors"
          >
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <h3 className="text-base font-medium text-white">{token.name}</h3>
                  <span
                    className={`px-2 py-0.5 rounded text-xs font-medium ${
                      token.enabled
                        ? 'bg-green-500/20 text-green-400'
                        : 'bg-dark-600 text-dark-400'
                    }`}
                  >
                    {token.enabled ? '已启用' : '已禁用'}
                  </span>
                </div>
                <div className="flex items-center gap-4 text-sm text-dark-400">
                  <div className="flex items-center gap-1">
                    <Clock size={14} />
                    <span>创建：{token.createdAt}</span>
                  </div>
                  {token.lastUsed && (
                    <div className="flex items-center gap-1">
                      <Clock size={14} />
                      <span>使用：{token.lastUsed}</span>
                    </div>
                  )}
                  {token.expiresAt && (
                    <div className="flex items-center gap-1">
                      <Clock size={14} />
                      <span>过期：{token.expiresAt}</span>
                    </div>
                  )}
                  <div className="flex items-center gap-1">
                    <Globe size={14} />
                    <span>请求：{token.requests.toLocaleString()}</span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => handleCopy(token.token, token.id)}
                  className="p-2 hover:bg-dark-700 rounded-lg transition-colors"
                  title="复制 Token"
                >
                  {copiedId === token.id ? (
                    <Check size={18} className="text-green-400" />
                  ) : (
                    <Copy size={18} className="text-dark-400" />
                  )}
                </button>
                <button
                  onClick={() => handleToggle(token.id)}
                  className="p-2 hover:bg-dark-700 rounded-lg transition-colors"
                  title={token.enabled ? '禁用' : '启用'}
                >
                  <Shield
                    size={18}
                    className={token.enabled ? 'text-green-400' : 'text-dark-500'}
                  />
                </button>
                <button
                  onClick={() => handleDelete(token.id)}
                  className="p-2 hover:bg-red-500/20 rounded-lg transition-colors"
                  title="删除"
                >
                  <Trash2 size={18} className="text-red-400" />
                </button>
              </div>
            </div>
            <div className="mt-3 pt-3 border-t border-dark-700/50">
              <div className="flex items-center gap-2">
                <code className="flex-1 px-3 py-2 bg-dark-900 rounded-lg text-sm text-dark-300 font-mono overflow-x-auto">
                  {token.token}
                </code>
              </div>
            </div>
          </div>
        ))}
      </div>

      {tokens.length === 0 && (
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-12 text-center">
          <Key size={48} className="mx-auto mb-4 text-dark-600" />
          <h3 className="text-lg font-medium text-dark-400 mb-2">暂无 Token</h3>
          <p className="text-sm text-dark-500 mb-4">创建您的第一个 API Token 开始使用</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm text-white font-medium transition-colors"
          >
            创建 Token
          </button>
        </div>
      )}

      {/* Create Modal Placeholder */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
          <div className="bg-dark-800 border border-dark-700 rounded-xl p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold mb-4">创建 Token</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-dark-400 mb-2">Token 名称</label>
                <input
                  type="text"
                  placeholder="例如：开发环境 Token"
                  className="w-full px-3 py-2 bg-dark-900 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
              <div>
                <label className="block text-sm text-dark-400 mb-2">过期时间</label>
                <input
                  type="date"
                  className="w-full px-3 py-2 bg-dark-900 border border-dark-700 rounded-lg text-white text-sm focus:outline-none focus:border-blue-500"
                />
              </div>
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="ipRestriction"
                  className="w-4 h-4 rounded bg-dark-900 border-dark-700"
                />
                <label htmlFor="ipRestriction" className="text-sm text-dark-400">
                  限制 IP 访问
                </label>
              </div>
            </div>
            <div className="flex items-center gap-3 mt-6">
              <button
                onClick={() => setShowCreateModal(false)}
                className="flex-1 px-4 py-2 bg-dark-700 hover:bg-dark-600 rounded-lg text-sm text-white transition-colors"
              >
                取消
              </button>
              <button className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm text-white font-medium transition-colors">
                创建
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
