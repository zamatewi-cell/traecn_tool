# 多 Agent 协作系统 - 任务分配总览

> 本文档追踪 7 个 Agent 的任务分配和进度状态

**最后更新**: 2026-03-15  
**项目状态**: ? On Track (Phase 0 - 15% 完成)  
**下次同步**: 每日 09:00

---

## ? 执行摘要

**项目名称**: Trae CN Reverse Proxy  
**当前阶段**: Phase 0 - 协议逆向工程 (15% 完成)  
**目标**: 将 Trae CN 的 AI 能力反代为 OpenAI 兼容 API  
**架构**: 多 Agent 协作系统 (7 个 Agent)  
**技术栈**: Go 1.22+, Event Bus, Workflow Engine

### 关键里程碑
- ? M0: 项目启动完成 (2026-03-15)
- ? M1: 协议分析完成 (目标：2026-03-20) - 15%
- ? M2: 基础功能可用 (目标：2026-03-25)
- ? M3: 生产就绪 (目标：2026-04-01)

### 总体进度
```
总体进度：████████???????? 15%

Phase 0: ████████?????????? 15% (协议分析)
Phase 1: ??????????????????  0% (认证模块)
Phase 2: ████?????????????? 20% (API 框架)
Phase 3: ████?????????????? 20% (队列框架)
Phase 4: ??????????????????  0% (负载均衡)
Phase 5: ??????????????????  0% (管理面板)
```

---

## Agent 团队

### 1. PM Agent (项目经理)
**职责**: 项目规划、任务分解、进度追踪、跨 Agent 协调  
**状态**: ? Active  
**工作目录**: `docs/agents/pm-agent/`

#### 当前任务
- [x] PM-0.1: 创建项目路线图
- [x] PM-0.2: 定义 Agent 职责边界
- [x] PM-0.3: 建立任务优先级系统
- [ ] PM-0.4: 设置进度报告机制
- [ ] PM-0.5: 风险管理文档

#### 关键决策
- ? 采用多 Agent 架构
- ? 使用 Go 1.22+ 作为主要语言
- ? Event Bus 作为通信机制

---

### 2. Protocol Agent (协议工程师)
**职责**: 协议分析、逆向工程、API 端点提取  
**状态**: ? Active  
**工作目录**: `docs/agents/protocol-agent/`

#### 当前任务
- [x] P-0.1: 分析 Trae CN 安装目录结构
- [x] P-0.2: 提取 product.json 配置
- [x] P-0.3: 分析 aiserver/server.js
- [x] P-0.4: 提取所有 API 端点
- [ ] P-0.5: 深度分析 extension.js
- [ ] P-0.6: 识别加密/签名机制
- [ ] P-0.7: 编写协议分析文档

#### 已发现端点
```go
const (
    EndpointChatCompletion  = "/api/ide/v1/chat_completion"
    EndpointLLMRawChat      = "/api/ide/v1/llm_raw_chat"
    EndpointModelList       = "/api/ide/v1/model_list"
    EndpointGetDetailParam  = "/api/ide/v1/get_detail_param"
    EndpointAgentCreateTask = "/api/agent/v3/create_agent_task"
    EndpointAgentCommitTool = "/api/agent/v3/commit_toolcall_result"
    // ... 更多端点
)
```

#### 认证机制
- Token 存储：`%APPDATA%\Trae CN\storage.json`
- Token 类型：JWT Token
- 刷新机制：Refresh Token
- 认证头：`X-Auth-Token`, `X-JWT-Token`

---

### 3. Auth Agent (认证专家)
**职责**: 认证流程、Token 管理、多账号管理  
**状态**: ? Active  
**工作目录**: `docs/agents/auth-agent/`

#### 当前任务
- [x] A-0.1: 定位 Token 存储位置
- [x] A-0.2: 分析 Token 结构
- [x] A-0.3: 实现 TokenProvider 基础
- [ ] A-0.4: 实现 Token 自动刷新
- [ ] A-0.5: 实现多账号轮询
- [ ] A-0.6: 测试 Token 有效性

#### Token 结构
```go
type TraeAuth struct {
    Token            string    `json:"token"`
    RefreshToken     string    `json:"refreshToken"`
    ExpiredAt        time.Time `json:"expiredAt"`
    RefreshExpiredAt time.Time `json:"refreshExpiredAt"`
    UserID           string    `json:"userId"`
    // ...
}
```

#### TokenProvider 功能
- ? 多账号管理
- ? Round-robin 轮询
- ? Token 过期检测
- ? 自动刷新
- ? 故障转移

