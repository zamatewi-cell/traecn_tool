# 多 Agent 工作流系统架构文档

## 概述

本系统实现了一个自动化、多 Agent 协作的工作流优化框架，支持并行执行、上下文隔离、任务自主跟踪和跨 Agent 进度感知。

## 核心组件

### 1. ContextManager (上下文管理器)

**位置**: `internal/workflow/context_manager.go`

**职责**:
- 提供严格的上下文隔离机制
- 支持四种作用域：Global、Agent、Task、Temporary
- 基于优先级的自动清理 (Low=1, Normal=5, High=10, Critical=100)
- LRU 风格的访问追踪和过期淘汰
- 线程安全的读写操作 (RWMutex)

**使用示例**:
```go
cm := NewContextManager(1000) // 最大 1000 条上下文

// 添加 Agent 上下文
item := &ContextItem{
    Scope:    Agent,
    Priority: High,
    Content:  "协议分析结果...",
    AgentID:  "protocol-agent",
}
cm.Add(item)

// 获取特定 Agent 的上下文
contexts := cm.GetByAgent("protocol-agent")

// 获取统计信息
stats := cm.GetStats()
```

### 2. TaskManager (任务管理器)

**位置**: `internal/workflow/task_manager.go`

**职责**:
- 任务生命周期管理 (创建、更新、取消、重试)
- 优先级调度 (Background=1, Normal=5, High=10, Urgent=100)
- 任务状态追踪 (Pending, Running, Paused, Completed, Failed, Cancelled)
- 按 Agent 和状态分类的任务队列
- 持久化到文件系统

**任务状态机**:
```
Pending → Running → Completed
              ↓
              Failed → (Retry) → Pending
              ↓
              Cancelled
```

**使用示例**:
```go
tm := NewTaskManager("./docs")

task := &Task{
    ID:          "task-001",
    Type:        "protocol-analysis",
    Description: "分析 Trae CN 认证协议",
    Priority:    PriorityHigh,
    AgentID:     "protocol-agent",
    MaxRetries:  3,
}

tm.CreateTask(task)
tm.UpdateTaskStatus(task.ID, TaskRunning, 50, nil, "")
```

### 3. EventBus (事件总线)

**位置**: `internal/workflow/event_bus.go`

**职责**:
- 发布/订阅模式的事件系统
- 支持多种事件类型 (任务、Agent、系统)
- 异步事件分发
- 历史记录管理
- 跨 Agent 通信桥梁

**事件类型**:
- **任务事件**: task.created, task.started, task.progress, task.completed, task.failed, task.cancelled
- **Agent 事件**: agent.started, agent.stopped, agent.paused, agent.resumed, agent.error
- **系统事件**: system.shutdown, system.config

**使用示例**:
```go
eb := NewEventBus(1000)

// 订阅任务完成事件
eb.Subscribe(EventTaskCompleted, func(event *Event) {
    log.Printf("任务完成：%s", event.Source)
})

// 发布事件
eb.Publish(&Event{
    Type:   EventTaskProgress,
    Source: "protocol-agent",
    Data: map[string]interface{}{
        "task_id":  "task-001",
        "progress": 75,
    },
})
```

### 4. BaseAgent (基础 Agent)

**位置**: `internal/agent/base.go`

**职责**:
- 提供 Agent 基础实现
- 状态管理 (Idle, Running, Paused, Completed, Failed)
- 进度追踪
- 上下文访问接口
- 事件发布集成

**Agent 接口**:
```go
type Agent interface {
    GetID() string
    GetName() string
    GetType() string
    Initialize() error
    Run(ctx context.Context, task *Task) error
    Shutdown() error
    GetState() AgentState
    GetProgress() Progress
    GetCurrentTask() *Task
    SetContextManager(cm *ContextManager)
    GetContextManager() *ContextManager
}
```

**使用示例**:
```go
agent := NewBaseAgent("protocol-001", "ProtocolAnalyzer", "protocol", "./docs")
agent.Initialize()
agent.SetContextManager(cm)

// 更新进度
agent.UpdateProgress(50, "Analyzing", "处理认证协议...")

// 添加上下文
agent.AddContext(Agent, High, "协议分析结果", map[string]interface{}{
    "endpoint": "/api/chat",
})
```

### 5. WorkflowEngine (工作流引擎)

**位置**: `internal/workflow/engine.go`

**职责**:
- Agent 注册和生命周期管理
- 任务分配和调度
- 自动任务分配循环
- 组件协调 (ContextManager, TaskManager, EventBus)
- 系统统计和监控

