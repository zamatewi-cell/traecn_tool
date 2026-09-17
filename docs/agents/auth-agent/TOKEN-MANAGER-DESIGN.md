# Token 管理器技术设计文档

**版本**: 1.0  
**日期**: 2026-03-15  
**作者**: Auth-Agent  
**状态**: ? 设计完成，等待实现

---

## 1. 概述

Token 管理器负责 Trae CN 认证 Token 的全生命周期管理，包括生成、存储、刷新、验证等功能。

### 1.1 核心职责

- Token 生成和刷新
- Token 存储和管理
- 过期检测和自动刷新
- 多账号 Token 管理
- Token 使用统计

### 1.2 设计目标

- **高可用**: 99.9% Token 可用性
- **低延迟**: Token 获取 < 1ms
- **自动化**: 自动刷新，无需人工干预
- **安全性**: Token 加密存储
- **可扩展**: 支持多账号管理

---

## 2. 架构设计

### 2.1 组件图

```
┌─────────────────────────────────────────────────┐
│              TokenManager                        │
├─────────────────────────────────────────────────┤
│  + tokens: map[string]*TokenInfo                │
│  + mu: sync.RWMutex                             │
│  + generator: TokenGenerator                    │
│  + config: TokenConfig                          │
├─────────────────────────────────────────────────┤
│  + GetToken(accountID) string                   │
│  + RefreshToken(accountID) error                │
│  + ValidateToken(token) bool                    │
│  + GetTokenInfo(accountID) *TokenInfo           │
│  + StartAutoRefresh()                           │
│  + StopAutoRefresh()                            │
└─────────────────────────────────────────────────┘
           │
           │ 依赖
           ▼
┌─────────────────────────────────────────────────┐
│            TokenGenerator (接口)                 │
├─────────────────────────────────────────────────┤
│  + Generate(params) (string, error)             │
│  + Refresh(oldToken) (string, error)            │
│  + Validate(token) bool                         │
└─────────────────────────────────────────────────┘
```

### 2.2 数据流

```
请求 Token
    │
    ▼
检查缓存 ──── 存在且有效 ────> 返回 Token
    │
    └── 不存在或过期
           │
           ▼
      刷新 Token ────> 成功 ────> 更新缓存 ────> 返回
           │
           └── 失败 ────> 重试 (最多 3 次) ────> 失败告警
```

---

## 3. 数据结构

### 3.1 TokenInfo

```go
type TokenInfo struct {
    Token       string      // Token 字符串
    ExpiresAt   time.Time   // 过期时间
    RefreshAt   time.Time   // 建议刷新时间 (过期前 5 分钟)
    AccountID   string      // 账号 ID
    Status      TokenStatus // Token 状态
    CreatedAt   time.Time   // 创建时间
    UpdatedAt   time.Time   // 最后更新时间
    RefreshCount int        // 刷新次数
    Metadata    map[string]interface{} // 扩展元数据
}
```

### 3.2 TokenStatus

```go
type TokenStatus string

const (
    TokenActive     TokenStatus = "active"      // 活跃可用
    TokenExpiring   TokenStatus = "expiring"    // 即将过期 (<5 分钟)
    TokenExpired    TokenStatus = "expired"     // 已过期
    TokenRefreshing TokenStatus = "refreshing"  // 刷新中
    TokenInvalid    TokenStatus = "invalid"     // 无效 (刷新失败)
)
```

### 3.3 TokenConfig

```go
type TokenConfig struct {
    // 刷新策略
    RefreshThreshold time.Duration // 刷新阈值 (默认 5 分钟)
    RetryCount       int           // 重试次数 (默认 3)
    RetryInterval    time.Duration // 重试间隔 (默认 1 秒)
    
    // 存储策略
    StorageType      string        // 存储类型：memory/file
    StoragePath      string        // 文件存储路径
    EncryptStorage   bool          // 是否加密存储
    
    // 监控策略
    EnableMetrics    bool          // 启用指标收集
    AlertOnFailure   bool          // 刷新失败告警
}
```

### 3.4 TokenGenerator 接口

```go
type TokenGenerator interface {
    // Generate 生成新 Token
    Generate(params GenerateParams) (string, error)
    
    // Refresh 刷新 Token
    Refresh(oldToken string) (string, error)
    
    // Validate 验证 Token 有效性
    Validate(token string) bool
    
    // GetExpiresIn 获取 Token 有效期 (秒)
    GetExpiresIn(token string) int
}

type GenerateParams struct {
    DeviceID  string
    Timestamp int64
    Nonce     string
    AccountID string
}
```

---

## 4. 核心算法

### 4.1 Token 获取流程

