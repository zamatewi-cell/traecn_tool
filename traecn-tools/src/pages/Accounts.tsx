import React, { useState } from 'react';
import { useAppStore } from '../store';
import type { Account, DeviceFingerprint } from '../types';
import {
  Search, LayoutList, LayoutGrid, Plus, RefreshCw,
  Upload, Download, MoreHorizontal, Fingerprint, Tag,
  ArrowRightLeft, Flame, Eye, Trash2, Ban, Copy,
  ChevronDown, X, Check, GripVertical,
} from 'lucide-react';

type ViewMode = 'list' | 'grid';
type FilterTab = 'all' | 'active' | 'disabled' | string;

export default function Accounts() {
  const { accounts, addAccount, updateAccount, removeAccount, switchAccount, setAccountDisabled } = useAppStore();
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [searchQuery, setSearchQuery] = useState('');
  const [filterTab, setFilterTab] = useState<FilterTab>('all');
  const [showFingerprintModal, setShowFingerprintModal] = useState<string | null>(null);
  const [editingLabel, setEditingLabel] = useState<string | null>(null);
  const [labelInput, setLabelInput] = useState('');
  const [showContextMenu, setShowContextMenu] = useState<string | null>(null);

  const filteredAccounts = accounts.filter((a) => {
    const matchesSearch = !searchQuery ||
      a.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      a.label.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesFilter =
      filterTab === 'all' ||
      (filterTab === 'active' && !a.disabled) ||
      (filterTab === 'disabled' && a.disabled) ||
      a.tags.includes(filterTab);
    return matchesSearch && matchesFilter;
  });

  const handleAddAccount = async () => {
    try {
      const result = await window.electronAPI?.addAccountOAuth();
      if (result?.success && result.data) {
        const fp = await window.electronAPI?.generateFingerprint();
        const newAccount: Account = {
          id: crypto.randomUUID(),
          email: result.data.email || result.data.user_id || '新账号',
          label: '',
          token: result.data.token || '',
          refreshToken: result.data.refresh_token || '',
          userId: result.data.user_id || '',
          host: result.data.host || '',
          expiredAt: result.data.expired_at || '',
          refreshExpiredAt: result.data.refresh_expired_at || '',
          username: result.data.username || '',
          scope: result.data.scope || 'personal',
          region: result.data.region || '',
          aiRegion: result.data.ai_region || '',
          isCurrent: accounts.length === 0,
          isPro: false,
          disabled: false,
          fingerprint: fp || null,
          fingerprintHistory: fp ? [fp] : [],
          lastUsed: new Date().toISOString(),
          createdAt: new Date().toISOString(),
          tags: [],
        };
        addAccount(newAccount);
      }
    } catch (e) {
      console.error('OAuth failed:', e);
    }
  };

  const handleReadLocal = async () => {
    try {
      const result = await window.electronAPI?.readTraeStorage();
      if (result?.success && result.data) {
        const existing = accounts.find(a => a.userId === result.data!.userId);
        if (existing) {
          updateAccount(existing.id, {
            token: result.data.token || existing.token,
            refreshToken: result.data.refreshToken || existing.refreshToken,
            expiredAt: result.data.expiredAt || existing.expiredAt,
          });
        } else {
          const fp = await window.electronAPI?.generateFingerprint();
          const newAccount: Account = {
            id: crypto.randomUUID(),
            email: result.data.username || '本地账号',
            label: '从本地导入',
            token: result.data.token || '',
            refreshToken: result.data.refreshToken || '',
            userId: result.data.userId || '',
            host: result.data.host || '',
            expiredAt: result.data.expiredAt || '',
            refreshExpiredAt: result.data.refreshExpiredAt || '',
            username: result.data.username || '',
            scope: result.data.scope || 'personal',
            region: result.data.region || '',
            aiRegion: result.data.aiRegion || '',
            isCurrent: accounts.length === 0,
            isPro: false,
            disabled: false,
            fingerprint: fp || null,
            fingerprintHistory: fp ? [fp] : [],
            lastUsed: new Date().toISOString(),
            createdAt: new Date().toISOString(),
            tags: ['本地'],
          };
          addAccount(newAccount);
        }
      }
    } catch (e) {
      console.error('Read local failed:', e);
    }
  };

  const handleLabelSave = (accountId: string) => {
    updateAccount(accountId, { label: labelInput });
    setEditingLabel(null);
    setLabelInput('');
  };

  const maskSecret = (secret?: string) => {
    if (!secret) return secret;
    if (secret.length <= 10) return '*** (已脱敏)';
    return `${secret.slice(0, 6)}***${secret.slice(-4)} (已脱敏保护)`;
  };

  const sanitizeAccountForExport = (acc: any) => ({
    ...acc,
    token: acc.token ? maskSecret(acc.token) : acc.token,
    refreshToken: acc.refreshToken ? maskSecret(acc.refreshToken) : acc.refreshToken,
  });

  const handleExport = () => {
    const sanitized = accounts.map(sanitizeAccountForExport);
    const data = JSON.stringify(sanitized, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `traecn-accounts-sanitized-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const allTags = [...new Set(accounts.flatMap(a => a.tags))];

  return (
    <div className="p-6 space-y-4 max-w-7xl">
      {/* Toolbar */}
      <div className="flex items-center gap-3">
        {/* Search */}
        <div className="relative flex-shrink-0 w-48">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-400" />
          <input
            type="text"
            placeholder="搜索邮箱..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-3 py-2 bg-dark-800 border border-dark-600 rounded-lg text-sm text-dark-200 placeholder:text-dark-500 focus:outline-none focus:border-blue-500"
          />
        </div>

        {/* View toggle */}
        <div className="flex items-center gap-0.5 bg-dark-800 border border-dark-600 rounded-lg p-0.5">
          <button onClick={() => setViewMode('list')}
            className={`p-1.5 rounded ${viewMode === 'list' ? 'bg-dark-600 text-white' : 'text-dark-400'}`}>
            <LayoutList size={16} />
          </button>
          <button onClick={() => setViewMode('grid')}
            className={`p-1.5 rounded ${viewMode === 'grid' ? 'bg-dark-600 text-white' : 'text-dark-400'}`}>
            <LayoutGrid size={16} />
          </button>
        </div>

        {/* Filter tabs */}
        <div className="flex items-center gap-1.5">
          <FilterBadge label="全部" count={accounts.length} active={filterTab === 'all'} onClick={() => setFilterTab('all')} color="green" />
          <FilterBadge label="活跃" count={accounts.filter(a => !a.disabled).length} active={filterTab === 'active'} onClick={() => setFilterTab('active')} color="blue" />
          <FilterBadge label="已禁用" count={accounts.filter(a => a.disabled).length} active={filterTab === 'disabled'} onClick={() => setFilterTab('disabled')} color="red" />
          {allTags.map(tag => (
            <FilterBadge key={tag} label={tag} count={accounts.filter(a => a.tags.includes(tag)).length}
              active={filterTab === tag} onClick={() => setFilterTab(tag)} color="purple" />
          ))}
        </div>

        <div className="flex-1" />

        {/* Actions */}
        <button onClick={handleAddAccount}
          className="flex items-center gap-1.5 px-3 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm transition-colors">
          <Plus size={16} />
        </button>
        <button onClick={handleReadLocal}
          className="flex items-center gap-1.5 px-3 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors"
          title="从本地 Trae CN 读取">
          <RefreshCw size={16} />
        </button>
        <button className="flex items-center gap-1.5 px-3 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors"
          title="导入">
          <Upload size={16} />
          导入
        </button>
        <button onClick={handleExport}
          className="flex items-center gap-1.5 px-3 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors"
          title="导出">
          <Download size={16} />
          导出
        </button>
      </div>

      {/* Account List */}
      {viewMode === 'list' ? (
        <div className="bg-dark-800/30 border border-dark-700/50 rounded-xl overflow-hidden">
          {/* Table Header */}
          <div className="grid grid-cols-[40px_1fr_1fr_150px_180px] gap-4 px-4 py-3 bg-dark-800/50 border-b border-dark-700/50 text-xs text-dark-400 uppercase tracking-wide">
            <div />
            <div>邮箱</div>
            <div>模型状态</div>
            <div>最后使用</div>
            <div className="text-right">操作</div>
          </div>

          {/* Table Body */}
          {filteredAccounts.length === 0 ? (
            <div className="text-center py-12 text-dark-400">
              暂无账号，点击 + 添加
            </div>
          ) : (
            filteredAccounts.map((account) => (
              <div key={account.id}
                className="grid grid-cols-[40px_1fr_1fr_150px_180px] gap-4 px-4 py-3 border-b border-dark-700/30 hover:bg-dark-800/30 transition-colors items-center group">
                {/* Drag + Select */}
                <div className="flex items-center gap-1">
                  <GripVertical size={14} className="text-dark-600 cursor-grab opacity-0 group-hover:opacity-100" />
                </div>

                {/* Email + Tags */}
                <div className="flex flex-col gap-1.5 min-w-0">
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="font-medium text-dark-200 truncate">{account.email}</span>
                  </div>
                  <div className="flex items-center gap-1.5 flex-wrap">
                    {account.isCurrent && (
                      <span className="px-1.5 py-0.5 bg-blue-500/20 text-blue-400 rounded text-xs">当前</span>
                    )}
                    {account.isPro && (
                      <span className="px-1.5 py-0.5 bg-blue-500/20 text-blue-300 rounded text-xs">◆ PRO</span>
                    )}
                    {account.disabled && (
                      <span className="px-1.5 py-0.5 bg-red-500/20 text-red-400 rounded text-xs">? 反代已禁用</span>
                    )}
                    {!account.disabled && (
                      <span className="px-1.5 py-0.5 bg-dark-600 text-dark-300 rounded text-xs">○ FREE</span>
                    )}
                    {account.tags.map(tag => (
                      <span key={tag} className="px-1.5 py-0.5 bg-orange-500/20 text-orange-400 rounded text-xs">{tag}</span>
                    ))}
                    {/* Editable label */}
                    {editingLabel === account.id ? (
                      <div className="flex items-center gap-1">
                        <input
                          type="text"
                          value={labelInput}
                          onChange={(e) => setLabelInput(e.target.value)}
                          onKeyDown={(e) => e.key === 'Enter' && handleLabelSave(account.id)}
                          placeholder="输入自定义标签"
                          className="px-1.5 py-0.5 bg-dark-700 border border-orange-500/50 rounded text-xs w-28 focus:outline-none focus:border-orange-400"
                          autoFocus
                        />
                        <button onClick={() => handleLabelSave(account.id)}
                          className="text-green-400 hover:text-green-300">
                          <Check size={12} />
                        </button>
                        <button onClick={() => setEditingLabel(null)}
                          className="text-dark-400 hover:text-dark-300">
                          <X size={12} />
                        </button>
                      </div>
                    ) : (
                      account.label && (
                        <span className="px-1.5 py-0.5 bg-purple-500/20 text-purple-400 rounded text-xs">{account.label}</span>
                      )
                    )}
                  </div>
                </div>

                {/* Model Status */}
                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center gap-2">
                    <span className="text-green-400 text-xs">?</span>
                    <span className="text-xs text-dark-300">全部模型</span>
                    <span className="text-xs text-green-400 font-medium">可用</span>
                  </div>
                  <div className="h-1.5 w-full bg-dark-700 rounded-full overflow-hidden">
                    <div className="h-full w-full bg-green-500 rounded-full" />
                  </div>
                </div>

                {/* Last Used */}
                <div className="text-xs text-dark-400">
                  {account.lastUsed ? new Date(account.lastUsed).toLocaleDateString('zh-CN') : '-'}
                </div>

                {/* Actions */}
                <div className="flex items-center gap-1 justify-end opacity-0 group-hover:opacity-100 transition-opacity">
                  <ActionButton icon={<RefreshCw size={14} />} title="刷新" onClick={() => {}} />
                  <ActionButton icon={<Fingerprint size={14} />} title="设备指纹" onClick={() => setShowFingerprintModal(account.id)} />
                  <ActionButton icon={<Tag size={14} />} title="编辑标签"
                    onClick={() => { setEditingLabel(account.id); setLabelInput(account.label); }} />
                  <ActionButton icon={<ArrowRightLeft size={14} />} title="切换到此账号" onClick={() => switchAccount(account.id)} />
                  <ActionButton icon={<Flame size={14} />} title="预热" onClick={() => {}} />
                  <ActionButton icon={<Download size={14} />} title="导出" onClick={() => {
                    const sanitized = sanitizeAccountForExport(account);
                    const data = JSON.stringify(sanitized, null, 2);
                    const blob = new Blob([data], { type: 'application/json' });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `${account.email || account.id}-sanitized.json`;
                    a.click();
                    URL.revokeObjectURL(url);
                  }} />
                  <ActionButton icon={<Eye size={14} />} title="详情" onClick={() => {}} />
                  <ActionButton icon={<Ban size={14} />} title={account.disabled ? '启用反代' : '禁用反代'}
                    onClick={() => setAccountDisabled(account.id, !account.disabled)}
                    className={account.disabled ? 'text-red-400' : ''} />
                  <ActionButton icon={<Trash2 size={14} />} title="删除"
                    onClick={() => { if (confirm('确定要删除此账号？')) removeAccount(account.id); }}
                    className="text-red-400 hover:!bg-red-500/20" />
                </div>
              </div>
            ))
          )}
        </div>
      ) : (
        /* Grid View */
        <div className="grid grid-cols-3 gap-4">
          {filteredAccounts.map((account) => (
            <div key={account.id} className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 card-hover">
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center text-sm font-bold">
                    {(account.email[0] || 'A').toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-dark-200 truncate">{account.email}</div>
                    <div className="flex items-center gap-1.5 mt-0.5">
                      {account.isPro && <span className="text-xs text-blue-400">PRO</span>}
                      {account.disabled && <span className="text-xs text-red-400">已禁用</span>}
                    </div>
                  </div>
                </div>
                <button className="p-1 hover:bg-dark-600 rounded transition-colors">
                  <MoreHorizontal size={14} className="text-dark-400" />
                </button>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-dark-400">模型状态</span>
                  <span className="text-green-400">全部可用</span>
                </div>
                <div className="h-1.5 bg-dark-700 rounded-full overflow-hidden">
                  <div className="h-full w-full bg-green-500 rounded-full" />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {filteredAccounts.length > 0 && (
        <div className="flex items-center justify-between text-xs text-dark-400">
          <span>显示第 1 到 {filteredAccounts.length} 条，共 {filteredAccounts.length} 条</span>
          <div className="flex items-center gap-2">
            <span>每页</span>
            <select className="bg-dark-800 border border-dark-600 rounded px-2 py-1 text-dark-300">
              <option>10 条</option>
              <option>25 条</option>
              <option>50 条</option>
            </select>
          </div>
        </div>
      )}

      {/* Fingerprint Modal */}
      {showFingerprintModal && (
        <FingerprintModal
          account={accounts.find(a => a.id === showFingerprintModal)!}
          onClose={() => setShowFingerprintModal(null)}
          onUpdate={(fp) => {
            const acc = accounts.find(a => a.id === showFingerprintModal);
            if (acc) {
              updateAccount(acc.id, {
                fingerprint: fp,
                fingerprintHistory: [...(acc.fingerprintHistory || []), fp],
              });
            }
          }}
        />
      )}
    </div>
  );
}

function FilterBadge({ label, count, active, onClick, color }: {
  label: string; count: number; active: boolean; onClick: () => void; color: string;
}) {
  const colors: Record<string, string> = {
    green: active ? 'bg-green-500/20 text-green-400 border-green-500/50' : 'bg-dark-800 text-dark-400 border-dark-600',
    blue: active ? 'bg-blue-500/20 text-blue-400 border-blue-500/50' : 'bg-dark-800 text-dark-400 border-dark-600',
    red: active ? 'bg-red-500/20 text-red-400 border-red-500/50' : 'bg-dark-800 text-dark-400 border-dark-600',
    purple: active ? 'bg-purple-500/20 text-purple-400 border-purple-500/50' : 'bg-dark-800 text-dark-400 border-dark-600',
  };

  return (
    <button onClick={onClick}
      className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs border transition-colors ${colors[color]}`}>
      {label}
      <span className="font-semibold">{count}</span>
    </button>
  );
}

