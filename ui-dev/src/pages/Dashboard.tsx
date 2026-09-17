import { useState, useEffect } from 'react';

export default function Dashboard() {
  const [stats, setStats] = useState({
    totalAccounts: 0,
    activeProxies: 0,
    requestsToday: 0,
    successRate: 0,
  });

  useEffect(() => {
    // TODO: Fetch real stats from backend
    setStats({
      totalAccounts: 12,
      activeProxies: 5,
      requestsToday: 1247,
      successRate: 98.5,
    });
  }, []);

  const statCards = [
    { title: '总账号数', value: stats.totalAccounts, icon: '?', color: 'bg-blue-500' },
    { title: '活跃代理', value: stats.activeProxies, icon: '?', color: 'bg-green-500' },
    { title: '今日请求', value: stats.requestsToday, icon: '?', color: 'bg-purple-500' },
    { title: '成功率', value: `${stats.successRate}%`, icon: '?', color: 'bg-orange-500' },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-white">仪表盘</h1>
        <div className="flex items-center space-x-3">
          <span className="text-sm text-slate-400">最后更新：刚刚</span>
          <button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors">
            刷新数据
          </button>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statCards.map((stat, index) => (
          <div
            key={index}
            className="bg-slate-800 rounded-xl p-6 border border-slate-700 hover:border-slate-600 transition-colors"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-slate-400 mb-1">{stat.title}</p>
                <p className="text-3xl font-bold text-white">{stat.value}</p>
              </div>
              <div className={`w-12 h-12 ${stat.color} rounded-lg flex items-center justify-center text-2xl`}>
                {stat.icon}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Recent Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Requests */}
        <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-white mb-4">最近请求</h2>
          <div className="space-y-3">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="flex items-center justify-between py-2 border-b border-slate-700 last:border-0">
                <div className="flex items-center space-x-3">
                  <span className="w-2 h-2 bg-green-500 rounded-full"></span>
                  <span className="text-sm text-slate-300">POST /api/v1/chat</span>
                </div>
                <span className="text-xs text-slate-500">{i * 5}分钟前</span>
              </div>
            ))}
          </div>
        </div>

        {/* System Status */}
        <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-white mb-4">系统状态</h2>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-400">API 服务</span>
              <span className="px-2 py-1 bg-green-500/20 text-green-400 text-xs rounded-full">运行中</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-400">代理服务</span>
              <span className="px-2 py-1 bg-green-500/20 text-green-400 text-xs rounded-full">运行中</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-400">数据库连接</span>
              <span className="px-2 py-1 bg-green-500/20 text-green-400 text-xs rounded-full">正常</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-400">内存使用</span>
              <span className="text-sm text-white">245 MB / 512 MB</span>
            </div>
            <div className="w-full bg-slate-700 rounded-full h-2">
              <div className="bg-blue-500 h-2 rounded-full" style={{ width: '48%' }}></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
