# trae-proxy 系统架构设计

**版本**: v0.1.0  
**日期**: 2026-03-15  
**作者**: PM-Agent  
**状态**: Draft

---

## 系统概述

**trae-proxy** 是一个反向代理工具，将 Trae CN 的 AI 模型以 OpenAI 兼容 API 形式暴露。

---

## 架构目标

### 功能性需求

1. **OpenAI 兼容**: 支持标准 OpenAI API 格式
2. **多模型支持**: 支持 14 个 Trae CN 模型
3. **流式响应**: 支持 SSE 流式输出
4. **账号管理**: 多账号池和轮换
5. **请求队列**: 优先级调度和限流

---

### 非功能性需求

1. **性能**: P99 延迟 < 500ms
2. **并发**: 支持 100+ QPS
3. **稳定性**: 7x24 小时无故障
4. **可扩展**: 易于添加新模型和功能
5. **易部署**: 单一二进制文件

---

## 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                     客户端层                              │
│  (ChatBox, NextChat, OpenAI SDK, LangChain, etc.)       │
└─────────────────────────────────────────────────────────┘
                          │
                          │ OpenAI API
                          ▼
┌─────────────────────────────────────────────────────────┐
│                   API 层 (API Layer)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  /v1/    │  │  /v1/    │  │  /v1/    │              │
│  │  models  │  │  chat/   │  │complet-  │              │
│  │          │  │complet-  │  │  ions    │              │
│  │          │  │  ions    │  │          │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
                          │
                          │ 认证 + 转换
                          ▼
┌─────────────────────────────────────────────────────────┐
│                 核心服务层 (Core Services)               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │   Auth   │  │  Queue   │  │Transform-│              │
│  │ Manager  │  │ Manager  │  │   er     │              │
│  │          │  │          │  │          │              │
│  │ - Token  │  │ - 队列    │  │ - OpenAI │              │
│  │ - 刷新    │  │ - 调度    │  │   ?      │              │
│  │ - 账号池  │  │ - 限流    │  │ Trae CN  │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
                          │
                          │ Trae CN 协议
                          ▼
┌─────────────────────────────────────────────────────────┐
│                协议适配层 (Protocol Adapter)             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  HTTP    │  │   SSE    │  │  Auth    │              │
│  │  Client  │  │  Parser  │  │  Header  │              │
│  │          │  │          │  │  Builder │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
                          │
                          │ HTTPS
                          ▼
┌─────────────────────────────────────────────────────────┐
│                  Trae CN 后端服务                         │
│              (api.trae.cn, auth.trae.cn)                │
└─────────────────────────────────────────────────────────┘
```

---

## 模块设计

### 1. API 层 (api/)

**职责**: 实现 OpenAI 兼容的 HTTP API

**组件**:
```
api/
├── server.go          # HTTP 服务器
├── routes.go          # 路由定义
├── middleware/        # 中间件
│   ├── auth.go        # API Key 验证
│   ├── cors.go        # CORS
│   ├── logger.go      # 日志
│   └── recovery.go    # 错误恢复
└── handlers/          # 处理器
    ├── models.go      # GET /v1/models
    ├── chat.go        # POST /v1/chat/completions
    └── completions.go # POST /v1/completions
```

**关键接口**:
```go
type APIHandler interface {
    GetModels(w http.ResponseWriter, r *http.Request)
    CreateChatCompletion(w http.ResponseWriter, r *http.Request)
    CreateCompletion(w http.ResponseWriter, r *http.Request)
}
```

---

### 2. 认证管理器 (auth/)

**职责**: Token 管理、账号池、认证中间件

**组件**:
```
auth/
├── manager.go         # Token 管理器
├── pool.go            # 账号池
├── middleware.go      # 认证中间件
└── storage.go         # 账号存储
```

**关键类型**:
```go
type TokenManager struct {
    tokens    map[string]*TokenInfo
    mu        sync.RWMutex
    generator TokenGenerator
}

