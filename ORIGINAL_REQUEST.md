# Original User Request

## 2026-09-17T06:50:11Z

在保留现有 5 个预设模型兼容性的基础上，攻坚 Trae CN 底层 aha 传输协议与本地 ZMQ/IPC 管道通信机制，实现 Trae IDE 16 个新一代内置模型（如 Seed-2.1、GLM-5.3、DeepSeek-V4.1-Flash、Kimi-K3 等）的 OpenAI 兼容 API 代理转发。

Working directory: d:\codelearn\vscode\reverse_proxy
Integrity mode: development

## Reference Material & Resources
- 现有打通成果：internal/proxy/masticate.go（AES-256-GCM 算法）、internal/proxy/proxy.go
- 抓包分析样本：scripts/captured/agent_task_req_body.bin（魔数 F3 C1 C0 37，长度 223KB）、agent_task_resp.bin、flows.bin
- 客户端核心代码参考：D:\Trae CN\resources\app\extensions\ai-completion\resource\aiserver\cueMain.js、server.js、main.js（含 ZmqClient、NetworkProxy、@aha-kit/net）
- 探测与实验脚本：scripts/probe4023/main.go、scripts/probe_endpoints/main.go

## Requirements

### R1. 存量预设模型通道无损保留
现有基于 /api/ide/v1/llm_raw_chat 与 masticate 加密的 5 个预设模型（seed_m8、Doubao_1_5_thinking_pro、deepseek-R1、deepseek-V3、deepseek-V3-0324）保持完全可用，不得破坏现有单元测试与端到端代理能力。

### R2. 本地 ZMQ / IPC 管道桥接代理（阶段一）
探测并连接本地运行中 Trae IDE 暴露的 ZMQ / IPC 管道（参考环境变量 TRAE_NET_ZMQ_ENDPOINT 或本地端口如 51000/49201，以及命名管道 aha_doctor_ipc_server_*、agent-code-toolhost-*），提取客户端通信契约，实现代理对本地 Trae 传输通道的借道转发。

### R3. aha 二进制传输协议逆向（阶段二）
分析 agent_task_req_body.bin 中以 F3 C1 C0 37 魔数开头的二进制帧结构及其对应的响应帧 agent_task_resp.bin，解析外层协议包头、序列化格式（Protobuf/FlatBuffers/二进制压缩）与加密分层，为脱机完全独立运行提供底层支持。

### R4. 统一网关与模型路由整合
将新打通的通道整合入 trae-proxy 网关：
- 将 16 个新内置模型注册进注册表（包含 DeepSeek-V4.1-Flash、Seed-Code、GLM-5.3 等）；
- 针对用户请求的模型名称智能路由到对应通道（预设模型走现有 HTTPS，内置新模型走 aha/ZMQ 通道）；
- 保持标准的 /v1/chat/completions 与 /v1/models 端点接口，支持流式 SSE 增量输出与错误平滑转换。

## Acceptance Criteria

### 1. 存量功能回归测试
- [ ] 执行 go test ./... 全部通过，现有测试用例 100% 保持绿色。

### 2. 新模型实测通路与流式响应
- [ ] 至少针对 2 个新内置模型（例如 DeepSeek-V4.1-Flash 与 Seed-Code / GLM-5.3）发起实际对话测试。
- [ ] 返回状态码为 HTTP 200，无 4023 MODEL_NOT_EXISTED 或 the model is unknown 错误。
- [ ] 在流式（stream: true）模式下，能正确输出增量内容（SSE chunks）并以 [DONE] 正常结束。

### 3. 模型列表与智能路由验证
- [ ] 调用 /v1/models 能列出包含新内置模型的完整清单。
- [ ] 当请求新模型与旧预设模型时，网关能自动正确派发至对应的上游后端通道。

## 2026-09-19T02:56:04Z

将 Trae CN 转 OpenAI 代理网关与桌面客户端 (traecn_tool) 全面推进至 v1.0.0 生产发布状态，涵盖跨平台 CI/CD 自动化构建、Electron 独立打包验证（NSIS 安装包与免安装 Portable）、占位与假按钮清理、高质量开源 README 与 Release 资产发布。

