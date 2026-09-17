# OpenAI API 层快速指南

**最后更新**: 2026-03-15  
**状态**: ? 已完成

---

## ? 快速启动

### 1. 启动 Proxy 服务

```bash
# 确保已配置 config.json
go run cmd/trae-proxy/main.go
```

### 2. 验证服务健康

```bash
curl http://localhost:8080/health
```

预期响应:
```json
{"status": "ok", "version": "0.1.0"}
```

---

## ? 测试 API

### 方法 1: 使用测试脚本 (推荐)

```bash
# 安装依赖
cd scripts
npm install openai

# 运行测试
node test_openai_api.js
```

### 方法 2: 使用 cURL

```bash
# 获取模型列表
curl http://localhost:8080/v1/models

# 非流式聊天
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v3.1-terminus",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# 流式聊天
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v3.1-terminus",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

### 方法 3: 使用 Python SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-test"
)

response = client.chat.completions.create(
    model="deepseek-v3.1-terminus",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

### 方法 4: 使用 Node.js SDK

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'sk-test'
});

const response = await client.chat.completions.create({
  model: 'deepseek-v3.1-terminus',
  messages: [{ role: 'user', content: 'Hello!' }]
});

console.log(response.choices[0].message.content);
```

---

## ? 配置 API Keys

编辑 `config.json`:

```json
{
  "openai_api": {
    "enabled": true,
    "port": 8080,
    "api_keys": ["sk-your-key-1", "sk-your-key-2"]
  }
}
```

重启服务后生效。

---

## ? 完整文档

详细 API 文档请查看：[`docs/api-reference.md`](docs/api-reference.md)

---

## ? 已实现功能

- [x] GET /v1/models - 模型列表
- [x] POST /v1/chat/completions - 聊天完成（流式/非流式）
- [x] POST /v1/completions - 文本完成
- [x] GET /v1/queue/status - 队列状态
- [x] GET /health - 健康检查
- [x] API Key 认证
- [x] CORS 支持
- [x] 请求日志
- [x] SSE 流式传输

---

## ? 故障排查

### 服务无法启动

检查端口是否被占用:
```bash
netstat -ano | findstr :8080
```

### 收到 401 错误

确认请求头包含正确的 API Key:
```bash
curl -H "Authorization: Bearer sk-your-key" ...
```

### 模型不支持

查看支持的模型列表:
```bash
curl http://localhost:8080/v1/models
```

---

## ? 监控

### 查看请求日志

服务启动后会自动记录所有请求:
```
[INFO] POST /v1/chat/completions - 200 OK - 1.234s - model=deepseek-v3.1-terminus, stream=true
```

### 查看队列状态

```bash
curl http://localhost:8080/v1/queue/status
```

---

## ? 下一步

1. **Test-Agent**: 编写单元测试和集成测试
2. **UI-Agent**: 创建管理界面
3. **PM-Agent**: 协调各模块集成测试

---

**维护者**: API-Agent  
**联系**: @API-Agent
