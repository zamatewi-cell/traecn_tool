# Changelog / 变更日志

本项目的所有重要变更均记录在此文件中。
格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/)。

---

## [1.0.2] - 2026-09-20

### 概述
v1.0.2 是一个重要的系统稳定性、进程生命周期与协议韧性加固补丁，全面闭环了审计报告中确认的 **全部 10 项 P2 级缺陷**（覆盖生命周期竞态、凭据脱敏防混淆、协议流截断处理、客户端取消链路透传、Anthropic 并行工具规整、凭据池唯一主键隔离、熔断器计数收敛、SQLite 数据路径治理、CI/CD 动态版本注入以及活动账号优先级调度）。

### 修复 (Fixed)
- **桌面代理进程生命周期治理与竞态防御 ([P2-1])**：
  - 在 Electron 主进程引入串行生命周期队列锁（`runWithProxyLock`），杜绝并发快速重启/启停导致的竞态；
  - 终结代理子进程增加实例比对守卫（`proxyProcess === proc`），彻底消除旧进程 `exit` 异步回调误将新拉起子进程句柄置为 null 导致的孤儿进程失控问题；
  - 补充对子进程 `error` 与 `stdin.error` 的异常捕获监听，增加 TCP 端口 readiness 探活探测，确保代理服务完全就绪后再向前端报告成功。
- **脱敏凭据导入强拦截与全量备份恢复解耦 ([P2-2])**：
  - 账号管理页导出脱敏数据时增加 `_exportType: 'sanitized_accounts_export'` 元数据，导入时增加 `isMaskedSecret` 强正则校验，严格阻断将带星号掩码的脱敏占位符作为真实 Token 存入凭据库；
  - 设置页导出全量配置时标记 `_backupType: 'full_backup'`，解耦保存文件对话框标题；
  - 设置页新增「从备份恢复数据」功能，支持一键安全还原账号列表与代理网络配置。
- **协议流截断感知与 incomplete 状态对齐 ([P2-3])**：
  - AgentTask 流式传输扩充结构化 `AgentTurnCompletionEvent`，在上游未发送结束帧即提前断开（EOF）时坚决返回 `io.ErrUnexpectedEOF`，拒绝向客户端伪造 `stop` 成功；
  - 消除协议层默认初始化的假 `finish_reason: "stop"`；
  - 对齐 OpenAI Responses 协议，当上游因 `length` / `max_tokens` 截断时，统一将响应对象及流式终止状态映射为 `status: "incomplete"` 并携带 `incomplete_details: {"reason": "max_output_tokens"}`。
- **客户端断开/取消全链路透传 ([P2-4])**：
  - 并发限流器 `Limiter` 扩充 `AcquireContext(ctx)` 机制，感知客户端 Context 取消并即刻释放槽位，杜绝队头死锁与排队槽位永久阻塞；
  - 入口网关处理程序全面绑定 `r.Context()` 至内部上游请求对象；
  - `doChatCompletion` 与 `doAgentTaskCompletion` 全面迁移至 `http.NewRequestWithContext(ctx, ...)`，确保客户端断开连接时立即向 Trae CN 上游传播取消信号，停止空耗用户配额；
  - 流式退出时通过 `defer` 及时清空排队状态。
- **Anthropic 并行工具碎片原子规整 ([P2-5])**：
  - 引入流式工具调用累加器 `streamToolCallAccumulator`，缓冲并发到达的多个工具调用片段；
  - 在流结束前通过 `flushToolCalls` 严格按 ContentBlock 规范原子化连续下发 `Block(N) Start -> Delta -> Stop` 事件，避免提前关闭 block 0 导致向已关闭块写入增量的协议违背，并将尾部 `stop_reason` 正确映射为 `"tool_use"`。
- **凭据池同名账号唯一主键隔离 ([P2-6])**：
  - `AccountConfig`、`Account` 与 `AccountInfo` 统一增加 `ID` 字段；
  - 网关上报上游鉴权状态时，`ReportFailure` 与 `ReportSuccess` 优先基于账号稳定唯一的 `ID` 进行索引，消除同名（如两个 label 均为 `default`）账号之间的误杀与误停用。
- **熔断器计数收敛与真实上游状态对齐 ([P2-7])**：
  - 彻底删除 `GetToken()` 内部第 304 行盲目调用 `acc.breaker.RecordSuccess()` 的逻辑缺陷，改由网关在真实收到上游成功响应后显式调用 `ReportSuccess` 反馈；
  - 确保连续 401 认证失败能够正确累加 failure 计数并在达到阈值时触发熔断器进入 `Open` 状态，阻断死循环空耗。
