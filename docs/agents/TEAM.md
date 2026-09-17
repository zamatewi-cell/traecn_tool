# Trae-Proxy 项目开发 Agent 团队

## 项目概述

**目标**: 开发 Trae CN 反向代理工具，将 Trae CN 的 AI 模型以 OpenAI 兼容 API 形式暴露

**技术栈**: Go 1.22+

**当前阶段**: Phase 0 - 协议逆向工程

---

## Agent 团队架构

```
┌─────────────────────────────────────────────────────────┐
│              Project Manager Agent                       │
│  - 项目协调、任务分配、进度追踪、决策仲裁                │
└─────────────────────────────────────────────────────────┘
              │
    ┌─────────┼─────────┬─────────────┬──────────────┐
    │         │         │             │              │
    ▼         ▼         ▼             ▼              ▼
┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐
│Protocol│ │  API   │ │  Auth  │ │ Queue  │ │  Test    │
│ Agent  │ │ Agent  │ │ Agent  │ │ Agent  │ │  Agent   │
└────────┘ └────────┘ └────────┘ └────────┘ └──────────┘
    │         │         │             │              │
    │         │         │             │              │
    ▼         ▼         ▼             ▼              ▼
协议逆向  API 实现  认证管理  队列调度  质量保障
```

---

## 各 Agent 职责

### 1. Project Manager Agent (PM-Agent)

**职责**:
- 项目整体规划和任务分解
- 分配任务给专业 Agent
- 追踪各 Agent 进度
- 解决跨 Agent 协作问题
- 决策技术选型和架构调整

**工作目录**: `docs/agents/pm-agent/`

**输出**:
- 项目计划文档
- 任务分配记录
- 进度报告
- 决策日志

---

### 2. Protocol Reverse Engineer Agent (Protocol-Agent)

**职责**:
- 抓包分析 Trae CN 网络请求
- 逆向认证流程和 Token 机制
- 分析请求/响应数据结构
- 破解加密算法和签名机制
- 编写协议规范文档

**工作目录**: `docs/agents/protocol-agent/`

**输出**:
- 协议分析文档
- API 端点列表
- 认证流程图
- 加密算法说明
- 请求/响应示例

**当前任务**:
- ? 抓包工具配置 (mitmproxy)
- ? 基础端点探测
- ? 认证流程逆向
- ? Token 生成算法
- ? 加密参数破解

---

### 3. API Development Agent (API-Agent)

**职责**:
- 实现 OpenAI 兼容 API 层
- 协议转换 (OpenAI ? Trae CN)
- 实现流式 SSE 响应
- 错误处理和重试机制
- API 文档编写

**工作目录**: `docs/agents/api-agent/`

**输出**:
- API 路由实现
- 协议转换代码
- SSE 流式处理
- 错误码映射表
- API 使用文档

**依赖**: Protocol-Agent 的协议分析结果

---

### 4. Authentication Manager Agent (Auth-Agent)

**职责**:
- Token 管理和刷新机制
- 账号池管理
- 认证状态监控
- 多账号轮换策略
- 防封号机制设计

**工作目录**: `docs/agents/auth-agent/`

**输出**:
- Token 管理器代码
- 账号池实现
- 认证中间件
- 刷新策略文档
- 风控规避方案

**依赖**: Protocol-Agent 的认证流程分析

---

### 5. Queue Manager Agent (Queue-Agent)

**职责**:
- 请求队列管理
- 优先级调度算法
- 并发控制
- 限流和降级策略
- 排队状态透传

**工作目录**: `docs/agents/queue-agent/`

**输出**:
- 队列管理器代码
- 调度算法实现
- 限流配置
- 排队状态 SSE 扩展
- 并发测试报告

---

### 6. Testing & QA Agent (Test-Agent)

**职责**:
- 编写单元测试
- 集成测试
- 性能测试
- 协议兼容性测试
- Bug 追踪和回归测试

**工作目录**: `docs/agents/test-agent/`

**输出**:
- 测试用例
- 测试脚本
- 性能基准报告
- Bug 列表
- 质量报告

**依赖**: 所有开发 Agent 的产出

---

## 协作流程

### Phase 0: 协议逆向 (当前阶段)

```
PM-Agent: 制定逆向计划
    ↓
Protocol-Agent: 抓包分析 → 输出协议文档
    ↓
┌────────────────────────────────────┐
│ Protocol-Agent 交付物               │
│ - API 端点列表                      │
│ - 认证流程                          │
│ - 请求/响应格式                     │
│ - 加密算法                          │
└────────────────────────────────────┘
    ↓
    ├→ Auth-Agent: 实现认证模块
    ├→ API-Agent: 实现协议转换
    └→ Test-Agent: 编写协议测试
```

### Phase 1: 核心开发

```
Auth-Agent: Token 管理 → 交付认证模块
API-Agent: 协议转换 → 交付 API 层
Queue-Agent: 队列调度 → 交付队列管理
    ↓
Test-Agent: 集成测试 → 交付测试报告
    ↓
PM-Agent: 验收评审
```

### Phase 2: 优化完善

```
Test-Agent: 性能测试 → 发现瓶颈
    ↓
Queue-Agent: 优化调度算法
API-Agent: 优化转换效率
Auth-Agent: 优化刷新策略
    ↓
PM-Agent: 发布准备
```

---

## 任务分配机制

### 任务卡片格式

```markdown
## Task: [任务类型] 任务名称

**ID**: TASK-001  
**优先级**: High/Normal/Low  
**分配给**: @Protocol-Agent  
**依赖**: 无 / TASK-000  
**截止日期**: 2026-03-20  
**状态**: Pending / In Progress / Review / Done  

### 描述
详细说明任务目标和交付物

### 验收标准
- [ ] 标准 1
- [ ] 标准 2

### 技术提示
- 提示 1
- 提示 2
```

