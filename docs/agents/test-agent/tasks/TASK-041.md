# Task: 编写集成测试

**ID**: TASK-041  
**优先级**: High  
**分配给**: @Test-Agent  
**依赖**: TASK-040  
**截止日期**: 2026-03-21  
**状态**: ? Pending  

---

## 描述

编写端到端集成测试，验证完整请求流程和系统交互。

---

## 验收标准

- [ ] 完整聊天流程测试
- [ ] 多账号轮询测试
- [ ] 流式响应测试
- [ ] 错误处理测试
- [ ] 并发请求测试

---

## 测试场景

### 场景 1: 单账号完整流程

```go
func TestSingleAccount_FullFlow(t *testing.T) {
    // 1. 启动服务器
    // 2. 发送聊天请求
    // 3. 验证响应格式
    // 4. 验证 Token 使用
}
```

### 场景 2: 多账号轮询

```go
func TestMultiAccount_RoundRobin(t *testing.T) {
    // 1. 配置 3 个账号
    // 2. 发送 10 个请求
    // 3. 验证轮询分配
    // 4. 验证负载均衡
}
```

### 场景 3: 流式响应

```go
func TestStreamingResponse(t *testing.T) {
    // 1. 发送流式请求
    // 2. 接收 SSE 事件
    // 3. 验证事件格式
    // 4. 验证 [DONE] 标记
}
```

### 场景 4: 错误处理

```go
func TestErrorHandling(t *testing.T) {
    // 1. 测试无效 Token
    // 2. 测试限流错误
    // 3. 测试后端错误
    // 4. 验证错误响应格式
}
```

---

## 测试工具

- **httptest**: Go 标准库 HTTP 测试
- **testcontainers**: 容器化测试环境
- **k6**: 负载测试（可选）

---

## 测试数据

### 配置文件

```json
{
  "listen_addr": ":0",  // 随机端口
  "accounts": [
    {"name": "test1", "token": "mock_token_1"},
    {"name": "test2", "token": "mock_token_2"}
  ]
}
```

### Mock 后端

```go
type MockTraeBackend struct {
    responses []string
    errors    []error
}

func (m *MockTraeBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 模拟 Trae CN 后端响应
}
```

---

## 依赖关系

**上游**: 
- TASK-040 (单元测试)
- TASK-010 (API 实现)
- TASK-020 (Token 管理)

**下游**:
- TASK-042 (压力测试)

---

## 交付物

1. **集成测试文件** (`tests/integration/*_test.go`)
2. **测试夹具** (`tests/fixtures/`)
3. **Mock 服务器** (`tests/mocks/`)
4. **测试脚本** (`scripts/run_tests.sh`)

---

**创建时间**: 2026-03-15  
**验收人**: PM-Agent
