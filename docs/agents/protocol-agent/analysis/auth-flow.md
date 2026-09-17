# Trae CN 认证流程分析

**最后更新**: 2026-03-15  
**负责人**: @Protocol-Agent  
**状态**: ? In Progress (60%)

---

## ? 概述

Trae CN 使用基于 JWT 的认证机制，包含以下关键组件：

1. **Access Token**: 短期有效的访问令牌
2. **Refresh Token**: 用于刷新 Access Token
3. **Device ID**: 设备唯一标识
4. **Signature**: 请求签名

---

## ? 认证流程

### 完整认证时序图

```
┌──────────┐         ┌──────────┐         ┌──────────┐
│  Client  │         │ Trae API │         │  Auth Svr│
└────┬─────┘         └────┬─────┘         └────┬─────┘
     │                    │                    │
     │  1. 登录请求        │                    │
     │───────────────────>│                    │
     │                    │  2. 验证凭据        │
     │                    │───────────────────>│
     │                    │                    │
     │                    │  3. 返回 Token      │
     │                    │<───────────────────│
     │  4. Token+Refresh  │                    │
     │<───────────────────│                    │
     │                    │                    │
     │  5. 携带 Token 请求   │                    │
     │───────────────────>│                    │
     │  6. 验证 Token      │                    │
     │<───────────────────│                    │
     │                    │                    │
     │  7. Token 过期      │                    │
     │───────────────────>│                    │
     │  8. 401 Unauthorized│                    │
     │<───────────────────│                    │
     │                    │                    │
     │  9. 刷新 Token      │                    │
     │───────────────────>│                    │
     │                    │ 10. 刷新请求        │
     │                    │───────────────────>│
     │                    │                    │
     │                    │ 11. 新 Token        │
     │                    │<───────────────────│
     │ 12. 新 Token+Refresh│                    │
     │<───────────────────│                    │
     │                    │                    │
```

---

## ? 请求头分析

### 认证相关 Header

从实际抓包数据中提取的完整 Header 列表：

| Header | 必需 | 说明 | 示例值 |
|--------|------|------|--------|
| `Authorization` | ? | Bearer Token (备用方式) | `Bearer eyJhbGciOiJSUzI1NiIs...` |
| `X-Ide-Token` | ? | 主要认证 Token | `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...` |
| `X-Device-Id` | ? | 设备唯一标识 | `2262131830954826` |
| `X-Machine-Id` | ? | 机器指纹 ID | `323072c1635648e2034a597de45cecfb28ee6fd5d74339197d47c77336d36ca3` |
| `X-Request-ID` | ? | 请求追踪 ID | `req_6cdadef1-5b0e-4e58-af12-ac97b720e86d` |
| `X-Custom-Trace-Id` | ? | 自定义追踪 ID | `5439d047f64b0a71fa09d721b97e3163` |
| `X-Tt-Trace-Id` | ? | 字节系追踪 ID | `00-e2a634220d809659c381b4a74ab7ffff-e2a634220d809659-01` |
| `X-Request-Pin` | ? | 请求 PIN 码 | `9ceba7caf2dde2cb` |
| `X-Request-At` | ? | 请求时间戳 | `1773329068` |
| `User-Agent` | ? | 客户端标识 | `TraeClient/TTNet` |

### 设备相关 Header

| Header | 说明 | 示例值 |
|--------|------|--------|
| `X-Device-Brand` | 设备品牌 | `QNLAS` |
| `X-Device-Type` | 设备类型 | `windows` |
| `X-Device-Cpu` | CPU 类型 | `Intel` |
| `X-Os-Version` | 操作系统版本 | `Windows 11 Home China` |

### 应用相关 Header

| Header | 说明 | 示例值 |
|--------|------|--------|
| `App-Version` | 应用版本 | `3.3.37` |
| `X-App-Id` | 应用 ID | `6eefa01c-1036-4c7e-9ca5-d891f63bfcd8` |
| `X-App-Version` | 应用版本标识 | `default` |
| `X-App-Version-Code` | 应用版本号 | `20260212` |
| `X-Ide-Version` | IDE 版本 | `3.3.37` |
| `X-Ide-Version-Code` | IDE 版本号 | `20260212` |
| `Package-Type` | 包类型 | `stable_cn` |

### 网络相关 Header

| Header | 说明 | 示例值 |
|--------|------|--------|
| `X-Bridge-Transport` | 传输协议 | `aha` |
| `X-Ahanet-Timeout` | 超时时间 (秒) | `86400` |
| `X-Lgw-Req-Sdk-Type` | SDK 类型 | `3` |
| `X-Ss-Dp` | 字节系标识 | `787976` |
| `X-Trae-Request-Id` | Trae 请求 ID | `fcf01b6e-b8d1-4679-8785-14fd1f1f022a` |
| `X-Custom-Repo-Urls` | 自定义仓库地址 | `https://github.com/zamatewi-cell/sky-pojo.git` |

