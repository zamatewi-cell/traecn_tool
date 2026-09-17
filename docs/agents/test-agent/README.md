# Testing & QA Agent 工作区

## 角色定位

**Test Agent** - 测试工程师，负责质量保证和测试验证

---

## 当前状态

**状态**: ? Available (等待任务分配)  
**依赖**: 各开发 Agent 的产出  
**预计开始**: 2026-03-20

---

## 待接收任务

### 预期 TASK-040: 编写单元测试

**依赖**: 各模块代码完成

**工作内容**:
1. API 层单元测试
2. 认证模块测试
3. 队列模块测试
4. 转换器测试

**覆盖率目标**: > 80%

---

### 预期 TASK-041: 集成测试

**工作内容**:
1. 完整请求流程测试
2. 多账号轮换测试
3. 并发请求测试
4. 错误处理测试

**测试场景**:
```
正常流程：请求 → 认证 → 队列 → 转换 → 响应
异常流程：Token 过期 → 自动刷新 → 重试 → 成功
边界情况：队列满 → 拒绝请求 → 返回 429
```

---

### 预期 TASK-042: 性能基准测试

**工作内容**:
1. 响应延迟测试
2. 吞吐量测试
3. 并发能力测试
4. 资源占用测试

**性能目标**:
- P50 延迟：< 200ms
- P99 延迟：< 500ms
- 吞吐量：> 100 QPS
- 内存占用：< 500MB

---

### 预期 TASK-043: 协议兼容性测试

**工作内容**:
1. OpenAI API 兼容性测试
2. 各客户端兼容性测试
3. 模型响应正确性测试

**测试客户端**:
- ChatBox
- NextChat
- OpenAI 官方 SDK
- LangChain

---

## 测试框架

### 单元测试

使用 Go 原生 `testing` 包:

```go
func TestTokenManager_Generate(t *testing.T) {
    tm := NewTokenManager(config)
    
    token, err := tm.Generate("test-account")
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if token == "" {
        t.Error("expected non-empty token")
    }
}
```

---

### 集成测试

使用 `testify` 和 `httptest`:

```go
func TestChatEndpoint(t *testing.T) {
    server := httptest.NewServer(api.NewRouter())
    defer server.Close()
    
    client := openai.NewClient(server.URL)
    
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: "deepseek-v3.1-terminus",
        Messages: []openai.ChatCompletionMessage{
            {Role: "user", Content: "Hello"},
        },
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.Choices[0].Message.Content)
}
```

---

### 性能测试

使用 `testing` 包的 benchmark:

```go
func BenchmarkChatEndpoint(b *testing.B) {
    server := setupTestServer()
    defer server.Close()
    
    client := openai.NewClient(server.URL)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := client.CreateChatCompletion(ctx, req)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## 测试用例库

### 认证模块

```go
// TC-AUTH-001: 正常 Token 生成
func TestTokenGeneration_Success(t *testing.T)

// TC-AUTH-002: Token 过期自动刷新
func TestTokenRefresh_Auto(t *testing.T)

// TC-AUTH-003: 无效 API Key 拒绝
func TestAPIKey_Invalid(t *testing.T)

// TC-AUTH-004: 账号池轮换
func TestAccountPool_Rotation(t *testing.T)
```

---

### API 模块

```go
// TC-API-001: 获取模型列表
func TestGetModels_Success(t *testing.T)

// TC-API-002: 聊天补全 - 非流式
func TestChatCompletions_NonStream(t *testing.T)

// TC-API-003: 聊天补全 - 流式
func TestChatCompletions_Stream(t *testing.T)

// TC-API-004: 错误请求处理
func TestChatCompletions_InvalidRequest(t *testing.T)
```

---

### 队列模块

```go
// TC-QUEUE-001: 优先级调度
func TestQueue_PriorityScheduling(t *testing.T)

// TC-QUEUE-002: 限流触发
func TestLimiter_Trigger(t *testing.T)

// TC-QUEUE-003: 熔断器状态转换
func TestCircuitBreaker_StateTransition(t *testing.T)

// TC-QUEUE-004: 排队状态推送
func TestQueueStatus_SSE(t *testing.T)
```

---

## 自动化测试

### CI/CD 集成

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

---

### 测试报告

生成 HTML 报告:

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## Bug 追踪

### Bug 模板

```markdown
## Bug: [简短描述]

**ID**: BUG-001  
**严重程度**: Critical/Major/Minor  
**发现日期**: 2026-03-20  
**发现者**: Test-Agent  

### 复现步骤

1. 步骤 1
2. 步骤 2
3. 步骤 3

### 预期行为

应该发生什么

### 实际行为

实际发生了什么

### 环境信息

- OS: Windows 11
- Go: 1.22
- Version: v0.1.0

### 日志

```
错误日志内容
```

### 修复建议

如何修复
```

---

## 质量指标

### 代码质量

- [ ] 单元测试覆盖率 > 80%
- [ ] 关键路径覆盖率 100%
- [ ] 无 Critical/Major Bug
- [ ] 代码审查通过率 100%

### 性能质量

- [ ] P99 延迟 < 500ms
- [ ] 吞吐量 > 100 QPS
- [ ] 内存占用 < 500MB
- [ ] CPU 占用 < 50%

### 稳定性质量

- [ ] 7x24 小时无故障
- [ ] 错误恢复率 > 99%
- [ ] 数据一致性 100%

---

## 依赖关系

### 需要各开发 Agent 提供

1. **可测试的代码**
   - 接口抽象
   - 依赖注入
   - Mock 支持

2. **测试数据**
   - 测试账号
   - 请求/响应示例
   - 边界条件说明

3. **文档**
   - API 文档
   - 配置说明
   - 部署指南

---

## 测试环境

### 本地环境

```bash
# 开发环境
go test -v ./...

# 覆盖率
go test -cover ./...

# 性能测试
go test -bench=. -benchmem ./...
```

### CI 环境

- GitHub Actions
- 自动触发测试
- 覆盖率上报

---

## 工作日志

### (待开始)

---

## 联系信息

- **工作目录**: `docs/agents/test-agent/`
- **状态**: ? Available
- **依赖**: 各开发 Agent

---

**最后更新**: 2026-03-15 20:30  
**下次更新**: 任务分配后
