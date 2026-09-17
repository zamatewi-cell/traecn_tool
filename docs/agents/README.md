# Trae-Proxy 多 Agent 协同开发项目

**项目状态**: ? 进行中  
**当前阶段**: Phase 0 - 协议逆向工程 (60%)  
**启动日期**: 2026-03-15

---

## ? 项目目标

开发 Trae CN 反向代理工具，将 Trae CN 的 AI 模型以 OpenAI 兼容 API 形式暴露。

**技术栈**: Go 1.22+ + React + Electron

---

## ? Agent 团队

本项目采用多 Agent 协同开发模式，共 7 个 Agent 分工合作：

| Agent | 职责 | 状态 | 当前任务 |
|-------|------|------|----------|
| **PM-Agent** | 项目协调 | ? Running | 项目协调 |
| **Protocol-Agent** | 协议逆向 | ? Running (60%) | TASK-001 |
| **API-Agent** | API 开发 | ? Blocked | 等待协议文档 |
| **Auth-Agent** | 认证管理 | ? Blocked | 等待认证分析 |
| **Queue-Agent** | 队列管理 | ? Available | 待分配 |
| **Test-Agent** | 质量保障 | ? Available | 待分配 |
| **UI-Agent** | 界面开发 | ? Completed (80%) | UI-005 |

---

## ? 文档导航

### 新手入门
- **[快速开始](./QUICKSTART.md)** - 5 分钟快速上手
- **[项目启动](./PROJECT_START.md)** - 项目概况和里程碑
- **[任务看板](./TASK_BOARD.md)** - 实时任务进度

### 协作规范
- **[团队协作](./COLLABORATION_GUIDE.md)** - 协作流程和规范
- **[团队职责](./TEAM.md)** - Agent 角色和职责
- **[每日站会](./pm-agent/daily-standup.md)** - 每日站会报告

### 技术文档
- **[项目架构](./ARCHITECTURE.md)** - 技术架构设计
- **[事件日志](./event-log.md)** - 系统事件记录
- **[知识库](../knowledge-base/)** - 技术知识库

### 任务卡片
- **[TASK-001](./tasks/TASK-001.md)** - 认证流程逆向 (? High)
- **[TASK-002](./tasks/TASK-002.md)** - SSE 流式响应分析 (? High)
- **[TASK-003](./tasks/TASK-003.md)** - 账号池管理设计 (? Medium)
- **[TASK-004](./tasks/TASK-004.md)** - API 路由映射设计 (? Medium)
- **[TASK-005](./tasks/TASK-005.md)** - 队列调度算法 (? Low)
- **[TASK-006](./tasks/TASK-006.md)** - 测试用例编写 (? Low)

### Agent 工作区
- **[PM-Agent](./pm-agent/)** - 项目管理
- **[Protocol-Agent](./protocol-agent/)** - 协议分析
- **[API-Agent](./api-agent/)** - API 开发
- **[Auth-Agent](./auth-agent/)** - 认证管理
- **[Queue-Agent](./queue-agent/)** - 队列管理
- **[Test-Agent](./test-agent/)** - 质量保障
- **[UI-Agent](../../ui-dev/)** - 界面开发

---

## ? 项目进度

### Phase 0: 协议逆向工程 (60%)
```
████████████████???????? 60%
```
**时间**: 2026-03-12 ~ 2026-03-20

- ? 抓包环境搭建
- ? 基础端点探测
- ? 认证流程逆向 (60%)
- ? 加密算法破解
- ? 协议文档完成

### Phase 1: 核心开发 (0%)
```
???????????????????????? 0%
```
**时间**: 2026-03-21 ~ 2026-03-27

- ? 认证模块实现
- ? API 层实现
- ? 队列管理实现
- ? 集成测试

### Phase 2: 优化完善 (0%)
```
???????????????????????? 0%
```
**时间**: 2026-03-28 ~ 2026-04-03

### Phase 3: 发布准备 (0%)
```
???????????????????????? 0%
```
**时间**: 2026-04-04 ~ 2026-04-09

---

## ?? 会议安排

### 每日站会
**时间**: 每日 09:00  
**主持**: PM-Agent  
**内容**:
- 各 Agent 汇报昨日完成
- 今日计划
- 阻塞和风险

### 周会
**时间**: 每周一 14:00  
**主持**: PM-Agent  
**内容**:
- 周进度回顾
- 里程碑检查
- 技术讨论

---

## ? 快速开始

### 新 Agent 入职

1. **阅读文档**
   - [快速开始](./QUICKSTART.md)
   - [项目启动](./PROJECT_START.md)
   - [团队职责](./TEAM.md)

