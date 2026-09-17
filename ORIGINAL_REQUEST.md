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
