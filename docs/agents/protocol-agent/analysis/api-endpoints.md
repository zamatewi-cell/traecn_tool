# Trae CN API 端点分析

**分析时间**: 2026-03-15  
**分析工具**: `scripts/analyze_endpoints.js`, `scripts/dump_auth.js`, `scripts/test_*.js`  
**数据来源**: 抓包数据、逆向分析

---

## ? 端点总览

| 端点 | 方法 | 用途 | 认证 | 状态 |
|------|------|------|------|------|
| `/api/auth/refresh_token` | POST | 刷新 Token | ? | ? 已分析 |
| `/api/user/profile` | GET | 获取用户信息 | ? | ? 待分析 |
| `/api/agent/v3/create_agent_task` | POST | 创建 Agent 任务 | ? | ? 待分析 |
| `/api/agent/task/{id}` | GET | 查询任务状态 | ? | ? 待分析 |
| `/api/v1/ai/chat/completions` | POST | AI 对话（SSE） | ? | ? 待分析 |
| `/api/v1/model/list` | GET | 获取模型列表 | ? | ? 待分析 |

---

## ? 1. 认证端点

### POST /api/auth/refresh_token

**用途**: 使用 Refresh Token 获取新的 Access Token

**请求**:
```http
POST /api/auth/refresh_token
Content-Type: application/json
User-Agent: TraeClient/TTNet
X-Device-Id: 2262131830954826
X-Machine-Id: 323072c1635648e2034a597de45cecfb28ee6fd5d74339197d47c77336d36ca3
X-Request-ID: req_6cdadef1-5b0e-4e58-af12-ac97b720e86d
```

**请求体**:
```json
{
  "refresh_token": "ghWFoX9c6QOLBcvXNl-hCCO9WukmJXlL0ELumBrXnKI=.189c1f657dd38abf",
  "user_id": "4355622541471866"
}
```

**响应**:
```json
{
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiredAt": "2026-03-26T14:47:54.040Z"
}
```

**错误响应**:
```json
{
  "error": {
    "code": "invalid_refresh_token",
    "message": "Refresh Token 已过期或无效",
    "type": "authentication_error"
  }
}
```

---

## ? 2. 用户端点

### GET /api/user/profile

**用途**: 获取当前登录用户的详细信息

**请求**:
```http
GET /api/user/profile
X-Ide-Token: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
X-Device-Id: 2262131830954826
X-Machine-Id: 323072c1635648e2034a597de45cecfb28ee6fd5d74339197d47c77336d36ca3
X-Request-ID: req_xxx
User-Agent: TraeClient/TTNet
```

**响应** (预期):
```json
{
  "user": {
    "id": "4355622541471866",
    "tenant_id": "7o2d894p7dr0o4",
    "email": "user@example.com",
    "name": "用户名",
    "avatar": "https://...",
    "plan": "free|pro|enterprise",
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

---

## ? 3. Agent 任务端点

### POST /api/agent/v3/create_agent_task

**用途**: 创建一个新的 Agent 任务（Builder、Solo Coder 等）

**请求**:
```http
POST /api/agent/v3/create_agent_task
Content-Type: application/json
X-Ide-Token: <JWT Token>
X-Device-Id: <Device ID>
X-Machine-Id: <Machine ID>
X-Request-ID: <Request ID>
X-Custom-Trace-Id: <Trace ID>
X-App-Id: 6eefa01c-1036-4c7e-9ca5-d891f63bfcd8
App-Version: 3.3.37
User-Agent: TraeClient/TTNet
```

**请求体**:
```json
{
  "agent_type": "builder_with_mcp",
  "prompt": "创建一个待办事项应用",
  "model_config": {
    "model": "Doubano1-1.5",
    "temperature": 0.7,
    "max_tokens": 4096
  },
  "context": {
    "workspace": "d:\\code\\my-project",
    "files": [
      {
        "path": "src/index.ts",
        "content": "..."
      }
    ],
    "selected_code": {
      "file": "src/app.ts",
      "start_line": 10,
      "end_line": 20,
      "content": "..."
    }
  },
  "conversation_id": "conv_xxx",
  "parent_message_id": "msg_xxx"
}
```

**响应** (预期):
```json
{
  "task_id": "task_abc123",
  "status": "queued",
  "position": 5,
  "estimated_time": 120,
  "created_at": "2026-03-15T14:30:00Z"
}
```

**错误响应**:
```json
{
  "error": {
    "code": "invalid_agent_type",
    "message": "不支持的 Agent 类型",
    "type": "validation_error"
  }
}
```

### GET /api/agent/task/{id}

**用途**: 查询 Agent 任务的状态和结果

**请求**:
```http
GET /api/agent/task/task_abc123
X-Ide-Token: <JWT Token>
X-Device-Id: <Device ID>
X-Machine-Id: <Machine ID>
X-Request-ID: <Request ID>
```

**响应** (预期):
```json
{
  "task": {
    "id": "task_abc123",
    "status": "completed|processing|failed",
    "agent_type": "builder_with_mcp",
    "prompt": "创建一个待办事项应用",
    "result": {
      "files": [
        {
          "path": "src/index.ts",
          "content": "...",
          "action": "create|modify|delete"
        }
      ],
      "summary": "已创建待办事项应用",
      "next_steps": ["运行 npm install", "启动开发服务器"]
    },
    "created_at": "2026-03-15T14:30:00Z",
    "completed_at": "2026-03-15T14:32:00Z"
  }
}
```

---

## ? 4. AI 对话端点 (SSE)

### POST /api/v1/ai/chat/completions

**用途**: 发送 AI 对话请求，接收 SSE 流式响应

**请求**:
```http
POST /api/v1/ai/chat/completions
Content-Type: application/json
X-Ide-Token: <JWT Token>
X-Device-Id: <Device ID>
X-Machine-Id: <Machine ID>
X-Request-ID: <Request ID>
Accept: text/event-stream
```

**请求体**:
```json
{
  "model": "Doubano1-1.5",
  "messages": [
    {
      "role": "system",
      "content": "你是一个编程助手"
    },
    {
      "role": "user",
      "content": "如何反转字符串？"
    }
  ],
  "stream": true,
  "temperature": 0.7,
  "max_tokens": 2048
}
```

**SSE 响应流**:
```
data: {"type":"queue_update","position":5,"estimated_time":120}