### 任务状态流转

```
Pending → In Progress → Review → Done
              ↓             ↓
          Blocked      Rejected
              ↓             ↓
          Pending      In Progress
```

---

## 进度追踪

### 每日站会 (Daily Standup)

每个 Agent 输出日报:

```markdown
## Daily Report - 2026-03-15

### Protocol-Agent
**昨日完成**:
- ? 完成 /api/chat 端点分析
- ? 提取认证头信息

**今日计划**:
- ? 破解 X-Bogus 签名算法
- ? 分析 SSE 流式响应格式

**阻塞问题**:
- ? 需要 PM-Agent 协调获取测试账号
```

### 里程碑追踪

```
Phase 0 - 协议逆向 (2026-03-12 ~ 2026-03-20)
├─ ? 抓包环境搭建 (2026-03-12)
├─ ? 基础端点探测 (2026-03-13)
├─ ? 认证流程逆向 (2026-03-14 ~ 2026-03-16)
├─ ? 加密算法破解 (2026-03-17 ~ 2026-03-19)
└─ ? 协议文档完成 (2026-03-20)

Phase 1 - 核心开发 (2026-03-21 ~ 2026-04-05)
Phase 2 - 优化完善 (2026-04-06 ~ 2026-04-15)
Phase 3 - 发布准备 (2026-04-16 ~ 2026-04-20)
```

---

## 知识管理

### 共享知识库

所有 Agent 共享的知识存储在 `docs/knowledge-base/`:

```
docs/knowledge-base/
├── protocol/           # 协议相关知识
│   ├── api-endpoints.md
│   ├── auth-flow.md
│   └── encryption.md
├── architecture/       # 架构设计
│   ├── system-design.md
│   └── module-interfaces.md
├── decisions/          # 技术决策
│   ├── adr-001-language-choice.md
│   └── adr-002-queue-strategy.md
└── troubleshooting/    # 问题排查
    ├── common-issues.md
    └── debugging-guide.md
```

### 决策记录 (ADR)

每个重大决策都记录 ADR (Architecture Decision Record):

```markdown
# ADR-001: 选择 Go 作为实现语言

## 状态
Accepted

## 背景
需要选择一种语言来实现反向代理

## 决策
使用 Go 1.22+

## 理由
- 并发性能优秀
- 编译为单一二进制，部署简单
- HTTP 库成熟
- 学习曲线低
```

---

## 沟通机制

### Agent 间通信

通过共享文件和事件日志进行异步通信:

```markdown
## Event Log - 2026-03-15

[09:00] Protocol-Agent → All: 开始分析 /api/chat 端点
[10:30] Protocol-Agent → Auth-Agent: 发现认证头 X-Token，请分析
[11:00] Auth-Agent → Protocol-Agent: 收到，正在逆向 Token 生成逻辑
[14:00] Protocol-Agent → All: 协议文档 v0.1 已发布，请查阅
[14:30] API-Agent → Protocol-Agent: 文档清晰，开始实现转换层
```

### 冲突解决

当 Agent 间出现分歧时:

1. 相关 Agent 直接沟通
2. 无法达成一致 → 提交 PM-Agent 仲裁
3. PM-Agent 决策 → 记录到决策日志
4. 所有 Agent 执行决策

---

## 质量保证

### 代码审查

每个 Agent 的代码必须经过另一个 Agent 审查:

```
API-Agent 提交代码 → Protocol-Agent 审查 → 提出意见 → 修改 → 合并
```

### 测试覆盖

Test-Agent 负责确保:
- 单元测试覆盖率 > 80%
- 关键路径 100% 覆盖
- 所有 API 端点有集成测试

### 文档完整性

所有交付物必须包含:
- 代码注释
- API 文档
- 使用示例
- 故障排查指南

---

## 工具链

### 开发工具

- **IDE**: VS Code + Go 扩展
- **抓包**: mitmproxy, Wireshark
- **调试**: Delve, Chrome DevTools
- **测试**: go test, pytest

### 协作工具

- **任务追踪**: GitHub Issues / 本地 Markdown
- **文档**: Markdown + Mermaid 图表
- **版本控制**: Git
- **通信**: 事件日志文件

---

## 启动指南

### 1. 初始化项目

```bash
cd d:\codelearn\vscode\reverse_proxy
git init
go mod init trae-proxy
```

### 2. 创建 Agent 工作目录

```bash
mkdir -p docs/agents/{pm-agent,protocol-agent,api-agent,auth-agent,queue-agent,test-agent}
mkdir -p docs/knowledge-base
```

### 3. 分配初始任务

PM-Agent 创建第一批任务卡片，分配到各 Agent

### 4. 开始执行

各 Agent 根据任务卡片开始工作，每日更新进度

---

## 成功标准

### Phase 0 完成标准

- ? 完整的协议文档
- ? 认证流程可复现
- ? 加密算法可破解
- ? 能手动构造合法请求

### Phase 1 完成标准

- ? OpenAI 兼容 API 可用
- ? 支持所有 14 个模型
- ? 流式响应正常
- ? 账号池管理正常

### Phase 2 完成标准

- ? 性能达标 (P99 < 500ms)
- ? 并发支持 (100+ QPS)
- ? 稳定性测试通过 (24h 无故障)

---

## 联系与协作

各 Agent 通过以下方式协作:

1. **任务卡片**: 在各自目录下创建 `tasks/` 子目录
2. **进度日志**: 每日更新 `daily-report.md`
3. **事件通知**: 写入 `event-log.md` 通知其他 Agent
4. **文档共享**: 公共知识写入 `docs/knowledge-base/`

---

**最后更新**: 2026-03-15  
**维护者**: PM-Agent