2. **确认角色**
   - 查看 [TEAM.md](./TEAM.md) 确认职责
   - 查看 [TASK_BOARD.md](./TASK_BOARD.md) 了解进度

3. **开始工作**
   - 查看分配的任务卡片
   - 更新任务进度
   - 参加每日站会

### 常用命令

```bash
# 查看任务状态
cat docs/agents/TASK_BOARD.md

# 查看特定任务
cat docs/agents/tasks/TASK-001.md

# 更新进度
# 编辑对应的任务卡片

# 运行后端
go run cmd/trae-proxy/main.go

# 运行前端
cd ui-dev/
npm run dev
```

---

## ? 获取帮助

遇到问题时：
1. 查看相关文档
2. 询问 PM-Agent
3. 在站会上提出
4. 联系相关 Agent

---

## ? 更新日志

### 2026-03-15
- ? 创建多 Agent 协同开发模式
- ? 分配 7 个 Agent 角色
- ? 创建任务看板
- ? 创建 6 个任务卡片
- ? Protocol-Agent 开始认证流程逆向 (60%)
- ? UI-Agent 完成基础页面 (80%)

---

**最后更新**: 2026-03-15 14:30  
**下次更新**: 2026-03-16 09:00 (每日站会)  
**维护人**: PM-Agent
- **自动清理**: 达到上限时清理低优先级项

### Event (事件)

事件是 Agent 间的通信机制:
- **发布/订阅**: 松耦合通信
- **异步分发**: 非阻塞通知
- **历史记录**: 可查询过去事件

## 创建自定义 Agent

### 步骤 1: 定义 Agent 类型

```go
package main

import (
    "context"
    "d:\codelearn\vscode\reverse_proxy\internal\agent"
    "d:\codelearn\vscode\reverse_proxy\internal\workflow"
)

type MyCustomAgent struct {
    *agent.BaseAgent
}

func NewMyCustomAgent(id string) *MyCustomAgent {
    return &MyCustomAgent{
        BaseAgent: agent.NewBaseAgent(id, "MyCustomAgent", "custom", "./docs"),
    }
}
```

### 步骤 2: 实现 Run 方法

```go
func (a *MyCustomAgent) Run(ctx context.Context, task *workflow.Task) error {
    // 调用父类初始化
    if err := a.BaseAgent.Run(ctx, task); err != nil {
        return err
    }

    // 更新进度
    a.UpdateProgress(10, "Starting", "开始执行...")

    // 执行实际工作
    for i := 0; i < 10; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            progress := (i + 1) * 10
            a.UpdateProgress(progress, "Working", fmt.Sprintf("处理第 %d 步", i+1))
            
            // 添加上下文
            a.AddContext(workflow.Agent, workflow.Normal, 
                fmt.Sprintf("步骤 %d 完成", i+1), nil)
            
            time.Sleep(100 * time.Millisecond)
        }
    }

    return nil
}
```

### 步骤 3: 注册到引擎

```go
engine := workflow.NewWorkflowEngine(config)
customAgent := NewMyCustomAgent("custom-001")
engine.RegisterAgent(customAgent)
```

## 任务管理

### 创建任务

```go
task := &workflow.Task{
    ID:          "my-task-001",
    Type:        "data-processing",
    Description: "处理数据集",
    Priority:    workflow.PriorityHigh,
    AgentID:     "custom-001",
    MaxRetries:  3,
    TimeoutSeconds: 300,
    Metadata: map[string]interface{}{
        "dataset": "training-data",
        "batch_size": 32,
    },
}

engine.CreateTask(task)
```

### 查询任务

```go
// 获取单个任务
task, err := engine.GetTask("my-task-001")

// 获取待处理任务
pending := engine.GetPendingTasks()

// 获取特定 Agent 的任务
tasks, err := engine.GetTaskManager().GetTasksByAgent("custom-001")

// 获取特定状态的任务
running := engine.GetTaskManager().GetTasksByStatus(workflow.TaskRunning)
```

### 更新任务状态

```go
// 更新进度
engine.GetTaskManager().UpdateTaskStatus(
    "my-task-001",
    workflow.TaskRunning,
    50, // 50% 完成
    nil, // 结果
    "",  // 错误信息
)

// 标记完成
engine.GetTaskManager().UpdateTaskStatus(
    "my-task-001",
    workflow.TaskCompleted,
    100,
    resultData,
    "",
)

// 标记失败
engine.GetTaskManager().UpdateTaskStatus(
    "my-task-001",
    workflow.TaskFailed,
    0,
    nil,
    "处理出错：数据格式不正确",
)
```

