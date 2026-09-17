# Trae-Proxy 多 Agent 协同开发 - 文档结构总览

**更新时间**: 2026-03-15  
**维护人**: PM-Agent

---

## ? 完整文档结构

```
docs/agents/
│
├── ? README.md                          # ? 主入口文档（从这里开始）
├── ? PROJECT_START.md                   # ? 项目启动和目标
├── ? TASK_BOARD.md                      # ? 实时任务看板
├── ? QUICKSTART.md                      # ? 5 分钟快速上手
├── ? COLLABORATION_GUIDE.md             # ? 协作流程和规范
├── ? TEAM.md                            # ? 团队职责分工
├── ? ARCHITECTURE.md                    # ?? 技术架构设计
├── ? event-log.md                       # ? 系统事件日志
├── ? STARTUP_REPORT.md                  # ? 项目启动报告
│
├── ? tasks/                             # ? 任务卡片目录
│   ├── TASK-001.md                       # ? 认证流程逆向
│   ├── TASK-002.md                       # ? SSE 流式响应分析
│   ├── TASK-003.md                       # ? 账号池管理设计
│   ├── TASK-004.md                       # ? API 路由映射设计
│   ├── TASK-005.md                       # ? 队列调度算法
│   └── TASK-006.md                       # ? 测试用例编写
│
├── ? pm-agent/                          # ? PM-Agent 工作区
│   ├── README.md                         # PM 职责说明
│   └── daily-standup.md                  # ? 每日站会报告
│
├── ? protocol-agent/                    # ? Protocol-Agent 工作区
│   ├── README.md                         # 协议逆向职责
│   └── analysis/                         # ? 分析文档
│       └── auth-flow.md                  # ? 认证流程分析
│
├── ? api-agent/                         # ? API-Agent 工作区
│   └── README.md                         # API 开发职责
│
├── ? auth-agent/                        # ? Auth-Agent 工作区
│   └── README.md                         # 认证管理职责
│
├── ? queue-agent/                       # ? Queue-Agent 工作区
│   └── README.md                         # 队列管理职责
│
├── ? test-agent/                        # ? Test-Agent 工作区
│   └── README.md                         # 质量保障职责
│
└── ? ui-agent/                          # ? UI-Agent 工作区
    └── README.md                         # UI 开发职责
    (实际代码在 ../../ui-dev/)
```

---

## ? 文档分类索引

### ? 新手入门（必读）

| 文档 | 用途 | 阅读时间 | 链接 |
|------|------|----------|------|
| **README.md** | 项目概况和导航 | 5 分钟 | [README.md](./README.md) |
| **QUICKSTART.md** | 5 分钟快速上手 | 5 分钟 | [QUICKSTART.md](./QUICKSTART.md) |
| **PROJECT_START.md** | 项目目标和里程碑 | 10 分钟 | [PROJECT_START.md](./PROJECT_START.md) |

### ? 协作规范（重要）

| 文档 | 用途 | 阅读时间 | 链接 |
|------|------|----------|------|
| **COLLABORATION_GUIDE.md** | 协作流程和规范 | 15 分钟 | [COLLABORATION_GUIDE.md](./COLLABORATION_GUIDE.md) |
| **TEAM.md** | 团队职责分工 | 10 分钟 | [TEAM.md](./TEAM.md) |
| **TASK_BOARD.md** | 实时任务进度 | 5 分钟 | [TASK_BOARD.md](./TASK_BOARD.md) |

### ? 进度追踪（每日更新）

| 文档 | 用途 | 更新频率 | 链接 |
|------|------|----------|------|
| **TASK_BOARD.md** | 任务看板 | 每日 | [TASK_BOARD.md](./TASK_BOARD.md) |
| **pm-agent/daily-standup.md** | 每日站会 | 每日 | [daily-standup.md](./pm-agent/daily-standup.md) |
| **tasks/TASK-XXX.md** | 任务卡片 | 实时 | [tasks/](./tasks/) |

### ?? 技术文档（参考）

