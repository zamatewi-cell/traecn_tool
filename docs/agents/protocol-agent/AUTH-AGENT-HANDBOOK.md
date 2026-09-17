# Auth-Agent 快速参考

**创建时间**: 2026-03-15 15:30  
**用途**: TASK-001 交付物快速索引  
**目标读者**: Auth-Agent 实现 TASK-101 Token 管理器

---

## ? 快速开始

### 第一步：了解 Token 结构 (5 分钟)

**阅读**: [`auth-flow.md`](auth-flow.md)

**关键信息**:
```
Token 类型：RS256 JWT
格式：Header.Payload.Signature (Base64URL 编码)

Header:
{
  "alg": "RS256",
  "typ": "JWT"
}

Payload:
{
  "data": {
    "id": "用户 ID",
    "source": "refresh_token",
    "source_id": "Refresh Token",
    "tenant_id": "租户 ID",
    "type": "user"
  },
  "exp": 过期时间戳，
  "iat": 签发时间戳
}
```

### 第二步：运行参考脚本 (10 分钟)

**测试 Token 解析**:
```bash
cd d:\codelearn\vscode\reverse_proxy
node scripts/parse_token.js
```

**预期输出**:
```
=== Trae CN Token 解析 ===
Token 长度：1004 字符
Header: { alg: 'RS256', typ: 'JWT' }
Payload: { ... }
过期时间：2026-03-26 14:27:54
剩余有效期：11 天
状态：? 有效
```

**测试完整认证流程**:
```bash
node scripts/auth_flow.js
```

**测试内容**:
- Token 加载
- 有效期检查
- Token 刷新 (如需要)
- 认证请求测试

### 第三步：开始实现 (2 天)

**参考**:
- JavaScript 实现：[`scripts/auth_flow.js`](../../../../scripts/auth_flow.js)
- 详细设计：Auth-Agent 的 `TOKEN-MANAGER-DESIGN.md`
- 任务要求：`TASK-101.md`

---

## ? 关键文件索引

### 协议文档

| 文件 | 内容 | 用途 |
|------|------|------|
| [`auth-flow.md`](auth-flow.md) | 认证流程详解 | ??? 必读 |
| [`TASK-001-COMPLETE.md`](tasks/TASK-001-COMPLETE.md) | 完成报告 | ??? 必读 |
| [`api-endpoints.md`](analysis/api-endpoints.md) | API 端点规格 | ?? 参考 |

### 参考代码

| 文件 | 功能 | 参考价值 |
|------|------|----------|
| [`scripts/parse_token.js`](../../../../scripts/parse_token.js) | JWT 解析 | ??? 直接参考 |
| [`scripts/auth_flow.js`](../../../../scripts/auth_flow.js) | 完整认证流程 | ??? 直接参考 |
| [`internal/auth/token.go`](../../../../internal/auth/token.go) | Go 实现 (待创建) | - |

### 任务文档

| 文件 | 内容 |
|------|------|
| [`tasks/TASK-001.md`](tasks/TASK-001.md) | TASK-001 详情 |
| [`tasks/TASK-002.md`](tasks/TASK-002.md) | API 端点分析 (进行中) |

---

## ? 核心知识点

### 1. JWT Token 解析

**Go 库推荐**: `github.com/golang-jwt/jwt/v5`

**示例代码**:
```go
import "github.com/golang-jwt/jwt/v5"

type TraeClaims struct {
    Data struct {
        ID       string `json:"id"`
        Source   string `json:"source"`
        SourceID string `json:"source_id"` // Refresh Token
        TenantID string `json:"tenant_id"`
        Type     string `json:"type"`
    } `json:"data"`
    jwt.RegisteredClaims
}

func ParseToken(tokenString string) (*TraeClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &TraeClaims{}, nil) // 验证逻辑待实现
    if err != nil {
        return nil, err
    }
    return token.Claims.(*TraeClaims), nil
}
```

### 2. Token 刷新接口

**端点**: `POST /api/auth/refresh_token`

**请求**:
```go
type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token"`
}

type RefreshTokenResponse struct {
    Data struct {
        Token             string `json:"token"`
        RefreshToken      string `json:"refresh_token"`
        ExpiredAt         int64  `json:"expired_at"`
        RefreshExpiredAt  int64  `json:"refresh_expired_at"`
    } `json:"data"`
}
```

**Header**:
```go
headers := map[string]string{
    "X-Ide-Token": currentToken,
    "X-Device-Id": deviceID,
    "X-Machine-Id": machineID,
    "X-Request-ID": requestID,
    "User-Agent": "TraeClient/TTNet",
}
```

### 3. Token 存储

**内存存储**:
```go
type TokenManager struct {
    mu           sync.RWMutex
    accessToken  string
    refreshToken string
    expiresAt    time.Time
    refreshExpiresAt time.Time
}
```

**文件持久化**:
```go
// Windows 路径
storagePath := filepath.Join(os.Getenv("APPDATA"), "Trae CN", "User", "globalStorage", "storage.json")

// 读取
data, err := os.ReadFile(storagePath)
var storage map[string]interface{}
json.Unmarshal(data, &storage)
authInfo := storage["iCubeAuthInfo://icube.cloudide"].(string)

// 解析 authInfo JSON 获取 token 信息
```

### 4. 并发安全

**使用 `sync.RWMutex`**:
```go
func (tm *TokenManager) GetToken() (string, error) {
    tm.mu.RLock()
    defer tm.mu.RUnlock()
    
    if time.Now().After(tm.expiresAt.Add(-3 * 24 * time.Hour)) {
        // 需要刷新
        return "", ErrTokenExpired
    }
    return tm.accessToken, nil
}