---

## ? Token 生成算法

### 已知信息

**Token 格式**: JWT (JSON Web Token)

**结构**:
```
Header.Payload.Signature
```

**Header** (Base64 编码):
```json
{
  "alg": "RS256",
  "typ": "JWT"
}
```

**Payload** (Base64 编码):
```json
{
  "data": {
    "id": "4355622541471866",
    "source": "refresh_token",
    "source_id": "ghWFoX9c6QOLBcvXNl-hCCO9WukmJXlL0ELumBrXnKI=.189c1f657dd38abf",
    "tenant_id": "7o2d894p7dr2o4",
    "type": "user"
  },
  "exp": 1774536474,
  "iat": 1773326874
}
```

### Token 字段说明

从实际 Token 解析出的关键字段：

| 字段 | 说明 | 示例值 |
|------|------|--------|
| `data.id` | 用户 ID | `4355622541471866` |
| `data.source` | Token 来源 | `refresh_token` |
| `data.source_id` | Refresh Token | `ghWFoX9c6QOLBcvXNl-hCCO9WukmJXlL0ELumBrXnKI=.189c1f657dd38abf` |
| `data.tenant_id` | 租户 ID | `7o2d894p7dr2o4` |
| `data.type` | 用户类型 | `user` |
| `exp` | 过期时间戳 | `1774536474` (2026-03-26) |
| `iat` | 签发时间戳 | `1773326874` (2026-03-12) |

### Refresh Token 格式

```
<base64url_encoded_data>.<signature>
```

示例：`ghWFoX9c6QOLBcvXNl-hCCO9WukmJXlL0ELumBrXnKI=.189c1f657dd38abf`

**特点**:
- 由两部分组成，用 `.` 分隔
- 第一部分：Base64URL 编码的数据
- 第二部分：16 进制签名
- 有效期：约 6 个月 (从 2026-03-12 到 2026-09-08)

### Access Token 有效期

- **Access Token**: 约 14 天
- **Refresh Token**: 约 6 个月
- **刷新策略**: 建议在过期前 1 天刷新
  "device_id": "abc123-def456-ghi789",
  "exp": 1710507600,
  "iat": 1710504000,
  "scope": "user"
}
```

**Signature**:
```
HMACSHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  secret_key
)
```

### 待破解部分

- [ ] **Secret Key**: 签名密钥来源
- [ ] **Device ID 生成规则**: 如何生成合法的设备 ID
- [ ] **Timestamp 容差**: 服务器接受的时间窗口

---

## ? 刷新机制

### 刷新流程

1. **检测 Token 过期**: 收到 401 响应
2. **调用刷新接口**: `POST /api/auth/refresh`
3. **提供 Refresh Token**: 在请求体中
4. **获取新 Token**: 返回新的 Access Token 和 Refresh Token

### 刷新接口

**端点**: `POST /api/auth/refresh`

**请求体**:
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "device_id": "abc123-def456-ghi789"
}
```

**响应**:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 3600
}
```

---

## ? 错误处理

### 认证错误码

| 错误码 | HTTP 状态 | 说明 | 处理方式 |
|--------|----------|------|----------|
| `TOKEN_EXPIRED` | 401 | Token 过期 | 刷新 Token |
| `TOKEN_INVALID` | 401 | Token 无效 | 重新登录 |
| `DEVICE_MISMATCH` | 403 | 设备不匹配 | 重新登录 |
| `SIGNATURE_INVALID` | 401 | 签名无效 | 重新生成签名 |
| `RATE_LIMITED` | 429 | 请求限流 | 等待后重试 |

---

## ?? 分析工具

### 使用的脚本

1. **find_token.js**: 定位 Token 生成函数
2. **test_auth.js**: 测试认证流程
3. **analyze_headers.js**: 分析请求头

### 抓包数据

- `captured/auth-requests.json`: 认证请求抓包
- `captured/refresh-flows.json`: 刷新流程抓包

---

## ? 待办事项

- [ ] 还原完整的 Token 生成算法
- [ ] 提取 Secret Key 来源
- [ ] 分析 Device ID 生成规则
- [ ] 测试 Timestamp 容差
- [ ] 验证刷新机制
- [ ] 编写可执行认证脚本

---

## ? 相关链接

- [TASK-001](../tasks/TASK-001.md) - 认证流程逆向任务
- [JWT 规范](https://jwt.io/) - JWT 标准文档