| 文档 | 用途 | 目标读者 | 链接 |
|------|------|----------|------|
| **ARCHITECTURE.md** | 技术架构 | 全体 Agent | [ARCHITECTURE.md](./ARCHITECTURE.md) |
| **event-log.md** | 事件日志 | PM-Agent | [event-log.md](./event-log.md) |
| **protocol-agent/analysis/** | 协议分析 | API/Auth-Agent | [analysis/](./protocol-agent/analysis/) |

### ? 任务文档（执行）

| 文档 | 负责人 | 优先级 | 状态 | 链接 |
|------|--------|--------|------|------|
| **TASK-001.md** | Protocol-Agent | ? High | ? 60% | [TASK-001.md](./tasks/TASK-001.md) |
| **TASK-002.md** | Protocol-Agent | ? High | ? 0% | [TASK-002.md](./tasks/TASK-002.md) |
| **TASK-003.md** | Auth-Agent | ? Medium | ? 0% | [TASK-003.md](./tasks/TASK-003.md) |
| **TASK-004.md** | API-Agent | ? Medium | ? 0% | [TASK-004.md](./tasks/TASK-004.md) |
| **TASK-005.md** | Queue-Agent | ? Low | ? 0% | [TASK-005.md](./tasks/TASK-005.md) |
| **TASK-006.md** | Test-Agent | ? Low | ? 0% | [TASK-006.md](./tasks/TASK-006.md) |

---

## ? 按角色查看文档

### PM-Agent

**必读文档**:
- [TASK_BOARD.md](./TASK_BOARD.md) - 任务看板
- [daily-standup.md](./pm-agent/daily-standup.md) - 站会报告
- [COLLABORATION_GUIDE.md](./COLLABORATION_GUIDE.md) - 协作规范

**工作文档**:
- 更新任务看板
- 编写站会报告
- 分配任务卡片

### Protocol-Agent

**必读文档**:
- [tasks/TASK-001.md](./tasks/TASK-001.md) - 当前任务
- [tasks/TASK-002.md](./tasks/TASK-002.md) - 下一任务
- [COLLABORATION_GUIDE.md](./COLLABORATION_GUIDE.md) - 协作规范

**工作文档**:
- [analysis/auth-flow.md](./protocol-agent/analysis/auth-flow.md) - 认证分析
- 编写分析文档

### API-Agent

**必读文档**:
- [tasks/TASK-004.md](./tasks/TASK-004.md) - 任务卡片
- [analysis/auth-flow.md](./protocol-agent/analysis/auth-flow.md) - 认证分析（等待交付）
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 架构设计

**准备工作**:
- 技术预研
- 等待协议文档

### Auth-Agent

**必读文档**:
- [tasks/TASK-003.md](./tasks/TASK-003.md) - 任务卡片
- [analysis/auth-flow.md](./protocol-agent/analysis/auth-flow.md) - 认证分析（等待交付）
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 架构设计

**准备工作**:
- 技术预研
- 等待认证分析

### Queue-Agent

**必读文档**:
- [tasks/TASK-005.md](./tasks/TASK-005.md) - 任务卡片
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 架构设计

**准备工作**:
- 队列算法研究
- 限流方案设计

### Test-Agent

**必读文档**:
- [tasks/TASK-006.md](./tasks/TASK-006.md) - 任务卡片
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 架构设计

**准备工作**:
- 测试计划编写
- 测试框架准备

### UI-Agent

**必读文档**:
- UI 相关任务卡片
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 架构设计

**工作目录**:
- [../../ui-dev/](../../ui-dev/) - 实际代码目录

---

## ? 文档更新频率

### 每日更新
- [TASK_BOARD.md](./TASK_BOARD.md) - PM-Agent
- [pm-agent/daily-standup.md](./pm-agent/daily-standup.md) - PM-Agent
- [tasks/TASK-XXX.md](./tasks/) - 各 Agent

### 实时更新
- [event-log.md](./event-log.md) - PM-Agent
- [analysis/auth-flow.md](./protocol-agent/analysis/auth-flow.md) - Protocol-Agent

### 按需更新
- [ARCHITECTURE.md](./ARCHITECTURE.md) - 技术变更时
- [TEAM.md](./TEAM.md) - 团队调整时
- [COLLABORATION_GUIDE.md](./COLLABORATION_GUIDE.md) - 流程优化时

---

## ? 快速查找

### 我想知道...

**项目概况** → [README.md](./README.md) 或 [PROJECT_START.md](./PROJECT_START.md)

**如何开始** → [QUICKSTART.md](./QUICKSTART.md)

**我的任务** → [TASK_BOARD.md](./TASK_BOARD.md) → [tasks/](./tasks/)

**当前进度** → [TASK_BOARD.md](./TASK_BOARD.md)

**协作规范** → [COLLABORATION_GUIDE.md](./COLLABORATION_GUIDE.md)

**团队职责** → [TEAM.md](./TEAM.md)

**技术架构** → [ARCHITECTURE.md](./ARCHITECTURE.md)

**认证分析** → [protocol-agent/analysis/auth-flow.md](./protocol-agent/analysis/auth-flow.md)

**每日站会** → [pm-agent/daily-standup.md](./pm-agent/daily-standup.md)

**项目报告** → [STARTUP_REPORT.md](./STARTUP_REPORT.md)

---

## ? 文档维护

### 责任人

- **README.md**: PM-Agent
- **TASK_BOARD.md**: PM-Agent
- **任务卡片**: 对应负责人
- **分析文档**: Protocol-Agent
- **站会报告**: PM-Agent

### 命名规范

- 文件名：`kebab-case.md` (如 `TASK-001.md`)
- 目录名：`kebab-case` (如 `protocol-agent`)
- 任务卡片：`TASK-XXX.md`

### 版本控制

所有文档通过 Git 版本控制，重要变更需要：
1. 提交信息清晰
2. 更新文档末尾的"最后更新"时间
3. 必要时在文档中添加更新日志

---

## ? 外部链接

- [项目根目录](../../)
- [UI 开发目录](../../ui-dev/)
- [知识库](../knowledge-base/)
- [协议文档](../knowledge-base/protocol/)

---

**最后更新**: 2026-03-15 14:30  
**维护人**: PM-Agent  
**下次审查**: 2026-03-22 (周会)