Working directory: d:\codelearn\vscode\reverse_proxy
Integrity mode: development

## Requirements

### R1. 跨平台 CI/CD 自动化流水线 (.github/workflows/build.yml)
- 搭建生产级 GitHub Actions 持续集成与发布流水线，监听 push (main 分支)、pull_request 以及 v* 标签推送；
- 设立自动化测试门禁：包含 Go 单元测试矩阵（go test ./...）与前端 Vite 编译校验（npm --prefix traecn-tools run build）；
- Go 核心代理（trae-proxy）跨平台交叉编译：自动化构建出 Windows (x64/arm64)、macOS (Intel/Apple Silicon)、Linux (x64/arm64) 独立二进制可执行文件与压缩包；
- 自动化打包 Windows 桌面客户端（NSIS Setup 安装包与 Portable 便携式单文件）；
- 当推送版本标签（如 v1.0.0）时，自动创建 GitHub Release，上传所有架构产物，并附带 SHA256 校验和清单（checksums.txt）。

### R2. Electron 桌面端 Clean Checkout 与构建分发验证
- 完善 traecn-tools/electron-builder.yml，增加 portable 免安装目标，确保产出 Setup.exe 与 Portable.exe；
- 校验在全新克隆（clean checkout）与自动化构建环境下，依赖安装无缺失、无开发机绝对路径残留、依赖的 trae-proxy.exe 能正确被打包进 resources/bin；
- 验证打包后的桌面应用能平稳拉起后台 Go 代理服务，通信与安全鉴权一切正常。

### R3. 假按钮与占位/未接入 UI 全面收口清理
- 全面排查桌面客户端（traecn-tools/src/pages：Overview、Accounts、Settings 等）与嵌入式控制台（internal/openai/dashboard.html）；
- 坚决删除或隐藏所有无实质后端接口支撑、点击无实际功能对应或误导用户的假按钮、占位图表和假开关；
- 确保界面暴露的每一个按钮、输入框与下拉框均具备 100% 真实的功能闭环和明确的交互反馈。

### R4. 正式 v1.0.0 生产文档与发布资产完备
- 编写生产级开源 README.md，图文并茂，包含：
  - 项目简介与架构图（Go 核心网关 + Electron 客户端双模态）；
  - 快速开始（独立二进制运行、桌面端一键安装、Docker/源码部署）；
  - 主流 AI 编程工具接入配置指南（Claude Code、Cursor、VS Code / Continue、Aider 等客户端的 BaseURL 与密钥配置说明）；
  - 安全最佳实践（局域网暴露防呆、无 Key 跨站 CSRF 防护、凭据 fail-closed 加密存储）；
- 编写详实的 CHANGELOG.md，记录 v1.0.0 的里程碑功能与安全加固历史。

### R5. 全链路版本与测试验证对齐
- 确保根目录、Go main.go、internal/version/version.go、traecn-tools/package.json 中的版本号严格统一为 v1.0.0；
- 运行全量单元测试与端到端回归脚本，确保在全部整改后 100% 绿灯通过。

## Acceptance Criteria

### 1. CI/CD 流水线可用性
- [ ] .github/workflows/build.yml 语法与 Action 引用完全符合规范，路径与依赖配置正确。
- [ ] 包含完整的测试检查任务（test）与跨平台构建发布任务（release），产物命名清晰规范。

### 2. 桌面端打包与独立运行验证
- [ ] traecn-tools 执行 npm run dist:win 能成功生成 NSIS 安装包与 Portable 免安装可执行程序，构建 0 错误。
- [ ] 打包出来的应用内嵌代理二进制路径正确，能够在干净环境下启动。

### 3. UI 交互 100% 真实闭环
- [ ] 桌面端与控制台完成假按钮排查，不存在无法响应或虚假操作项。
- [ ] 现存功能（账号导入导出/解密、代理启停、实时监控、会话流水查询、API Key 鉴权设置）保持完全可用。

### 4. 交付文档完整度
- [ ] 根目录 README.md 与 CHANGELOG.md 完整详实，排版专业。
