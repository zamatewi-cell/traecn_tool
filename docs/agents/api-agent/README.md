# API Development Agent 工作区

## 角色定位

**API Agent** - API 开发工程师，负责实现 OpenAI 兼容的 API 层

---

## 当前状态

**状态**: ? Available (等待任务分配)  
**依赖**: Protocol-Agent 的协议分析结果  
**预计开始**: 2026-03-17

---

## 待接收任务

### 预期 TASK-010: 实现 OpenAI 兼容 API 层

**依赖**: TASK-001 (认证流程逆向完成)

**工作内容**:
1. 实现 `/v1/chat/completions` 端点
2. 实现 `/v1/completions` 端点
3. 实现 `/v1/models` 端点
4. 实现流式 SSE 响应

**技术栈**:
- Go `net/http` 或 `gin` 框架
- SSE 流式处理
- JSON 序列化/反序列化

---

### 预期 TASK-011: 协议转换层实现

**依赖**: Protocol-Agent 的协议文档

**工作内容**:
1. OpenAI 请求 → Trae CN 协议转换
2. Trae CN 响应 → OpenAI 格式转换
3. 错误码映射
4. 字段映射表

**转换示例**:
```go
// OpenAI 请求
{
  "model": "deepseek-v3.1-terminus",
  "messages": [...],
  "stream": true
}

// ↓ 转换

// Trae CN 请求
{
  "model_id": "ds_v31",
  "conversation_id": "...",
  "messages": [...],
  "stream": true
}
```

---

## 技术预研

### 框架选择

**候选**:
1. **标准库 net/http**
   - 优点：无依赖，轻量
   - 缺点：路由需要手动实现

2. **Gin**
   - 优点：性能好，生态成熟
   - 缺点：需要额外依赖

**建议**: 使用 Gin，提升开发效率

---

### SSE 实现方案

**方案 1**: 手动实现
```go
w.Header().Set("Content-Type", "text/event-stream")
flusher := w.(http.Flusher)
fmt.Fprintf(w, "data: %s\n\n", jsonData)
flusher.Flush()
```

**方案 2**: 使用库
```go
import "github.com/r3labs/sse/v2"
```

**建议**: 手动实现，依赖少

---

## 开发计划

### Phase 1: 基础框架 (预计 2 天)

**Day 1**:
- [ ] 项目初始化
- [ ] 路由框架搭建
- [ ] 中间件实现 (CORS, Auth)

**Day 2**:
- [ ] `/v1/models` 端点
- [ ] 基础错误处理
- [ ] 日志系统

---

### Phase 2: 核心功能 (预计 3 天)

**Day 3**:
- [ ] `/v1/chat/completions` 非流式
- [ ] 协议转换逻辑

**Day 4**:
- [ ] `/v1/chat/completions` 流式
- [ ] SSE 处理

**Day 5**:
- [ ] `/v1/completions` 端点
- [ ] 统一错误处理

---

### Phase 3: 优化完善 (预计 2 天)

**Day 6**:
- [ ] 性能优化
- [ ] 并发测试

**Day 7**:
- [ ] 文档编写
- [ ] 代码审查

---

## 接口设计

### GET /v1/models

**响应**:
```json
{
  "object": "list",
  "data": [
    {
      "id": "deepseek-v3.1-terminus",
      "object": "model",
      "created": 1234567890,
      "owned_by": "trae-proxy"
    }
  ]
}
```

---

### POST /v1/chat/completions

**请求**:
```json
{
  "model": "deepseek-v3.1-terminus",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true
}
```

**响应 (流式)**:
```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"}}]}

data: [DONE]
```

---

## 依赖管理

### 需要 Protocol-Agent 提供

1. **协议文档**
   - API 端点列表
   - 请求/响应格式
   - 认证头信息

2. **测试数据**
   - 真实请求示例
   - 响应示例
   - 错误情况

3. **测试账号**
   - 用于联调测试

---

### 需要 Auth-Agent 提供

1. **Token 管理器**
   - Token 生成接口
   - 刷新接口

2. **认证中间件**
   - API Key 验证
   - Token 自动刷新

---

## 代码结构

```
internal/
├── api/
│   ├── server.go          # HTTP 服务器
│   ├── routes.go          # 路由定义
│   ├── middleware/        # 中间件
│   │   ├── auth.go        # 认证中间件
│   │   ├── cors.go        # CORS 中间件
│   │   └── logger.go      # 日志中间件
│   └── handlers/          # 处理器
│       ├── models.go      # /v1/models
│       ├── chat.go        # /v1/chat/completions
│       └── completions.go # /v1/completions
├── transformers/          # 协议转换
│   ├── request.go         # 请求转换
│   └── response.go        # 响应转换
└── models/                # 数据模型
    ├── openai.go          # OpenAI 模型
    └── trae.go            # Trae CN 模型
```

---

## 测试计划

### 单元测试

- [ ] 路由测试
- [ ] 转换器测试
- [ ] 中间件测试

### 集成测试

- [ ] 完整请求流程测试
- [ ] 流式响应测试
- [ ] 错误处理测试

### 性能测试

- [ ] 并发请求测试
- [ ] 响应延迟测试
- [ ] 内存泄漏测试

---

## 工作日志

### (待开始)

---

## 联系信息

- **工作目录**: `docs/agents/api-agent/`
- **状态**: ? Available
- **依赖**: Protocol-Agent (TASK-001)

---

**最后更新**: 2026-03-15 20:30  
**下次更新**: 任务分配后
