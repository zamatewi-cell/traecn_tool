import React, { useState } from 'react';
import { Settings as SettingsIcon, Globe, Moon, Bell, Power, Database, Download, Trash2, Shield, Zap, HardDrive, AlertTriangle } from 'lucide-react';

export default function SettingsPage() {
  const [language, setLanguage] = useState('zh');
  const [theme, setTheme] = useState('dark');
  const [autoStart, setAutoStart] = useState(false);
  const [minimizeToTray, setMinimizeToTray] = useState(true);
  const [showNotifications, setShowNotifications] = useState(true);
  const [logRetention, setLogRetention] = useState('7');
  const [dataDir, setDataDir] = useState('');

  const handleOpenDataDir = async () => {
    try {
      const dir = await window.electronAPI?.getDataDir();
      if (dir) {
        setDataDir(dir);
        window.electronAPI?.openPath(dir);
      }
    } catch (error) {
      console.error('Failed to open data directory:', error);
    }
  };

  const handleExportData = async () => {
    alert('数据导出功能开发中...');
  };

  const handleClearData = async () => {
    if (confirm('确定要清除所有数据吗？此操作不可恢复！')) {
      alert('清除数据功能开发中...');
    }
  };

  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center gap-2 mb-6">
        <SettingsIcon size={20} className="text-blue-400" />
        <h1 className="text-lg font-semibold">设置</h1>
      </div>

      <div className="space-y-6">
        {/* General Settings */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Globe size={16} className="text-blue-400" />
            <h2 className="text-sm font-semibold text-dark-200">通用设置</h2>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">语言</div>
              <div className="text-xs text-dark-500">选择界面显示语言</div>
            </div>
            <select
              value={language}
              onChange={(e) => setLanguage(e.target.value)}
              className="px-3 py-1.5 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200 focus:outline-none focus:border-blue-500"
            >
              <option value="zh">中文</option>
              <option value="en">English</option>
            </select>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">主题</div>
              <div className="text-xs text-dark-500">选择界面主题风格</div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setTheme('dark')}
                className={`flex items-center gap-2 px-3 py-1.5 rounded-lg border text-sm transition-colors ${
                  theme === 'dark'
                    ? 'bg-blue-500/20 border-blue-500 text-blue-400'
                    : 'bg-dark-900 border-dark-600 text-dark-400 hover:border-dark-500'
                }`}
              >
                <Moon size={14} />
                深色
              </button>
              <button
                onClick={() => setTheme('light')}
                disabled
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg border border-dark-600 bg-dark-900 text-dark-500 cursor-not-allowed text-sm"
                title="即将支持"
              >
                <span>??</span>
                浅色
              </button>
            </div>
          </div>
        </div>

        {/* System Settings */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Power size={16} className="text-purple-400" />
            <h2 className="text-sm font-semibold text-dark-200">系统设置</h2>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">开机自启动</div>
              <div className="text-xs text-dark-500">系统启动时自动运行应用</div>
            </div>
            <button
              onClick={() => setAutoStart(!autoStart)}
              className={`relative w-12 h-6 rounded-full transition-colors ${
                autoStart ? 'bg-green-500' : 'bg-dark-600'
              }`}
            >
              <div
                className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
                  autoStart ? 'left-7' : 'left-1'
                }`}
              />
            </button>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">最小化到系统托盘</div>
              <div className="text-xs text-dark-500">关闭窗口时隐藏到托盘区域</div>
            </div>
            <button
              onClick={() => setMinimizeToTray(!minimizeToTray)}
              className={`relative w-12 h-6 rounded-full transition-colors ${
                minimizeToTray ? 'bg-green-500' : 'bg-dark-600'
              }`}
            >
              <div
                className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
                  minimizeToTray ? 'left-7' : 'left-1'
                }`}
              />
            </button>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">通知提醒</div>
              <div className="text-xs text-dark-500">显示系统通知和提醒</div>
            </div>
            <button
              onClick={() => setShowNotifications(!showNotifications)}
              className={`relative w-12 h-6 rounded-full transition-colors ${
                showNotifications ? 'bg-green-500' : 'bg-dark-600'
              }`}
            >
              <div
                className={`absolute top-1 w-4 h-4 bg-white rounded-full transition-transform ${
                  showNotifications ? 'left-7' : 'left-1'
                }`}
              />
            </button>
          </div>
        </div>

        {/* Proxy Settings */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Zap size={16} className="text-yellow-400" />
            <h2 className="text-sm font-semibold text-dark-200">代理设置</h2>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">日志保留天数</div>
              <div className="text-xs text-dark-500">超过天数的日志将自动清理</div>
            </div>
            <select
              value={logRetention}
              onChange={(e) => setLogRetention(e.target.value)}
              className="px-3 py-1.5 bg-dark-900 border border-dark-600 rounded-lg text-sm text-dark-200 focus:outline-none focus:border-blue-500"
            >
              <option value="1">1 天</option>
              <option value="3">3 天</option>
              <option value="7">7 天</option>
              <option value="14">14 天</option>
              <option value="30">30 天</option>
              <option value="90">90 天</option>
            </select>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">IP 访问控制</div>
              <div className="text-xs text-dark-500">管理允许访问的 IP 地址</div>
            </div>
            <button className="px-3 py-1.5 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm text-dark-200 transition-colors">
              配置
            </button>
          </div>
        </div>

        {/* Data Management */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Database size={16} className="text-green-400" />
            <h2 className="text-sm font-semibold text-dark-200">数据管理</h2>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">数据目录</div>
              <div className="text-xs text-dark-500">{dataDir || '点击按钮打开数据存储位置'}</div>
            </div>
            <button
              onClick={handleOpenDataDir}
              className="flex items-center gap-2 px-4 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors"
            >
              <HardDrive size={14} />
              打开目录
            </button>
          </div>

          <div className="flex gap-3 pt-2">
            <button
              onClick={handleExportData}
              className="flex items-center gap-2 px-4 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors"
            >
              <Download size={14} />
              导出数据
            </button>
            <button
              onClick={handleClearData}
              className="flex items-center gap-2 px-4 py-2 bg-red-500/20 hover:bg-red-500/30 border border-red-500/30 text-red-400 rounded-lg text-sm transition-colors"
            >
              <Trash2 size={14} />
              清除数据
            </button>
          </div>

          <div className="flex items-start gap-2 p-3 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
            <AlertTriangle size={16} className="text-yellow-400 mt-0.5 flex-shrink-0" />
            <div className="text-xs text-yellow-200">
              <div className="font-medium mb-1">注意</div>
              <div>清除数据将删除所有配置、账号信息和日志记录，此操作不可恢复。请谨慎操作！</div>
            </div>
          </div>
        </div>

        {/* About */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-3">
          <div className="flex items-center gap-2 mb-2">
            <Shield size={16} className="text-blue-400" />
            <h2 className="text-sm font-semibold text-dark-200">关于</h2>
          </div>
          <div className="space-y-2 text-sm text-dark-400">
            <div className="flex justify-between">
              <span>版本</span>
              <span className="text-dark-300">1.0.0</span>
            </div>
            <div className="flex justify-between">
              <span>框架</span>
              <span className="text-dark-300">Electron + React</span>
            </div>
            <div className="flex justify-between">
              <span>构建时间</span>
              <span className="text-dark-300">{new Date().toLocaleDateString('zh-CN')}</span>
            </div>
            <div className="flex justify-between">
              <span>技术支持</span>
              <span className="text-blue-400 cursor-pointer hover:underline">查看文档</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
