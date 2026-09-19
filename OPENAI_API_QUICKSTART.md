# OpenAI API 层快速指南

**最后更新**: 2026-09-19  
**版本**: 1.0.0  
**状态**: 已就绪 (Production Ready)

---

## 快速启动

### 1. 启动 Proxy 服务

```bash
# 确保已配置 config.json，或直接无参启动（自动嗅探本地 Trae CN 凭据）
go run cmd/trae-proxy/main.go
# 或直接运行编译好的可执行程序
./trae-proxy.exe
```

### 2. 验证服务健康

```bash
curl http://localhost:9090/health
```

预期响应:
```json
{"status": "ok", "version": "1.0.0"}
```

---

## 测试 API

### 方法 1: 使用测试脚本

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
curl http://localhost:9090/v1/models

# 非流式聊天
curl -X POST http://localhost:9090/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Seed-Code",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'

# 流式聊天
curl -X POST http://localhost:9090/v1/chat/completions \
  -H "Authorization: Bearer sk-test" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Seed-Code",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

### 方法 3: 使用 Python SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:9090/v1",
    api_key="sk-test"
)

response = client.chat.completions.create(
    model="Seed-Code",
    messages=[{"role": "user", "content": "Hello!"}]
)

print(response.choices[0].message.content)
```

### 方法 4: 使用 Node.js SDK

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:9090/v1',
  apiKey: 'sk-test'
});

const response = await client.chat.completions.create({
  model: 'Seed-Code',
  messages: [{ role: 'user', content: 'Hello!' }]
});

console.log(response.choices[0].message.content);
```

---

## 配置 API Keys

编辑 `config.json`:

```json
{
  "listen_addr": ":9090",
  "log_level": "info",
  "api_key": "sk-your-key",
  "protect": {
    "max_concurrent": 4
  }
}
```

重启服务后生效。如果设置了 `api_key`，客户端请求必须携带 `Authorization: Bearer <api_key>`。

---

## 完整文档

详细 API 文档请查看：[`docs/api-reference.md`](docs/api-reference.md)

---

## 已实现功能

- [x] GET /v1/models - 模型列表（包含 16 个内置模型与 5 个预设模型）
- [x] POST /v1/chat/completions - 聊天完成（流式/非流式，支持思考链分流）
- [x] POST /v1/messages - Anthropic Messages 协议兼容
- [x] POST /v1/responses - Codex 协议兼容
- [x] GET /v1/queue/status - 队列状态
- [x] GET /v1/accounts - 账号池与凭据状态
- [x] GET /health - 健康检查（返回 version 1.0.0）
- [x] WebUI 可视化控制台（/ 或 /dashboard）
- [x] API Key 认证与 LAN 安全暴露防呆
- [x] CSRF 跨站防护与 CORS 支持
- [x] SQLite 请求流水与本地日志持久化
- [x] SSE 流式传输与首字延迟 (TTFT) 监控

---

## 故障排查

### 服务无法启动

检查端口是否被占用:
```powershell
netstat -ano | findstr :9090
```

### 收到 401 错误

确认请求头包含正确的 API Key:
```bash
curl -H "Authorization: Bearer sk-your-key" ...
```

### 模型不支持

查看支持的模型列表:
```bash
curl http://localhost:9090/v1/models
```

---

## 监控与日志

### 查看请求日志

服务启动后会自动记录所有请求:
```
[INFO] POST /v1/chat/completions - 200 OK - 1.234s - model=Seed-Code, stream=true
```

### 控制台在线查看

在浏览器访问 `http://localhost:9090/` 即可直接在暗黑主题仪表盘中查看实时请求流水与官方云端账单。
