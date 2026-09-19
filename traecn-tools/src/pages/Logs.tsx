import React, { useState, useEffect, useRef } from 'react';
import {
  ScrollText, Search, Download, Trash2, RefreshCw,
  Terminal, ArrowDown, Check, Copy, AlertCircle, Play, Square,
} from 'lucide-react';
import { useAppStore } from '../store';

export default function Logs() {
  const { proxyLogs, proxyRunning } = useAppStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [levelFilter, setLevelFilter] = useState<'all' | 'info' | 'warn' | 'error'>('all');
  const [autoScroll, setAutoScroll] = useState(true);
  const [copied, setCopied] = useState(false);
  const [, setRefreshKey] = useState(0);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  // 解析与过滤日志
  const filteredLogs = proxyLogs.filter((line) => {
    if (typeof line !== 'string') return false;
    const matchesSearch = searchTerm === '' || line.toLowerCase().includes(searchTerm.toLowerCase());
    if (!matchesSearch) return false;

    if (levelFilter === 'all') return true;
    const lower = line.toLowerCase();
    if (levelFilter === 'error') return lower.includes('error') || lower.includes('err') || lower.includes('fail') || lower.includes('panic');
    if (levelFilter === 'warn') return lower.includes('warn') || lower.includes('warning');
    if (levelFilter === 'info') return lower.includes('info') || lower.includes('listening') || lower.includes('route');
    return true;
  });

  // 统计指标
  const totalCount = proxyLogs.length;
  const errorCount = proxyLogs.filter(l => typeof l === 'string' && (l.toLowerCase().includes('error') || l.toLowerCase().includes('fail'))).length;
  const warnCount = proxyLogs.filter(l => typeof l === 'string' && l.toLowerCase().includes('warn')).length;
  const infoCount = totalCount - errorCount - warnCount;

  // 自动滚动到底部
  useEffect(() => {
    if (autoScroll && terminalEndRef.current) {
      terminalEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [filteredLogs, autoScroll]);

  // 真实刷新
  const handleRefresh = () => {
    setRefreshKey((k) => k + 1);
  };

  // 真实清空
  const handleClear = () => {
    if (confirm('确定要清空当前的代理日志记录吗？')) {
      useAppStore.setState({ proxyLogs: [] });
    }
  };

  // 真实导出
  const handleExport = () => {
    if (proxyLogs.length === 0) {
      alert('当前没有日志可供导出');
      return;
    }
    const logContent = proxyLogs.join('\n');
    const blob = new Blob([logContent], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `traecn-proxy-logs-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // 复制全部日志
  const handleCopyAll = async () => {
    if (proxyLogs.length === 0) return;
    try {
      await navigator.clipboard.writeText(proxyLogs.join('\n'));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (e) {
      console.error('Failed to copy logs:', e);
    }
  };

  // 获取日志行颜色
  const getLineStyle = (line: string) => {
    const lower = line.toLowerCase();
    if (lower.includes('error') || lower.includes('fail') || lower.includes('panic')) {
      return 'text-red-400 bg-red-950/20';
    }
    if (lower.includes('warn') || lower.includes('warning')) {
      return 'text-yellow-400 bg-yellow-950/20';
    }
    if (lower.includes('success') || lower.includes('listening') || lower.includes('healthy')) {
      return 'text-green-400';
    }
    if (lower.includes('route') || lower.includes('request') || lower.includes('http')) {
      return 'text-cyan-300';
    }
    return 'text-dark-200';
  };

  return (
    <div className="p-6 h-full flex flex-col space-y-4 max-w-7xl mx-auto">
      {/* 顶部标题与操作栏 */}
      <div className="flex items-center justify-between shrink-0">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-blue-500/10 rounded-lg text-blue-400">
            <Terminal size={20} />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white flex items-center gap-2">
              控制台与流量日志
              <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                proxyRunning ? 'bg-green-500/15 text-green-400 border border-green-500/30' : 'bg-dark-700 text-dark-400'
              }`}>
                {proxyRunning ? '● 代理监听中' : '○ 代理已停止'}
              </span>
            </h1>
            <p className="text-xs text-dark-400 mt-0.5">
              实时捕获 Go 核心代理网关标准输出流 (stdout/stderr)，支持快速筛选与审计
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleRefresh}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-800 hover:bg-dark-700 border border-dark-600 rounded-lg text-xs text-dark-200 transition-colors"
            title="重新渲染日志"
          >
            <RefreshCw size={13} />
            刷新
          </button>
          <button
            onClick={handleCopyAll}
            disabled={proxyLogs.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-800 hover:bg-dark-700 border border-dark-600 rounded-lg text-xs text-dark-200 transition-colors disabled:opacity-50"
            title="复制控制台全部日志"
          >
            {copied ? <Check size={13} className="text-green-400" /> : <Copy size={13} />}
            {copied ? '已复制' : '复制'}
          </button>
          <button
            onClick={handleExport}
            disabled={proxyLogs.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-800 hover:bg-dark-700 border border-dark-600 rounded-lg text-xs text-dark-200 transition-colors disabled:opacity-50"
            title="导出文本日志文件"
          >
            <Download size={13} />
            导出
          </button>
          <button
            onClick={handleClear}
            disabled={proxyLogs.length === 0}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 text-red-400 rounded-lg text-xs transition-colors disabled:opacity-50"
            title="清空当前日志流"
          >
            <Trash2 size={13} />
            清空
          </button>
        </div>
      </div>

      {/* 状态指标卡片 */}
      <div className="grid grid-cols-4 gap-3 shrink-0">
        <div className="bg-dark-900/60 border border-dark-700/50 rounded-xl p-3">
          <div className="text-xs text-dark-400 mb-0.5">总日志条数</div>
          <div className="text-xl font-bold text-white">{totalCount}</div>
        </div>
        <div className="bg-dark-900/60 border border-dark-700/50 rounded-xl p-3">
          <div className="text-xs text-dark-400 mb-0.5">常规记录 (INFO)</div>
          <div className="text-xl font-bold text-blue-400">{infoCount >= 0 ? infoCount : 0}</div>
        </div>
        <div className="bg-dark-900/60 border border-dark-700/50 rounded-xl p-3">
          <div className="text-xs text-dark-400 mb-0.5">潜在警告 (WARN)</div>
          <div className="text-xl font-bold text-yellow-400">{warnCount}</div>
        </div>
        <div className="bg-dark-900/60 border border-dark-700/50 rounded-xl p-3">
          <div className="text-xs text-dark-400 mb-0.5">异常拦截 (ERROR)</div>
          <div className="text-xl font-bold text-red-400">{errorCount}</div>
        </div>
      </div>

      {/* 筛选与搜索工具条 */}
      <div className="bg-dark-900/60 border border-dark-700/50 rounded-xl p-3 flex items-center justify-between gap-4 shrink-0">
        <div className="flex items-center gap-2 flex-1 max-w-md relative">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-400" />
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="搜索控制台关键字、模型名称、端口、状态码..."
            className="w-full pl-9 pr-3 py-1.5 bg-dark-950 border border-dark-700 rounded-lg text-xs text-dark-200 placeholder-dark-500 focus:outline-none focus:border-blue-500"
          />
        </div>

        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1 bg-dark-950 p-1 rounded-lg border border-dark-800 text-xs">
            {(['all', 'info', 'warn', 'error'] as const).map((lvl) => (
              <button
                key={lvl}
                onClick={() => setLevelFilter(lvl)}
                className={`px-2.5 py-1 rounded transition-colors ${
                  levelFilter === lvl
                    ? 'bg-blue-600 text-white font-medium'
                    : 'text-dark-400 hover:text-dark-200'
                }`}
              >
                {lvl === 'all' ? '全部' : lvl.toUpperCase()}
              </button>
            ))}
          </div>

          <label className="flex items-center gap-2 text-xs text-dark-300 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
              className="rounded bg-dark-800 border-dark-600 text-blue-600 focus:ring-0 focus:ring-offset-0"
            />
            <ArrowDown size={13} />
            自动滚屏
          </label>
        </div>
      </div>

      {/* 终端控制台核心视图 */}
      <div className="flex-1 min-h-0 bg-dark-950 border border-dark-700/60 rounded-xl flex flex-col overflow-hidden shadow-inner font-mono">
        <div className="px-4 py-2 bg-dark-900/80 border-b border-dark-800 flex items-center justify-between text-xs text-dark-400 shrink-0 select-none">
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-red-500/80 inline-block" />
            <span className="w-2.5 h-2.5 rounded-full bg-yellow-500/80 inline-block" />
            <span className="w-2.5 h-2.5 rounded-full bg-green-500/80 inline-block" />
            <span className="ml-2 text-dark-300 font-sans font-medium">trae-proxy.log</span>
          </div>
          <span>显示 {filteredLogs.length} / {totalCount} 行</span>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-1 text-xs leading-relaxed select-text">
          {filteredLogs.length > 0 ? (
            filteredLogs.map((line, idx) => (
              <div key={idx} className={`flex items-start gap-3 py-0.5 px-2 rounded hover:bg-dark-800/40 transition-colors ${getLineStyle(line)}`}>
                <span className="text-dark-600 select-none w-10 text-right shrink-0 font-sans text-[11px]">{idx + 1}</span>
                <span className="whitespace-pre-wrap break-all flex-1">{line}</span>
              </div>
            ))
          ) : (
            <div className="h-full flex flex-col items-center justify-center text-dark-500 py-16">
              <ScrollText size={36} className="mb-3 opacity-30 text-blue-400" />
              <p className="text-sm">
                {totalCount === 0 ? '暂无代理控制台日志' : '未找到匹配的日志行'}
              </p>
              <p className="text-xs text-dark-600 mt-1 font-sans">
                {totalCount === 0
                  ? '启动「API 反代」后，Go 核心网关的标准输出流将在此实时呈现'
                  : '请尝试修改搜索词或重置级别筛选'}
              </p>
            </div>
          )}
          <div ref={terminalEndRef} />
        </div>
      </div>
    </div>
  );
}