data: {"type":"task_created","task_id":"chat_abc123"}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710504000,"model":"Doubano1-1.5","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710504000,"model":"Doubano1-1.5","choices":[{"index":0,"delta":{"content":"要"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710504000,"model":"Doubano1-1.5","choices":[{"index":0,"delta":{"content":"反转"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710504000,"model":"Doubano1-1.5","choices":[{"index":0,"delta":{"content":"字符串"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710504000,"model":"Doubano1-1.5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

**SSE 事件类型**:

| 类型 | 说明 | 数据格式 |
|------|------|----------|
| `queue_update` | 排队状态更新 | `{"type":"queue_update","position":5,"estimated_time":120}` |
| `task_created` | 任务创建成功 | `{"type":"task_created","task_id":"xxx"}` |
| `chat.completion.chunk` | 对话内容分块 | OpenAI 兼容格式 |
| `[DONE]` | 流式传输完成 | 纯文本 `[DONE]` |

**错误处理**:

SSE 流中的错误也会通过 `data:` 推送：

```
data: {"error":{"code":"rate_limit_exceeded","message":"请求频率超限","type":"rate_limit_error","retry_after":60}}
```

---

## ? 5. 模型配置端点

### GET /api/v1/model/list

**用途**: 获取可用的 AI 模型列表

**请求**:
```http
GET /api/v1/model/list
X-Ide-Token: <JWT Token>
X-Device-Id: <Device ID>
X-Machine-Id: <Machine ID>
```

**响应** (预期):
```json
{
  "models": [
    {
      "id": "Doubano1-1.5",
      "name": "Doubao 1.5",
      "provider": "bytedance",
      "max_tokens": 8192,
      "supports_vision": false,
      "supports_function_call": true,
      "context_window": 32768
    },
    {
      "id": "GPT-4",
      "name": "GPT-4",
      "provider": "openai",
      "max_tokens": 8192,
      "supports_vision": true,
      "supports_function_call": true,
      "context_window": 8192
    }
  ]
}
```

---

## ? 6. 通用请求头

所有认证请求都需要包含以下 Header：

### 必需 Header

| Header | 说明 | 示例 |
|--------|------|------|
| `X-Ide-Token` | JWT Access Token | `eyJhbGciOiJSUzI1NiIs...` |
| `X-Device-Id` | 设备唯一标识 | `2262131830954826` |
| `X-Machine-Id` | 机器指纹 ID | `323072c1635648e2034a597de45cecfb28ee6fd5d74339197d47c77336d36ca3` |
| `X-Request-ID` | 请求追踪 ID | `req_6cdadef1-5b0e-4e58-af12-ac97b720e86d` |
| `User-Agent` | 客户端标识 | `TraeClient/TTNet` |