---

### 4. API Agent (API 工程师)
**职责**: API 实现、OpenAI 格式转换、请求处理  
**状态**: ? Active  
**工作目录**: `docs/agents/api-agent/`

#### 当前任务
- [x] API-0.1: 实现 /v1/models 端点
- [x] API-0.2: 实现 /v1/chat/completions 基础
- [ ] API-0.3: 完善 OpenAI 格式转换
- [ ] API-0.4: 实现流式响应 (SSE)
- [ ] API-0.5: 实现错误处理
- [ ] API-0.6: 实现请求日志

#### 已实现端点
```
GET  /v1/models         - 获取模型列表
POST /v1/chat/completions - 聊天补全
GET  /v1/queue/status   - 排队状态
GET  /health            - 健康检查
GET  /                  - 根路径
```

#### 支持的模型
- doubao-seed-1.6, doubao-1.5-pro
- deepseek-v3, deepseek-r1
- glm-5, glm-4-plus
- kimi-k2, minimax-m1
- qwen3-coder, qwen3
- gemini-2.5-pro, claude-sonnet-4, gpt-4.1

---

### 5. Queue Agent (调度工程师)
**职责**: 请求队列、并发控制、限流降级  
**状态**: ? Active  
**工作目录**: `docs/agents/queue-agent/`

#### 当前任务
- [x] Q-0.1: 实现基础请求队列
- [x] Q-0.2: 实现限流器 (RateLimiter)
- [x] Q-0.3: 实现熔断器 (CircuitBreaker)
- [ ] Q-0.4: 实现 MLFQ 调度算法
- [ ] Q-0.5: 集成排队状态监控
- [ ] Q-0.6: 实现 SSE 进度推送

#### 队列设计
```go
type RequestQueue struct {
    Requests []*QueuedRequest
    mu       sync.Mutex
}

type RateLimiter struct {
    Tokens     float64
    MaxTokens  float64
    RefillRate float64 // tokens/秒
}

type CircuitBreakerStatus struct {
    State        string // closed/open/half-open
    FailureCount int
    Threshold    int
}
```

---

### 6. Test Agent (测试工程师)
**职责**: 测试用例、质量验证、自动化测试  
**状态**: ? Active  
**工作目录**: `docs/agents/test-agent/`

#### 当前任务
- [x] T-0.1: 创建测试脚本模板
- [ ] T-0.2: 编写 API 端点测试
- [ ] T-0.3: 编写认证流程测试
- [ ] T-0.4: 编写压力测试
- [ ] T-0.5: 自动化测试流水线
- [ ] T-0.6: 编写集成测试

#### 测试脚本目录
```
scripts/
├── test_api.js              - API 端点测试
├── test_chat_v3.js          - 聊天 API 测试
├── test_connect.js          - 连接测试
├── test_endpoints.js        - 端点发现
├── test_model_config.js     - 模型配置测试
├── test_prompts.js          - Prompt 测试
└── ... (更多测试脚本)
```

---

### 7. UI Agent (UI 设计师)
**职责**: 管理面板、监控仪表板、用户界面  
**状态**: ? Pending (等待 Phase 1 完成后开始)  
**工作目录**: `docs/agents/ui-agent/`

#### 预期任务
- [ ] UI-0.1: 设计管理面板 UI
- [ ] UI-0.2: 实现账号管理界面
- [ ] UI-0.3: 实现监控仪表板
- [ ] UI-0.4: 实现日志查看器
- [ ] UI-0.5: 实现配置编辑器

#### 技术栈
- **前端**: React + TypeScript + Vite
- **UI 库**: shadcn/ui + TailwindCSS
- **状态管理**: Zustand
- **图表**: Recharts

---

## 项目里程碑

### M1: 协议分析完成 (Phase 0)
**目标日期**: 2026-03-20  
**负责人**: Protocol Agent + PM Agent  
**状态**: ? 进行中 (15%)

**交付物**:
- [ ] 完整的 API 端点文档
- [ ] 认证机制分析文档
- [ ] 至少 10 个完整请求分析
- [ ] mitmproxy 抓包数据

---

### M2: 基础功能可用 (Phase 1-2)
**目标日期**: 2026-03-25  
**负责人**: Auth Agent + API Agent  
**状态**: ? 待开始

**交付物**:
- [ ] Token 自动刷新机制
- [ ] /v1/chat/completions 完整实现
- [ ] 流式响应 (SSE) 支持
- [ ] 至少 3 个模型可用

