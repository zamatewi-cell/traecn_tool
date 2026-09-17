import React, { useState, useEffect } from 'react';
import { ScrollText, Search, Filter, Download, Trash2, RefreshCw, ChevronLeft, ChevronRight } from 'lucide-react';
import { useAppStore } from '../store';

interface ProxyLog {
  timestamp: string;
  method: string;
  path: string;
  status: number;
  duration: number;
  model?: string;
  accountId?: string;
}

export default function Logs() {
  const { proxyLogs } = useAppStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [filterMethod, setFilterMethod] = useState<string>('all');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const [logs, setLogs] = useState<ProxyLog[]>([]);
  const pageSize = 20;

  // Simulate logs from proxyLogs state
  useEffect(() => {
    // In production, this would fetch from actual proxy logs
    const mockLogs: ProxyLog[] = proxyLogs.length > 0 ? proxyLogs : [];
    setLogs(mockLogs);
  }, [proxyLogs]);

  const filteredLogs = logs.filter(log => {
    const matchesSearch = searchTerm === '' || 
      log.path.toLowerCase().includes(searchTerm.toLowerCase()) ||
      log.model?.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesMethod = filterMethod === 'all' || log.method === filterMethod;
    const matchesStatus = filterStatus === 'all' || 
      (filterStatus === 'success' && log.status >= 200 && log.status < 300) ||
      (filterStatus === 'error' && log.status >= 400);
    return matchesSearch && matchesMethod && matchesStatus;
  });

  const totalPages = Math.ceil(filteredLogs.length / pageSize);
  const paginatedLogs = filteredLogs.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  const stats = {
    total: logs.length,
    success: logs.filter(l => l.status >= 200 && l.status < 300).length,
    error: logs.filter(l => l.status >= 400).length,
    avgDuration: logs.length > 0 ? Math.round(logs.reduce((sum, l) => sum + l.duration, 0) / logs.length) : 0,
  };

  const getMethodColor = (method: string) => {
    switch (method) {
      case 'GET': return 'text-blue-400 bg-blue-500/10';
      case 'POST': return 'text-green-400 bg-green-500/10';
      case 'PUT': return 'text-yellow-400 bg-yellow-500/10';
      case 'DELETE': return 'text-red-400 bg-red-500/10';
      default: return 'text-dark-400 bg-dark-700/50';
    }
  };

  const getStatusColor = (status: number) => {
    if (status >= 200 && status < 300) return 'text-green-400';
    if (status >= 300 && status < 400) return 'text-yellow-400';
    if (status >= 400) return 'text-red-400';
    return 'text-dark-400';
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <ScrollText size={20} className="text-blue-400" />
          <h1 className="text-lg font-semibold">流量日志</h1>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-2 px-3 py-2 bg-dark-700 hover:bg-dark-600 rounded-lg text-sm text-dark-300 transition-colors">
            <RefreshCw size={14} />
            刷新
          </button>
          <button className="flex items-center gap-2 px-3 py-2 bg-dark-700 hover:bg-dark-600 rounded-lg text-sm text-dark-300 transition-colors">
            <Download size={14} />
            导出
          </button>
          <button className="flex items-center gap-2 px-3 py-2 bg-dark-700 hover:bg-dark-600 rounded-lg text-sm text-dark-300 transition-colors">
            <Trash2 size={14} />
            清空
          </button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="text-xs text-dark-400 mb-1">总请求数</div>
          <div className="text-2xl font-bold text-white">{stats.total}</div>
        </div>
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="text-xs text-dark-400 mb-1">成功</div>
          <div className="text-2xl font-bold text-green-400">{stats.success}</div>
        </div>
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="text-xs text-dark-400 mb-1">失败</div>
          <div className="text-2xl font-bold text-red-400">{stats.error}</div>
        </div>
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4">
          <div className="text-xs text-dark-400 mb-1">平均延迟</div>
          <div className="text-2xl font-bold text-cyan-400">{stats.avgDuration}ms</div>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-4 mb-4">
        <div className="flex items-center gap-3">
          <div className="flex-1 relative">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-500" />
            <input
              type="text"
              placeholder="搜索路径、模型..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-dark-900 border border-dark-700 rounded-lg text-sm text-white placeholder-dark-500 focus:outline-none focus:border-blue-500"
            />
          </div>
          <div className="flex items-center gap-2">
            <Filter size={16} className="text-dark-500" />
            <select
              value={filterMethod}
              onChange={(e) => setFilterMethod(e.target.value)}
              className="px-3 py-2 bg-dark-900 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-blue-500"
            >
              <option value="all">所有方法</option>
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
            </select>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="px-3 py-2 bg-dark-900 border border-dark-700 rounded-lg text-sm text-white focus:outline-none focus:border-blue-500"
            >
              <option value="all">所有状态</option>
              <option value="success">成功</option>
              <option value="error">失败</option>
            </select>
          </div>
        </div>
      </div>

      {/* Logs Table */}
      <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl overflow-hidden">
        <table className="w-full">
          <thead className="bg-dark-900/50 border-b border-dark-700">
            <tr>
              <th className="text-left py-3 px-4 text-xs font-medium text-dark-400">时间</th>
              <th className="text-left py-3 px-4 text-xs font-medium text-dark-400">方法</th>
              <th className="text-left py-3 px-4 text-xs font-medium text-dark-400">路径</th>
              <th className="text-left py-3 px-4 text-xs font-medium text-dark-400">模型</th>
              <th className="text-left py-3 px-4 text-xs font-medium text-dark-400">状态</th>
              <th className="text-left py-3 px-4 text-xs font-medium text-dark-400">延迟</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-dark-700/50">
            {paginatedLogs.length > 0 ? (
              paginatedLogs.map((log, index) => (
                <tr key={index} className="hover:bg-dark-700/30 transition-colors">
                  <td className="py-3 px-4 text-sm text-dark-300">
                    {new Date(log.timestamp).toLocaleString('zh-CN')}
                  </td>
                  <td className="py-3 px-4">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${getMethodColor(log.method)}`}>
                      {log.method}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-sm text-dark-300 font-mono">{log.path}</td>
                  <td className="py-3 px-4 text-sm text-dark-400">{log.model || '-'}</td>
                  <td className={`py-3 px-4 text-sm font-medium ${getStatusColor(log.status)}`}>
                    {log.status}
                  </td>
                  <td className="py-3 px-4 text-sm text-dark-400">{log.duration}ms</td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="py-12 text-center text-dark-500">
                  暂无日志记录
                </td>
              </tr>
            )}
          </tbody>
        </table>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t border-dark-700">
            <div className="text-sm text-dark-500">
              第 {currentPage} 页，共 {totalPages} 页，总计 {filteredLogs.length} 条记录
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                disabled={currentPage === 1}
                className="p-2 rounded-lg hover:bg-dark-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <ChevronLeft size={16} />
              </button>
              <button
                onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                disabled={currentPage === totalPages}
                className="p-2 rounded-lg hover:bg-dark-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <ChevronRight size={16} />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
