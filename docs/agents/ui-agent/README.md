# UI Development Agent

## ? Role

The UI Development Agent is responsible for designing, implementing, and maintaining all user interface components for the TraeCN Tools application.

## ? Responsibilities

### Primary Tasks
1. **Component Development**: Build reusable React components with TypeScript
2. **Page Implementation**: Create full pages (Dashboard, Accounts, API Proxy, etc.)
3. **Styling**: Implement responsive designs with Tailwind CSS
4. **Electron Integration**: Ensure UI works seamlessly in Electron desktop app
5. **State Management**: Manage application state with Zustand
6. **OAuth Flow**: Implement GitHub OAuth login UI
7. **API Integration**: Connect UI to backend API endpoints

### Quality Standards
- ? Type-safe TypeScript code
- ? Responsive mobile-first design
- ? Accessible (WCAG 2.1 AA compliant)
- ? Performance optimized (lazy loading, code splitting)
- ? Consistent design system
- ? Cross-browser compatible

## ? Workspace

**Active Directory**: `ui-dev/`

All UI development work happens in the [`ui-dev/`](../ui-dev/) directory.

## ? Task Board

### Completed Tasks

**UI-000: Project Setup** ?
- [x] Create directory structure
- [x] Configure build tools (Vite, TypeScript, Tailwind)
- [x] Setup Electron main process
- [x] Create React entry point and App component
- [x] Install npm dependencies

**UI-001: Dashboard Page** ?
- [x] Stats cards with metrics
- [x] Recent requests list
- [x] System status panel
- [x] Responsive grid layout

**UI-002: Accounts Management** ?
- [x] Account list table
- [x] Status badges (active/inactive/expired)
- [x] Usage progress bars
- [x] Pagination controls

**UI-003: Layout Component** ?
- [x] Sidebar navigation
- [x] Title bar with window controls
- [x] Responsive layout
- [x] Electron IPC integration

### In Progress

**UI-004: Additional Pages** ?
- [x] API Proxy page (placeholder)
- [x] IP Management page (placeholder)
- [x] Logs page (placeholder)
- [x] Settings page (placeholder)

### Pending Tasks

**UI-005: API Integration** ?
- [ ] Create API client utility
- [ ] Implement authentication hooks
- [ ] Add error handling
- [ ] Loading states

**UI-006: Advanced Features** ?
- [ ] Real-time updates (WebSocket)
- [ ] Dark/Light theme toggle
- [ ] Keyboard shortcuts
- [ ] Export functionality

## ?? Tech Stack

- **React 18.2** - UI framework
- **TypeScript 5.3** - Type safety
- **Vite 5.0** - Build tooling
- **Tailwind CSS 3.4** - Styling
- **React Router 6** - Navigation
- **Electron 28** - Desktop wrapper
- Source code: `ui-dev/src/`
- Components: `ui-dev/src/components/`
- Pages: `ui-dev/src/pages/`
- Electron: `ui-dev/electron/`

## ? Tech Stack

| Category | Technology |
|----------|-----------|
| Framework | React 18 + TypeScript |
| Build | Vite |
| Styling | Tailwind CSS |
| Desktop | Electron |
| State | Zustand |
| HTTP | Axios |
| Icons | Lucide React |
| Testing | Vitest + React Testing Library |

## ? Current Projects

### 1. Dashboard Page
- **Status**: In Progress
- **Components**: Stats cards, activity charts, quick actions
- **Priority**: High

### 2. Account Management
- **Status**: In Progress
- **Components**: Account list, OAuth login, token management
- **Priority**: High

### 3. API Proxy Configuration
- **Status**: Planned
- **Components**: Proxy settings, endpoint mapping, rule editor
- **Priority**: Medium

### 4. Device Fingerprint
- **Status**: Planned
- **Components**: Device list, fingerprint viewer, device editor
- **Priority**: Medium

### 5. Settings Page
- **Status**: Planned
- **Components**: General settings, advanced config, theme toggle
- **Priority**: Low

## ? Development Workflow

### 1. Task Assignment
Tasks are assigned via the central TaskManager in `internal/workflow/task_manager.go`

### 2. Component Development
```bash
# Start dev server
cd ui-dev
npm run dev

# Build components
npm run build
```

### 3. Testing
```bash
# Run unit tests
npm run test

# Run E2E tests
npm run test:e2e
```

### 4. Build & Package
```bash
# Build Electron app
npm run electron:build

# Create installer
npm run electron:package
```

## ? Task Board

| Task ID | Description | Status | Priority |
|---------|-------------|--------|----------|
| UI-001 | Dashboard layout | In Progress | High |
| UI-002 | Account list page | In Progress | High |
| UI-003 | OAuth login flow | Planned | High |
| UI-004 | API proxy config | Planned | Medium |
| UI-005 | Device management | Planned | Medium |
| UI-006 | Settings page | Planned | Low |

## ? Related Agents

- **API-Agent**: Provides backend API for UI to consume
- **Auth-Agent**: Handles OAuth tokens and authentication
- **Protocol-Agent**: Defines API contracts and endpoints
- **Test-Agent**: Tests UI components and E2E flows

## ? Progress Tracking

Progress is tracked in the central event log:
- See: [`../docs/agents/event-log.md`](../docs/agents/event-log.md)

## ? Design Principles

1. **User-Centric**: Design for ease of use and clarity
2. **Consistency**: Follow established design patterns
3. **Performance**: Optimize for fast load times
4. **Accessibility**: Ensure everyone can use the application
5. **Maintainability**: Write clean, documented code

## ? Resources

- [React Documentation](https://react.dev)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [Electron Docs](https://www.electronjs.org/docs)
- [Vite Guide](https://vitejs.dev/guide/)

---

**Last Updated**: 2026-03-15
**Agent Status**: Active