## 上下文管理

### 添加上下文

```go
cm := engine.GetContextManager()

// Agent 私有上下文
cm.Add(&workflow.ContextItem{
    Scope:    workflow.Agent,
    Priority: workflow.High,
    Content:  "协议分析结果",
    AgentID:  "protocol-001",
    Metadata: map[string]interface{}{
        "version": "1.0",
    },
})

// 全局共享上下文
cm.Add(&workflow.ContextItem{
    Scope:    workflow.Global,
    Priority: workflow.Critical,
    Content:  "系统配置",
    Metadata: map[string]interface{}{
        "api_version": "v1",
    },
})

// 临时上下文 (1 小时后过期)
cm.Add(&workflow.ContextItem{
    Scope:     workflow.Temporary,
    Priority:  workflow.Low,
    Content:   "缓存数据",
    ExpiresAt: time.Now().Add(time.Hour),
})
```

### 查询上下文

```go
// 获取 Agent 上下文
contexts := cm.GetByAgent("protocol-001")

// 获取任务上下文
contexts := cm.GetByTask("task-001")

// 获取全局上下文
contexts := cm.GetGlobal()

// 获取单个上下文
item, err := cm.Get("context-id-123")
```

### 清理上下文

```go
// 删除单个上下文
cm.Remove("context-id-123")

// 清理 Agent 所有上下文
cm.ClearAgent("protocol-001")
```

## 事件系统

### 订阅事件

```go
eventBus := engine.GetEventBus()

// 订阅单一事件类型
eventBus.Subscribe(workflow.EventTaskCompleted, func(event *workflow.Event) {
    fmt.Printf("任务完成：%s\n", event.Data["task_id"])
})

// 订阅多种事件类型
eventBus.SubscribeMultiple([]workflow.EventType{
    workflow.EventTaskStarted,
    workflow.EventTaskCompleted,
    workflow.EventTaskFailed,
}, func(event *workflow.Event) {
    fmt.Printf("任务事件：%s - %s\n", event.Type, event.Source)
})
```

### 发布事件

```go
eventBus.Publish(&workflow.Event{
    Type:   workflow.EventTaskProgress,
    Source: "custom-001",
    Data: map[string]interface{}{
        "task_id":  "my-task-001",
        "progress": 75,
        "status":   "Processing",
        "details":  "正在处理第 3 批数据",
    },
})
```

### 查询历史事件

```go
// 获取最近 10 条事件
events := eventBus.GetRecentEvents(10)

// 获取特定类型的历史事件
events := eventBus.GetHistory(workflow.EventTaskCompleted, 20)
```

## 监控和调试

### 获取系统统计

```go
stats := engine.GetStats()

// Agent 统计
fmt.Printf("运行 Agents: %d\n", stats.Agents)
for id, state := range stats.AgentStates {
    fmt.Printf("  %s: %s\n", id, state)
}

// 任务统计
fmt.Printf("任务 - 总计：%d, 待处理：%d, 运行中：%d, 已完成：%d\n",
    stats.Tasks.Total, stats.Tasks.Pending, 
    stats.Tasks.Running, stats.Tasks.Completed)

// 事件统计
fmt.Printf("事件 - 总数：%d, 订阅者：%d\n",
    stats.Events.TotalEvents, stats.Events.SubscriberCount)

// 上下文统计
fmt.Printf("上下文 - 项数：%d, 使用率：%d%%\n",
    stats.Contexts.TotalItems, stats.Contexts.UsagePercent)
```

### 工作空间管理

```go
workspace := workflow.NewAgentWorkspace("custom-001", "./docs")

// 初始化
workspace.Initialize()

// 获取信息
info := workspace.GetWorkspaceInfo()
fmt.Printf("任务数：%d, 日志数：%d, 日志大小：%d bytes\n",
    info.TaskCount, info.LogCount, info.LogSize)

// 清理旧日志
workspace.CleanupOldLogs(7) // 保留 7 天
```

## 最佳实践

### 1. Agent 设计原则

- **单一职责**: 每个 Agent 只做一件事
- **无状态**: 状态通过 ContextManager 管理
- **可中断**: 定期检查 `ctx.Done()`
- **可观测**: 频繁更新进度和日志

### 2. 任务设计原则

- **原子性**: 任务应该小而专注
- **可重试**: 为可能失败的任务设置 `AutoRetry`
- **超时**: 总是设置合理的 `TimeoutSeconds`
- **依赖**: 明确声明任务依赖关系

### 3. 上下文使用原则

