import React, { useState, useEffect } from 'react';
import {
  Settings as SettingsIcon,
  Database, Download, Upload, Trash2, Shield, HardDrive, AlertTriangle, ExternalLink,
} from 'lucide-react';
import { useAppStore } from '../store';

export default function SettingsPage() {
  const { accounts, proxyConfig } = useAppStore();

  const [dataDir, setDataDir] = useState('');
  const [appVersion, setAppVersion] = useState('v1.0.1');

  useEffect(() => {
    window.electronAPI?.getDataDir().then((dir) => {
      if (dir) setDataDir(dir);
    }).catch(() => {});

    if (window.electronAPI?.getAppVersion) {
      window.electronAPI.getAppVersion().then((ver: string) => {
        if (ver) setAppVersion(`v${ver.replace(/^v/, '')}`);
      }).catch(() => {});
    }
  }, []);

  const handleOpenDataDir = async () => {
    try {
      const dir = await window.electronAPI?.getDataDir();
      if (dir) {
        setDataDir(dir);
        await window.electronAPI?.openPath(dir);
      }
    } catch (error) {
      console.error('Failed to open data directory:', error);
    }
  };

  // ST1: 真实导出全部数据 (全量数据备份)
  const handleExportData = async () => {
    try {
      const backupData = {
        _backupType: 'full_backup',
        version: appVersion,
        exportedAt: new Date().toISOString(),
        accounts,
        proxyConfig,
      };

      const defaultFileName = `traecn-tools-full-backup-${new Date().toISOString().slice(0, 10)}.json`;

      if (window.electronAPI?.exportAccounts) {
        const res = await window.electronAPI.exportAccounts({
          data: backupData,
          defaultFileName,
          title: '导出完整配置与数据备份',
        });
        if (res.canceled) return;
        if (res.success) {
          alert('数据备份已成功导出至：' + res.filePath);
        } else if (res.error) {
          alert('导出失败: ' + res.error);
        }
        return;
      }

      // Web 备用方案
      const blob = new Blob([JSON.stringify(backupData, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = defaultFileName;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e: any) {
      alert('导出遇到异常: ' + e?.message);
    }
  };

  // ST1.5: 真实从备份恢复数据
  const handleRestoreData = async () => {
    try {
      if (!window.electronAPI?.importAccounts) {
        alert('当前环境不支持从本地文件恢复数据');
        return;
      }
      const res = await window.electronAPI.importAccounts();
      if (res.canceled) return;
      if (!res.success || !res.data) {
        alert('恢复失败: ' + (res.error || '未能解析备份数据'));
        return;
      }

      const backup = res.data;
      if (backup._exportType === 'sanitized_accounts_export') {
        alert('【恢复中止】您选择的文件是「脱敏账号列表」，其中的凭据已脱敏，无法用于数据恢复！请选择全量数据备份文件。');
        return;
      }

      const accountsToRestore = backup.accounts || (Array.isArray(backup) ? backup : []);
      const proxyConfigToRestore = backup.proxyConfig || null;

      if (!accountsToRestore.length && !proxyConfigToRestore) {
        alert('所选文件不包含有效的账号或代理配置数据！');
        return;
      }

      if (!confirm(`确定要从该备份中恢复数据吗？\n包含 ${accountsToRestore.length} 个账号及相关设置，当前未保存的配置将被覆盖。`)) {
        return;
      }

      // 步骤 1: 停止正在运行的代理服务
      if (window.electronAPI?.stopProxy) {
        await window.electronAPI.stopProxy();
      }

      const currentConfig = useAppStore.getState().proxyConfig;
      const mergedConfig = proxyConfigToRestore ? { ...currentConfig, ...proxyConfigToRestore } : currentConfig;

      // 步骤 2: 持久化存储
      if (window.electronAPI?.saveData) {
        await window.electronAPI.saveData({
          accounts: accountsToRestore,
          proxyConfig: mergedConfig,
          settings: { language: 'zh', theme: 'dark' },
        });
      }

      // 步骤 3: 更新 Store
      useAppStore.setState({
        accounts: accountsToRestore,
        proxyConfig: mergedConfig,
        proxyRunning: false,
      });

      alert(`数据恢复成功！已还原 ${accountsToRestore.length} 个账号及代理配置。`);
    } catch (e: any) {
      alert('恢复数据遇到异常: ' + e?.message);
    }
  };

  // ST2: 真实清除所有数据 (强同步前置守卫版)
  const handleClearData = async () => {
    if (!confirm('【高危确认】确定要清除所有本地数据与配置吗？此操作将立即停止运行中的代理服务，并重置全部账号与代理设置，不可撤销！')) {
      return;
    }

    try {
      // 步骤 1: 强同步前置守卫 —— 无条件调用 stopProxy() 彻底终止后台子进程
      if (window.electronAPI?.stopProxy) {
        const stopRes = await window.electronAPI.stopProxy();
        if (stopRes && !stopRes.success && stopRes.error !== '服务未运行') {
          throw new Error(`停止正在运行的代理服务失败: ${stopRes.error}，为防止凭据残留已中止重置！`);
        }
      }

      // 步骤 2: 清空本地持久化存储
      const emptyConfig = {
        listenPort: 8045,
        requestTimeout: 120,
        autoStart: false,
        allowLan: false,
        authEnabled: true,
        authMode: 'auto' as const,
        apiKey: '',
        webUiPassword: '',
        userAgentOverride: false,
        userAgentValue: '',
      };

      if (window.electronAPI?.saveData) {
        await window.electronAPI.saveData({
          accounts: [],
          proxyConfig: emptyConfig,
          settings: { language: 'zh', theme: 'dark' },
        });
      }

      // 步骤 3: 完整重置前端 Store 状态机 (显式包含 proxyRunning: false)
      useAppStore.setState({
        accounts: [],
        currentAccountId: null,
        proxyConfig: emptyConfig,
        proxyLogs: [],
        proxyRunning: false,
      });

      alert('所有本地数据已成功清空，后台代理服务已彻底终止并重置为初始状态');
    } catch (e: any) {
      alert('清除数据失败: ' + e?.message);
    }
  };

  // ST6: 真实打开官方文档
  const handleOpenDoc = () => {
    if (window.electronAPI?.openExternal) {
      window.electronAPI.openExternal('https://trae.cn');
    } else {
      window.open('https://trae.cn', '_blank');
    }
  };

  return (
    <div className="p-6 max-w-4xl space-y-6">
      <div className="flex items-center gap-2 mb-2">
        <SettingsIcon size={20} className="text-blue-400" />
        <h1 className="text-lg font-semibold">系统设置</h1>
      </div>

      <div className="space-y-6">

        {/* Data Management (ST1/ST2: 真实导出与真实清除) */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-4">
          <div className="flex items-center gap-2 mb-2">
            <Database size={16} className="text-green-400" />
            <h2 className="text-sm font-semibold text-dark-200">存储与数据管理</h2>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-dark-200">数据目录</div>
              <div className="text-xs text-dark-500 font-mono truncate max-w-md">{dataDir || '点击按钮打开存储位置'}</div>
            </div>
            <button
              onClick={handleOpenDataDir}
              className="flex items-center gap-2 px-4 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors text-dark-200"
            >
              <HardDrive size={14} />
              打开目录
            </button>
          </div>

          <div className="flex flex-wrap gap-3 pt-2">
            <button
              onClick={handleExportData}
              className="flex items-center gap-2 px-4 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors text-dark-200"
            >
              <Download size={14} />
              导出数据备份
            </button>
            <button
              onClick={handleRestoreData}
              className="flex items-center gap-2 px-4 py-2 bg-dark-700 hover:bg-dark-600 border border-dark-600 rounded-lg text-sm transition-colors text-dark-200"
            >
              <Upload size={14} />
              从备份恢复数据
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
              <div className="font-medium mb-0.5">注意</div>
              <div>清除数据将永久重置所有账号与自定义代理配置，操作前建议先执行「导出数据备份」。</div>
            </div>
          </div>
        </div>

        {/* About (ST6: 真实打开官方文档) */}
        <div className="bg-dark-800/50 border border-dark-700/50 rounded-xl p-5 space-y-3">
          <div className="flex items-center gap-2 mb-2">
            <Shield size={16} className="text-blue-400" />
            <h2 className="text-sm font-semibold text-dark-200">关于 TraeCN Tools</h2>
          </div>
          <div className="space-y-2 text-sm text-dark-400">
            <div className="flex justify-between">
              <span>版本号</span>
              <span className="text-dark-300 font-mono">{appVersion} (Production Release)</span>
            </div>
            <div className="flex justify-between">
              <span>应用架构</span>
              <span className="text-dark-300">Electron + Vite + React 18 + Tailwind CSS</span>
            </div>
            <div className="flex justify-between">
              <span>核心网关</span>
              <span className="text-dark-300">Go trae-proxy (Masticate & AgentTask Engine)</span>
            </div>
            <div className="flex justify-between items-center">
              <span>官方站点与指南</span>
              <button
                onClick={handleOpenDoc}
                className="text-blue-400 hover:text-blue-300 transition-colors flex items-center gap-1 text-xs"
              >
                <span>访问 Trae CN 官网</span>
                <ExternalLink size={12} />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