---

### M3: 生产就绪 (Phase 3-4)
**目标日期**: 2026-04-01  
**负责人**: Queue Agent + Test Agent  
**状态**: ? 待开始

**交付物**:
- [ ] 多账号负载均衡
- [ ] 排队系统完善
- [ ] 管理面板上线
- [ ] 完整的测试覆盖

---

## 依赖关系图

```
Phase 0 (协议分析)
  ├─ Protocol Agent (主导)
  └─ Auth Agent (协助)
        ↓
Phase 1 (认证模块)
  ├─ Auth Agent (主导)
  ├─ Protocol Agent (协助)
  └─ Test Agent (验证)
        ↓
Phase 2 (聊天 API)
  ├─ API Agent (主导)
  ├─ Auth Agent (Token 支持)
  └─ Test Agent (验证)
        ↓
Phase 3 (排队系统)
  ├─ Queue Agent (主导)
  ├─ API Agent (集成)
  └─ Test Agent (压力测试)
        ↓
Phase 4 (负载均衡)
  ├─ Auth Agent (多账号)
  ├─ Queue Agent (调度)
  └─ Test Agent (验证)
        ↓
Phase 5 (管理面板)
  └─ UI Agent (主导)
```

---

## 通信机制

### Event Bus 事件类型
```go
// 任务事件
"task.started"      // 任务开始
"task.completed"    // 任务完成
"task.failed"       // 任务失败
"task.progress"     // 进度更新

// Agent 事件
"agent.ready"       // Agent 就绪
"agent.blocked"     // Agent 被阻塞
"agent.idle"        // Agent 空闲

// 项目事件
"milestone.reached" // 里程碑达成
"risk.identified"   // 风险识别
"decision.made"     // 决策制定
```

### 上下文作用域
```go
// Global: 所有 Agent 共享
- 项目配置
- API 端点列表
- 模型配置
- 认证信息

// Agent: 单个 Agent 私有
- Agent 状态
- 工作进度
- 临时数据

// Task: 任务特定
- 任务参数
- 执行结果
- 错误信息
```

---

## 进度追踪

### 总体进度
```
Phase 0: ████████???????? 15%
Phase 1: ????????????????  0%
Phase 2: ████???????????? 20% (基础框架)
Phase 3: ████???????????? 20% (基础框架)
Phase 4: ????????????????  0%
Phase 5: ????????????????  0%
```

### Agent 活跃度
| Agent | 任务数 | 完成 | 进行中 | 待开始 | 活跃度 |
|-------|--------|------|--------|--------|--------|
| PM | 5 | 3 | 1 | 1 | ? |
| Protocol | 7 | 4 | 0 | 3 | ? |
| Auth | 6 | 3 | 0 | 3 | ? |
| API | 6 | 2 | 0 | 4 | ? |
| Queue | 6 | 3 | 0 | 3 | ? |
| Test | 6 | 1 | 0 | 5 | ? |
| UI | 5 | 0 | 0 | 5 | ? |

---

## 风险管理

| 风险 | 概率 | 影响 | 状态 | 缓解措施 |
|------|------|------|------|----------|
| Trae CN 协议变更 | ? 中 | ? 高 | 监控 | 持续监控版本更新 |
| Token 失效机制复杂 | ? 中 | ? 中 | 已识别 | 实现多种刷新策略 |
| 排队时间过长 | ? 高 | ? 中 | 处理中 | 多账号负载均衡 |
| 法律合规风险 | ? 低 | ? 高 | 已识别 | 仅限个人学习使用 |
| Electron 源码加密 | ? 低 | ? 中 | 已排除 | 确认无 asar 打包 |

---

## 下一步行动

### 本周 (2026-03-15 ~ 2026-03-21)
1. **Protocol Agent**: 深度分析 extension.js
2. **Auth Agent**: 实现 Token 自动刷新
3. **API Agent**: 完善流式响应
4. **Queue Agent**: 实现 MLFQ 调度
5. **Test Agent**: 编写 API 端点测试
6. **PM Agent**: 设置进度报告机制

### 下周 (2026-03-22 ~ 2026-03-28)
1. 完成 Phase 0 协议分析
2. 开始 Phase 1 认证模块
3. 开始 Phase 2 聊天 API
4. UI Agent 开始设计

---

**最后更新**: 2026-03-15  
**下次同步**: 2026-03-16 09:00  
**项目状态**: ? On Track  
**风险等级**: ? Low
