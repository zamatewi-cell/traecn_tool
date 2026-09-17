export interface Account {
  id: number;
  name: string;
  email: string;
  status: 'active' | 'inactive' | 'expired';
  expiresAt: string;
  usage: number;
}

export interface ProxyConfig {
  id: number;
  name: string;
  host: string;
  port: number;
  protocol: 'http' | 'https' | 'socks4' | 'socks5';
  username?: string;
  password?: string;
  isActive: boolean;
}

export interface LogEntry {
  id: number;
  timestamp: string;
  level: 'info' | 'warn' | 'error';
  message: string;
  source: string;
}

export interface SystemStats {
  totalAccounts: number;
  activeProxies: number;
  requestsToday: number;
  successRate: number;
  memoryUsage: number;
  cpuUsage: number;
}