- **SQLite 数据库持久化路径锁定 ([P2-8])**：
  - Go 代理核心新增 `-db-path` 命令行参数支持；
  - Electron 启动子进程时显式传入 `-db-path <userData>/data/trae_proxy.db` 并锁定子进程 `cwd: app.getPath('userData')`，彻底消除不同快捷方式与启动工作目录漂移导致的数据库分散问题。
- **CI/CD 桌面构建链与版本动态注入 ([P2-9])**：
  - `.github/workflows/build.yml` 中 `build-desktop` 作业改用 PowerShell 动态从 Git Tag 解析版本号，并通过 `-ldflags` 动态注入内嵌 `trae-proxy.exe`，解除 `1.0.0` 硬编码；
  - 桌面客户端前端设置页通过 IPC 动态获取应用版本号，消除纯静态硬编码。
- **当前活动账号（Active Account）优先级调度 ([P2-10])**：
  - 配置层与凭据池引入 `ActiveAccountID` 与 `is_current` 标记；
  - `GetToken()` 实现优先调度逻辑：配置了活动账号且该账号健康可用时 100% 优先派发；活动账号熔断或失效时自动平滑降级（fallback）至其余健康账号的 Round-Robin 调度并输出告警。

---

## [1.0.1] - 2026-09-20

### 概述
v1.0.1 是一个关键的安全与协议规范紧急热修复补丁（Hotfix），全面闭环了协议映射、流式规约、凭据生命周期回写、访问控制鉴权与进程治理领域的 6 项 P1 级缺陷。

### 修复 (Fixed)
- **AgentTask 工具调用主动防御 ([P1-1])**：
  - 针对 AgentTask 转发通道目前尚未支持工具语义的问题，增加前置安全守卫 `ValidateAgentTaskRequest`；
  - 当客户端在 AgentTask 模式下携带 `tools`、`tool_choice` 或工具调用历史时，在扣减凭据前立即拒绝并返回标准 HTTP 400（`unsupported_channel_feature`），坚决阻断因静默丢弃工具导致的无限循环调用与配额浪费。
- **OpenAI Responses 流式规范补齐与 ID 稳定 ([P1-2])**：
  - 严格对齐 OpenAI Responses 协议规约，流式分块中补齐 `content_part.done`（携带完整 `part` 内容）与 `output_item.done` 终态事件；
  - 针对 `output_item.done` 实现多态组装：`reasoning` 输出项完整组装 `summary` 且严格排除 role/content；`function_call` 组装完整 arguments；`message` 组装完整 role 与 content；
  - 引入 `stableIDs` 映射机制，确保流式生命周期内 `output_item.added` 与 `output_item.done` 间的 ID 严格一致；
  - 调整终态事件时序，严格遵循先 `response.done` 后 `response.completed` 派发。
- **凭据无盘回写与生命周期闭环 ([P1-3])**：
  - 配置层扩充 `RefreshToken`、`ExpiresAt` 等凭据持久化字段；
  - Go 代理池成功刷新 Token 后，通过受保护的标准输出流向 Electron 发射结构化事件帧 `__TRAE_EVENT__:<json>`；
  - Electron 主进程采用 `readline` 逐行解析捕获凭据更新事件，通过 DPAPI 安全持久化至加密存储，并通过 `account-updated` 实时同步渲染进程与配置状态，杜绝明文日志泄露。
- **API 鉴权空密钥短路安全阻断 ([P1-4])**：
  - 消除空 API Key 导致鉴权绕过的严重安全隐患；
  - Electron 启动前校验强制要求开启鉴权时必须配置非空密钥；
  - Go 代理服务启动参数严格阻断显式空密钥 `-api-key ""` 并清洗有效 Key 列表；
  - 中间件层严格清洗空白字符并拒绝空串 Bearer Token，规范返回 HTTP 401。
- **显式控制防止意外探测开发者本机私钥 ([P1-5])**：
  - 引入显式配置项 `AutoDiscover *bool`，明确区分“显式空账号列表”与“自动探测本地 Trae 凭据”两种语义；
  - 当配置了空账号列表且未显式声明开启自动探测时，报错退出并阻止向 Trae 默认目录嗅探开发者未知私钥。
- **清除数据残留幽灵进程彻底治理 ([P1-6])**：
  - 设置页面“清除数据”操作增加强同步异步前置守卫，必须等待后台代理子进程完全终止成功后方可执行数据清除与 UI 重置；
  - Windows 环境下通过 `taskkill /pid <PID> /T /F` 强制杀死完整子进程树，杜绝后台幽灵进程占用端口或持续运行。

---

## [1.0.0] - 2026-09-19

