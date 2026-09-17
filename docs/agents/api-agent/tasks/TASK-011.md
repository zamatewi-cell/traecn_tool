# Task: 协议转换层实现

**ID**: TASK-011  
**优先级**: High  
**分配给**: @API-Agent  
**依赖**: TASK-001, TASK-010  
**截止日期**: 2026-03-20  
**状态**: ? Complete  

---

## 描述

实现 OpenAI 请求格式到 Trae CN 协议格式的转换层，以及反向响应转换。

---

## 验收标准

- [x] OpenAI 请求 → Trae CN 请求转换
- [x] Trae CN 响应 → OpenAI 响应转换
- [x] 错误码映射表
- [x] 字段映射文档
- [x] API 文档编写

---

## 转换示例

### 请求转换

```go
// OpenAI 请求
{
  "model": "deepseek-v3.1-terminus",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true,
  "temperature": 0.7,
  "max_tokens": 1000
}

// ↓ 转换为 Trae CN 请求
{
  "model_id": "ds_v31",
  "conversation_id": "conv_xxx",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true,
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
```

### 响应转换

```go
// Trae CN 响应
{
  "id": "trae_xxx",
  "choices": [{
    "delta": {"content": "Hello"}
  }]
}

// ↓ 转换为 OpenAI 响应
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion.chunk",
  "model": "deepseek-v3.1-terminus",
  "choices": [{
    "delta": {"content": "Hello"}
  }]
}
```

---

## 技术实现

**代码结构**:
```
internal/transformers/
├── request.go         # 请求转换
├── response.go        # 响应转换
├── error_map.go       # 错误码映射
└── field_map.go       # 字段映射表
```

**映射表**:
- 模型名称映射 (OpenAI ? Trae CN)
- 参数名称映射 (temperature, max_tokens 等)
- 角色映射 (system, user, assistant)
- 错误码映射 (400, 401, 429, 500 等)

---

## 依赖关系

**上游**: 
- TASK-001 (Protocol-Agent: Trae CN 协议规格)
- TASK-010 (API-Agent: API 基础框架)

**下游**:
- TASK-040 (Test-Agent: 转换器测试)

---

## 交付物

1. **转换器实现** (`internal/transformers/`)
2. **映射表文档** (`docs/transformers/mapping.md`)
3. **测试用例** (请求/响应转换测试)

---

**创建时间**: 2026-03-15  
**验收人**: PM-Agent
