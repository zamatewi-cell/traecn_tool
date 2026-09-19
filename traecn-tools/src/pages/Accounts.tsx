import React, { useState } from 'react';
import { useAppStore } from '../store';
import type { Account, DeviceFingerprint } from '../types';
import {
  Search, LayoutList, LayoutGrid, Plus, RefreshCw,
  Upload, Download, Fingerprint, Tag,
  ArrowRightLeft, Trash2, Ban,
  ChevronLeft, ChevronRight, X, Check,
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
  const [pageSize, setPageSize] = useState<number>(10);
  const [currentPage, setCurrentPage] = useState<number>(1);

  const filteredAccounts = accounts.filter((a) => {
    const matchesSearch = !searchQuery ||
      a.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (a.label && a.label.toLowerCase().includes(searchQuery.toLowerCase())) ||
      (a.username && a.username.toLowerCase().includes(searchQuery.toLowerCase()));
    const matchesFilter =
      filterTab === 'all' ||
      (filterTab === 'active' && !a.disabled) ||
      (filterTab === 'disabled' && a.disabled) ||
      (a.tags && a.tags.includes(filterTab));
    return matchesSearch && matchesFilter;
  });

  // 分页计算
  const totalPages = Math.max(1, Math.ceil(filteredAccounts.length / pageSize));
  const paginatedAccounts = filteredAccounts.slice((currentPage - 1) * pageSize, currentPage * pageSize);

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
            lastUsed: new Date().toISOString(),
          });
          alert('已成功同步本地 Trae CN 最新凭据');
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
          alert('成功从本地 Trae 读取并创建新账号');
        }
      } else {
        alert(result?.error || '未在本地找到 Trae 登录凭证');
      }
    } catch (e: any) {
      alert('读取本地凭据遇到错误: ' + e?.message);
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

  const handleImport = async () => {
    try {
      if (window.electronAPI?.importAccounts) {
        const res = await window.electronAPI.importAccounts();
        if (res.canceled) return;
        if (!res.success || !res.data) {
          alert('导入失败: ' + (res.error || '未能解析账号数据'));
          return;
        }
        const dataArr = res.data.accounts || (Array.isArray(res.data) ? res.data : [res.data]);
        let addedCount = 0;
        dataArr.forEach((acc: any) => {
          if (acc && (acc.email || acc.userId || acc.token)) {
            addAccount({
              id: crypto.randomUUID(),
              email: acc.email || `imported_${Date.now()}`,
              token: acc.token || '',
              refreshToken: acc.refreshToken || '',
              userId: acc.userId || '',
              username: acc.username || '',
              region: acc.region || '',
              aiRegion: acc.aiRegion || '',
              label: acc.label || '导入账号',
              tags: acc.tags || ['导入'],
              disabled: Boolean(acc.disabled),
              isPro: Boolean(acc.isPro),
              isCurrent: false,
              fingerprint: acc.fingerprint || null,
              fingerprintHistory: acc.fingerprintHistory || [],
              lastUsed: acc.lastUsed || new Date().toISOString(),
              createdAt: new Date().toISOString(),
              host: acc.host || '',
              expiredAt: acc.expiredAt || '',
              refreshExpiredAt: acc.refreshExpiredAt || '',
              scope: acc.scope || 'personal',
            });
            addedCount++;
          }
        });
        alert(`成功导入 ${addedCount} 个账号配置`);
      } else {
        alert('当前环境不支持本地文件导入');
      }
    } catch (e: any) {
      alert('导入失败: ' + e?.message);
    }
  };

  const handleExport = async () => {
    const sanitized = accounts.map(sanitizeAccountForExport);
    const defaultFileName = `traecn-accounts-sanitized-${new Date().toISOString().slice(0, 10)}.json`;

    if (window.electronAPI?.exportAccounts) {
      const res = await window.electronAPI.exportAccounts({ data: sanitized, defaultFileName });
      if (!res.canceled && res.success) {
        alert('脱敏信息导出成功: ' + res.filePath);
      } else if (res.error) {
        alert('导出失败: ' + res.error);
      }
      return;
    }

    const data = JSON.stringify(sanitized, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = defaultFileName;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleExportSingle = async (acc: any) => {
    const sanitized = sanitizeAccountForExport(acc);
    const defaultFileName = `${acc.email || acc.id}-sanitized.json`;

    if (window.electronAPI?.exportAccounts) {
      const res = await window.electronAPI.exportAccounts({ data: sanitized, defaultFileName });
      if (!res.canceled && res.success) {
        alert(`账号 ${acc.email || acc.id} 脱敏信息导出成功: ${res.filePath}`);
      } else if (res.error) {
        alert('导出失败: ' + res.error);
      }
      return;
    }

    const data = JSON.stringify(sanitized, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = defaultFileName;
    a.click();
    URL.revokeObjectURL(url);
  };

  const allTags = [...new Set(accounts.flatMap(a => a.tags || []))];

  return (
    <div className="p-6 space-y-4 max-w-7xl">
      {/* Toolbar */}
      <div className="flex items-center gap-3">
        {/* Search */}
        <div className="relative flex-shrink-0 w-48">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-400" />
          <input
            type="text"
            placeholder="搜索邮箱或昵称..."
            value={searchQuery}
            onChange={(e) => { setSearchQuery(e.target.value); setCurrentPage(1); }}
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
          <FilterBadge label="全部" count={accounts.length} active={filterTab === 'all'} onClick={() => { setFilterTab('all'); setCurrentPage(1); }} color="green" />
          <FilterBadge label="活跃" count={accounts.filter(a => !a.disabled).length} active={filterTab === 'active'} onClick={() => { setFilterTab('active'); setCurrentPage(1); }} color="blue" />
          <FilterBadge label="已禁用" count={accounts.filter(a => a.disabled).length} active={filterTab === 'disabled'} onClick={() => { setFilterTab('disabled'); setCurrentPage(1); }} color="red" />
          {allTags.map(tag => (
            <FilterBadge key={tag} label={tag} count={accounts.filter(a => (a.tags || []).includes(tag)).length}
              active={filterTab === tag} onClick={() => { setFilterTab(tag); setCurrentPage(1); }} color="purple" />
          ))}
        </div>

        <div className="flex-1" />

        {/* Actions */}
        <button
          onClick={handleAddAccount}
          className="flex items-center gap-1.5 px-3 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg text-sm transition-colors text-white"
          title="OAuth 添加新账号"
        >
          <Plus size={16} />
          添加账号
        </button>
        <button
          onClick={handleReadLocal}
          className="flex items-center gap-1.5 px-3 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors text-dark-200"
          title="从本地已安装的 Trae CN 凭证读取"
        >
          <RefreshCw size={16} />
          从本地同步
        </button>
        <button
          onClick={handleImport}
          className="flex items-center gap-1.5 px-3 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors text-dark-200"
          title="导入 JSON 配置文件"
        >
          <Upload size={16} />
          导入
        </button>
        <button
          onClick={handleExport}
          className="flex items-center gap-1.5 px-3 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors text-dark-200"
          title="导出账号脱敏数据"
        >
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
            <div>邮箱与身份</div>
            <div>模型状态</div>
            <div>最后使用</div>
            <div className="text-right">操作</div>
          </div>

          {/* Table Body */}
          {paginatedAccounts.length === 0 ? (
            <div className="text-center py-12 text-dark-400">
              暂无账号，点击右上角「添加账号」或「从本地同步」导入
            </div>
          ) : (
            paginatedAccounts.map((account) => (
              <div
                key={account.id}
                className="grid grid-cols-[40px_1fr_1fr_150px_180px] gap-4 items-center px-4 py-3 hover:bg-dark-800/50 transition-colors group border-b border-dark-700/30 last:border-b-0"
              >
                {/* Current Indicator */}
                <div className="flex justify-center">
                  {account.isCurrent ? (
                    <div className="w-2.5 h-2.5 rounded-full bg-green-400 shadow-[0_0_8px_rgba(74,222,128,0.5)]" title="当前生效账号" />
                  ) : (
                    <div className="w-2.5 h-2.5 rounded-full bg-dark-600" />
                  )}
                </div>

                {/* Email & Label */}
                <div className="flex items-center gap-2 min-w-0">
                  <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center text-xs font-bold text-white shrink-0">
                    {(account.email[0] || 'A').toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <span className="text-sm font-medium text-dark-200 truncate">{account.email}</span>
                      {account.isPro && (
                        <span className="px-1.5 py-0.5 bg-blue-500/20 text-blue-400 border border-blue-500/30 rounded text-[10px] font-semibold shrink-0">
                          PRO
                        </span>
                      )}
                      {account.disabled && (
                        <span className="px-1.5 py-0.5 bg-red-500/20 text-red-400 border border-red-500/30 rounded text-[10px] shrink-0">
                          已禁用
                        </span>
                      )}
                    </div>
                    {editingLabel === account.id ? (
                      <div className="flex items-center gap-1 mt-0.5">
                        <input
                          value={labelInput}
                          onChange={(e) => setLabelInput(e.target.value)}
                          onKeyDown={(e) => { if (e.key === 'Enter') handleLabelSave(account.id); }}
                          className="px-1.5 py-0.5 bg-dark-700 border border-orange-500/50 rounded text-xs w-28 focus:outline-none focus:border-orange-400 text-white"
                          autoFocus
                        />
                        <button onClick={() => handleLabelSave(account.id)} className="text-green-400 hover:text-green-300">
                          <Check size={12} />
                        </button>
                        <button onClick={() => setEditingLabel(null)} className="text-dark-400 hover:text-dark-300">
                          <X size={12} />
                        </button>
                      </div>
                    ) : (
                      account.label && (
                        <span className="inline-block px-1.5 py-0.5 bg-purple-500/20 text-purple-400 rounded text-xs mt-0.5">
                          {account.label}
                        </span>
                      )
                    )}
                  </div>
                </div>

                {/* Model Status (A7: 真实状态指示) */}
                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center gap-2">
                    <span className={`text-xs ${account.disabled ? 'text-red-400' : 'text-green-400'}`}>
                      ●
                    </span>
                    <span className="text-xs text-dark-300">全部模型</span>
                    <span className={`text-xs font-medium ${account.disabled ? 'text-red-400' : 'text-green-400'}`}>
                      {account.disabled ? '反代已禁用' : '通道就绪'}
                    </span>
                  </div>
                  <div className="h-1.5 w-full bg-dark-700 rounded-full overflow-hidden">
                    <div className={`h-full w-full rounded-full transition-all ${
                      account.disabled ? 'bg-red-500/60' : 'bg-green-500'
                    }`} />
                  </div>
                </div>

                {/* Last Used */}
                <div className="text-xs text-dark-400">
                  {account.lastUsed ? new Date(account.lastUsed).toLocaleDateString('zh-CN') : '-'}
                </div>

                {/* Actions (A1-A3: 清理假预热/假详情，刷新绑定真实动作) */}
                <div className="flex items-center gap-1 justify-end opacity-0 group-hover:opacity-100 transition-opacity">
                  <ActionButton
                    icon={<RefreshCw size={14} />}
                    title="刷新使用时间"
                    onClick={() => {
                      updateAccount(account.id, { lastUsed: new Date().toISOString() });
                    }}
                  />
                  <ActionButton
                    icon={<Fingerprint size={14} />}
                    title="设备指纹"
                    onClick={() => setShowFingerprintModal(account.id)}
                  />
                  <ActionButton
                    icon={<Tag size={14} />}
                    title="编辑标签"
                    onClick={() => { setEditingLabel(account.id); setLabelInput(account.label || ''); }}
                  />
                  <ActionButton
                    icon={<ArrowRightLeft size={14} />}
                    title="切换到此账号"
                    onClick={() => switchAccount(account.id)}
                    className={account.isCurrent ? 'text-green-400' : ''}
                  />
                  <ActionButton
                    icon={<Download size={14} />}
                    title="导出脱敏信息"
                    onClick={() => handleExportSingle(account)}
                  />
                  <ActionButton
                    icon={<Ban size={14} />}
                    title={account.disabled ? '启用反代' : '禁用反代'}
                    onClick={() => setAccountDisabled(account.id, !account.disabled)}
                    className={account.disabled ? 'text-red-400' : ''}
                  />
                  <ActionButton
                    icon={<Trash2 size={14} />}
                    title="删除"
                    onClick={() => { if (confirm(`确定要删除账号 ${account.email} 吗？`)) removeAccount(account.id); }}
                    className="text-red-400 hover:!bg-red-500/20"
                  />
                </div>
              </div>
            ))
          )}
        </div>
      ) : (
        /* Grid View (A4: 卡片视图真实交互) */
        <div className="grid grid-cols-3 gap-4">
          {paginatedAccounts.map((account) => (
            <div
              key={account.id}
              className={`bg-dark-800/50 border rounded-xl p-4 card-hover transition-colors ${
                account.isCurrent ? 'border-green-500/50 shadow-[0_0_15px_rgba(74,222,128,0.1)]' : 'border-dark-700/50'
              }`}
            >
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2 min-w-0">
                  <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center text-sm font-bold text-white shrink-0">
                    {(account.email[0] || 'A').toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <div className="text-sm font-medium text-dark-200 truncate">{account.email}</div>
                    <div className="flex items-center gap-1.5 mt-0.5">
                      {account.isPro && <span className="text-xs text-blue-400 font-semibold">PRO</span>}
                      {account.disabled && <span className="text-xs text-red-400">已禁用</span>}
                      {account.isCurrent && <span className="text-xs text-green-400 font-medium">● 生效中</span>}
                    </div>
                  </div>
                </div>

                {/* 卡片快速切换或删除 */}
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => switchAccount(account.id)}
                    title={account.isCurrent ? '当前已生效' : '切换到该账号'}
                    disabled={account.isCurrent}
                    className="p-1 hover:bg-dark-600 rounded transition-colors text-dark-300 hover:text-white disabled:opacity-40"
                  >
                    <ArrowRightLeft size={14} />
                  </button>
                  <button
                    onClick={() => { if (confirm(`确定要删除此账号？`)) removeAccount(account.id); }}
                    title="删除账号"
                    className="p-1 hover:bg-red-500/20 rounded transition-colors text-dark-400 hover:text-red-400"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between text-xs">
                  <span className="text-dark-400">模型状态</span>
                  <span className={account.disabled ? 'text-red-400' : 'text-green-400'}>
                    {account.disabled ? '已禁用' : '全部可用'}
                  </span>
                </div>
                <div className="h-1.5 bg-dark-700 rounded-full overflow-hidden">
                  <div className={`h-full w-full rounded-full ${account.disabled ? 'bg-red-500/60' : 'bg-green-500'}`} />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination (A6: 真实分页联动) */}
      {filteredAccounts.length > 0 && (
        <div className="flex items-center justify-between text-xs text-dark-400 pt-2">
          <span>
            显示第 {(currentPage - 1) * pageSize + 1} 到 {Math.min(currentPage * pageSize, filteredAccounts.length)} 条，
            共 {filteredAccounts.length} 条记录
          </span>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <span>每页显示</span>
              <select
                value={pageSize}
                onChange={(e) => {
                  setPageSize(Number(e.target.value));
                  setCurrentPage(1);
                }}
                className="bg-dark-800 border border-dark-600 rounded px-2 py-1 text-dark-300 focus:outline-none"
              >
                <option value={10}>10 条</option>
                <option value={25}>25 条</option>
                <option value={50}>50 条</option>
              </select>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                disabled={currentPage <= 1}
                className="p-1.5 bg-dark-800 hover:bg-dark-700 disabled:opacity-40 rounded transition-colors"
                title="上一页"
              >
                <ChevronLeft size={14} />
              </button>
              <span className="px-2">{currentPage} / {totalPages}</span>
              <button
                onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                disabled={currentPage >= totalPages}
                className="p-1.5 bg-dark-800 hover:bg-dark-700 disabled:opacity-40 rounded transition-colors"
                title="下一页"
              >
                <ChevronRight size={14} />
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Fingerprint Modal (A5: 恢复原始绑定) */}
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
    <button
      onClick={onClick}
      title={title}
      className={`p-1.5 hover:bg-dark-700 rounded-lg text-dark-400 hover:text-dark-200 transition-colors ${className}`}
    >
      {icon}
    </button>
  );
}

function FingerprintModal({
  account,
  onClose,
  onUpdate,
}: {
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

  // A5: 恢复原始逻辑
  const handleRestoreOriginal = () => {
    if (account.fingerprintHistory && account.fingerprintHistory.length > 0) {
      onUpdate(account.fingerprintHistory[0]);
      alert('已成功恢复到初始记录的设备指纹');
    } else if (account.fingerprint) {
      alert('当前指纹已是唯一的初始指纹');
    } else {
      handleGenerate();
    }
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
              className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-500 rounded-lg text-xs transition-colors disabled:opacity-50 text-white">
              ⚡ 生成并绑定
            </button>
            <button
              onClick={handleRestoreOriginal}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-xs transition-colors text-dark-200"
            >
              ◇ 恢复原始
            </button>
            <button onClick={handleOpenDir}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-xs transition-colors text-dark-200">
              📁 打开储存目录
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
            <h3 className="text-sm font-semibold text-dark-200 mb-3">历史指纹记录</h3>
            {(account.fingerprintHistory || []).length === 0 ? (
              <p className="text-xs text-dark-500">暂无历史记录</p>
            ) : (
              <div className="space-y-3">
                {account.fingerprintHistory.map((hist, idx) => (
                  <div key={idx} className="bg-dark-900/50 border border-dark-700/50 rounded-lg p-3">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-xs text-dark-400 font-mono">#{idx + 1}</span>
                      <button
                        onClick={() => { onUpdate(hist); alert('已应用此历史指纹'); }}
                        className="text-xs text-blue-400 hover:underline"
                      >
                        应用此指纹
                      </button>
                    </div>
                    <FingerprintDetails fp={hist} />
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
    <div className="space-y-1.5 text-xs font-mono">
      <div className="flex justify-between">
        <span className="text-dark-500">Device ID:</span>
        <span className="text-dark-300 truncate max-w-[200px]" title={fp.deviceId}>{fp.deviceId ? `${fp.deviceId.slice(0, 16)}...` : '-'}</span>
      </div>
      <div className="flex justify-between">
        <span className="text-dark-500">Machine ID:</span>
        <span className="text-dark-300 truncate max-w-[200px]" title={fp.machineId}>{fp.machineId ? `${fp.machineId.slice(0, 16)}...` : '-'}</span>
      </div>
      <div className="flex justify-between">
        <span className="text-dark-500">MAC:</span>
        <span className="text-dark-300">{fp.macAddress || '-'}</span>
      </div>
    </div>
  );
}
