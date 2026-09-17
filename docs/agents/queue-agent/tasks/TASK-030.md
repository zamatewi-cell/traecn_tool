# Task: 实现请求队列管理

**ID**: TASK-030  
**优先级**: Normal  
**分配给**: @Queue-Agent  
**依赖**: TASK-010 (API 层完成)  
**截止日期**: 2026-03-20  
**状态**: ? Pending  

---

## 描述

实现请求队列管理系统，包括优先级队列、调度算法、并发控制和公平性保证。

---

## 验收标准

- [ ] 多级优先级队列实现
- [ ] 请求调度算法（MLFQ）
- [ ] 并发控制（信号量/通道）
- [ ] 公平性保证（防饥饿）
- [ ] 队列状态监控

---

## 技术设计

### 队列结构

```go
type RequestQueue struct {
    mu       sync.Mutex
    queues   map[Priority]*deque  // 按优先级分队列
    maxLen   int
    workers  int
    semaphore chan struct{}       // 并发控制
}

type Priority int

const (
    PriorityVIP    Priority = 100  // VIP 用户
    PriorityHigh   Priority = 10   // 高优先级
    PriorityNormal Priority = 5    // 普通
    PriorityLow    Priority = 1    // 低优先级
)

type QueuedRequest struct {
    ID        string
    Priority  Priority
    AccountID string
    Model     string
    Request   *http.Request
    Response  chan<- Response
    AddedAt   time.Time
}
```

### 调度算法：多级反馈队列 (MLFQ)

```
Priority 100: [VIP 请求] → 立即处理（抢占式）
Priority 10:  [高优请求] → VIP 空闲时处理
Priority 5:   [普通请求] → 按 FIFO 处理
Priority 1:   [低优请求] → 系统空闲时处理
```

### 并发控制

```go
func (q *RequestQueue) StartWorkers() {
    for i := 0; i < q.workers; i++ {
        go q.worker()
    }
}

func (q *RequestQueue) worker() {
    for req := range q.dequeue() {
        q.semaphore <- struct{}{}  // 获取信号量
        q.process(req)
        <-q.semaphore              // 释放信号量
    }
}
```

---

## 代码结构

```
internal/queue/
├── queue.go          # 现有 Monitor（待扩展）
├── scheduler.go      # 调度器实现
├── priority.go       # 优先级定义
└── monitor.go        # 队列监控
```

---

## 当前状态

**已有实现**:
- ? `internal/queue/queue.go` - Monitor 基础
- ? 队列状态跟踪

**待完成**:
- ? 优先级队列实现
- ? 调度器
- ? 并发控制
- ? 排队状态 SSE 推送

---

## 依赖关系

**上游**: 
- TASK-010 (API-Agent: 请求入口)
- TASK-020 (Auth-Agent: 账号状态)

**下游**:
- TASK-041 (Test-Agent: 并发测试)

---

## 交付物

1. **队列实现** (`internal/queue/scheduler.go`)
2. **调度策略** (`internal/queue/policy.go`)
3. **监控接口** (`GET /v1/queue/status`)
4. **配置示例** (并发数、队列长度限制)

---

**创建时间**: 2026-03-15  
**验收人**: PM-Agent
