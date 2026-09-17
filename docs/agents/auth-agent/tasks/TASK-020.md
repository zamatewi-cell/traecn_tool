# Task: 实现 Token 管理器

**ID**: TASK-020  
**优先级**: High  
**分配给**: @Auth-Agent  
**依赖**: TASK-001 (认证流程逆向完成)  
**截止日期**: 2026-03-18  
**状态**: ? Pending  

---

## 描述

实现多账号 Token 管理器，支持 Token 生成、刷新、存储和自动轮换。

---

## 验收标准

- [ ] Token 生成逻辑实现
- [ ] Token 刷新机制（主动 + 被动）
- [ ] Token 加密存储
- [ ] 过期检测和自动刷新
- [ ] 多账号轮询负载均衡
- [ ] 单元测试覆盖

---

## 技术设计

### 数据结构

```go
type TokenManager struct {
    mu       sync.RWMutex
    accounts map[string]*Account  // 账号 ID -> Account
    index    int                  // 轮询索引
}

type Account struct {
    ID          string
    Name        string
    Token       string
    RefreshToken string
    ExpiresAt   time.Time
    Status      AccountStatus
    Stats       AccountStats
}

type AccountStatus int

const (
    StatusActive AccountStatus = iota
    StatusInUse
    StatusCooldown
    StatusDisabled
    StatusRateLimited
)

type AccountStats struct {
    TotalRequests   int
    SuccessCount    int
    ErrorCount      int
    LastUsedAt      time.Time
    CooldownUntil   time.Time
}
```

### Token 刷新策略

```go
// 主动刷新：过期前 5 分钟
func (tm *TokenManager) shouldRefresh(account *Account) bool {
    return time.Now().After(account.ExpiresAt.Add(-5 * time.Minute))
}

// 被动刷新：使用过期 Token 请求失败后
func (tm *TokenManager) refreshToken(accountID string) error {
    // 调用刷新接口获取新 Token
}
```

---

## 代码结构

```
internal/auth/
├── token.go           # 现有 TokenProvider（待重构）
├── manager.go         # TokenManager 实现
├── account.go         # Account 模型
├── storage.go         # 加密存储
└── middleware.go      # 认证中间件
```

---

## 当前状态

**已有实现**:
- ? `internal/auth/token.go` - TokenProvider 基础
- ? 多账号轮询机制
- ? Token 存储加载

**待完成**:
- ? Token 自动刷新逻辑
- ? 账号健康管理
- ? 加密存储实现
- ? 认证中间件

---

## 依赖关系

**上游**: 
- TASK-001 (Protocol-Agent: Token 生成算法)

**下游**:
- TASK-010 (API-Agent: API 层使用 Token)
- TASK-030 (Queue-Agent: 基于账号状态的调度)

---

## 交付物

1. **Token 管理器实现** (`internal/auth/manager.go`)
2. **加密存储** (`internal/auth/storage.go`)
3. **认证中间件** (`internal/auth/middleware.go`)
4. **配置示例** (多账号配置模板)

---

**创建时间**: 2026-03-15  
**验收人**: PM-Agent