```go
func (tm *TokenManager) GetToken(accountID string) (string, error) {
    // 1. 读取缓存
    tm.mu.RLock()
    info, exists := tm.tokens[accountID]
    tm.mu.RUnlock()
    
    if !exists {
        // 2. 不存在，生成新 Token
        return tm.generateToken(accountID)
    }
    
    // 3. 检查状态
    now := time.Now()
    
    if info.Status == TokenExpired || info.Status == TokenInvalid {
        // 4. 已过期或无效，刷新
        return tm.refreshToken(accountID)
    }
    
    if now.After(info.ExpiresAt) {
        // 5. 超过过期时间，刷新
        return tm.refreshToken(accountID)
    }
    
    if now.After(info.RefreshAt) && info.Status == TokenActive {
        // 6. 到达刷新时间，异步刷新
        go tm.refreshTokenAsync(accountID)
        return info.Token, nil
    }
    
    // 7. Token 有效，返回
    return info.Token, nil
}
```

### 4.2 Token 刷新流程

```go
func (tm *TokenManager) refreshToken(accountID string) (string, error) {
    // 1. 标记为刷新中
    tm.mu.Lock()
    if info, exists := tm.tokens[accountID]; exists {
        info.Status = TokenRefreshing
    }
    tm.mu.Unlock()
    
    // 2. 执行刷新 (带重试)
    var newToken string
    var err error
    
    for i := 0; i < tm.config.RetryCount; i++ {
        newToken, err = tm.generator.Refresh(accountID)
        if err == nil {
            break
        }
        time.Sleep(tm.config.RetryInterval)
    }
    
    // 3. 更新 Token 信息
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    if err != nil {
        // 刷新失败，标记为无效
        if info, exists := tm.tokens[accountID]; exists {
            info.Status = TokenInvalid
            info.UpdatedAt = time.Now()
        }
        
        // 告警
        if tm.config.AlertOnFailure {
            tm.alertTokenRefreshFailed(accountID, err)
        }
        
        return "", fmt.Errorf("refresh token failed: %w", err)
    }
    
    // 4. 刷新成功，更新信息
    expiresIn := tm.generator.GetExpiresIn(newToken)
    now := time.Now()
    
    if info, exists := tm.tokens[accountID]; exists {
        info.Token = newToken
        info.ExpiresAt = now.Add(time.Duration(expiresIn) * time.Second)
        info.RefreshAt = now.Add(time.Duration(expiresIn-tm.config.RefreshThreshold) * time.Second)
        info.Status = TokenActive
        info.UpdatedAt = now
        info.RefreshCount++
    } else {
        // 新建 TokenInfo
        tm.tokens[accountID] = &TokenInfo{
            Token:       newToken,
            ExpiresAt:   now.Add(time.Duration(expiresIn) * time.Second),
            RefreshAt:   now.Add(time.Duration(expiresIn-tm.config.RefreshThreshold) * time.Second),
            AccountID:   accountID,
            Status:      TokenActive,
            CreatedAt:   now,
            UpdatedAt:   now,
            RefreshCount: 1,
        }
    }
    
    // 5. 持久化 (如果启用)
    if tm.config.StorageType == "file" {
        tm.persistTokens()
    }
    
    return newToken, nil
}
```

### 4.3 自动刷新后台任务

```go
func (tm *TokenManager) StartAutoRefresh() {
    ticker := time.NewTicker(1 * time.Minute)
    go func() {
        for range ticker.C {
            tm.checkAndRefresh()
        }
    }()
}

func (tm *TokenManager) checkAndRefresh() {
    tm.mu.RLock()
    defer tm.mu.RUnlock()
    
    now := time.Now()
    
    for accountID, info := range tm.tokens {
        // 检查是否需要刷新
        if info.Status == TokenActive && now.After(info.RefreshAt) {
            // 异步刷新
            go func(id string) {
                _, err := tm.refreshToken(id)
                if err != nil {
                    log.Printf("Auto refresh failed for %s: %v", id, err)
                }
            }(accountID)
        }
    }
}
```

---

## 5. 存储方案

### 5.1 内存存储

**优点**:
- 快速访问 (< 1ms)
- 无需序列化
- 并发安全

**缺点**:
- 重启丢失
- 内存占用

**实现**:
```go
type MemoryStorage struct {
    tokens map[string]*TokenInfo
    mu     sync.RWMutex
}
```

### 5.2 文件存储 (可选)

**优点**:
- 持久化
- 重启恢复

**缺点**:
- IO 延迟
- 需要序列化

**实现**:
```go
type FileStorage struct {
    filePath string
    encrypt  bool
    key      []byte // AES 密钥
}

func (fs *FileStorage) Save(tokens map[string]*TokenInfo) error {
    // 1. 序列化
    data, _ := json.Marshal(tokens)
    
    // 2. 加密 (可选)
    if fs.encrypt {
        data, _ = aesEncrypt(data, fs.key)
    }
    
    // 3. 写入文件
    return os.WriteFile(fs.filePath, data, 0600)
}

func (fs *FileStorage) Load() (map[string]*TokenInfo, error) {
    // 1. 读取文件
    data, err := os.ReadFile(fs.filePath)
    if err != nil {
        return nil, err
    }
    
    // 2. 解密 (可选)
    if fs.encrypt {
        data, _ = aesDecrypt(data, fs.key)
    }
    
    // 3. 反序列化
    var tokens map[string]*TokenInfo
    err = json.Unmarshal(data, &tokens)
    return tokens, err
}
```

