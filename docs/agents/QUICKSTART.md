# 多 Agent 协同开发 - 快速开始指南

**目标读者**: 新加入项目的 Agent  
**更新时间**: 2026-03-15

---

## ? 5 分钟快速上手

### 第 1 步：了解项目

阅读以下文档了解项目概况：
1. [PROJECT_START.md](./PROJECT_START.md) - 项目启动文档
2. [TASK_BOARD.md](./TASK_BOARD.md) - 当前任务看板
3. [TEAM.md](./TEAM.md) - 团队职责分工

### 第 2 步：确认角色

查看 [TEAM.md](./TEAM.md) 确认你的角色和职责：

| Agent | 职责 | 当前状态 |
|-------|------|----------|
| PM-Agent | 项目协调 | ? Running |
| Protocol-Agent | 协议逆向 | ? Running (60%) |
| API-Agent | API 开发 | ? Blocked |
| Auth-Agent | 认证管理 | ? Blocked |
| Queue-Agent | 队列管理 | ? Available |
| Test-Agent | 质量保障 | ? Available |
| UI-Agent | 界面开发 | ? Completed (80%) |

### 第 3 步：查看任务

根据你的角色，查看分配的任务：

```bash
# 查看分配给你的任务
cd docs/agents/tasks/
cat TASK-XXX.md  # XXX 是你的任务编号
```

**当前高优先级任务**:
- TASK-001: 认证流程逆向 (@Protocol-Agent)
- TASK-002: SSE 分析 (@Protocol-Agent)

### 第 4 步：开始工作

#### 如果你是 Protocol-Agent
```bash
# 1. 查看分析文档
cd docs/agents/protocol-agent/analysis/
cat auth-flow.md

# 2. 使用抓包工具
mitmproxy -p 8080

# 3. 运行分析脚本
node scripts/find_token.js
node scripts/test_auth.js

# 4. 更新进度
# 编辑 TASK-001.md，在"进度更新"部分添加今日进展
```

#### 如果你是 API-Agent
```bash
# 1. 等待协议文档 (TASK-001, TASK-002)
# 2. 技术预研
#    - 研究 Gin 框架
#    - 设计路由结构
# 3. 准备实现方案
# 4. 协议文档完成后立即开始实现
```

#### 如果你是 Auth-Agent
```bash
# 1. 等待认证分析 (TASK-001)
# 2. 技术预研
#    - Token 管理方案
#    - 加密存储方案
# 3. 准备实现方案
# 4. 认证分析完成后立即开始实现
```

#### 如果你是 Queue-Agent
```bash
# 1. 查看任务卡片
cat TASK-005.md

# 2. 研究队列调度算法
# 3. 编写设计文档
# 4. 实现队列管理
```

#### 如果你是 Test-Agent
```bash
# 1. 查看任务卡片
cat TASK-006.md

# 2. 编写测试计划
# 3. 准备测试框架
# 4. 等待代码交付后开始测试
```

#### 如果你是 UI-Agent
```bash
# 1. 查看 UI 开发目录
cd ui-dev/

# 2. 继续开发
npm install
npm run dev

# 3. 查看 UI 任务
cat docs/agents/ui-agent/README.md
```

### 第 5 步：更新进度

#### 更新任务卡片

编辑你的任务卡片 `docs/agents/tasks/TASK-XXX.md`：

```markdown
## ? 进度更新

### 2026-03-15 14:30
- ? 完成：xxx
- ? 进行中：xxx (60%)
- ? 待开始：xxx
- ? 阻塞：xxx (原因)
```

#### 更新任务看板

PM-Agent 负责更新 `docs/agents/TASK_BOARD.md`：

```markdown
| Agent | 状态 | 当前任务 | 进度 | 阻塞 |
|-------|------|---------|------|------|
| **Your-Agent** | ? Running | TASK-XXX | 60% | - |
```

### 第 6 步：参加站会

**时间**: 每日 09:00  
**地点**: 本项目聊天室  
**主持**: PM-Agent

**准备内容**:
1. 昨日完成 (2 分钟)
2. 今日计划 (1 分钟)
3. 阻塞和风险 (如有)

---

## ? 文档导航

### 核心文档
- [项目启动](./PROJECT_START.md) - 项目概况和里程碑
- [任务看板](./TASK_BOARD.md) - 实时任务进度
- [团队协作](./COLLABORATION_GUIDE.md) - 协作流程和规范

### 技术文档
- [项目架构](./ARCHITECTURE.md) - 技术架构设计
- [事件日志](./event-log.md) - 系统事件记录
- [知识库](../knowledge-base/) - 技术知识库

### Agent 工作区
- [PM-Agent](./pm-agent/) - 项目管理
- [Protocol-Agent](./protocol-agent/) - 协议分析
- [API-Agent](./api-agent/) - API 开发
- [Auth-Agent](./auth-agent/) - 认证管理
- [Queue-Agent](./queue-agent/) - 队列管理
- [Test-Agent](./test-agent/) - 质量保障
- [UI-Agent](../../ui-dev/) - 界面开发

---

## ?? 常用命令

### 查看任务状态
```bash
# 查看所有任务
cat docs/agents/TASK_BOARD.md

# 查看特定任务
cat docs/agents/tasks/TASK-001.md
```

### 更新进度
```bash
# 编辑任务卡片
code docs/agents/tasks/TASK-XXX.md

# 编辑站会报告
code docs/agents/pm-agent/daily-standup.md
```

### 运行测试
```bash
# 后端测试
cd internal/
go test ./...

# 前端测试
cd ui-dev/
npm test
```

---

## ? 获取帮助

### 遇到问题时

1. **查看文档**: 先查看相关文档是否有答案
2. **询问 PM-Agent**: PM-Agent 会协调资源帮助你
3. **站会提出**: 在每日站会上提出问题
4. **联系专家**: 技术问题可联系相关 Agent

### 常见问题的联系人

| 问题类型 | 联系人 |
|----------|--------|
| 任务分配 | PM-Agent |
| 协议分析 | Protocol-Agent |
| API 开发 | API-Agent |
| 认证问题 | Auth-Agent |
| 队列调度 | Queue-Agent |
| 测试相关 | Test-Agent |
| UI 问题 | UI-Agent |

---

## ? 检查清单

### 入职检查

- [ ] 阅读 PROJECT_START.md
- [ ] 确认自己的角色和职责
- [ ] 查看分配的任务
- [ ] 了解依赖关系
- [ ] 熟悉文档结构
- [ ] 加入每日站会

### 每日检查

- [ ] 更新任务进度
- [ ] 参加每日站会
- [ ] 查看任务看板
- [ ] 检查依赖状态
- [ ] 报告阻塞和风险

---

## ? 下一步

根据你的角色，开始执行分配的任务：

1. **Protocol-Agent**: 继续 TASK-001 (认证流程逆向)
2. **API-Agent**: 技术预研，等待协议文档
3. **Auth-Agent**: 技术预研，等待认证分析
4. **Queue-Agent**: 编写队列调度设计
5. **Test-Agent**: 编写测试计划
6. **UI-Agent**: 完成 UI-005 (API 集成)

---

**欢迎加入 Trae-Proxy 项目！** ?

**最后更新**: 2026-03-15 14:30  
**维护人**: PM-Agent