### 概述
Trae CN 转 OpenAI 代理网关与桌面客户端 (`traecn_tool`) 正式迈入 **v1.0.0 生产级里程碑**！
本版本实现了从早期的实验性本地协议解密工具，向工业级双模态代理网关（独立 CLI/Daemon + Electron 桌面端）的全面跃升，覆盖跨平台自动化构建、主流 AI 编程工具无缝接入、桌面端双形态分发与生产级安全防呆架构。

### 新增 (Added)
- **多模型统一路由与注册表**：
  - 打通 Trae IDE 内置传输协议与本地通道，注册并完整支持 16 个全新一代大语言模型（如 `DeepSeek-V4.1-Flash`、`Seed-Code`、`GLM-5.3`、`Kimi-K3`、`Qwen3.7-Plus` 等）；
  - 完美兼容存量 5 个经典预设模型（`seed_m8`、`Doubao_1_5_thinking_pro`、`deepseek-R1`、`deepseek-V3`、`deepseek-V3-0324`），网关自动按模型类型进行上游通道分发。
- **深度思考链流式分流**：
  - 针对推理类模型（如 DeepSeek R1 / V4-Pro），支持 `<think>` 标签跨分片切割，并在流式响应中将思考过程通过 `delta.reasoning_content` 进行实时独立分流输出。
- **单文件内嵌现代化 WebUI 仪表盘**：
  - 服务端内嵌自包含暗黑主题管理控制台（`/dashboard`），零外部网络 CDN 依赖；
  - 提供即时健康状态探测、API Key 一键生成与管理、21 个模型全景矩阵筛选；
  - 在线流式 Playground 调试面板，实时监控首字延迟 (TTFT)、吐字速率 (TPS)、提示词缓存命中率 (Prompt Cache) 与上下文窗口占用；
  - 本地 SQLite 会话流水与 Trae 官方云端账单双视图分页查看。
- **自动化跨平台 CI/CD 流水线**：
  - 建立 GitHub Actions 持续集成与发布工作流 (`.github/workflows/build.yml`)；
  - 包含 Go 1.25 自动化测试与前端 Vite 编译双重质量门禁；
  - 基于纯 Go（零 CGO 依赖）在 `ubuntu-latest` 上并行交叉编译全部 6 大操作系统/芯片架构（`windows/amd64`, `windows/arm64`, `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`）；
  - 针对版本标签推送自动产出 Release 资产包，并自动计算全量 SHA256 校验和清册 (`checksums.txt`)。
- **Windows 桌面端双模态打包分发**：
  - 支持生成标准的 NSIS 安装向导程序 (`TraeCN.Tools_1.0.0_x64-setup.exe`) 与免安装即开即用的便携版程序 (`TraeCN.Tools_1.0.0_x64-portable.exe`)；
  - 桌面客户端内嵌网关进程生命周期守护，实现随应用自启与退出自动回收。

### 安全与防呆 (Security)
- **Localhost 环回绑定防呆**：网关默认严格绑定本地环回地址（`127.0.0.1` / `localhost`），防止意外暴露于外部网络。
- **局域网暴露防呆拦截 (LAN Guard)**：
  - 增加启动参数 `-allow-lan`；
  - 当尝试绑定非本地环回地址或启用局域网共享时，强制要求必须配置高强度 API Key，未配置密钥时坚决拒绝启动。
- **跨站 CSRF 攻击防护**：
  - 在无 Key 本地开发运行模式下，网关内置 Origin 白名单拦截机制，拦截来自非受信公网域名的跨站偷渡请求（HTTP 403），杜绝恶意网页在后台静默发起对话或探测本地资产。
- **凭据零落盘与安全存储**：
  - Trae 本地凭据通过内存管道在主进程与子进程间按需注入，杜绝明文写入临时文件；
  - 支持多账号轮换与令牌过期前 5 分钟自动刷新。

### 变更与修复 (Changed & Fixed)
- **桌面端 UI 全面收口清理**：
  - 排查并修复了 28 处交互缺陷，彻底下线了缺乏后端支撑的假页面（`Stats.tsx`、`Tokens.tsx`、`IpManagement.tsx`）；
  - 修复了 `Logs.tsx` 中因对象属性解析错误导致的白屏崩溃隐患，重构为流式文本终端日志视图；
  - 为所有按钮与下拉框补齐真实的交互与持久化响应。
- **文档与编码全链路规范化**：
  - 修正了历史文档中的旧版本遗留，统一将版本号定义为 `v1.0.0`；
  - 统一全工程前端源码与 Markdown 文档为标准 UTF-8（无 BOM）编码。
- **测试用例全量通过**：
  - 涵盖协议编解码、账号池、设备指纹、SQLite 持久化、SSE 流式解析及端到端安全验证在内的全部 32 个 Go 测试套件 100% 保持绿灯通过。