type AccountPool struct {
    accounts []Account
    mu       sync.RWMutex
    strategy RotationStrategy
}
```

**工作流程**:
```
请求 → 验证 API Key → 获取 Token → 注入请求头 → 转发
            ↓           ↓
        失败返回    自动刷新
```

---

### 3. 队列管理器 (queue/)

**职责**: 请求调度、限流、熔断

**组件**:
```
queue/
├── manager.go         # 队列管理器
├── scheduler.go       # 调度器
├── limiter.go         # 限流器
└── breaker.go         # 熔断器
```

**关键算法**:

**多级反馈队列 (MLFQ)**:
```
Priority 100: [VIP] → 立即处理
Priority 10:  [高优] → VIP 空闲时处理
Priority 5:   [普通] → FIFO
Priority 1:   [低优] → 系统空闲时处理
```

**令牌桶限流**:
```go
type TokenBucket struct {
    tokens     float64
    capacity   float64
    refillRate float64
}
```

---

### 4. 协议转换器 (transformer/)

**职责**: OpenAI ? Trae CN 协议转换

**组件**:
```
transformer/
├── request.go         # 请求转换
├── response.go        # 响应转换
└── sse.go             # SSE 流式处理
```

**转换示例**:
```go
// OpenAI 请求
{
  "model": "deepseek-v3.1-terminus",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": true
}

// ↓ 转换

// Trae CN 请求
{
  "model_id": "ds_v31",
  "conversation_id": "conv_123",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": true
}
```

---

### 5. 协议适配器 (protocol/)

**职责**: 与 Trae CN 后端通信

**组件**:
```
protocol/
├── client.go          # HTTP 客户端
├── sse.go             # SSE 解析器
└── auth.go            # 认证头构建
```

**关键功能**:
- 建立 HTTPS 连接
- 解析 SSE 流式响应
- 处理错误和重试

---

## 数据流

### 完整请求流程

```
1. 客户端发送 OpenAI 格式请求
   ↓
2. API 层接收请求
   ↓
3. Auth 中间件验证 API Key
   ↓
4. Queue 管理器分配优先级
   ↓
5. Transformer 转换请求格式
   ↓
6. Protocol 客户端发送 Trae CN
   ↓
7. 接收 Trae CN 响应 (流式)
   ↓
8. Transformer 转换响应格式
   ↓
9. API 层 SSE 推送给客户端
   ↓
10. 完成
```

---

### 流式响应处理

```
Trae CN SSE 事件流:
data: {"content": "Hello"}
data: {"content": " World"}
data: [DONE]

↓ 转换

OpenAI SSE 事件流:
data: {"id":"chatcmpl-123","choices":[{"delta":{"content":"Hello"}}]}
data: {"id":"chatcmpl-123","choices":[{"delta":{"content":" World"}}]}
data: [DONE]
```

---

## 错误处理

### 错误码映射

| Trae CN 错误 | HTTP 状态码 | OpenAI 错误 |
|-------------|-----------|------------|
| 401 Unauthorized | 401 | invalid_api_key |
| 403 Forbidden | 403 | access_denied |
| 429 Rate Limited | 429 | rate_limit_exceeded |
| 500 Internal Error | 500 | internal_error |
| 503 Service Unavailable | 503 | service_unavailable |

---

### 重试策略

```go
type RetryConfig struct {
    MaxRetries   int           // 最大重试次数
    InitialDelay time.Duration // 初始延迟
    MaxDelay     time.Duration // 最大延迟
    Multiplier   float64       // 延迟倍增器
}

// 指数退避重试
// 1s → 2s → 4s → 8s → 16s
```

---

## 配置管理

### 配置文件结构

```yaml
# config.yaml

# API 配置
api:
  port: 8080
  host: "0.0.0.0"
  
# API Key 配置
auth:
  api_keys:
    - "sk-key1"
    - "sk-key2"
  