### 可选 Header

| Header | 说明 | 示例 |
|--------|------|------|
| `X-Custom-Trace-Id` | 自定义追踪 ID | `5439d047f64b0a71fa09d721b97e3163` |
| `X-Tt-Trace-Id` | 字节系追踪 ID | `00-e2a634220d809659c381b4a74ab7ffff-e2a634220d809659-01` |
| `X-App-Id` | 应用 ID | `6eefa01c-1036-4c7e-9ca5-d891f63bfcd8` |
| `App-Version` | 应用版本 | `3.3.37` |
| `X-Device-Brand` | 设备品牌 | `QNLAS` |
| `X-Device-Type` | 设备类型 | `windows` |
| `X-Os-Version` | 操作系统版本 | `Windows 11 Home China` |

---

## ? 7. 错误码汇总

### 认证错误 (4xx)

| 错误码 | HTTP 状态码 | 说明 | 解决方案 |
|--------|-----------|------|----------|
| `invalid_token` | 401 | Token 无效或过期 | 刷新 Token |
| `invalid_refresh_token` | 401 | Refresh Token 无效 | 重新登录 |
| `missing_auth_header` | 401 | 缺少认证 Header | 添加 X-Ide-Token |
| `insufficient_permissions` | 403 | 权限不足 | 升级账户 |

### 限流错误 (429)

| 错误码 | HTTP 状态码 | 说明 | 解决方案 |
|--------|-----------|------|----------|
| `rate_limit_exceeded` | 429 | 请求频率超限 | 等待 retry_after 秒 |
| `quota_exceeded` | 429 | 配额用尽 | 升级套餐或等待重置 |

### 服务器错误 (5xx)

| 错误码 | HTTP 状态码 | 说明 | 解决方案 |
|--------|-----------|------|----------|
| `internal_error` | 500 | 服务器内部错误 | 重试 |
| `service_unavailable` | 503 | 服务不可用 | 等待后重试 |
| `gateway_timeout` | 504 | 网关超时 | 检查网络或重试 |

### 验证错误 (400)

| 错误码 | HTTP 状态码 | 说明 | 解决方案 |
|--------|-----------|------|----------|
| `invalid_agent_type` | 400 | Agent 类型无效 | 检查 agent_type 参数 |
| `invalid_model` | 400 | 模型不存在 | 检查 model 参数 |
| `missing_required_field` | 400 | 缺少必填字段 | 检查请求体 |
| `invalid_json` | 400 | JSON 格式错误 | 检查 JSON 语法 |

---

## ?? 参考实现

### Go 语言 HTTP 客户端

```go
package api

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
    tokenMgr   *TokenManager
}

func NewClient(baseURL string, tokenMgr *TokenManager) *Client {
    return &Client{
        baseURL:  baseURL,
        tokenMgr: tokenMgr,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *Client) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
    var reqBody io.Reader
    if body != nil {
        data, err := json.Marshal(body)
        if err != nil {
            return nil, err
        }
        reqBody = bytes.NewReader(data)
    }

    req, err := http.NewRequest(method, c.baseURL+endpoint, reqBody)
    if err != nil {
        return nil, err
    }

    // 获取有效 Token
    token, err := c.tokenMgr.GetValidToken()
    if err != nil {
        return nil, err
    }

    // 设置认证 Header
    req.Header.Set("X-Ide-Token", token)
    req.Header.Set("X-Device-Id", c.tokenMgr.GetDeviceID())
    req.Header.Set("X-Machine-Id", c.tokenMgr.GetMachineID())
    req.Header.Set("X-Request-ID", generateUUID())
    req.Header.Set("User-Agent", "TraeClient/TTNet")

    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }

    return c.httpClient.Do(req)
}

func generateUUID() string {
    // 实现 UUID 生成
    return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
}
```

---

## ? 下一步行动

1. ? 完成认证端点分析 (TASK-001)
2. ? 完成所有 API 端点分析 (TASK-002)
3. ? 分析 SSE 流式响应格式细节
4. ? 实现 API 客户端参考代码
5. ? 编写集成测试用例

---

## ? 相关资源

- [认证流程文档](./auth-flow.md)
- [SSE 协议分析](./sse-protocol.md) (待创建)
- [错误处理规范](./error-handling.md) (待创建)
- [`scripts/auth_flow.js`](../../../scripts/auth_flow.js) - 认证流程实现
- [`scripts/parse_token.js`](../../../scripts/parse_token.js) - Token 解析工具
