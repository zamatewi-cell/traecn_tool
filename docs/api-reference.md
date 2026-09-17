# Trae CN Proxy API 文档

**版本**: 0.1.0  
**最后更新**: 2026-03-15  
**维护者**: API-Agent  

---

## ? 概述

Trae CN Proxy 提供 OpenAI 兼容的 API 接口，允许使用标准 OpenAI SDK 访问 Trae CN 服务。

### 基础信息

- **Base URL**: `http://localhost:8080`
- **认证方式**: Bearer Token (API Key)
- **数据格式**: JSON
- **流式传输**: Server-Sent Events (SSE)

### 支持的模型

| 模型名称 | Trae CN ID | 描述 |
|---------|-----------|------|
| `deepseek-v3.1-terminus` | `ds_v31` | DeepSeek V3.1 最新版 |
| `deepseek-v3` | `ds_v3` | DeepSeek V3 |
| `deepseek-r1` | `ds_r1` | DeepSeek R1 |

---

## ? 认证

所有 API 请求需要在 `Authorization` 头中提供 API Key：

```http
Authorization: Bearer your-api-key
```

### 配置 API Keys

在启动配置中设置：

```json
{
  "openai_api": {
    "enabled": true,
    "port": 8080,
    "api_keys": ["sk-your-key-1", "sk-your-key-2"]
  }
}
```

---

## ? API 端点

### 1. GET /v1/models

获取可用模型列表。

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
      "id": "deepseek-v3.1-terminus",
      "object": "model",
      "created": 1234567890,
      "owned_by": "trae"
    },
    {
      "id": "deepseek-v3",
      "object": "model",
      "created": 1234567890,
      "owned_by": "trae"
    }
  ]
}
```

---

### 2. POST /v1/chat/completions

创建聊天完成请求（支持流式和非流式）。

**请求**:
```http
POST /v1/chat/completions
Authorization: Bearer sk-xxx
Content-Type: application/json
```

**请求体**:
```json
{
  "model": "deepseek-v3.1-terminus",
  "messages": [
    {
      "role": "system",
      "content": "You are a helpful assistant."
    },
    {
      "role": "user",
      "content": "Hello, world!"
    }
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1000
}
```

**参数说明**:

| 参数 | 类型 | 必填 | 默认值 | 描述 |
|------|------|------|--------|------|
| `model` | string | ? | - | 模型名称 |
| `messages` | array | ? | - | 消息列表 |
| `stream` | boolean | ? | false | 是否启用流式 |
| `temperature` | number | ? | 1.0 | 温度参数 |
| `max_tokens` | integer | ? | null | 最大 token 数 |
| `top_p` | number | ? | 1.0 | Top-p 采样 |
| `frequency_penalty` | number | ? | 0.0 | 频率惩罚 |
| `presence_penalty` | number | ? | 0.0 | 存在惩罚 |

**Message 格式**:

```json
{
  "role": "user|assistant|system",
  "content": "消息内容"
}
```

#### 非流式响应

**响应**:
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "deepseek-v3.1-terminus",
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
    "prompt_tokens": 20,
    "completion_tokens": 15,
    "total_tokens": 35
  }
}
```

#### 流式响应

设置 `"stream": true` 启用 SSE 流式传输。

**响应格式** (SSE):
```
data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1234567890,"model":"deepseek-v3.1-terminus","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1234567890,"model":"deepseek-v3.1-terminus","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","created":1234567890,"model":"deepseek-v3.1-terminus","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: [DONE]
```

**Chunk 格式**:
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion.chunk",
  "created": 1234567890,
  "model": "deepseek-v3.1-terminus",
  "choices": [
    {
      "index": 0,
      "delta": {
        "role": "assistant",
        "content": "..."
      },
      "finish_reason": null
    }
  ]
}
```

---

### 3. POST /v1/completions

Legacy Completions API（兼容旧版 OpenAI API）。

**请求**:
```http
POST /v1/completions
Authorization: Bearer sk-xxx
Content-Type: application/json
```

**请求体**:
```json
{
  "model": "deepseek-v3.1-terminus",
  "prompt": "Once upon a time",
  "max_tokens": 100
}
```

**响应**:
```json
{
  "id": "cmpl-abc123",
  "object": "text_completion",
  "created": 1234567890,
  "model": "deepseek-v3.1-terminus",
  "choices": [
    {
      "text": ", there was a brave knight...",
      "index": 0,
      "finish_reason": "length"
    }
  ],
  "usage": {
    "prompt_tokens": 4,
    "completion_tokens": 10,
    "total_tokens": 14
  }
}
```

---

### 4. GET /v1/queue/status

获取队列状态信息。

**请求**:
```http
GET /v1/queue/status?model=deepseek-v3.1-terminus
Authorization: Bearer sk-xxx
```

**响应**:
```json
{
  "model": "deepseek-v3.1-terminus",
  "queue_length": 5,
  "estimated_wait_seconds": 120,
  "active_requests": 3,
  "total_capacity": 100
}
```

---

### 5. GET /health

健康检查端点。

**请求**:
```http
GET /health
```

**响应**:
```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

