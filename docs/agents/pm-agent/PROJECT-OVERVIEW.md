# Trae Proxy 项目总览

**项目状态**: Phase 0 - 协议逆向工程 (60% 完成)  
**创建日期**: 2026-03-12  
**最后更新**: 2026-03-15 20:30  
**技术栈**: Go 1.22+  
**目标**: 将 Trae CN 的 AI 模型以 OpenAI 兼容 API 形式对外暴露

---

## ? 项目进度

### 整体进度：60%

```
Phase 0: 协议逆向工程     ████████████????????  60%
Phase 1: 核心功能开发     ████????????????????  20%
Phase 2: 测试优化         ????????????????????   0%
Phase 3: 文档部署         ????????????????????   0%
```

### 各模块进度

| 模块 | 负责人 | 进度 | 状态 |
|------|--------|------|------|
| 协议分析 | Protocol-Agent | 60% | ? 进行中 |
| API 层 | API-Agent | 40% | ? 进行中 |
| 认证管理 | Auth-Agent | 30% | ? 进行中 |
| 队列调度 | Queue-Agent | 10% | ? 待开始 |
| 测试 QA | Test-Agent | 0% | ? 待开始 |
| UI 界面 | UI-Agent | 0% | ? 待开始 |

---

## ?? 团队结构

### 7 个 Agent 工程师

```
┌─────────────────────────────────────────┐
│           PM-Agent (项目经理)            │
│  - 任务分配、进度跟踪、协调沟通          │
└─────────────────┬───────────────────────┘
                  │
    ┌─────────────┼─────────────┐
    │             │             │
┌───▼───┐   ┌────▼────┐   ┌───▼───┐
│Protocol│   │  API    │   │ Auth  │
│ -逆向  │   │ -开发   │   │ -认证 │
└───┬───┘   └────┬────┘   └───┬───┘
    │            │            │
    └────────────┼────────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
┌───▼───┐   ┌───▼───┐   ┌───▼───┐
│ Queue │   │ Test  │   │  UI   │
│ -调度 │   │ -测试 │   │ -界面 │
└───────┘   └───────┘   └───────┘
```

---

## ? 任务列表

### ? 高优先级任务

| ID | 任务 | 负责人 | 依赖 | 截止 | 状态 |
|----|------|--------|------|------|------|
| TASK-001 | 完成认证流程逆向 | Protocol-Agent | 无 | 03-16 | ? 60% |
| TASK-010 | 实现 OpenAI 兼容 API 层 | API-Agent | TASK-001 | 03-19 | ? 0% |
| TASK-020 | 实现 Token 管理器 | Auth-Agent | TASK-001 | 03-18 | ? 0% |
| TASK-040 | 编写单元测试 | Test-Agent | TASK-010,020 | 03-20 | ? 0% |
| TASK-041 | 编写集成测试 | Test-Agent | TASK-040 | 03-21 | ? 0% |

### ? 中优先级任务

| ID | 任务 | 负责人 | 依赖 | 截止 | 状态 |
|----|------|--------|------|------|------|
| TASK-002 | SSE 流式响应格式分析 | Protocol-Agent | TASK-001 | 03-17 | ? 0% |
| TASK-011 | 协议转换层实现 | API-Agent | TASK-001,010 | 03-20 | ? 0% |
| TASK-021 | 实现账号池管理 | Auth-Agent | TASK-020 | 03-19 | ? 0% |
| TASK-030 | 实现请求队列管理 | Queue-Agent | TASK-010 | 03-20 | ? 0% |
| TASK-031 | 实现限流和降级 | Queue-Agent | TASK-030 | 03-21 | ? 0% |
| TASK-042 | 压力测试和性能分析 | Test-Agent | TASK-041 | 03-22 | ? 0% |

### ? 低优先级任务

| ID | 任务 | 负责人 | 依赖 | 截止 | 状态 |
|----|------|--------|------|------|------|
| TASK-050 | 实现 Web 管理界面 | UI-Agent | TASK-010 | 03-25 | ? 0% |

---

## ? 里程碑

### M1: 协议逆向完成 (预计 03-17)

- [x] Token 生成函数定位
- [ ] Token 生成算法还原
- [ ] 认证流程文档
- [ ] SSE 格式分析

### M2: 核心功能可用 (预计 03-20)

- [ ] OpenAI 兼容 API 层
- [ ] Token 管理器
- [ ] 协议转换层
- [ ] 基础单元测试

### M3: 生产就绪 (预计 03-25)

- [ ] 队列调度系统
- [ ] 限流降级
- [ ] 完整测试覆盖
- [ ] 性能优化
- [ ] Web 管理界面

---

## ? 项目结构

```
reverse_proxy/
├── cmd/trae-proxy/          # 主程序入口
│   └── main.go
├── internal/
│   ├── agent/               # Agent 实现（待删除）
│   ├── api/                 # API 层（待创建）
│   ├── auth/                # 认证管理 ?
│   ├── config/              # 配置管理 ?
│   ├── device/              # 设备信息 ?
│   ├── models/              # 模型定义 ?
│   ├── openai/              # OpenAI 服务器 ?
│   ├── proxy/               # 代理核心 ?
│   ├── queue/               # 队列管理 ?
│   ├── sse/                 # SSE 处理 ?
│   ├── transformers/        # 协议转换（待创建）
│   └── workflow/            # 工作流（待创建）
├── ui-dev/                  # Web 界面（待开发）
├── docs/
│   ├── agents/              # Agent 工作区
│   │   ├── api-agent/
│   │   ├── auth-agent/
│   │   ├── pm-agent/
│   │   ├── protocol-agent/
│   │   ├── queue-agent/
│   │   ├── test-agent/
│   │   └── ui-agent/
│   ├── knowledge-base/      # 知识库
│   └── PRD.md               # 产品需求文档
├── scripts/                 # 工具脚本
└── tests/                   # 测试文件（待创建）
```

