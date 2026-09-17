# Task: 实现限流和降级

**ID**: TASK-031  
**优先级**: Normal  
**分配给**: @Queue-Agent  
**依赖**: TASK-030  
**截止日期**: 2026-03-21  
**状态**: ? Pending  

---

## 描述

实现多层限流机制和自动降级策略，保护系统免受过载影响。

---

## 验收标准

- [ ] 全局限流（令牌桶算法）
- [ ] 每用户限流（滑动窗口）
- [ ] 每账号限流（防封号）
- [ ] 自动降级策略
- [ ] 排队状态透传（SSE）

---

## 限流配置

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

### 令牌桶实现

```go
type TokenBucket struct {
    mu         sync.Mutex
    tokens     float64
    capacity   float64
    refillRate float64  // tokens/second
    lastRefill time.Time
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    tb.tokens = min(tb.capacity, tb.tokens+elapsed*tb.refillRate)
    tb.lastRefill = now
    
    if tb.tokens >= 1 {
        tb.tokens--
        return true
    }
    return false
}
```

### 降级策略

```go
type DegradationPolicy struct {
    QueueThreshold    int           // 队列长度阈值
    ErrorThreshold    float64       // 错误率阈值
    CooldownDuration  time.Duration // 降级冷却时间
}

func (dp *DegradationPolicy) ShouldDegrade(stats *QueueStats) bool {
    return stats.QueueLength > dp.QueueThreshold ||
           stats.ErrorRate > dp.ErrorThreshold
}
```

---

## 排队状态 SSE 推送

```
data: {"object":"queue.status","position":5,"estimated_wait":"30s"}
data: {"object":"queue.status","position":3,"estimated_wait":"20s"}
data: {"object":"queue.status","position":1,"estimated_wait":"10s"}
data: {"object":"queue.status","position":0,"estimated_wait":"0s"}
data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"Hello"}}]}
```

---

## 依赖关系

**上游**: 
- TASK-030 (队列管理)

**下游**:
- TASK-042 (Test-Agent: 压力测试)

---

## 交付物

1. **限流器实现** (`internal/queue/ratelimit.go`)
2. **降级策略** (`internal/queue/degradation.go`)
3. **SSE 状态推送** (`internal/queue/sse.go`)
4. **配置模板** (限流参数配置)

---

**创建时间**: 2026-03-15  
**验收人**: PM-Agent