function ActionButton({ icon, title, onClick, className = '' }: {
  icon: React.ReactNode; title: string; onClick: () => void; className?: string;
}) {
  return (
    <button onClick={onClick} title={title}
      className={`p-1.5 hover:bg-dark-600 rounded transition-colors text-dark-400 hover:text-dark-200 ${className}`}>
      {icon}
    </button>
  );
}

function FingerprintModal({ account, onClose, onUpdate }: {
  account: Account;
  onClose: () => void;
  onUpdate: (fp: DeviceFingerprint) => void;
}) {
  const [generating, setGenerating] = useState(false);

  const handleGenerate = async () => {
    setGenerating(true);
    try {
      const fp = await window.electronAPI?.generateFingerprint();
      if (fp) onUpdate(fp);
    } finally {
      setGenerating(false);
    }
  };

  const handleRestore = (fp: DeviceFingerprint) => {
    onUpdate(fp);
  };

  const handleOpenDir = async () => {
    const dir = await window.electronAPI?.getDataDir();
    if (dir) window.electronAPI?.openPath(dir);
  };

  const fp = account.fingerprint;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div className="bg-dark-800 border border-dark-600 rounded-2xl w-[700px] max-h-[85vh] overflow-y-auto shadow-2xl" onClick={(e) => e.stopPropagation()}>
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-dark-700/50">
          <div className="flex items-center gap-3">
            <Fingerprint size={20} className="text-blue-400" />
            <h2 className="text-lg font-semibold">设备指纹</h2>
            <span className="px-2 py-0.5 bg-dark-700 text-dark-300 rounded-lg text-xs">{account.email}</span>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-dark-600 rounded-lg transition-colors">
            <X size={18} className="text-dark-400" />
          </button>
        </div>

        <div className="p-6 space-y-6">
          {/* Actions */}
          <div className="flex items-center gap-3">
            <h3 className="text-sm font-semibold text-dark-200">设备指纹操作</h3>
            <div className="flex-1" />
            <button onClick={handleGenerate} disabled={generating}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-500 rounded-lg text-xs transition-colors disabled:opacity-50">
              ?? 生成并绑定
            </button>
            <button className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-xs transition-colors">
              ◇ 恢复原始
            </button>
            <button onClick={handleOpenDir}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-xs transition-colors">
              ? 打开储存目录
            </button>
          </div>

          {/* Current vs Bound fingerprints */}
          <div className="grid grid-cols-2 gap-4">
            {/* Current Storage */}
            <div className="bg-dark-900/50 border border-dark-700/50 rounded-xl p-4">
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-medium">当前存储</h4>
                <span className="px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">已生效</span>
              </div>
              <p className="text-xs text-dark-500 mb-3">读取自 storage.json（切换账号时应用绑定后更新）</p>
              {fp ? (
                <FingerprintDetails fp={fp} />
              ) : (
                <p className="text-xs text-dark-500">暂无指纹数据</p>
              )}
            </div>

            {/* Account Bound */}
            <div className="bg-dark-900/50 border border-dark-700/50 rounded-xl p-4">
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-medium">账号绑定</h4>
                <span className="px-2 py-0.5 bg-orange-500/20 text-orange-400 rounded text-xs">待应用</span>
              </div>
              <p className="text-xs text-dark-500 mb-3">生成/恢复后保存为绑定，切换账号时写入 storage.json</p>
              {fp ? (
                <FingerprintDetails fp={fp} />
              ) : (
                <p className="text-xs text-dark-500">暂无绑定数据</p>
              )}
            </div>
          </div>

          {/* History */}
          <div>
            <h3 className="text-sm font-semibold text-dark-200 mb-3">历史指纹（可选恢复/删除）</h3>
            {(account.fingerprintHistory || []).length === 0 ? (
              <p className="text-xs text-dark-500">暂无历史记录</p>
            ) : (
              <div className="space-y-3">
                {account.fingerprintHistory.map((hist, idx) => (
                  <div key={idx} className="bg-dark-900/50 border border-dark-700/50 rounded-lg p-3">
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-medium text-dark-300">{hist.label}</span>
                        {idx === account.fingerprintHistory.length - 1 && (
                          <span className="px-1.5 py-0.5 bg-blue-500/20 text-blue-400 rounded text-xs">当前</span>
                        )}
                      </div>
                      <button onClick={() => handleRestore(hist)}
                        className="text-xs text-blue-400 hover:text-blue-300 transition-colors">
                        恢复
                      </button>
                    </div>
                    <div className="text-xs text-dark-500 mb-1">{new Date(hist.createdAt).toLocaleString('zh-CN')}</div>
                    <div className="text-xs text-dark-500 font-mono space-y-0.5">
                      <div>machineId: {hist.machineId}</div>
                      <div>macMachineId: {hist.macMachineId}</div>
                      <div>devDeviceId: {hist.devDeviceId}</div>
                      <div>sqmId: {hist.sqmId}</div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function FingerprintDetails({ fp }: { fp: DeviceFingerprint }) {
  return (
    <div className="space-y-2">
      <FpField label="machineId" value={fp.machineId} />
      <FpField label="macMachineId" value={fp.macMachineId} />
      <FpField label="devDeviceId" value={fp.devDeviceId} />
      <FpField label="sqmId" value={fp.sqmId} />
    </div>
  );
}

function FpField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <div className="flex items-start gap-2">
      <span className="text-xs font-semibold text-dark-400 w-24 shrink-0">{label}:</span>
      <span className="text-xs text-dark-300 font-mono break-all flex-1">{value}</span>
      <button onClick={handleCopy} className="text-dark-500 hover:text-dark-300 shrink-0" title="复制">
        {copied ? <Check size={12} className="text-green-400" /> : <Copy size={12} />}
      </button>
    </div>
  );
}
