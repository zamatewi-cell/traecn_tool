# Queue Manager Agent 工作区

## 角色定位

**Queue Agent** - 队列管理工程师，负责请求调度和并发控制

---

## 当前状态

**状态**: ? Available (任务完成)  
**依赖**: 无  
**最后更新**: 2026-03-15

---

## 已完成任务

### ? TASK-030: 实现请求队列管理

**完成内容**:
- ? 多级优先级队列实现
- ? 请求调度算法（MLFQ）
- ? 并发控制（信号量/通道）
- ? 公平性保证（防饥饿）
- ? 队列状态监控

**实现文件**:
- `internal/queue/priority.go` - 优先级定义
- `internal/queue/scheduler.go` - MLFQ 调度器
- `internal/queue/queue.go` - 队列基础结构

### ? TASK-031: 实现限流和降级

**完成内容**:
- ? 全局限流（令牌桶算法）
- ? 每用户限流（滑动窗口）
- ? 每账号限流（防封号）
- ? 自动降级策略
- ? 熔断器模式实现

**实现文件**:
- `internal/queue/limiter.go` - 限流器和熔断器
- `internal/queue/queue.go` - 令牌桶和滑动窗口

### ? TASK-032: 实现排队状态 SSE 推送

**完成内容**:
- ? 排队位置计算
- ? 预计等待时间估算
- ? SSE 状态推送
- ? 客户端健康管理

**实现文件**:
- `internal/queue/sse.go` - SSE 推送模块

### ? TASK-033: 集成队列管理器

**完成内容**:
- ? 统一队列管理器
- ? 组件协调
- ? 健康检查
- ? 综合状态监控

**实现文件**:
- `internal/queue/manager.go` - 队列管理器
- `internal/queue/example_test.go` - 使用示例

---

## 技术实现

### 队列结构

```go
type RequestQueue struct {
    mu       sync.Mutex
    queues   map[Priority]*deque  // 按优先级分队列
    maxLen   int
    workers  int
    semaphore chan struct{}       // 并发控制
}
```

### 调度算法：多级反馈队列 (MLFQ)

```
Priority 100: [VIP 请求] → 立即处理（抢占式）
Priority 10:  [高优请求] → VIP 空闲时处理
Priority 5:   [普通请求] → 按 FIFO 处理
Priority 1:   [低优请求] → 系统空闲时处理
```

### 限流配置

```yaml
rate_limit:
  global:
    requests_per_second: 100
    burst: 150
  per_user:
    requests_per_minute: 60
  per_account:
    requests_per_minute: 30
    concurrent_requests: 3
```

### SSE 推送格式

```
data: {"object":"queue.status","position":5,"estimated_wait":"30s"}
data: {"object":"queue.status","position":3,"estimated_wait":"20s"}
data: {"object":"queue.status","position":1,"estimated_wait":"10s"}
data: {"object":"queue.status","position":0,"estimated_wait":"0s"}
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"Hello"}}]}
```

---

## 使用示例

### 基本使用

```go
// 创建管理器
config := queue.DefaultManagerConfig()
manager := queue.NewManager(config)

// 启动
manager.Start(processor)
defer manager.Stop()

// 提交请求
req := &queue.QueuedRequest{
    ID:        "req-1",
    Priority:  queue.PriorityNormal,
    AccountID: "account-1",
    Model:     "claude-3-5-sonnet",
    Response:  make(chan queue.Response, 1),
}

if err := manager.SubmitRequest(req); err != nil {
    // 处理错误
}
```

### SSE 推送

```go
// SSE 端点
http.HandleFunc("/queue/status", 
    manager.GetSSEBroker().SSEHandler("claude-3-5-sonnet"))

// 广播状态
manager.GetSSEBroker().BroadcastQueueStatus(
    "claude-3-5-sonnet",
    position,
    estimatedWait,
)
```

---

## 监控指标

### 队列统计

- `TotalLength`: 总队列长度
- `ByPriority`: 按优先级分类
- `SemaphoreAvailable`: 可用并发数
- `SemaphoreCapacity`: 总并发容量

### 熔断器状态

- `State`: CLOSED / OPEN / HALF-OPEN
- `FailureCount`: 失败次数
- `SuccessCount`: 成功次数
- `LastFailure`: 最后失败时间

### 降级状态

- `IsDegraded`: 是否降级
- `DegradationLevel`: 降级等级 (0-3)
- `CooldownRemaining`: 冷却剩余时间

---

## 下一步计划

- [ ] 持久化队列（Redis）
- [ ] 分布式队列支持
- [ ] 动态配置更新
- [ ] 详细日志记录
- [ ] Prometheus 指标导出
- 账号不可用：自动切换到备用账号
- 连续失败：触发熔断机制

---

### 限流实现

**令牌桶算法**:
```go
type TokenBucket struct {
    tokens     float64
    capacity   float64
    refillRate float64  // 每秒补充的令牌数
    lastRefill time.Time
    mu         sync.Mutex
}

func (b *TokenBucket) Allow() bool {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    now := time.Now()
    b.tokens = min(b.capacity, b.tokens+b.refillRate*now.Sub(b.lastRefill).Seconds())
    b.lastRefill = now
    
    if b.tokens >= 1 {
        b.tokens--
        return true
    }
    return false
}
```

---

### 熔断机制

**状态机**:
```
Closed → Open (失败率 > 阈值)
   ↑       ↓
   └── Half-Open (测试请求)
```

**实现**:
```go
type CircuitBreaker struct {
    state      State
    failures   int
    threshold  int
    timeout    time.Duration
    lastFail   time.Time
}

func (cb *CircuitBreaker) Allow() bool {
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        if time.Since(cb.lastFail) > cb.timeout {
            cb.state = StateHalfOpen
            return true
        }
        return false
    case StateHalfOpen:
        return true // 允许一个请求测试
    }
    return false
}
```

---

## 代码结构

```
internal/
├── queue/
│   ├── manager.go         # 队列管理器
│   ├── scheduler.go       # 调度器
│   ├── limiter.go         # 限流器
│   └── breaker.go         # 熔断器
├── models/
│   └── queue.go           # 队列模型
└── config/
    └── queue.go           # 队列配置
```

---

## 监控指标

### 队列相关

- 队列长度 (按优先级)
- 平均等待时间
- 最大等待时间
- 请求丢弃数

### 限流相关

- 触发限流次数
- 当前令牌数
- 请求拒绝率

### 熔断相关

- 熔断状态
- 熔断次数
- 恢复时间

---

## 依赖关系

### 需要 API-Agent 配合

1. **请求入口**: 队列集成到 API 处理流程
2. **状态推送**: SSE 通道复用

### 需要 Auth-Agent 配合

1. **账号状态**: 获取账号健康状态
2. **配额信息**: 获取账号剩余配额

---

## 测试计划

### 功能测试

- [ ] 优先级调度测试
- [ ] 限流触发测试
- [ ] 熔断机制测试

### 性能测试

- [ ] 高并发测试 (1000+ QPS)
- [ ] 长队列测试 (10000+ 请求)
- [ ] 内存泄漏测试

### 稳定性测试

- [ ] 7x24 小时运行
- [ ] 故障恢复测试
- [ ] 边界条件测试

---

## 工作日志

### (待开始)

---

## 联系信息

- **工作目录**: `docs/agents/queue-agent/`
- **状态**: ? Available
- **依赖**: 无

---

**最后更新**: 2026-03-15 20:30  
**下次更新**: 任务分配后