# Trae CN 配置
trae:
  base_url: "https://api.trae.cn"
  timeout: 30s
  
# 账号池配置
pool:
  accounts:
    - id: "account1"
      token: "xxx"
      weight: 100
  
# 队列配置
queue:
  max_size: 1000
  workers: 10
  
# 限流配置
rate_limit:
  global_rps: 100
  per_user_rpm: 60
```

---

## 监控指标

### 业务指标

```go
// 请求统计
requests_total{endpoint, method, status}
request_duration_seconds{endpoint, quantile}

// 队列统计
queue_length{priority}
queue_wait_seconds{priority}

// 账号统计
pool_accounts_total
pool_accounts_active
pool_accounts_cooldown

// Token 统计
tokens_total
tokens_expiring_soon
token_refresh_total
token_refresh_failures
```

---

### 系统指标

```go
// Go 运行时
go_goroutines
go_heap_bytes
go_gc_duration_seconds

// 系统
process_cpu_seconds
process_resident_memory_bytes
```

---

## 部署架构

### 单机部署

```
┌─────────────────┐
│  trae-proxy     │
│  :8080          │
└─────────────────┘
       │
       ▼
┌─────────────────┐
│  Trae CN        │
│  api.trae.cn    │
└─────────────────┘
```

---

### 高可用部署

```
┌─────────────────┐
│   Load Balancer │
│   (Nginx/HA)    │
└─────────────────┘
       │
    ┌──┴──┐
    ▼     ▼
┌─────────┐ ┌─────────┐
│trae-    │ │trae-    │
│proxy-1  │ │proxy-2  │
└─────────┘ └─────────┘
    │           │
    └─────┬─────┘
          ▼
    ┌──────────┐
    │ Trae CN  │
    │ Backend  │
    └──────────┘
```

---

## 安全设计

### API Key 管理

- **哈希存储**: bcrypt 哈希后存储
- **权限分离**: 不同 Key 不同权限
- **审计日志**: 记录使用情况

---

### Token 安全

- **内存存储**: 运行时 Token 不落地
- **加密传输**: HTTPS 加密
- **自动刷新**: 避免 Token 过期

---

### 访问控制

```yaml
# RBAC 配置
roles:
  admin:
    - "*"  # 所有权限
  user:
    - "POST /v1/chat/completions"
    - "GET /v1/models"
  readonly:
    - "GET /v1/models"
```

---

## 扩展性设计

### 添加新模型

1. 在配置文件中添加模型映射
2. 在 Transformer 中添加转换规则
3. 更新 `/v1/models` 端点

---

### 添加新功能

**插件化设计**:
```go
type Plugin interface {
    Name() string
    Init(config Config) error
    HandleRequest(req *Request) (*Response, error)
}
```

---

## 性能优化

### 连接池

```go
type HTTPClientPool struct {
    clients map[string]*http.Client
    mu      sync.RWMutex
}

// 复用 HTTP 连接，减少握手开销
```

---

### 缓存

```go
// 模型列表缓存
type ModelCache struct {
    models   []Model
    expiresAt time.Time
    mu       sync.RWMutex
}
```

---

### 并发处理

```go
// 使用 goroutine 池处理请求
type WorkerPool struct {
    jobs   chan Job
    workers int
}
```

---

## 测试策略

### 单元测试

- 覆盖所有核心函数
- Mock 外部依赖
- 目标覆盖率 > 80%

---

### 集成测试

- 完整请求流程
- 多账号轮换
- 并发场景

---

### 性能测试

- 基准测试 (go test -bench)
- 压力测试 (wrk, ab)
- 长时间运行测试

---

## 参考资料

- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [Gin 框架文档](https://gin-gonic.com/)
- [Go 并发编程](https://go.dev/tour/concurrency/1)
- [反向代理模式](https://en.wikipedia.org/wiki/Reverse_proxy)

---

**最后更新**: 2026-03-15  
**维护者**: PM-Agent  
**下次评审**: 2026-03-20