---

## 6. 监控和告警

### 6.1 指标收集

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

func (tm *TokenManager) GetMetrics() TokenMetrics {
    // 实现指标收集
}
```

### 6.2 告警事件

```go
type TokenAlert struct {
    Type      AlertType
    AccountID string
    Message   string
    Timestamp time.Time
    Error     error
}

type AlertType string

const (
    AlertTokenExpired     AlertType = "token_expired"
    AlertRefreshFailed    AlertType = "refresh_failed"
    AlertTokenInvalid     AlertType = "token_invalid"
    AlertMultipleFailures AlertType = "multiple_failures"
)

func (tm *TokenManager) alertTokenRefreshFailed(accountID string, err error) {
    alert := TokenAlert{
        Type:      AlertRefreshFailed,
        AccountID: accountID,
        Message:   fmt.Sprintf("Token refresh failed for %s", accountID),
        Timestamp: time.Now(),
        Error:     err,
    }
    
    // 发送告警 (日志、邮件、Webhook 等)
    log.Printf("[ALERT] %v", alert)
}
```

---

## 7. 使用示例

### 7.1 基本使用

```go
// 1. 创建 Token 管理器
config := TokenConfig{
    RefreshThreshold: 5 * time.Minute,
    RetryCount:       3,
    RetryInterval:    1 * time.Second,
    StorageType:      "memory",
    EnableMetrics:    true,
    AlertOnFailure:   true,
}

manager := NewTokenManager(config, generator)

// 2. 启动自动刷新
manager.StartAutoRefresh()

// 3. 获取 Token (自动刷新)
token, err := manager.GetToken("account_1")
if err != nil {
    log.Printf("Get token failed: %v", err)
}

// 4. 使用 Token 发送请求
req, _ := http.NewRequest("GET", "https://api.example.com", nil)
req.Header.Set("Authorization", "Bearer "+token)
client.Do(req)

// 5. 查看指标
metrics := manager.GetMetrics()
log.Printf("Active tokens: %d", metrics.ActiveTokens)
```

### 7.2 多账号管理

```go
// 管理多个账号的 Token
accounts := []string{"account_1", "account_2", "account_3"}

for _, accountID := range accounts {
    token, err := manager.GetToken(accountID)
    if err != nil {
        log.Printf("Account %s token error: %v", accountID, err)
        continue
    }
    log.Printf("Account %s token: %s...", accountID, token[:20])
}
```

---

## 8. 测试计划

### 8.1 单元测试

- [ ] TestTokenManager_GetToken
- [ ] TestTokenManager_RefreshToken
- [ ] TestTokenManager_AutoRefresh
- [ ] TestTokenManager_MultiAccount
- [ ] TestTokenManager_ConcurrentAccess
- [ ] TestTokenManager_StoragePersistence

### 8.2 集成测试

- [ ] TestTokenManager_EndToEnd
- [ ] TestTokenManager_RefreshRetry
- [ ] TestTokenManager_AlertOnFailure

### 8.3 性能测试

- [ ] BenchmarkTokenManager_GetToken
- [ ] BenchmarkTokenManager_Concurrent

---

## 9. 依赖清单

### 9.1 需要 Protocol-Agent 交付

1. **Token 生成算法**
   - 输入参数
   - 生成步骤
   - 示例代码

2. **Token 刷新接口**
   - API 端点
   - 请求格式
   - 响应格式

3. **Token 格式说明**
   - Token 结构
   - 有效期
   - 验证方法

### 9.2 外部依赖

- Go 1.22+
- 标准库：sync, time, crypto/aes, encoding/json

---

## 10. 实现计划

### Phase 1: 核心功能 (2 天)

- [ ] TokenManager 基础结构
- [ ] Token 获取和刷新
- [ ] 内存存储
- [ ] 单元测试

### Phase 2: 高级功能 (1 天)

- [ ] 自动刷新后台任务
- [ ] 文件持久化
- [ ] 指标收集
- [ ] 告警机制

### Phase 3: 集成测试 (1 天)

- [ ] 集成测试
- [ ] 性能测试
- [ ] 文档完善

---

## 11. 风险和挑战

### 11.1 技术风险

- **Token 算法复杂度**: 可能需要逆向工程
- **刷新频率**: 过高可能触发风控
- **并发安全**: 多 goroutine 访问需要仔细处理

### 11.2 缓解措施

- 与 Protocol-Agent 紧密协作
- 实现请求频率限制
- 充分的并发测试

---

## 12. 验收标准

- [ ] 所有单元测试通过
- [ ] 集成测试通过
- [ ] 性能达标 (获取 < 1ms, 刷新成功率 > 99%)
- [ ] 文档完整
- [ ] 代码审查通过

---

**文档结束**
