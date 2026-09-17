import React, { useState } from 'react';
import { Globe, Shield, ShieldAlert, ShieldCheck, Plus, Trash2, ToggleLeft, ToggleRight, Search, Filter } from 'lucide-react';

interface IpRule {
  id: string;
  ip: string;
  type: 'whitelist' | 'blacklist';
  description: string;
  enabled: boolean;
  createdAt: string;
}

export default function IpManagement() {
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<'all' | 'whitelist' | 'blacklist'>('all');
  const [showAddModal, setShowAddModal] = useState(false);
  const [newIp, setNewIp] = useState('');
  const [newType, setNewType] = useState<'whitelist' | 'blacklist'>('whitelist');
  const [newDescription, setNewDescription] = useState('');

  // Mock data - replace with actual data from backend
  const [ipRules, setIpRules] = useState<IpRule[]>([
    {
      id: '1',
      ip: '192.168.1.0/24',
      type: 'whitelist',
      description: 'Internal network',
      enabled: true,
      createdAt: '2024-01-15',
    },
    {
      id: '2',
      ip: '10.0.0.0/8',
      type: 'whitelist',
      description: 'Private network range',
      enabled: true,
      createdAt: '2024-01-16',
    },
    {
      id: '3',
      ip: '203.0.113.0/24',
      type: 'blacklist',
      description: 'Blocked external network',
      enabled: false,
      createdAt: '2024-01-17',
    },
  ]);

  const filteredRules = ipRules.filter(rule => {
    const matchesSearch = rule.ip.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         rule.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = filterType === 'all' || rule.type === filterType;
    return matchesSearch && matchesType;
  });

  const toggleRule = (id: string) => {
    setIpRules(rules =>
      rules.map(rule =>
        rule.id === id ? { ...rule, enabled: !rule.enabled } : rule
      )
    );
  };

  const deleteRule = (id: string) => {
    setIpRules(rules => rules.filter(rule => rule.id !== id));
  };

  const addRule = () => {
    if (!newIp.trim()) return;

    const rule: IpRule = {
      id: Date.now().toString(),
      ip: newIp,
      type: newType,
      description: newDescription || `${newType === 'whitelist' ? 'Whitelist' : 'Blacklist'} rule`,
      enabled: true,
      createdAt: new Date().toISOString().split('T')[0],
    };

    setIpRules([...ipRules, rule]);
    setNewIp('');
    setNewType('whitelist');
    setNewDescription('');
    setShowAddModal(false);
  };

  const stats = {
    total: ipRules.length,
    whitelist: ipRules.filter(r => r.type === 'whitelist').length,
    blacklist: ipRules.filter(r => r.type === 'blacklist').length,
    active: ipRules.filter(r => r.enabled).length,
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Globe size={20} className="text-blue-400" />
          <h1 className="text-lg font-semibold">IP 管理</h1>
        </div>
        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
        >
          <Plus size={16} />
          添加规则
        </button>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-dark-400 text-sm">总规则数</span>
            <Shield size={16} className="text-blue-400" />
          </div>
          <div className="text-2xl font-bold text-dark-200">{stats.total}</div>
        </div>
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-dark-400 text-sm">白名单</span>
            <ShieldCheck size={16} className="text-green-400" />
          </div>
          <div className="text-2xl font-bold text-dark-200">{stats.whitelist}</div>
        </div>
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-dark-400 text-sm">黑名单</span>
            <ShieldAlert size={16} className="text-red-400" />
          </div>
          <div className="text-2xl font-bold text-dark-200">{stats.blacklist}</div>
        </div>
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-dark-400 text-sm">已启用</span>
            <ToggleRight size={16} className="text-purple-400" />
          </div>
          <div className="text-2xl font-bold text-dark-200">{stats.active}</div>
        </div>
      </div>

      {/* Filter Bar */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 mb-6">
        <div className="flex items-center gap-4">
          <div className="flex-1 relative">
            <Search size={16} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-dark-500" />
            <input
              type="text"
              placeholder="搜索 IP 或描述..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-dark-900 border border-dark-700 rounded-lg text-dark-200 placeholder-dark-500 focus:outline-none focus:border-blue-500"
            />
          </div>
          <div className="flex items-center gap-2">
            <Filter size={16} className="text-dark-500" />
            <select
              value={filterType}
              onChange={(e) => setFilterType(e.target.value as typeof filterType)}
              className="px-4 py-2 bg-dark-900 border border-dark-700 rounded-lg text-dark-200 focus:outline-none focus:border-blue-500"
            >
              <option value="all">全部</option>
              <option value="whitelist">白名单</option>
              <option value="blacklist">黑名单</option>
            </select>
          </div>
        </div>
      </div>

      {/* IP Rules Table */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead>
            <tr className="border-b border-dark-700/50">
              <th className="text-left py-3 px-4 text-dark-400 font-medium text-sm">IP 地址/范围</th>
              <th className="text-left py-3 px-4 text-dark-400 font-medium text-sm">类型</th>
              <th className="text-left py-3 px-4 text-dark-400 font-medium text-sm">描述</th>
              <th className="text-left py-3 px-4 text-dark-400 font-medium text-sm">创建日期</th>
              <th className="text-left py-3 px-4 text-dark-400 font-medium text-sm">状态</th>
              <th className="text-right py-3 px-4 text-dark-400 font-medium text-sm">操作</th>
            </tr>
          </thead>
          <tbody>
            {filteredRules.length === 0 ? (
              <tr>
                <td colSpan={6} className="py-12 text-center text-dark-500">
                  暂无 IP 规则
                </td>
              </tr>
            ) : (
              filteredRules.map((rule) => (
                <tr key={rule.id} className="border-b border-dark-700/30 hover:bg-dark-800/30 transition-colors">
                  <td className="py-3 px-4">
                    <code className="text-dark-200 bg-dark-900 px-2 py-1 rounded">{rule.ip}</code>
                  </td>
                  <td className="py-3 px-4">
                    <span className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium ${
                      rule.type === 'whitelist'
                        ? 'bg-green-500/20 text-green-400'
                        : 'bg-red-500/20 text-red-400'
                    }`}>
                      {rule.type === 'whitelist' ? <ShieldCheck size={12} /> : <ShieldAlert size={12} />}
                      {rule.type === 'whitelist' ? '白名单' : '黑名单'}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-dark-300">{rule.description}</td>
                  <td className="py-3 px-4 text-dark-400 text-sm">{rule.createdAt}</td>
                  <td className="py-3 px-4">
                    <button
                      onClick={() => toggleRule(rule.id)}
                      className={`flex items-center gap-1 px-2 py-1 rounded text-xs font-medium transition-colors ${
                        rule.enabled
                          ? 'bg-green-500/20 text-green-400 hover:bg-green-500/30'
                          : 'bg-dark-700 text-dark-400 hover:bg-dark-600'
                      }`}
                    >
                      {rule.enabled ? <ToggleRight size={14} /> : <ToggleLeft size={14} />}
                      {rule.enabled ? '已启用' : '已禁用'}
                    </button>
                  </td>
                  <td className="py-3 px-4 text-right">
                    <button
                      onClick={() => deleteRule(rule.id)}
                      className="p-2 hover:bg-red-500/20 hover:text-red-400 rounded-lg transition-colors text-dark-500"
                      title="删除规则"
                    >
                      <Trash2 size={16} />
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Add Rule Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50">
          <div className="bg-dark-800 border border-dark-700 rounded-xl p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold text-dark-200 mb-4">添加 IP 规则</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-dark-400 mb-2">IP 地址/范围</label>
                <input
                  type="text"
                  placeholder="例如：192.168.1.0/24"
                  value={newIp}
                  onChange={(e) => setNewIp(e.target.value)}
                  className="w-full px-4 py-2 bg-dark-900 border border-dark-700 rounded-lg text-dark-200 placeholder-dark-500 focus:outline-none focus:border-blue-500"
                  autoFocus
                />
              </div>

              <div>
                <label className="block text-sm text-dark-400 mb-2">规则类型</label>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setNewType('whitelist')}
                    className={`flex-1 py-2 px-4 rounded-lg border transition-colors ${
                      newType === 'whitelist'
                        ? 'bg-green-500/20 border-green-500 text-green-400'
                        : 'bg-dark-900 border-dark-700 text-dark-400 hover:border-dark-600'
                    }`}
                  >
                    白名单
                  </button>
                  <button
                    type="button"
                    onClick={() => setNewType('blacklist')}
                    className={`flex-1 py-2 px-4 rounded-lg border transition-colors ${
                      newType === 'blacklist'
                        ? 'bg-red-500/20 border-red-500 text-red-400'
                        : 'bg-dark-900 border-dark-700 text-dark-400 hover:border-dark-600'
                    }`}
                  >
                    黑名单
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-sm text-dark-400 mb-2">描述</label>
                <input
                  type="text"
                  placeholder="可选，例如：内部网络"
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  className="w-full px-4 py-2 bg-dark-900 border border-dark-700 rounded-lg text-dark-200 placeholder-dark-500 focus:outline-none focus:border-blue-500"
                />
              </div>
            </div>

            <div className="flex gap-3 mt-6">
              <button
                onClick={() => setShowAddModal(false)}
                className="flex-1 py-2 px-4 bg-dark-700 hover:bg-dark-600 text-dark-200 rounded-lg transition-colors"
              >
                取消
              </button>
              <button
                onClick={addRule}
                disabled={!newIp.trim()}
                className="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-dark-600 disabled:text-dark-400 text-white rounded-lg transition-colors"
              >
                添加
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