---

## ? 错误处理

### 错误响应格式

```json
{
  "error": {
    "message": "错误描述",
    "type": "错误类型",
    "param": null,
    "code": "错误代码"
  }
}
```

### 常见错误码

| HTTP 状态码 | 错误类型 | 描述 |
|------------|---------|------|
| 400 | `invalid_request` | 请求参数错误 |
| 401 | `invalid_auth` | 认证失败（API Key 无效） |
| 403 | `forbidden` | 权限不足 |
| 429 | `rate_limit` | 请求频率限制 |
| 500 | `internal_error` | 服务器内部错误 |
| 503 | `overloaded` | 服务过载（队列满） |

### 错误示例

**缺少 API Key**:
```json
{
  "error": {
    "message": "Missing Authorization header",
    "type": "invalid_auth",
    "code": "missing_authorization"
  }
}
```

**无效模型**:
```json
{
  "error": {
    "message": "Model 'gpt-4' is not supported",
    "type": "invalid_request",
    "code": "model_not_supported"
  }
}
```

**队列已满**:
```json
{
  "error": {
    "message": "Queue is full, please try again later",
    "type": "rate_limit",
    "code": "queue_full"
  }
}
```

---

## ? 配置选项

### OpenAI API 配置

```json
{
  "openai_api": {
    "enabled": true,
    "port": 8080,
    "api_keys": ["sk-xxx"],
    "cors": {
      "allowed_origins": ["*"],
      "allowed_methods": ["GET", "POST", "OPTIONS"],
      "allowed_headers": ["Authorization", "Content-Type"]
    },
    "logging": {
      "enabled": true,
      "include_body": true
    }
  }
}
```

| 配置项 | 类型 | 默认值 | 描述 |
|-------|------|--------|------|
| `enabled` | boolean | false | 是否启用 OpenAI API |
| `port` | integer | 8080 | 监听端口 |
| `api_keys` | array | [] | 允许的 API Keys |
| `cors.allowed_origins` | array | ["*"] | 允许的来源 |
| `logging.enabled` | boolean | true | 是否记录请求日志 |
| `logging.include_body` | boolean | false | 是否记录请求体 |

---

## ? 使用示例

### Python SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-xxx"  # 任意非空字符串
)

# 非流式
response = client.chat.completions.create(
    model="deepseek-v3.1-terminus",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)
print(response.choices[0].message.content)

# 流式
stream = client.chat.completions.create(
    model="deepseek-v3.1-terminus",
    messages=[
        {"role": "user", "content": "Tell me a story"}
    ],
    stream=True
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### Node.js SDK

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'sk-xxx'
});

// 非流式
const response = await client.chat.completions.create({
  model: 'deepseek-v3.1-terminus',
  messages: [{ role: 'user', content: 'Hello!' }]
});
console.log(response.choices[0].message.content);

// 流式
const stream = await client.chat.completions.create({
  model: 'deepseek-v3.1-terminus',
  messages: [{ role: 'user', content: 'Tell me a story' }],
  stream: true
});
for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0].delta.content || '');
}
```

### cURL

```bash
# 非流式
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v3.1-terminus",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# 流式
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v3.1-terminus",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

---

## ? 监控与日志

### 请求日志

每个请求都会记录以下信息：

```
[INFO] POST /v1/chat/completions - 200 OK - 1.234s - model=deepseek-v3.1-terminus, stream=true
```

包含：
- HTTP 方法和路径
- 响应状态码
- 请求耗时
- 模型名称
- 是否流式

### 队列监控

通过 `/v1/queue/status` 端点实时监控队列状态。

---

## ? 安全建议

1. **API Key 管理**
   - 定期轮换 API Keys
   - 不要在客户端暴露 API Keys
   - 使用环境变量存储敏感信息

2. **网络隔离**
   - 仅监听本地地址（127.0.0.1）
   - 使用防火墙限制访问
   - 考虑添加 IP 白名单

3. **速率限制**
   - 配置合理的队列大小
   - 监控异常请求模式
   - 实施请求频率限制

---

## ? 故障排查

### 常见问题

**Q: 收到 401 错误**
- 检查 Authorization 头格式：`Bearer sk-xxx`
- 确认 API Key 在配置中已设置
- 检查是否有前导/后随空格

**Q: 流式响应中断**
- 检查网络连接稳定性
- 查看服务器日志是否有错误
- 确认队列未满

**Q: 模型不支持**
- 确认模型名称拼写正确
- 查看 `/v1/models` 获取支持的模型列表
- 检查模型映射配置

---

## ? 更新日志

### v0.1.0 (2026-03-15)

- ? 初始版本发布
- ? OpenAI 兼容 API 层
- ? 流式 SSE 支持
- ? API Key 认证
- ? CORS 支持
- ? 请求日志

---

## ? 相关文档

- [项目 README](../../../README.md)
- [架构文档](../ARCHITECTURE.md)
- [故障排查指南](../../knowledge-base/troubleshooting/)

---

**维护者**: API-Agent  
**最后更新**: 2026-03-15