**使用示例**:
```go
engine := NewWorkflowEngine(WorkflowConfig{
    WorkDir:       "./docs",
    MaxAgents:     10,
    MaxTasks:      100,
    MaxContexts:   1000,
    MaxEventHistory: 1000,
})

// 注册 Agent
engine.RegisterAgent(protocolAgent)
engine.RegisterAgent(apiAgent)

// 创建任务
engine.CreateTask(&Task{
    ID:       "task-001",
    Type:     "protocol-analysis",
    AgentID:  "protocol-agent",
    Priority: PriorityHigh,
})

// 启动引擎
engine.Start()

// 获取统计
stats := engine.GetStats()
```

## 目录结构

```
docs/
└── agents/
    ├── protocol-agent/
    │   ├── config.yaml       # Agent 配置
    │   ├── tasks/            # 任务记录
    │   │   ├── task_001.json
    │   │   └── task_002.json
    │   ├── logs/             # 执行日志
    │   └── state.json        # 状态快照
    ├── api-agent/
    │   └── ...
    └── queue-agent/
        └── ...
```

## 工作流示例

### 场景：多 Agent 协作分析 Trae CN 协议

```go
// 1. 初始化引擎
engine := NewWorkflowEngine(WorkflowConfig{
    WorkDir: "./docs",
    MaxAgents: 5,
    MaxContexts: 1000,
})

// 2. 创建专业化 Agent
protocolAgent := NewProtocolAgent("protocol-001")
apiAgent := NewAPIAgent("api-001")
queueAgent := NewQueueAgent("queue-001")

// 3. 注册 Agent
engine.RegisterAgent(protocolAgent)
engine.RegisterAgent(apiAgent)
engine.RegisterAgent(queueAgent)

// 4. 创建任务链
tasks := []*Task{
    {
        ID: "task-protocol",
        Type: "protocol-reverse",
        Description: "逆向分析 Trae CN 协议",
        Priority: PriorityHigh,
        AgentID: "protocol-001",
    },
    {
        ID: "task-api",
        Type: "api-wrapper",
        Description: "构建 OpenAI 兼容 API",
        Priority: PriorityNormal,
        AgentID: "api-001",
        Dependencies: []string{"task-protocol"}, // 依赖协议分析完成
    },
}

for _, task := range tasks {
    engine.CreateTask(task)
}

// 5. 订阅事件以监控进度
engine.GetEventBus().Subscribe(EventTaskCompleted, func(event *Event) {
    log.Printf("任务完成：%s by %s", event.Data["task_id"], event.Source)
})

// 6. 启动引擎
engine.Start()

// 7. 监控统计
for {
    stats := engine.GetStats()
    log.Printf("运行中：%d Agents, %d 待处理，%d 运行中", 
        stats.Agents, stats.Tasks.Pending, stats.Tasks.Running)
    time.Sleep(5 * time.Second)
}
```

## 上下文隔离机制

### 作用域层级

1. **Global**: 全局共享上下文 (所有 Agent 可访问)
2. **Agent**: Agent 私有上下文 (仅该 Agent 可访问)
3. **Task**: 任务特定上下文 (仅任务执行期间有效)
4. **Temporary**: 临时上下文 (自动过期)

### 优先级清理策略

当上下文数量达到上限时:
1. 首先清理过期的 Temporary 上下文
2. 然后按优先级从低到高清理 (Low → Normal → High)
3. 同优先级下，清理最少访问的 (LRU)
4. 每次清理 10% 的容量

### 访问追踪

每个上下文项记录:
- `AccessCount`: 访问次数
- `LastAccessed`: 最后访问时间
- `CreatedAt`: 创建时间
- `ExpiresAt`: 过期时间 (Temporary 类型)

## 任务调度策略

### 优先级队列

任务按优先级排序，高优先级任务优先分配:
- **Urgent (100)**: 紧急任务，立即处理
- **High (10)**: 高优先级任务
- **Normal (5)**: 普通任务
- **Background (1)**: 后台任务，空闲时处理

### 自动分配

引擎每秒检查:
1. 待处理任务队列
2. 空闲 Agent 列表
3. 任务依赖关系
4. 自动分配匹配的任务给空闲 Agent

### 重试机制

支持自动重试的任务:
- 设置 `AutoRetry: true`
- 配置 `MaxRetries: 3`
- 失败后自动重新加入待处理队列
- 记录重试次数

## 跨 Agent 通信

### 事件驱动架构

