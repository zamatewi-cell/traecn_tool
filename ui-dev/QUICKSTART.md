# UI 开发快速启动指南

## ? 快速开始

### 1. 安装依赖

```bash
cd ui-dev
npm install
```

### 2. 开发模式

**仅前端开发**（浏览器中预览）：
```bash
npm run dev
```
访问 http://localhost:5173

**Electron 开发**（桌面应用中测试）：
```bash
npm run electron:dev
```

### 3. 构建生产版本

```bash
npm run build
```

构建产物在 `ui-dev/dist/` 目录。

### 4. 打包 Electron 应用

```bash
npm run electron:build
```

安装包在 `ui-dev/release/` 目录。

## ? 目录结构

```
ui-dev/
├── src/
│   ├── components/     # 可复用组件
│   │   └── Layout.tsx
│   ├── pages/          # 页面组件
│   │   ├── Dashboard.tsx
│   │   ├── Accounts.tsx
│   │   ├── ApiProxy.tsx
│   │   ├── IpManagement.tsx
│   │   ├── Logs.tsx
│   │   └── Settings.tsx
│   ├── styles/         # 样式文件
│   │   └── index.css
│   ├── types/          # TypeScript 类型定义
│   │   ├── electron.d.ts
│   │   └── index.ts
│   ├── utils/          # 工具函数
│   ├── hooks/          # 自定义 Hooks
│   ├── assets/         # 静态资源
│   ├── App.tsx         # 主应用组件
│   └── main.tsx        # 入口文件
├── electron/
│   ├── main.js         # Electron 主进程
│   └── preload.js      # 预加载脚本
├── package.json
├── vite.config.ts
├── tailwind.config.js
└── tsconfig.json
```

## ? 已实现页面

### ? Dashboard (仪表盘)
- 统计卡片（账号数、代理数、请求数、成功率）
- 最近请求列表
- 系统状态面板

### ? Accounts (账号管理)
- 账号列表表格
- 状态标签（活跃/未激活/已过期）
- 使用量进度条
- 分页控件

### ? Layout (布局)
- 侧边栏导航
- 标题栏（Electron 窗口控制）
- 响应式布局

### ? 开发中页面
- API Proxy (API 代理配置)
- IP Management (IP 代理池管理)
- Logs (系统日志)
- Settings (系统设置)

## ? 开发指南

### 添加新页面

1. 在 `src/pages/` 创建新组件：
```tsx
export default function NewPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-white">新页面</h1>
    </div>
  );
}
```

2. 在 [`App.tsx`](ui-dev/src/App.tsx) 添加路由：
```tsx
<Route path="new-page" element={<NewPage />} />
```

3. 在 [`Layout.tsx`](ui-dev/src/components/Layout.tsx) 添加导航项。

### 添加新组件

在 `src/components/` 创建可复用组件：

```tsx
interface ButtonProps {
  children: React.ReactNode;
  onClick?: () => void;
  variant?: 'primary' | 'secondary';
}

export default function Button({ children, onClick, variant = 'primary' }: ButtonProps) {
  const baseClasses = 'px-4 py-2 rounded-lg font-medium transition-colors';
  const variantClasses = {
    primary: 'bg-blue-600 hover:bg-blue-700 text-white',
    secondary: 'bg-slate-700 hover:bg-slate-600 text-slate-300',
  };

  return (
    <button onClick={onClick} className={`${baseClasses} ${variantClasses[variant]}`}>
      {children}
    </button>
  );
}
```

### 样式约定

- 使用 Tailwind 工具类
- 深色主题：`bg-slate-900`, `text-slate-300`
- 主色调：蓝色 (`bg-blue-600`, `text-blue-400`)
- 边框：`border-slate-700`
- 悬停效果：`hover:bg-slate-600`

## ? 待办事项

### 高优先级
1. [ ] API 客户端工具类
2. [ ] 认证 Hooks（useAuth）
3. [ ] 错误处理中间件
4. [ ] 加载状态组件

### 中优先级
5. [ ] WebSocket 实时更新
6. [ ] 主题切换（深色/浅色）
7. [ ] 键盘快捷键
8. [ ] 数据导出功能

### 低优先级
9. [ ] E2E 测试
10. [ ] 性能优化（懒加载）
11. [ ] 无障碍访问（A11y）
12. [ ] 国际化（i18n）

## ? 相关文档

- [UI Agent 职责定义](./README.md)
- [后端 API 文档](../api-agent/README.md)
- [协议规范](../protocol-agent/README.md)
