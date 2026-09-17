import React, { useState } from 'react';
import { BarChart3, TrendingUp, Activity, Zap } from 'lucide-react';

export default function Stats() {
  const [timeRange, setTimeRange] = useState<'today' | 'week' | 'month'>('week');

  // Mock data for demonstration
  const stats = {
    today: {
      totalTokens: 1250000,
      totalRequests: 3420,
      avgResponseTime: 1250,
      successRate: 99.2,
    },
    week: {
      totalTokens: 8750000,
      totalRequests: 23940,
      avgResponseTime: 1180,
      successRate: 99.5,
    },
    month: {
      totalTokens: 37500000,
      totalRequests: 102800,
      avgResponseTime: 1220,
      successRate: 99.3,
    },
  };

  const currentStats = stats[timeRange];

  const modelUsage = [
    { model: 'ByteDance', tokens: 2500000, percentage: 28.6, color: 'bg-blue-500' },
    { model: 'DeepSeek', tokens: 2100000, percentage: 24.0, color: 'bg-cyan-500' },
    { model: 'Zhipu', tokens: 1500000, percentage: 17.1, color: 'bg-purple-500' },
    { model: 'Moonshot', tokens: 1200000, percentage: 13.7, color: 'bg-green-500' },
    { model: 'MiniMax', tokens: 850000, percentage: 9.7, color: 'bg-yellow-500' },
    { model: 'Others', tokens: 600000, percentage: 6.9, color: 'bg-dark-500' },
  ];

  const dailyUsage = [
    { day: '周一', requests: 3200, tokens: 1200000 },
    { day: '周二', requests: 3800, tokens: 1450000 },
    { day: '周三', requests: 2900, tokens: 980000 },
    { day: '周四', requests: 4100, tokens: 1680000 },
    { day: '周五', requests: 3600, tokens: 1320000 },
    { day: '周六', requests: 2800, tokens: 950000 },
    { day: '周日', requests: 3500, tokens: 1170000 },
  ];

  const formatNumber = (num: number) => {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <BarChart3 size={20} className="text-blue-400" />
          <h1 className="text-lg font-semibold">Token 统计</h1>
        </div>
        <div className="flex items-center gap-2 bg-dark-800/50 border border-dark-700/50 rounded-lg p-1">
          <button
            onClick={() => setTimeRange('today')}
            className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
              timeRange === 'today' ? 'bg-blue-600 text-white' : 'text-dark-400 hover:text-white'
            }`}
          >
            今日
          </button>
          <button
            onClick={() => setTimeRange('week')}
            className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
              timeRange === 'week' ? 'bg-blue-600 text-white' : 'text-dark-400 hover:text-white'
            }`}
          >
            本周
          </button>
          <button
            onClick={() => setTimeRange('month')}
            className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${
              timeRange === 'month' ? 'bg-blue-600 text-white' : 'text-dark-400 hover:text-white'
            }`}
          >
            本月
          </button>
        </div>
      </div>

      {/* Overview Stats */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <Activity size={18} className="text-blue-400" />
            <span className="text-xs text-dark-400">总 Token</span>
          </div>
          <div className="text-2xl font-bold text-white">
            {formatNumber(currentStats.totalTokens)}
          </div>
          <div className="text-xs text-green-400 mt-1 flex items-center gap-1">
            <TrendingUp size={12} />
            +12.5%
          </div>
        </div>

        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <Zap size={18} className="text-yellow-400" />
            <span className="text-xs text-dark-400">总请求数</span>
          </div>
          <div className="text-2xl font-bold text-white">
            {formatNumber(currentStats.totalRequests)}
          </div>
          <div className="text-xs text-green-400 mt-1 flex items-center gap-1">
            <TrendingUp size={12} />
            +8.3%
          </div>
        </div>

        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <Activity size={18} className="text-cyan-400" />
            <span className="text-xs text-dark-400">平均响应</span>
          </div>
          <div className="text-2xl font-bold text-white">
            {currentStats.avgResponseTime}ms
          </div>
          <div className="text-xs text-red-400 mt-1 flex items-center gap-1">
            <TrendingUp size={12} />
            +3.2%
          </div>
        </div>

        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <div className="flex items-center gap-2 mb-2">
            <BarChart3 size={18} className="text-green-400" />
            <span className="text-xs text-dark-400">成功率</span>
          </div>
          <div className="text-2xl font-bold text-white">
            {currentStats.successRate}%
          </div>
          <div className="text-xs text-green-400 mt-1 flex items-center gap-1">
            <TrendingUp size={12} />
            +0.5%
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        {/* Model Usage Distribution */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <h2 className="text-base font-semibold mb-4">模型使用分布</h2>
          <div className="space-y-3">
            {modelUsage.map((item) => (
              <div key={item.model}>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm text-dark-300">{item.model}</span>
                  <span className="text-sm text-dark-400">
                    {formatNumber(item.tokens)} ({item.percentage}%)
                  </span>
                </div>
                <div className="w-full h-2 bg-dark-700 rounded-full overflow-hidden">
                  <div
                    className={`h-full ${item.color} transition-all duration-500`}
                    style={{ width: `${item.percentage}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Daily Usage Trend */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5">
          <h2 className="text-base font-semibold mb-4">每日使用趋势</h2>
          <div className="space-y-3">
            {dailyUsage.map((day) => (
              <div key={day.day}>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm text-dark-300">{day.day}</span>
                  <span className="text-sm text-dark-400">
                    {formatNumber(day.requests)} 请求
                  </span>
                </div>
                <div className="w-full h-2 bg-dark-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-gradient-to-r from-blue-500 to-cyan-500 transition-all duration-500"
                    style={{ width: `${(day.requests / 4500) * 100}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
