import React, { useEffect } from 'react';
import { HashRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Accounts from './pages/Accounts';
import ApiProxy from './pages/ApiProxy';
import Logs from './pages/Logs';
import Stats from './pages/Stats';
import Tokens from './pages/Tokens';
import IpManagement from './pages/IpManagement';
import SettingsPage from './pages/Settings';
import { useAppStore } from './store';

export default function App() {
  const init = useAppStore((s) => s.init);
  const loading = useAppStore((s) => s.loading);

  useEffect(() => {
    init();
  }, [init]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-dark-950">
        <div className="text-center">
          <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-cyan-400 rounded-xl flex items-center justify-center text-xl font-bold mx-auto mb-4">
            T
          </div>
          <div className="text-sm text-dark-400">Мгдижа...</div>
        </div>
      </div>
    );
  }

  return (
    <HashRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/accounts" element={<Accounts />} />
          <Route path="/proxy" element={<ApiProxy />} />
          <Route path="/logs" element={<Logs />} />
          <Route path="/stats" element={<Stats />} />
          <Route path="/tokens" element={<Tokens />} />
          <Route path="/ip" element={<IpManagement />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </HashRouter>
  );
}