func (tm *TokenManager) UpdateToken(token, refreshToken string, expiresAt time.Time) {
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    tm.accessToken = token
    tm.refreshToken = refreshToken
    tm.expiresAt = expiresAt
}
```

### 5. 定时刷新

**使用 `time.Timer`**:
```go
func (tm *TokenManager) StartAutoRefresh() {
    go func() {
        for {
            select {
            case <-tm.refreshTimer.C:
                tm.refreshToken()
            case <-tm.stopChan:
                return
            }
        }
    }()
}

func (tm *TokenManager) scheduleNextRefresh(expiresAt time.Time) {
    // 提前 3 天刷新
    refreshTime := expiresAt.Add(-3 * 24 * time.Hour)
    duration := time.Until(refreshTime)
    tm.refreshTimer = time.NewTimer(duration)
}
```

---

## ? 实现检查清单

### Phase 1: 基础功能 (Day 1)

- [ ] Token 解析器实现
  - [ ] JWT 解析 (使用 golang-jwt/jwt)
  - [ ] Claims 结构定义
  - [ ] 有效期验证
  
- [ ] Token 存储实现
  - [ ] 内存存储结构
  - [ ] 文件持久化
  - [ ] 加载/保存方法

- [ ] Token 刷新实现
  - [ ] HTTP 客户端
  - [ ] 刷新请求构造
  - [ ] 响应解析

### Phase 2: 高级功能 (Day 2)

- [ ] 并发安全
  - [ ] sync.RWMutex 集成
  - [ ] 读写锁优化
  
- [ ] 自动刷新
  - [ ] time.Timer 定时刷新
  - [ ] 主动刷新策略
  - [ ] 被动刷新策略

- [ ] 错误处理
  - [ ] 刷新失败重试
  - [ ] Token 完全过期处理
  - [ ] 网络错误处理

- [ ] 监控告警
  - [ ] 刷新失败告警
  - [ ] Token 即将过期告警
  - [ ] 指标收集

### Phase 3: 测试验证 (Day 2 下午)

- [ ] 单元测试
  - [ ] Token 解析测试
  - [ ] 刷新逻辑测试
  - [ ] 并发安全测试

- [ ] 集成测试
  - [ ] 真实 Token 测试
  - [ ] 刷新流程测试
  - [ ] 与 API-Agent 联调

---

## ? 实现优先级

### P0 (必须实现)

1. **Token 解析** - 基础功能
2. **Token 存储** - 内存 + 文件
3. **Token 刷新** - HTTP 客户端
4. **并发安全** - sync.RWMutex

### P1 (重要)

5. **自动刷新** - time.Timer
6. **错误处理** - 重试机制
7. **Header 构造** - 完整认证头

### P2 (可选)

8. **监控告警** - 指标收集
9. **日志记录** - 详细日志
10. **配置化** - 刷新阈值可配置

---

## ? 常见问题

### Q1: 如何验证 JWT 签名？

**A**: RS256 使用公钥验证。Trae 的公钥未知，但可以选择：
1. **跳过验证** (仅用于开发): `jwt.ParseWithClaims(token, claims, nil)`
2. **反向工程获取公钥**: 从 Trae CN 客户端提取
3. **信任 Token 内容**: 假设从官方客户端获取的 Token 有效

**推荐**: 开发阶段跳过验证，生产阶段实现完整验证

### Q2: Refresh Token 会变化吗？

**A**: 会！每次刷新后：
- Access Token 更新
- Refresh Token 也会更新 (新的 `data.source_id`)
- 需要同时保存新的 token 和 refresh_token

### Q3: 多久刷新一次 Token？

**A**: 策略：
- **主动刷新**: 剩余有效期 < 3 天
- **被动刷新**: Token 已过期但 Refresh Token 有效
- **强制刷新**: Refresh Token 也过期 → 需要重新登录

### Q4: Device-ID 和 Machine-ID 如何生成？

**A**: 
- **Device-ID**: 可以是随机 UUID (首次生成后持久化)
- **Machine-ID**: 可以使用机器指纹 (MAC 地址、CPU ID 等哈希)
- **关键**: 需要保持一致性，不能每次请求都变化

### Q5: 如何处理多个账号？

**A**: Token 管理器设计：
```go
type MultiAccountManager struct {
    accounts map[string]*TokenManager // userId -> TokenManager
    current  string // 当前使用的 userId
}
```

---

## ? 获取帮助

### 文档资源

- **认证流程**: [`auth-flow.md`](auth-flow.md)
- **完成报告**: [`TASK-001-COMPLETE.md`](tasks/TASK-001-COMPLETE.md)
- **API 端点**: [`api-endpoints.md`](analysis/api-endpoints.md)

### 参考代码

- **Token 解析**: [`scripts/parse_token.js`](../../../../scripts/parse_token.js)
- **完整流程**: [`scripts/auth_flow.js`](../../../../scripts/auth_flow.js)

### 联系方式

- **事件日志**: [`event-log.md`](../../event-log.md) - 留言
- **PM-Agent**: 项目协调
- **Protocol-Agent**: 协议问题咨询

---

## ? 下一步行动

1. **立即**: 阅读 [`auth-flow.md`](auth-flow.md)
2. **5 分钟**: 运行 `node scripts/parse_token.js`
3. **10 分钟**: 运行 `node scripts/auth_flow.js`
4. **30 分钟**: 设计 Go 实现方案
5. **1 小时**: 开始实现 TASK-101

**Auth-Agent 准备好了吗？开始吧！** ?

---

**最后更新**: 2026-03-15 15:30  
**维护者**: Protocol-Agent