---

## ? 技术架构

### 数据流

```
用户应用 (ChatBox/NextChat)
       │ OpenAI API
       ▼
┌──────────────────────────┐
│   API 层 (OpenAI 兼容)     │
│   - /v1/models           │
│   - /v1/chat/completions │
│   - /v1/completions      │
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────┐
│   协议转换层              │
│   - OpenAI → Trae CN     │
│   - Trae CN → OpenAI     │
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────┐
│   认证管理                │
│   - Token 轮询           │
│   - 自动刷新             │
│   - 账号健康             │
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────┐
│   队列调度                │
│   - 优先级队列           │
│   - 限流降级             │
│   - 并发控制             │
└──────────┬───────────────┘
           │
           ▼
┌──────────────────────────┐
│   Trae CN 后端            │
│   - 字节跳动 AI 服务       │
└──────────────────────────┘
```

---

## ? 当前状态详情

### Protocol-Agent (60%)

**进行中**:
- ? Token 生成函数定位
- ? 关键参数字段提取
- ? 认证 Header 识别
- ? Token 生成算法逆向
- ? Token 刷新机制分析

**下一步**:
- 完成 Token 生成脚本
- 编写认证流程文档
- 分析 SSE 流式格式

### API-Agent (40%)

**已完成**:
- ? HTTP 服务器框架
- ? 路由注册
- ? `/v1/models` 端点
- ? CORS 中间件

**待完成**:
- ? 完整的请求/响应转换
- ? 流式 SSE 处理
- ? 错误码映射
- ? API Key 认证

### Auth-Agent (30%)

**已完成**:
- ? TokenProvider 基础
- ? 多账号轮询
- ? Token 存储加载

**待完成**:
- ? Token 自动刷新
- ? 账号健康管理
- ? 加密存储
- ? 认证中间件

### Queue-Agent (10%)

**已完成**:
- ? Monitor 基础

**待完成**:
- ? 优先级队列
- ? 调度器
- ? 并发控制
- ? 限流降级

### Test-Agent (0%)

**待开始**:
- ? 单元测试
- ? 集成测试
- ? 压力测试

### UI-Agent (0%)

**待开始**:
- ? Web 界面设计
- ? 前端开发
- ? API 集成

---

## ? 快速开始

### 开发环境

```bash
# 1. 克隆仓库
git clone <repo-url>
cd reverse_proxy

# 2. 安装依赖
go mod download

# 3. 配置 Token（可选）
cp config.example.json config.json
# 编辑 config.json 填入 Token

# 4. 运行
go run cmd/trae-proxy/main.go

# 5. 测试 API
curl http://localhost:9090/v1/models
```

### 测试

```bash
# 运行单元测试
go test ./...

# 运行集成测试
go test ./tests/integration/...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## ? 开发规范

### Git 工作流

```bash
# 功能分支
git checkout -b feature/TASK-001

# 提交信息
git commit -m "feat: 实现 Token 管理器 (TASK-020)"
git commit -m "fix: 修复 Token 刷新逻辑 (TASK-020)"
git commit -m "test: 添加 Token 单元测试 (TASK-040)"

# 合并请求
git push origin feature/TASK-020
# 创建 Pull Request
```

### 代码规范

- 遵循 Go 官方代码风格
- 所有公开函数必须有文档注释
- 单元测试覆盖率 >= 80%
- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码质量

---

## ? 沟通协作

### 每日站会

- **时间**: 每天 09:30
- **内容**:
  - 昨天完成了什么
  - 今天计划做什么
  - 有什么阻碍

### 任务分配

PM-Agent 负责任务分配，各 Agent 根据任务卡片执行：

1. 查看分配给自己的任务
2. 更新任务状态（Not Started → In Progress → Completed）
3. 提交工作成果
4. 请求验收

### 问题上报

遇到问题时：
1. 查看相关文档
2. 在任务卡片中评论
3. @相关 Agent 协助
4. 升级到 PM-Agent

---

## ? 项目风险

### 技术风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| Token 生成算法复杂 | 中 | 高 | 多方案并行研究 |
| 字节跳动升级防护 | 中 | 高 | 持续监控和适配 |
| 并发性能瓶颈 | 低 | 中 | 提前压力测试 |
| 协议变更频繁 | 高 | 中 | 建立快速响应机制 |

### 进度风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| Protocol-Agent 延期 | 中 | 高 | 提前启动，留缓冲 |
| 测试资源不足 | 低 | 中 | 使用 Mock 和容器 |
| UI 开发复杂度高 | 中 | 低 | 使用成熟组件库 |

---

## ? 学习资源

### 协议逆向

- [Electron 应用逆向工程](https://www.electronjs.org/docs/latest/)
- [MITM 抓包教程](https://mitmproxy.org/)
- [JavaScript 反混淆](https://github.com/javascript-obfuscator/javascript-obfuscator)

### Go 开发

- [Go 官方文档](https://golang.org/doc/)
- [Go Web 编程](https://github.com/astaxie/build-web-application-with-golang)
- [Go 并发模式](https://github.com/golang-standards/project-layout)

### API 设计

- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [RESTful API 最佳实践](https://restfulapi.net/)
- [SSE 规范](https://html.spec.whatwg.org/multipage/server-sent-events.html)

---

## ? 许可证

MIT License - 详见 [LICENSE](LICENSE)

---

**最后更新**: 2026-03-15 20:30  
**下次更新**: 2026-03-16 09:30 (每日站会后)  
**维护人**: PM-Agent
