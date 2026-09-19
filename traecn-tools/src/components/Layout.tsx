import React from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import {
  LayoutDashboard, Users, Network, ScrollText,
  Settings, Minus, Square, X,
} from 'lucide-react';

const navItems = [
  { to: '/', icon: LayoutDashboard, label: '仪表盘' },
  { to: '/accounts', icon: Users, label: '账号管理' },
  { to: '/proxy', icon: Network, label: 'API 反代' },
  { to: '/logs', icon: ScrollText, label: '流量日志' },
  { to: '/settings', icon: Settings, label: '设置' },
];

export default function Layout() {
  return (
    <div className="flex flex-col h-screen bg-dark-950">
      {/* Title Bar */}
      <div className="drag-region flex items-center justify-between h-10 bg-dark-900 border-b border-dark-700/50 px-4 shrink-0">
        <div className="flex items-center gap-2 no-drag">
          <div className="w-6 h-6 bg-gradient-to-br from-blue-500 to-cyan-400 rounded-md flex items-center justify-center text-xs font-bold">
            T
          </div>
          <span className="text-sm font-semibold text-dark-200">TraeCN Tools</span>
        </div>
        <div className="flex items-center no-drag">
          <button onClick={() => window.electronAPI?.windowMinimize()}
            className="p-2 hover:bg-dark-700 rounded transition-colors">
            <Minus size={14} className="text-dark-300" />
          </button>
          <button onClick={() => window.electronAPI?.windowMaximize()}
            className="p-2 hover:bg-dark-700 rounded transition-colors">
            <Square size={12} className="text-dark-300" />
          </button>
          <button onClick={() => window.electronAPI?.windowClose()}
            className="p-2 hover:bg-red-500/80 rounded transition-colors">
            <X size={14} className="text-dark-300" />
          </button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <nav className="w-56 bg-dark-900/50 border-r border-dark-700/50 flex flex-col shrink-0">
          <div className="p-3 flex-1 space-y-0.5 overflow-y-auto">
            {navItems.map(({ to, icon: Icon, label }) => (
              <NavLink
                key={to}
                to={to}
                end={to === '/'}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm transition-colors-fast ${
                    isActive
                      ? 'bg-blue-500/15 text-blue-400 font-medium'
                      : 'text-dark-300 hover:bg-dark-700/50 hover:text-dark-200'
                  }`
                }
              >
                <Icon size={18} />
                <span>{label}</span>
              </NavLink>
            ))}
          </div>
          <div className="p-3 border-t border-dark-700/30">
            <div className="text-xs text-dark-500 text-center">v1.0.0</div>
          </div>
        </nav>

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto bg-dark-950">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
