# Trae CN Proxy API 参考文档

**版本**: 1.0.0  
**最后更新**: 2026-09-19  
**维护状态**: 生产就绪 (Production Ready)  

---

## 概述

Trae CN Proxy 提供标准 OpenAI / Anthropic / Codex 兼容的 HTTP API 接口，允许使用标准 OpenAI SDK、各类 AI 客户端（Cursor、Continue、Claude Code、Aider 等）无缝访问 Trae CN 服务。

### 基础信息

- **Base URL**: `http://localhost:9090` (OpenAI 协议路由为 `http://localhost:9090/v1`)
- **认证方式**: Bearer Token (`Authorization: Bearer <API_KEY>`)
- **数据格式**: JSON
- **流式传输**: Server-Sent Events (SSE)
- **Web 控制台**: `http://localhost:9090/` 或 `http://localhost:9090/dashboard`

### 核心支持模型清单

| 模型标识 (Model ID) | 常见别名 | 上游通道映射 | 描述 |
|---|---|---|---|
| `Seed-Code` | `doubao`, `seed` | `seed_m8` | 字节跳动代码优化主力模型 (默认) |
| `DeepSeek-V4.1-Flash` | | 内置通道 | DeepSeek 最新高吞吐 Flash 模型 |
| `DeepSeek-V4-Flash` | `deepseek-v4`, `deepseek-chat` | `deepseek-V3` | DeepSeek 高性能通用对话模型 |
| `DeepSeek-V4-Pro` | `deepseek-r1`, `deepseek-reasoner` | `deepseek-R1` | DeepSeek 深度推理模型（支持思考链） |
| `GLM-5.3` | `glm-5`, `glm` | 内置通道 | 智谱新一代通用大模型 |
| `GLM-5.3-Flash` | | 内置通道 | 智谱轻量极速模型 |
| `Kimi-K3` | `kimi` | 内置通道 | 月之暗面长上下文主力模型 |
| `Qwen3.7-Plus` | `qwen` | 内置通道 | 通义千问增强版代码与通用模型 |

*完整 21 个模型（16 个内置模型 + 5 个预设模型）支持调用 `GET /v1/models` 获取最新动态注册表。*

---

## 认证机制

如果配置了 API Key，所有受保护 API 请求须在 HTTP 请求头中提供：

```http
Authorization: Bearer your-api-key
```

### 安全与防呆策略

1. **Localhost 默认保护**：默认仅监听本地环回地址（`127.0.0.1` / `localhost`）。
2. **局域网暴露拦截**：若启动参数附加 `-allow-lan` 或绑定 `0.0.0.0`，系统强制要求必须在配置中配置高强度 `api_key`，否则拒绝启动，杜绝内网资产意外裸露风险。
3. **CSRF 防护**：在无 Key 本地运行模式下，网关对非受信外部网页 Origin 发起的跨站请求进行拦截（HTTP 403），防止恶意网页通过本地端口探测利用。

---

## API 端点

### 1. GET /v1/models

获取可用模型列表及其元数据。

**请求**:
```http
GET /v1/models
Authorization: Bearer sk-xxx
```

**响应**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "Seed-Code",
      "object": "model",
      "created": 1726704000,
      "owned_by": "trae"
    },
    {
      "id": "DeepSeek-V4.1-Flash",
      "object": "model",
      "created": 1726704000,
      "owned_by": "trae"
    },
    {
      "id": "DeepSeek-V4-Pro",
      "object": "model",
      "created": 1726704000,
      "owned_by": "trae"
    }
  ]
}
```

---

### 2. POST /v1/chat/completions

创建对话完成请求（兼容 OpenAI 标准，支持流式与非流式）。

**请求**:
```http
POST /v1/chat/completions
Authorization: Bearer sk-xxx
Content-Type: application/json

{
  "model": "Seed-Code",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false
}
```

**非流式响应**:
```json
{
  "id": "chatcmpl-87c2b5d4",
  "object": "chat.completion",
  "created": 1726704000,
  "model": "Seed-Code",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 10,
    "total_tokens": 25
  }
}
```

#### 流式响应 (SSE)

当 `"stream": true` 时，服务端以 `text/event-stream` 格式持续分块推送：

```
data: {"id":"chatcmpl-87c2b5d4","object":"chat.completion.chunk","created":1726704000,"model":"Seed-Code","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-87c2b5d4","object":"chat.completion.chunk","created":1726704000,"model":"Seed-Code","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-87c2b5d4","object":"chat.completion.chunk","created":1726704000,"model":"Seed-Code","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: [DONE]
```

*若使用推理模型（如 DeepSeek-V4-Pro / deepseek-r1），思考过程将通过 `delta.reasoning_content` 进行实时独立分流。*

---

### 3. POST /v1/messages

Anthropic Claude Messages 协议兼容端点。

**请求**:
```http
POST /v1/messages
x-api-key: sk-xxx
anthropic-version: 2023-06-01
Content-Type: application/json

{
  "model": "claude-3-5-sonnet",
  "messages": [
    {"role": "user", "content": "Hello Claude"}
  ],
  "max_tokens": 1024
}
```

---

### 4. POST /v1/responses

Codex 协议兼容端点。

---

### 5. GET /v1/queue/status

获取网关当前并发请求队列状态。

**请求**:
```http
GET /v1/queue/status
Authorization: Bearer sk-xxx
```

**响应**:
```json
{
  "active_requests": 1,
  "max_concurrent": 4,
  "queue_length": 0
}
```

---

### 6. GET /health

服务存活与版本健康检查端点。

**请求**:
```http
GET /health
```

**响应**:
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

---

## 错误处理规范

所有异常统一返回规范的 JSON 结构：

```json
{
  "error": {
    "message": "错误描述信息",
    "type": "invalid_request_error",
    "param": null,
    "code": "model_not_found"
  }
}
```

### 常用 HTTP 状态码

| HTTP 状态码 | 错误分类 (`type`) | 说明 |
|---|---|---|
| 400 | `invalid_request_error` | 请求体格式错误或缺少必填字段 |
| 401 | `authentication_error` | API Key 缺失或无效 |
| 403 | `forbidden` | CSRF 跨站非法请求阻断或无权限 |
| 429 | `rate_limit_error` | 并发限制或触发上游限频 |
| 500 | `api_error` | 网关处理或上游服务异常 |

---

## 更新日志

### v1.0.0 (2026-09-19)

- 正式里程碑发布；
- 自动化 CI/CD 跨平台交叉编译（Windows/macOS/Linux 6大架构支持）；
- 桌面端打包发布（NSIS 安装包与 Portable 便携版）；
- 统一模型通道分发（支持 16 个新内置模型与 5 个预设模型）；
- WebUI 仪表盘与 SQLite 持久化会话日志；
- LAN 暴露安全防呆与 CSRF 防护加固。