Agent 通过 EventBus 进行松耦合通信:

```go
// Agent A 发布进度
eventBus.Publish(&Event{
    Type: EventTaskProgress,
    Source: "protocol-agent",
    Data: map[string]interface{}{
        "task_id": "task-001",
        "progress": 80,
        "result": "协议分析完成",
    },
})

// Agent B 订阅并响应
eventBus.Subscribe(EventTaskProgress, func(event *Event) {
    if event.Data["task_id"] == "task-001" {
        // 准备接收协议分析结果
        prepareForNextStep()
    }
})
```

### 共享上下文

通过 Global 作用域共享关键信息:

```go
// Agent A 写入全局上下文
cm.Add(&ContextItem{
    Scope: Global,
    Priority: Critical,
    Content: "协议规范 v1.0",
    Metadata: map[string]interface{}{
        "version": "1.0",
        "author": "protocol-agent",
    },
})

// Agent B 读取全局上下文
globalContexts := cm.GetGlobal()
```

## 监控和调试

### 统计信息

```go
stats := engine.GetStats()

// Agent 统计
fmt.Printf("Agents: %d\n", stats.Agents)
for id, state := range stats.AgentStates {
    fmt.Printf("  %s: %s\n", id, state)
}

// 任务统计
fmt.Printf("Tasks: %d total, %d pending, %d running, %d completed, %d failed\n",
    stats.Tasks.Total, stats.Tasks.Pending, stats.Tasks.Running,
    stats.Tasks.Completed, stats.Tasks.Failed)

// 事件统计
fmt.Printf("Events: %d total, %d subscribers\n",
    stats.Events.TotalEvents, stats.Events.SubscriberCount)

// 上下文统计
fmt.Printf("Contexts: %d items, %d%% usage\n",
    stats.Contexts.TotalItems, stats.Contexts.UsagePercent)
```

### 日志记录

所有关键操作都有日志输出:
- Agent 生命周期事件
- 任务状态变更
- 上下文操作
- 事件发布/订阅

## 最佳实践

### 1. 上下文使用

? **推荐**:
- 使用适当的优先级 (Critical 用于关键数据)
- 为临时数据设置过期时间
- 定期清理不需要的上下文

? **避免**:
- 滥用 Global 作用域
- 不设置过期时间的 Temporary 上下文
- 忽略上下文大小限制

### 2. 任务设计

? **推荐**:
- 明确任务依赖关系
- 设置合理的超时时间
- 为关键任务启用自动重试

? **避免**:
- 循环依赖
- 无限重试
- 过长的任务执行时间

### 3. Agent 协作

? **推荐**:
- 通过事件总线通信
- 使用全局上下文共享关键信息
- 保持 Agent 职责单一

? **避免**:
- Agent 间直接耦合
- 过度共享上下文
- 一个 Agent 承担过多职责

## 扩展指南

### 创建专用 Agent

```go
type ProtocolAgent struct {
    *agent.BaseAgent
}

func NewProtocolAgent(id string) *ProtocolAgent {
    return &ProtocolAgent{
        BaseAgent: agent.NewBaseAgent(id, "ProtocolAnalyzer", "protocol", "./docs"),
    }
}

func (a *ProtocolAgent) Run(ctx context.Context, task *workflow.Task) error {
    // 调用父类 Run
    if err := a.BaseAgent.Run(ctx, task); err != nil {
        return err
    }

    // 实现具体逻辑
    a.UpdateProgress(10, "Starting", "开始协议分析...")
    
    // ... 实际工作 ...
    
    a.UpdateProgress(100, "Completed", "协议分析完成")
    return nil
}
```

### 自定义事件类型

```go
const (
    EventProtocolAnalyzed EventType = "protocol.analyzed"
    EventAPIReady         EventType = "api.ready"
)

// 发布自定义事件
eventBus.Publish(&Event{
    Type: EventProtocolAnalyzed,
    Source: "protocol-agent",
    Data: map[string]interface{}{
        "protocol_version": "1.0",
        "endpoints": []string{"/api/chat", "/api/completions"},
    },
})
```

## 故障排除

### 常见问题

1. **上下文泄漏**: 检查是否正确清理 Temporary 上下文
2. **任务卡住**: 检查超时设置和依赖关系
3. **Agent 无响应**: 检查 Agent 状态和错误日志
4. **事件丢失**: 检查订阅时机和历史记录大小

### 调试技巧

- 启用详细日志记录
- 监控统计信息变化
- 检查事件历史记录
- 审查上下文使用情况
