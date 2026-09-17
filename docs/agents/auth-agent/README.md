# Auth-Agent 工作区

## 角色定义

Auth-Agent 负责整个系统的认证和 Token 管理。

## 当前状态

? **TASK-101 Token Manager 已完成** (2026-03-17 19:30)

## 技术调研

? 完成
- JWT Token 结构分析（RS256 签名）
- Token 有效期验证（Access Token 14 天，Refresh Token 6 个月）
- Token 刷新流程（POST /api/auth/refresh_token）
- Token 存储位置（storage.json）

## 核心组件

### 1. Token Manager (`internal/auth/manager.go`)

**功能**:
- Token 的集中管理（内存 + 文件存储）
- 自动刷新（剩余<3 天自动刷新）
- 线程安全（sync.RWMutex）
- JWT 解析和验证

**核心方法**:
```go
// 获取 Token（自动刷新）
func (tm *TokenManager) GetToken(ctx context.Context, accountID string) (*TokenInfo, error)

// 手动刷新 Token
func (tm *TokenManager) RefreshToken(ctx context.Context, accountID string) error

// 检查 Token 状态
func (tm *TokenManager) CheckTokenStatus(token *TokenInfo) *TokenStatus

// 启动/停止自动刷新
func (tm *TokenManager) Start(ctx context.Context) error
func (tm *TokenManager) Stop() error
```

### 2. Token Storage (`internal/auth/storage.go`)

**接口**:
```go
type TokenStorage interface {
    Save(accountID string, token *TokenInfo) error
    Load(accountID string) (*TokenInfo, error)
    LoadAll() (map[string]*TokenInfo, error)
    Delete(accountID string) error
    Exists(accountID string) bool
}
```

**实现**:
- `FileTokenStorage`: 文件存储（持久化）
- `MemoryTokenStorage`: 内存存储（测试用）

### 3. Token Refresher (`internal/auth/refresher.go`)

**功能**:
- 调用 Trae CN API 刷新 Token
- HTTP 客户端管理
- 请求头设置

**API**:
```
POST /api/auth/refresh_token
Body: {"refresh_token": "xxx"}
Response: {"data": {"token": "new", "refresh_token": "new", ...}}
```

## 数据结构

### TokenInfo
```go
type TokenInfo struct {
    AccessToken       string    // Access Token
    RefreshToken      string    // Refresh Token
    ExpiresAt         time.Time // Access Token 过期时间
    RefreshExpiresAt  time.Time // Refresh Token 过期时间
    LastRefreshedAt   time.Time // 最后刷新时间
    AccountID         string    // 账户 ID
    TenantID          string    // 租户 ID
    UserID            string    // 用户 ID
    Source            string    // Token 来源
    SourceID          string    // 来源 ID
}
```

### TokenStatus
```go
type TokenStatus struct {
    IsValid        bool          // 是否有效
    IsExpired      bool          // 是否过期
    NeedsRefresh   bool          // 是否需要刷新
    TimeToExpiry   time.Duration // 距离过期时间
    CanRefresh     bool          // 是否可以刷新
    RefreshExpired bool          // Refresh Token 是否过期
}
```

## 测试

**运行测试**:
```bash
go test ./internal/auth/... -v
```

**测试结果**:
```
=== RUN   TestTokenManager_Basic
--- PASS: TestTokenManager_Basic (0.00s)
...
PASS
ok      github.com/zamatewi-cell/traecn_tool/internal/auth      2.484s
```

? **16 个测试用例全部通过**

## 任务列表

- [x] **TASK-101**: Token Manager 实现
  - [x] Token 解析和验证
  - [x] Token 刷新逻辑
  - [x] 内存存储
  - [x] 文件持久化
  - [x] 自动刷新后台任务
  - [x] 单元测试

- [ ] **TASK-020**: Token 管理器（原始任务）
  - [x] 核心实现完成
  - [ ] 集成到主程序

- [ ] **TASK-021**: Account Pool 账户池
  - [ ] 多账户管理
  - [ ] 负载均衡
  - [ ] 健康检查

- [ ] **TASK-022**: Auth Middleware 认证中间件
  - [ ] Gin 中间件
  - [ ] Token 验证
  - [ ] 权限检查

## 依赖关系

**需要**:
- Protocol-Agent 提供 Token 生成算法 ? (TASK-001 已交付)

**被依赖**:
- API-Agent: 使用 Token Manager 获取 Token
- Queue-Agent: 使用 Token Manager 管理多账户

## 快速开始

**其他 Agent 集成示例**:

```go
import "github.com/zamatewi-cell/traecn_tool/internal/auth"

// 1. 创建 Token Manager
config := auth.DefaultTokenConfig()
tm := auth.NewTokenManager(config)

// 2. 创建 Token Refresher
refresher := auth.NewTokenRefresher("https://proxy.example.com", headers)
tm.SetRefresher(refresher)

// 3. 创建文件存储
storage, _ := auth.NewFileTokenStorage("tokens.json")
tm.SetStorage(storage)

// 4. 启动自动刷新
ctx := context.Background()
tm.Start(ctx)

// 5. 获取 Token（自动刷新）
token, err := tm.GetToken(ctx, "account_1")
if err != nil {
    // 处理错误
}

// 使用 token.AccessToken
```

## 文档

- [TASK-001-DELIVERY.md](../protocol-agent/TASK-001-DELIVERY.md) - Protocol-Agent 交付
- [TOKEN-MANAGER-DESIGN.md](TOKEN-MANAGER-DESIGN.md) - 技术设计文档
- [QUICK-REFERENCE.md](QUICK-REFERENCE.md) - 快速参考

## 事件日志

查看最新状态更新：[event-log.md](../event-log.md)
