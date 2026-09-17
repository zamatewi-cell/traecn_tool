# Auth-Agent 快速参考

**最后更新**: 2026-03-15 18:20  
**状态**: ? 准备就绪，等待依赖

---

## ? 提供的服务

### 1. Token 管理

```go
// 获取 Token (自动刷新)
token, err := authManager.GetToken(accountID string) (string, error)

// 刷新 Token (手动)
err := authManager.RefreshToken(accountID string) error

// 验证 Token
valid := authManager.ValidateToken(token string) bool

// 获取 Token 信息
info := authManager.GetTokenInfo(accountID string) *TokenInfo
```

### 2. 账号池管理

```go
// 获取下一个可用账号
account := pool.NextAccount() *Account

// 获取所有可用账号
accounts := pool.GetAvailableAccounts() []*Account

// 标记账号使用
pool.MarkUsed(accountID string)

// 获取账号统计
stats := pool.GetStats() *AccountStats
```

### 3. 认证中间件

```go
// Gin 中间件
router.Use(authMiddleware())

// 自动注入 Token
// 验证 API Key
// 处理认证失败
```

---

## ? 配置示例

```json
{
  "auth": {
    "token_manager": {
      "refresh_threshold": 300,        // 刷新阈值 (秒)
      "retry_count": 3,                // 重试次数
      "retry_interval": 1,             // 重试间隔 (秒)
      "storage_type": "memory",        // 存储类型：memory/file
      "storage_path": "tokens.json",   // 文件存储路径
      "encrypt_storage": true,         // 加密存储
      "enable_metrics": true,          // 启用指标
      "alert_on_failure": true         // 失败告警
    },
    "account_pool": {
      "rotation_strategy": "round_robin", // 轮换策略
      "health_check_interval": 300,       // 健康检查间隔 (秒)
      "max_concurrent_requests": 10,      // 单账号最大并发
      "rate_limit_per_minute": 60         // 单账号限流
    }
  }
}
```

---

## ? 监控指标

### Token 指标

```go
type TokenMetrics struct {
    TotalTokens     int           // Token 总数
    ActiveTokens    int           // 活跃 Token 数
    ExpiringTokens  int           // 即将过期 Token 数
    ExpiredTokens   int           // 已过期 Token 数
    RefreshCount    int64         // 累计刷新次数
    RefreshFailures int64         // 刷新失败次数
    AvgRefreshTime  time.Duration // 平均刷新耗时
}
```

### 账号池指标

```go
type AccountPoolMetrics struct {
    TotalAccounts     int     // 账号总数
    ActiveAccounts    int     // 活跃账号数
    DisabledAccounts  int     // 禁用账号数
    TotalRequests     int64   // 总请求数
    AvgWaitTime       float64 // 平均等待时间
    QueueLength       int     // 当前队列长度
}
```

---

## ?? 错误处理

### 常见错误

```go
// Token 相关
ErrTokenNotFound      = errors.New("token not found")
ErrTokenExpired       = errors.New("token expired")
ErrTokenRefreshFailed = errors.New("token refresh failed")
ErrTokenInvalid       = errors.New("token invalid")

// 账号池相关
ErrNoAvailableAccount = errors.New("no available account")
ErrAccountDisabled    = errors.New("account disabled")
ErrAccountRateLimited = errors.New("account rate limited")
```

### 错误处理最佳实践

```go
token, err := authManager.GetToken("account_1")
if err != nil {
    switch err {
    case ErrTokenNotFound:
        // 处理：生成新 Token
    case ErrTokenRefreshFailed:
        // 处理：切换账号或降级
    case ErrTokenExpired:
        // 处理：强制刷新
    default:
        // 处理：记录日志，返回错误
    }
}
```

---

## ? 安全建议

### 1. Token 存储

- ? 使用加密存储 (AES-256)
- ? 文件权限设置为 0600
- ? 不在日志中打印完整 Token
- ? 不要硬编码 Token

### 2. Token 刷新

- ? 实现重试机制
- ? 设置合理的刷新阈值
- ? 监控刷新失败
- ? 避免频繁刷新触发风控

### 3. 账号管理

- ? 定期健康检查
- ? 实现账号轮换
- ? 限制单账号并发
- ? 不要过度使用单一账号

---

## ? 使用示例

### 基本使用

```go
package main

import (
    "log"
    "time"
    "your-module/internal/auth"
)

func main() {
    // 1. 创建配置
    config := auth.TokenConfig{
        RefreshThreshold: 5 * time.Minute,
        RetryCount:       3,
        RetryInterval:    1 * time.Second,
        StorageType:      "memory",
        EnableMetrics:    true,
        AlertOnFailure:   true,
    }
    
    // 2. 创建 Token 管理器
    generator := &TraeTokenGenerator{} // 实现 TokenGenerator 接口
    manager := auth.NewTokenManager(config, generator)
    
    // 3. 启动自动刷新
    manager.StartAutoRefresh()
    
    // 4. 获取 Token
    token, err := manager.GetToken("account_1")
    if err != nil {
        log.Fatalf("Get token failed: %v", err)
    }
    
    // 5. 使用 Token
    log.Printf("Token: %s...", token[:20])
}
```

### 多账号轮询

```go
// 创建账号池
pool := auth.NewAccountPool(config)

// 添加账号
pool.AddAccount(auth.Account{
    ID:       "account_1",
    Token:    "token_1",
    Weight:   10,
    Enabled:  true,
})

// 获取下一个账号
account := pool.NextAccount()
if account != nil {
    token, _ := manager.GetToken(account.ID)
    // 使用 token...
    pool.MarkUsed(account.ID)
}
```

### Gin 中间件

```go
r := gin.Default()

// 使用认证中间件
r.Use(authMiddleware(manager))

r.POST("/v1/chat/completions", func(c *gin.Context) {
    // 已自动验证 API Key 和注入 Token
    // 直接处理业务逻辑
})
```

---

## ? 与其他 Agent 的接口

### 依赖 Protocol-Agent

```go
// 需要 Protocol-Agent 提供
type TokenGenerator interface {
    Generate(params auth.GenerateParams) (string, error)
    Refresh(oldToken string) (string, error)
    Validate(token string) bool
    GetExpiresIn(token string) int
}
```

### 提供给 API-Agent

```go
// API-Agent 使用
token, err := manager.GetToken(accountID)
// 用于请求 Trae CN API
```

### 提供给 Queue-Agent

```go
// Queue-Agent 使用
account := pool.NextAccount()
// 用于请求队列调度
```

---

## ? 联系方式

- **工作区**: [`docs/agents/auth-agent/`](auth-agent/)
- **设计文档**: [`TOKEN-MANAGER-DESIGN.md`](auth-agent/TOKEN-MANAGER-DESIGN.md)
- **任务记录**: [`TASK-RECORD.md`](auth-agent/TASK-RECORD.md)
- **事件日志**: [`docs/agents/event-log.md`](../event-log.md)

---

**Auth-Agent 准备就绪，随时响应！** ?
