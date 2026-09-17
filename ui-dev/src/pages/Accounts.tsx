import { useState } from 'react';

interface Account {
  id: number;
  name: string;
  email: string;
  status: 'active' | 'inactive' | 'expired';
  expiresAt: string;
  usage: number;
}

export default function Accounts() {
  const [accounts] = useState<Account[]>([
    { id: 1, name: '账号 A', email: 'account_a@example.com', status: 'active', expiresAt: '2024-12-31', usage: 75 },
    { id: 2, name: '账号 B', email: 'account_b@example.com', status: 'active', expiresAt: '2024-11-30', usage: 45 },
    { id: 3, name: '账号 C', email: 'account_c@example.com', status: 'inactive', expiresAt: '2024-10-15', usage: 100 },
    { id: 4, name: '账号 D', email: 'account_d@example.com', status: 'expired', expiresAt: '2024-09-01', usage: 100 },
  ]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-500/20 text-green-400';
      case 'inactive':
        return 'bg-slate-500/20 text-slate-400';
      case 'expired':
        return 'bg-red-500/20 text-red-400';
      default:
        return 'bg-slate-500/20 text-slate-400';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'active':
        return '活跃';
      case 'inactive':
        return '未激活';
      case 'expired':
        return '已过期';
      default:
        return status;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">账号管理</h1>
        <button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors">
          添加账号
        </button>
      </div>

      {/* Accounts Table */}
      <div className="bg-slate-800 rounded-xl border border-slate-700 overflow-hidden">
        <table className="w-full">
          <thead className="bg-slate-700/50">
            <tr>
              <th className="px-6 py-4 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">账号</th>
              <th className="px-6 py-4 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">状态</th>
              <th className="px-6 py-4 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">到期时间</th>
              <th className="px-6 py-4 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">使用量</th>
              <th className="px-6 py-4 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-700">
            {accounts.map((account) => (
              <tr key={account.id} className="hover:bg-slate-700/30 transition-colors">
                <td className="px-6 py-4">
                  <div>
                    <div className="text-sm font-medium text-white">{account.name}</div>
                    <div className="text-sm text-slate-400">{account.email}</div>
                  </div>
                </td>
                <td className="px-6 py-4">
                  <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(account.status)}`}>
                    {getStatusText(account.status)}
                  </span>
                </td>
                <td className="px-6 py-4 text-sm text-slate-300">
                  {account.expiresAt}
                </td>
                <td className="px-6 py-4">
                  <div className="flex items-center space-x-2">
                    <div className="w-24 bg-slate-700 rounded-full h-2">
                      <div
                        className={`h-2 rounded-full ${
                          account.usage >= 100
                            ? 'bg-red-500'
                            : account.usage >= 75
                            ? 'bg-orange-500'
                            : 'bg-green-500'
                        }`}
                        style={{ width: `${account.usage}%` }}
                      ></div>
                    </div>
                    <span className="text-xs text-slate-400">{account.usage}%</span>
                  </div>
                </td>
                <td className="px-6 py-4">
                  <div className="flex items-center space-x-2">
                    <button className="text-blue-400 hover:text-blue-300 text-sm">编辑</button>
                    <button className="text-slate-400 hover:text-slate-300 text-sm">详情</button>
                    <button className="text-red-400 hover:text-red-300 text-sm">删除</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-slate-400">
          显示 1 到 {accounts.length} 条，共 {accounts.length} 条
        </div>
        <div className="flex items-center space-x-2">
          <button className="px-3 py-1 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm rounded transition-colors disabled:opacity-50" disabled>
            上一页
          </button>
          <button className="px-3 py-1 bg-blue-600 text-white text-sm rounded">1</button>
          <button className="px-3 py-1 bg-slate-700 hover:bg-slate-600 text-slate-300 text-sm rounded transition-colors disabled:opacity-50" disabled>
            下一页
          </button>
        </div>
      </div>
    </div>
  );
}
