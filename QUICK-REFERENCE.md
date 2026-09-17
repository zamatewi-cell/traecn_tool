# Trae-Proxy 项目快速参考

**最后更新**: 2026-03-15 17:30

---

## ? 项目一览

| 项目 | 信息 |
|------|------|
| **名称** | Trae-Proxy (Trae CN 反向代理) |
| **目标** | 将 Trae CN AI 模型以 OpenAI API 形式暴露 |
| **技术栈** | Go 1.22+, React 18, Electron |
| **阶段** | Phase 0 - 协议逆向 (25%) |
| **预计完成** | 2026-04-10 |
| **团队** | 7 个专业 Agent |

---

## ? 关键文档

### 必读文档

| 文档 | 路径 | 用途 |
|------|------|------|
| ? 团队架构 | [`docs/agents/TEAM.md`](docs/agents/TEAM.md) | 了解各 Agent 职责 |
| ? 协作协议 | [`docs/agents/COLLABORATION-PROTOCOL.md`](docs/agents/COLLABORATION-PROTOCOL.md) | 协作规范和交付标准 |
| ? 项目看板 | [`docs/agents/PROJECT-BOARD.md`](docs/agents/PROJECT-BOARD.md) | 查看进度和任务 |
| ? 启动报告 | [`docs/PROJECT-KICKOFF.md`](docs/PROJECT-KICKOFF.md) | 项目整体介绍 |
| ? 架构文档 | [`docs/agents/ARCHITECTURE.md`](docs/agents/ARCHITECTURE.md) | 系统架构说明 |

### 工作区文档

| Agent | 工作区 | 任务记录 |
|-------|--------|----------|
| PM-Agent | [`docs/agents/pm-agent/`](docs/agents/pm-agent/) | [`DECISION-LOG.md`](docs/agents/pm-agent/DECISION-LOG.md) |
| Protocol-Agent | [`docs/agents/protocol-agent/`](docs/agents/protocol-agent/) | 任务进行中 |
| API-Agent | [`docs/agents/api-agent/`](docs/agents/api-agent/) | 等待任务 |
| Auth-Agent | [`docs/agents/auth-agent/`](docs/agents/auth-agent/) | [`TASK-RECORD.md`](docs/agents/auth-agent/TASK-RECORD.md) |
| Queue-Agent | [`docs/agents/queue-agent/`](docs/agents/queue-agent/) | 等待任务 |
| Test-Agent | [`docs/agents/test-agent/`](docs/agents/test-agent/) | 等待任务 |
| UI-Agent | [`ui-dev/`](ui-dev/) | 已完成 UI-004 |

---

## ? 当前任务

### 高优先级

| 任务 | 执行者 | 进度 | 截止 | 状态 |
|------|--------|------|------|------|
| TASK-001: 认证流程逆向 | Protocol-Agent | 60% | 03-16 | ? |
| TASK-002: SSE 格式分析 | Protocol-Agent | 0% | 03-17 | ? |
| TASK-020: Token 管理器 | Auth-Agent | 预研 | 03-18 | ? |

### 中优先级

| 任务 | 执行者 | 依赖 | 截止 | 状态 |
|------|--------|------|------|------|
| TASK-010: API 层实现 | API-Agent | TASK-001 | 03-19 | ? |
| TASK-011: 协议转换 | API-Agent | TASK-001 | 03-20 | ? |
| TASK-021: 账号池管理 | Auth-Agent | TASK-020 | 03-19 | ? |

---

## ? 关键路径

```
Protocol-Agent (TASK-001)
    ↓
Auth-Agent (TASK-020)
    ↓
API-Agent (TASK-010)
    ↓
Test-Agent (TASK-040)
```

?? **注意**: TASK-001 是关键路径起点，延期会影响全局

---

## ? 沟通指南

### 如何协作

1. **查看事件日志**: [`docs/agents/event-log.md`](docs/agents/event-log.md)
2. **更新任务状态**: 编辑各自的任务记录
3. **请求协作**: 在事件日志中@相关 Agent
4. **问题升级**: 联系 PM-Agent

### 沟通模板

**任务完成通知**:
```
[时间] {Agent} → {目标 Agent}
? {任务 ID} 已完成

交付物:
- 文档链接
- 代码路径
- 使用说明

请查收并开始集成
```

**协作请求**:
```
[时间] {Agent} → {Agent}
? 协作请求

需要:
1. 具体需求 1
2. 具体需求 2

截止时间：{日期}
```

