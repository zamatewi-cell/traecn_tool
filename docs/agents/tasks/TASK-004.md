# TASK-004: API 路由映射设计

**分配给**: @API-Agent  
**优先级**: ? Medium  
**状态**: ? Pending  
**创建日期**: 2026-03-15  
**截止日期**: 2026-03-19  
**依赖**: TASK-001 ?

---

## ? 任务描述

设计 OpenAI API 到 Trae CN 协议的路由映射，包括：
1. 端点映射表
2. 参数转换规则
3. 错误码映射
4. 响应格式转换

---

## ? 验收标准

- [ ] 完整的端点映射表
- [ ] 参数转换规则文档
- [ ] 错误码映射表
- [ ] 提供协议转换代码
- [ ] 编写 API 文档

---

## ? 子任务

### 1. 端点映射表 (0%)
- [ ] `/v1/chat/completions` → Trae 端点
- [ ] `/v1/completions` → Trae 端点
- [ ] `/v1/models` → Trae 端点
- [ ] `/v1/embeddings` → Trae 端点 (如有)

### 2. 参数转换规则 (0%)
- [ ] messages 格式转换
- [ ] model 名称映射
- [ ] stream 参数处理
- [ ] temperature/top_p 等参数映射

### 3. 错误码映射 (0%)
- [ ] Trae 错误码 → OpenAI 错误码
- [ ] 错误信息翻译
- [ ] 重试策略映射

### 4. 响应格式转换 (0%)
- [ ] Chat Completion 响应转换
- [ ] Stream 响应转换
- [ ] Models 列表转换

---

## ? 工作文件

- `design/api-mapping.md` - API 映射设计
- `internal/proxy/translator.go` - 转换器实现
- `docs/api/openai-compatibility.md` - 兼容性文档

---

## ?? 技术设计

### 端点映射

| OpenAI API | Trae CN API | 方法 | 说明 |
|------------|-------------|------|------|
| `/v1/chat/completions` | `/api/chat` | POST | 对话补全 |
| `/v1/models` | `/api/models` | GET | 模型列表 |

### 参数转换示例

```go
// OpenAI → Trae
{
  "model": "deepseek-v3.1-terminus",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": true
}

// ↓ 转换

{
  "model_id": "ds_v31",
  "conversation_id": "uuid",
  "messages": [{"role": "user", "content": "Hello"}],
  "stream": true
}
```

---

## ? 进度更新

### 2026-03-15 (待开始)
- ? 等待 TASK-001 完成

---

## ? 相关链接

- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [协议分析](./TASK-001.md)