- **适当作用域**: 能用 Agent 就不用 Global
- **合理优先级**: 关键数据用 High/Critical
- **设置过期**: 临时数据必须设置 `ExpiresAt`
- **定期清理**: 使用 `ClearAgent` 清理离职 Agent

### 4. 事件使用原则

- **轻量级**: 事件数据保持简洁
- **幂等性**: 事件处理函数应该幂等
- **错误处理**: 订阅函数内捕获异常
- **避免循环**: 防止事件触发链形成循环

## 故障排除

### 常见问题

**Q: Agent 不执行任务**
- 检查 Agent 状态是否为 Idle
- 确认任务已分配给该 Agent
- 查看事件日志确认任务分配事件

**Q: 上下文泄漏**
- 检查 Temporary 上下文是否设置过期时间
- 使用 `cm.GetStats()` 监控上下文数量
- 离职 Agent 调用 `cm.ClearAgent(agentID)`

**Q: 任务卡住**
- 检查是否有超时设置
- 查看 Agent 日志确认执行状态
- 检查依赖任务是否完成

**Q: 事件丢失**
- 确认订阅在事件发布前完成
- 增加 `MaxEventHistory` 配置
- 使用 `GetHistory()` 查询历史

### 调试技巧

1. **启用详细日志**:
```go
log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
```

2. **监控关键指标**:
```go
go func() {
    for range time.Tick(5 * time.Second) {
        stats := engine.GetStats()
        log.Printf("Stats: Agents=%d, Tasks=%d, Contexts=%d",
            stats.Agents, stats.Tasks.Total, stats.Contexts.TotalItems)
    }
}()
```

3. **捕获异常**:
```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("Panic recovered: %v", r)
    }
}()
```

## 进阶用法

### 任务依赖链

```go
// 任务 A -> 任务 B -> 任务 C
tasks := []*workflow.Task{
    {
        ID:       "task-a",
        Type:     "step-1",
        Priority: workflow.PriorityHigh,
        AgentID:  "agent-1",
    },
    {
        ID:           "task-b",
        Type:         "step-2",
        Priority:     workflow.PriorityNormal,
        AgentID:      "agent-2",
        Dependencies: []string{"task-a"}, // 依赖 task-a
    },
    {
        ID:           "task-c",
        Type:         "step-3",
        Priority:     workflow.PriorityNormal,
        AgentID:      "agent-3",
        Dependencies: []string{"task-b"}, // 依赖 task-b
    },
}

// 引擎会自动按依赖顺序执行
```

### 并行任务执行

```go
// 创建多个独立任务
for i := 0; i < 10; i++ {
    engine.CreateTask(&workflow.Task{
        ID:       fmt.Sprintf("parallel-task-%d", i),
        Type:     "data-chunk",
        Priority: workflow.PriorityNormal,
        AgentID:  fmt.Sprintf("worker-%d", i%3), // 3 个 worker
    })
}

// 所有任务并行执行
```

### 自定义事件类型

```go
const (
    EventCustomStart workflow.EventType = "custom.start"
    EventCustomDone  workflow.EventType = "custom.done"
)

// 发布自定义事件
eventBus.Publish(&workflow.Event{
    Type:   EventCustomStart,
    Source: "my-agent",
    Data:   map[string]interface{}{"param": "value"},
})

// 订阅自定义事件
eventBus.Subscribe(EventCustomDone, func(event *workflow.Event) {
    // 处理自定义事件
})
```

## 性能优化

### 1. 上下文大小控制

```go
// 限制上下文总数
engine := workflow.NewWorkflowEngine(workflow.WorkflowConfig{
    MaxContexts: 1000, // 根据内存调整
})
```

### 2. 事件历史管理

```go
// 限制历史记录
config.MaxEventHistory = 500

// 定期清理
go func() {
    for range time.Tick(time.Hour) {
        eventBus.ClearHistory()
    }
}()
```

### 3. Agent 池化

```go
// 预创建 Agent 池
agentPool := make(chan agent.Agent, 10)
for i := 0; i < 10; i++ {
    agentPool <- NewWorkerAgent(fmt.Sprintf("worker-%d", i))
}

// 使用时从池中获取
worker := <-agentPool
// ... 使用 worker ...
// 使用后归还
agentPool <- worker
```

## 示例代码

完整示例请参考:
- `cmd/workflow-demo/main.go` - 演示程序
- `docs/agents/ARCHITECTURE.md` - 架构文档

## 支持

遇到问题？
1. 查看 `docs/agents/ARCHITECTURE.md` 了解架构
2. 检查日志输出定位问题
3. 使用 `engine.GetStats()` 诊断状态
4. 参考示例代码 `cmd/workflow-demo/main.go`