**进度更新**:
```
[时间] {Agent} → All
? 进度更新

任务：{任务 ID}
进度：{百分比}%
状态：{状态描述}
下一步：{计划}
```

---

## ?? 开发环境

### 后端 (Go)

```bash
# 进入项目目录
cd d:\codelearn\vscode\reverse_proxy

# 安装依赖
go mod tidy

# 运行演示
go run cmd/workflow-demo/main.go

# 运行测试
go test ./...
```

### 前端 (UI)

```bash
# 进入 UI 目录
cd ui-dev

# 安装依赖
npm install

# 开发模式
npm run dev

# 构建
npm run build
```

---

## ? 项目指标

### 进度指标

- **整体进度**: 25%
- **Phase 0**: 25% (协议逆向)
- **Phase 1**: 0% (核心开发)
- **Phase 2**: 0% (优化完善)
- **Phase 3**: 0% (发布准备)

### 质量指标

- **测试覆盖率**: 目标 > 80%
- **代码审查**: 进行中
- **Bug 数**: 0 (开发中)

### 性能指标

- **P50 延迟**: 目标 < 200ms
- **P99 延迟**: 目标 < 500ms
- **吞吐量**: 目标 > 100 QPS

---

## ?? 当前风险

### RISK-001: 协议逆向复杂度

- **概率**: 中
- **影响**: 高
- **状态**: 监控中
- **措施**: Protocol-Agent 专注攻关，每日同步

### RISK-002: Token 算法还原

- **概率**: 中
- **影响**: 高
- **状态**: 监控中
- **措施**: 多种方法并行，准备备选方案

---

## ? 重要日期

| 日期 | 事件 | 参与 |
|------|------|------|
| 2026-03-16 | TASK-001 截止 | Protocol-Agent |
| 2026-03-17 | TASK-002 开始 | Protocol-Agent |
| 2026-03-18 | TASK-020 截止 | Auth-Agent |
| 2026-03-20 | Phase 0 完成 | 全体 |
| 2026-03-21 | Phase 1 开始 | 全体 |
| 2026-04-10 | 项目完成 | 全体 |

---

## ? 快速入门

### 新 Agent 加入流程

1. **阅读文档**
   - [ ] [`TEAM.md`](docs/agents/TEAM.md) - 了解职责
   - [ ] [`COLLABORATION-PROTOCOL.md`](docs/agents/COLLABORATION-PROTOCOL.md) - 了解协作规范
   - [ ] [`ARCHITECTURE.md`](docs/agents/ARCHITECTURE.md) - 了解架构

2. **设置环境**
   - [ ] 克隆代码
   - [ ] 安装依赖
   - [ ] 运行测试

3. **接收任务**
   - [ ] 等待 PM-Agent 分配
   - [ ] 创建任务记录
   - [ ] 开始执行

4. **开始协作**
   - [ ] 在事件日志中自我介绍
   - [ ] 了解依赖关系
   - [ ] 建立沟通渠道

---

## ? 检查清单

### 任务开始前

- [ ] 已阅读相关文档
- [ ] 已理解任务需求
- [ ] 已确认依赖关系
- [ ] 已评估工作量
- [ ] 已创建任务记录

### 任务执行中

- [ ] 每日更新进度
- [ ] 遇到问题及时沟通
- [ ] 保持代码质量
- [ ] 编写测试用例
- [ ] 更新相关文档

### 任务完成后

- [ ] 代码审查通过
- [ ] 测试全部通过
- [ ] 文档完整
- [ ] 通知下游 Agent
- [ ] 标记任务完成

---

## ? 获取帮助

### 问题分类

**技术问题** → 相关领域 Agent  
**协作问题** → PM-Agent  
**文档问题** → 文档作者  
**环境问题** → 查看 README

### 联系方式

- **事件日志**: [`docs/agents/event-log.md`](docs/agents/event-log.md)
- **任务看板**: [`docs/agents/PROJECT-BOARD.md`](docs/agents/PROJECT-BOARD.md)
- **决策日志**: [`docs/agents/pm-agent/DECISION-LOG.md`](docs/agents/pm-agent/DECISION-LOG.md)

---

## ? 项目愿景

**让 Trae CN 的 AI 能力更易用、更高效、更可靠！**

---

**快速参考结束**

? **提示**: 将此文档加入书签，方便快速查阅！
