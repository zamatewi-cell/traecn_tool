# Task: API 端点协议分析

**ID**: TASK-002  
**优先级**: High  
**分配给**: @Protocol-Agent  
**依赖**: TASK-001 (认证流程逆向)  
**截止日期**: 2026-03-17  
**状态**: ? In Progress  
**最后更新**: 2026-03-15 14:45

---

## 描述

完整分析 Trae CN 的所有 API 端点，包括请求/响应格式、参数验证规则、错误码等，为 API-Agent 实现提供完整的协议规格说明。

---

## 验收标准

- [ ] 列出所有 API 端点及其用途
- [ ] 分析每个端点的请求格式（Header、Body）
- [ ] 分析每个端点的响应格式
- [ ] 识别参数验证规则
- [ ] 整理错误码和错误消息
- [ ] 分析 SSE 流式响应格式
- [ ] 编写 API 文档和参考实现

---

## 技术细节

### 已知端点

从抓包数据中识别的端点：

1. **认证相关**
   - `POST /api/auth/refresh_token` - 刷新 Token

2. **Agent 任务相关**
   - `POST /api/agent/v3/create_agent_task` - 创建 Agent 任务
   - `GET /api/agent/task/{task_id}` - 查询任务状态

3. **SSE 流式响应**
   - `POST /api/v1/ai/chat/completions` - AI 对话完成（SSE 流）

4. **用户相关**
   - `GET /api/user/profile` - 获取用户信息

### 请求格式分析

从实际抓包数据中提取的请求结构：

**创建 Agent 任务**:
```http
POST /api/agent/v3/create_agent_task
Content-Type: application/json
X-Ide-Token: <JWT Token>
X-Device-Id: <Device ID>
X-Machine-Id: <Machine ID>
X-Request-ID: <Request ID>
```

**请求体**:
```json
{
  "agent_type": "builder_with_mcp",
  "prompt": "用户提示词",
  "model_config": {
    "model": "Doubano1-1.5",
    "temperature": 0.7
  },
  "context": {
    "workspace": "工作区路径",
    "files": ["相关文件列表"]
  }
}
```

### SSE 流式响应格式

从抓包数据中提取的 SSE 事件：

```
data: {"type":"queue_update","position":5,"estimated_time":120}
data: {"type":"task_created","task_id":"abc123"}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":"stop"}]}
data: [DONE]
```

### 错误响应格式

**401 Unauthorized**:
```json
{
  "error": {
    "code": "invalid_token",
    "message": "Token 已过期或无效",
    "type": "authentication_error"
  }
}
```

**429 Too Many Requests**:
```json
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "请求频率超限",
    "type": "rate_limit_error",
    "retry_after": 60
  }
}
```

**500 Internal Server Error**:
```json
{
  "error": {
    "code": "internal_error",
    "message": "服务器内部错误",
    "type": "server_error"
  }
}
```

---

## 工作计划

### Phase 1: 端点发现 (20%)
- [x] 分析现有抓包数据
- [ ] 运行端点探测脚本
- [ ] 整理端点列表

### Phase 2: 协议分析 (60%)
- [ ] 分析每个端点的请求格式
- [ ] 分析每个端点的响应格式
- [ ] 识别参数验证规则
- [ ] 整理错误码

### Phase 3: 文档编写 (20%)
- [ ] 编写 API 文档
- [ ] 编写参考实现
- [ ] 编写测试用例

---

## 相关资源

- **抓包数据**: `scripts/captured/`
- **分析脚本**: `scripts/analyze_endpoints.js`
- **测试脚本**: `scripts/test_*.js`
- **API 文档**: `docs/agents/protocol-agent/analysis/api-endpoints.md` (待创建)

---

## 依赖关系

**上游**: TASK-001 (认证流程)  
**下游**: 
- API-Agent 的 TASK-010 (API 层实现)
- API-Agent 的 TASK-011 (协议转换)

---

## 交付物

1. **SSE 格式文档** (`docs/agents/protocol-agent/analysis/sse-format.md`)
2. **解析器参考** (`scripts/sse_parser.py`)
3. **测试数据** (完整的 SSE 流示例，包含正常和错误情况)

---

**创建时间**: 2026-03-15  
**验收人**: PM-Agent
