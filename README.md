# trae-proxy & traecn-tools

<div align="center">

**将 Trae CN 本地 AI 能力转换为标准 OpenAI / Anthropic / Codex 协议的高性能代理网关与桌面客户端**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Node Version](https://img.shields.io/badge/Node-20+-339933?style=flat&logo=node.js)](https://nodejs.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/badge/Release-v1.0.0-blue.svg)](https://github.com/zamatewi-cell/traecn_tool/releases)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)](#)

[快速开始](#-快速开始) • [主流客户端接入](#-主流-ai-编程工具接入指南) • [架构设计](#-系统架构) • [安全实践](#-生产安全最佳实践) • [更新日志](CHANGELOG.md)

</div>

---

## 📖 项目简介

`trae-proxy` 是一套工业级、双模态的 AI 模型代理网关系统。它能自动探测并安全解析本地 Trae CN IDE 的运行凭据，将其底层强大的大语言模型能力无缝封装为完全兼容 **OpenAI Chat Completions**、**Anthropic Messages** 及 **Codex** 的标准 HTTP/SSE 接口（默认监听 `http://localhost:9090/v1`）。

无论是使用 **Cursor**、**VS Code (Continue / Cline)**、**Claude Code**、**Aider** 还是各类日常应用（如 **Cherry Studio**、**NextChat**），均可即插即用接入，免去繁琐配置，充分享受最新一代大语言模型的编程加速体验。

---

## 🏛️ 系统架构

本项目采用 **Go 核心网关** 与 **Electron 桌面端** 双模态分层架构，既支持轻量级无头服务器/后台守护进程部署，又为桌面用户提供一键启停的可视化管理界面：

```text
+-----------------------------------------------------------------------------------------+
|                                    Client Ecosystem                                     |
|     Cursor  /  VS Code Continue  /  Claude Code  /  Aider  /  Cherry Studio  /  SDKs    |
+-----------------------------------------------------------------------------------------+
                                             │ (HTTP REST / SSE Stream)
                                             ▼
+-----------------------------------------------------------------------------------------+
|                         Dual-Modal Trae-Proxy Gateway (v1.0.0)                          |
|                                                                                         |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ 1. 核心网关协议层 (Core Gateway Engine)                                         │   |
|   │    • 兼容协议端点: /v1/chat/completions, /v1/models, /v1/messages, /v1/responses │   |
|   │    • 思考链分离提取: <think> 跨分片切割 -> delta.reasoning_content 流式分流     │   |
|   │    • 21 模型智能路由器: 自动分流 5 个预设模型 (HTTPS) 与 16 个内置模型 (IPC/aha)  │   |
|   │    • 上下文投影与流量控制: Context Projector 头尾截断与并发限流保护              │   |
|   │    • SQLite 异步日志引擎: 会话流水本地持久化与用量统计                          │   |
|   └─────────────────────────────────────────────────────────────────────────────────┘   |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ 2. 生产级安全防护内核 (Security Core)                                           │   |
|   │    • Localhost 环回绑定防呆 (127.0.0.1 默认隔离)                                │   |
|   │    • LAN 暴露强制鉴权 (外网暴露必须配置 API Key，无 Key 严禁启动)               │   |
|   │    • CSRF 跨站拦截引擎 (无 Key 模式严格阻断外部非受信 Origin 探测)              │   |
|   │    • 凭据内存管道注入 (Token 零明文落盘，AES-256 加密存储与提前刷新)            │   |
|   └─────────────────────────────────────────────────────────────────────────────────┘   |
|   ┌─────────────────────────────────────────────────────────────────────────────────┐   |
|   │ 3. 内嵌 WebUI 控制台 (Embedded Dashboard: http://localhost:9090/)                │   |
|   │    • 零外部 CDN 纯静态架构 / 在线流式 Playground / TTFT 与 TPS 实时监控        │   |
|   │    • 官方云端账单与本地会话流水双视图 / API Key 管理与 Base URL 一键复制        │   |
|   └─────────────────────────────────────────────────────────────────────────────────┘   |
+-----------------------------------------------------------------------------------------+
                                             ▲
                                             │ (Stdin Config / Process Guardian)
+-----------------------------------------------------------------------------------------+
|                       Desktop GUI Client (traecn-tools Electron)                        |
|   • Electron + Vite + React 18 + Tailwind CSS 极简暗黑界面                              |
|   • 进程生命周期守护: 自动拉起 resources/bin/trae-proxy.exe，退出自动回收防孤儿僵尸     |
|   • 分发形态: NSIS 安装包 (Setup.exe) 与即开即用免安装便携版 (Portable.exe)             |
+-----------------------------------------------------------------------------------------+
                                             │
                                             ▼
+-----------------------------------------------------------------------------------------+
|                        Upstream Trae IDE & Official AI Services                         |
|      Local storage.json (AES-128-CBC)  /  ZMQ IPC Channels  /  TOS Cloud Endpoints      |
+-----------------------------------------------------------------------------------------+
```

---

## ✨ 核心特性

- **多协议全栈兼容**：原生支持 OpenAI (`/v1/chat/completions`)、Anthropic (`/v1/messages`) 与 Codex (`/v1/responses`) 协议规范。
- **21 大模型矩阵与智能分流**：
  - 涵盖最新 16 个内置模型（如 `DeepSeek-V4.1-Flash`、`Seed-Code`、`GLM-5.3`、`Kimi-K3`、`Qwen3.7-Plus` 等）；
  - 完美共存并路由至 5 个经典预设模型（`seed_m8`、`Doubao_1_5_thinking_pro`、`deepseek-R1`、`deepseek-V3`、`deepseek-V3-0324`）。
- **深度思考过程流式分流**：支持推理模型（如 DeepSeek-R1）思维链 `<think>` 内容实时分离，并通过 `delta.reasoning_content` 单独吐字，前端支持一键折叠。
- **生产级安全防护基线**：
  - **局域网防呆**：禁止未设密码的网关意外对外暴露；
  - **CSRF 防护**：拦截外部网页通过浏览器偷偷探测调用本机代理；
  - **凭据零落盘**：内存安全传输，敏感凭据支持 AES-256 加密与自动刷新。
- **内置零依赖 Web 控制台**：单可执行文件打包所有 Web 资源，包含在线 Playground 性能指标监测（首字延迟 TTFT、生成速度 Tokens/s、提示词缓存命中率）。
- **多平台交叉编译**：纯 Go 编写（零 CGO），原生支持 Windows、macOS 与 Linux 的 x64 与 arm64 六大主流芯片架构。

---

## 🚀 快速开始

本项目提供多种便捷的使用方式，请根据您的场景自由选择：

### 方式一：独立二进制直接运行 (推荐)

从 [GitHub Releases](https://github.com/zamatewi-cell/traecn_tool/releases) 下载对应操作系统的预编译归档包（如 `trae-proxy_v1.0.0_windows_amd64.zip` 或 `trae-proxy_v1.0.0_linux_amd64.tar.gz`），解压后直接执行：

```bash
# Windows
.\trae-proxy.exe

# Linux / macOS
chmod +x ./trae-proxy
./trae-proxy
```

启动成功后，控制台将输出服务监听地址（默认 `http://localhost:9090`）。浏览器访问 `http://localhost:9090/` 即可直接打开管理面板。

---

### 方式二：Windows 桌面客户端安装 (GUI)

对于 Windows 桌面用户，可直接使用打包好的桌面客户端：
- **安装向导版**：下载 `TraeCN.Tools_1.0.0_x64-setup.exe`，双击根据向导安装，生成桌面快捷方式与开机启动项；
- **便携免安装版**：下载 `TraeCN.Tools_1.0.0_x64-portable.exe`，免安装即开即用，适合放入 U 盘或快速体验。

桌面端启动后会自动在后台托管并拉起核心代理，系统托盘支持一键启停和打开仪表盘。

---

### 方式三：Docker 容器化运行

利用轻量化容器快速启动无头网关（将宿主机 Trae 凭据挂载进容器）：

```bash
docker run -d \
  --name trae-proxy \
  -p 9090:9090 \
  -v ~/.config/Trae:/root/.config/Trae:ro \
  -e TRAE_PROXY_LISTEN=":9090" \
  ghcr.io/zamatewi-cell/trae-proxy:v1.0.0
```

---

### 方式四：从源码编译构建

开发机需具备 **Go 1.25+** 与 **Node.js 20+**：

```bash
# 1. 克隆代码仓库
git clone https://github.com/zamatewi-cell/traecn_tool.git
cd traecn_tool

# 2. 编译 Go 核心代理
go build -trimpath -ldflags="-s -w" -o trae-proxy.exe ./cmd/trae-proxy

# 3. 运行全量单元测试
go test ./...

# 4. (可选) 编译并启动桌面端
cd traecn-tools
npm install
npm run dev
```

---

## 🔌 主流 AI 编程工具接入指南

代理服务启动后，默认提供标准的 OpenAI 兼容接口：
- **API Base URL**: `http://localhost:9090/v1` (或 `http://127.0.0.1:9090/v1`)
- **API Key**: 任意非空字符串（如 `sk-trae-local`，若未开启全局鉴权可任意填写）

### 1. Cursor 配置

1. 打开 Cursor 设置界面：`Settings` -> `Models`；
2. 关闭其它非必需模型提供商，在 **OpenAI API Key** 区域输入任意密钥（如 `sk-local`）；
3. 勾选 **Override OpenAI Base URL**，填入：
   ```text
   http://localhost:9090/v1
   ```
4. 在模型列表中点击 **Add Model**，添加你想要使用的模型（例如 `Seed-Code` 或 `DeepSeek-V4.1-Flash`）。

---

### 2. VS Code / Continue 插件配置

编辑 Continue 配置文件 `~/.continue/config.json`（或在设置中打开）：

```json
{
  "models": [
    {
      "title": "Trae - Seed-Code",
      "provider": "openai",
      "model": "Seed-Code",
      "apiBase": "http://localhost:9090/v1",
      "apiKey": "sk-local"
    },
    {
      "title": "Trae - DeepSeek-V4-Pro (Thinking)",
      "provider": "openai",
      "model": "DeepSeek-V4-Pro",
      "apiBase": "http://localhost:9090/v1",
      "apiKey": "sk-local"
    },
    {
      "title": "Trae - GLM-5.3",
      "provider": "openai",
      "model": "GLM-5.3",
      "apiBase": "http://localhost:9090/v1",
      "apiKey": "sk-local"
    }
  ]
}
```

---

### 3. Claude Code (Anthropic CLI)

Claude Code 命令行工具可通过环境变量直接映射代理：

```bash
# 设置 Base URL 与占位 API Key
export ANTHROPIC_BASE_URL="http://localhost:9090"
export ANTHROPIC_API_KEY="sk-local"

# 启动 Claude Code
claude
```

*在 Windows PowerShell 下：*
```powershell
$env:ANTHROPIC_BASE_URL="http://localhost:9090"
$env:ANTHROPIC_API_KEY="sk-local"
claude
```

---

### 4. Aider 命令行编程助手

在终端中使用 Aider 搭配任意 Trae 模型进行自动化代码协同：

```bash
# 方式 A：通过命令行参数指定
aider \
  --openai-api-base http://localhost:9090/v1 \
  --openai-api-key sk-local \
  --model openai/Seed-Code

# 方式 B：通过环境变量指定
export OPENAI_API_BASE="http://localhost:9090/v1"
export OPENAI_API_KEY="sk-local"
aider --model openai/DeepSeek-V4.1-Flash
```

---

### 5. Cherry Studio / NextChat / 其它客户端

| 设置项 | 推荐填入值 | 说明 |
|---|---|---|
| **API 提供商 (Provider)** | `OpenAI` | 选择标准 OpenAI 协议 |
| **API 地址 (Base URL)** | `http://localhost:9090/v1` | 注意部分客户端末尾不需要带 `/v1`，视提示而定 |
| **API 密钥 (API Key)** | `sk-local` | 随意填写；若服务端启用了密码，则填入该密码 |
| **模型名称 (Model)** | `Seed-Code` / `deepseek-r1` / `glm` | 支持模型全名或别名 |

---

## 🛡️ 生产安全最佳实践

为保障用户凭据资产安全及防止内网资产意外暴露，`trae-proxy` 在 v1.0.0 中设计了严格的纵深防御体系：

### 1. Localhost 环回绑定防呆 (Loopback Pinning)
- 网关默认仅监听 `127.0.0.1:9090` 或 `localhost:9090`。
- 本地开发模式下，外部机器完全无法通过网络扫描或直接访问本机的 API 端口。

### 2. 局域网暴露警示与强制 Token (LAN Guard)
- 若用户希望在局域网内共享本网关（例如使用 `-allow-lan` 或显式配置监听 `0.0.0.0`），网关执行 **Fail-Closed 严格安全防呆**：
  - **必须**在 `config.json` 或命令行 `-api-key` 中设置高强度鉴权密钥；
  - 若检测到开放局域网但**未设置 API Key**，网关将输出醒目红色警告并**直接拒绝启动退出**，彻底规避局域网未授权调用与盗刷风险。

### 3. CSRF 跨站请求防范机制 (Origin Gate)
- 许多本地开发代理容易遭受来自浏览器的跨站探测攻击（如受害者访问了攻击者网页，网页内脚本静默向 `http://127.0.0.1:9090` 发送请求盗用大模型）。
- 在免密本地模式下，`trae-proxy` 内置严格的请求来源（Origin）过滤，非受信来源直接返回 `HTTP 403 Forbidden`，仅允许本地控制台与受信任 Origin 调用。

### 4. 凭据零明文落盘与安全内存管道
- Electron 桌面主进程与 Go 代理子进程之间通过专用内存管道传输敏感凭据与指令，杜绝将临时令牌写入临时文件；
- 本地存储支持 AES-256 加密封装，令牌在过期前 5 分钟自动静默刷新，遇到 401 时平滑熔断，保障会话稳定性。

---

## 🤖 支持的大模型清单

代理自动从本地环境与上游服务动态构建模型注册表，支持使用标准 **Model ID** 或 **易记别名**：

| 厂商 / 架构 | 模型标识 (Model ID) | 推荐别名 (Alias) | 上游通道模式 | 最大输出 | 上下文窗口 | 特性描述 |
|---|---|---|:---:|:---:|:---:|---|
| **ByteDance** | `Seed-Code` | `doubao`, `seed` | 预设 (HTTPS) | 16,000 | 256K | **默认主力模型**，针对代码生成深度调优 |
| **ByteDance** | `Seed-Evolving` | | 内置 (aha) | 16,000 | 256K | 具备自适应演化能力的综合模型 |
| **ByteDance** | `Seed-2.1-Turbo` | | 内置 (aha) | 16,000 | 256K | 极速响应版本，适合高频补全 |
| **DeepSeek** | `DeepSeek-V4.1-Flash` | | 内置 (aha) | 16,384 | 128K | 最新一代高吞吐闪电模型 |
| **DeepSeek** | `DeepSeek-V4-Flash` | `deepseek-v4`, `deepseek-chat` | 预设 (HTTPS) | 8,192 | 128K | 高性价比通用对话 |
| **DeepSeek** | `DeepSeek-V4-Pro` | `deepseek-r1`, `deepseek-reasoner` | 预设 (HTTPS) | 16,384 | 128K | **深度推理模型**，支持思考链输出 |
| **Zhipu AI** | `GLM-5.3` | `glm-5`, `glm` | 内置 (aha) | 16,384 | 128K | 智谱新一代旗舰通用大模型 |
| **Zhipu AI** | `GLM-5.3-Flash` | | 内置 (aha) | 8,192 | 128K | 极低时延，兼顾代码理解 |
| **Moonshot** | `Kimi-K3` | `kimi` | 内置 (aha) | 16,384 | 256K | 超长上下文理解与推理 |
| **MiniMax** | `MiniMax-M3` | `minimax` | 内置 (aha) | 16,384 | 1,024K | 百万级长文本上下文 |
| **Alibaba** | `Qwen3.7-Plus` | `qwen` | 内置 (aha) | 16,384 | 128K | 通义千问增强版通用模型 |
| **Alibaba** | `Qwen3.8-Max` | | 内置 (aha) | 16,384 | 256K | 通义千问旗舰级推理大模型 |

*完整 21 个模型清单可启动服务后调用 `GET /v1/models` 或在 Web 控制台中实时查看。*

---

## 💻 可视化管理控制台 (WebUI)

在浏览器访问 `http://localhost:9090/` 或 `http://localhost:9090/dashboard`，即可享受开箱即用的暗黑风格管理界面：

- 📊 **实时用量与健康监测**：即刻查看本地 Trae 会员状态（Free / Pro）、积分余额与官方权益包使用进度。
- ⚡ **在线流式 Playground 性能四联监控**：
  - **首字延迟 (TTFT)**：毫秒级度量首个 Token 到达速度；
  - **吐字速率 (TPS)**：实时计算 Tokens/s 输出效率；
  - **提示词缓存命中 (Prompt Cache)**：解析提示词复用节省情况；
  - **思考过程独立折叠**：完美渲染推理模型的 Thinking 思考历程。
- 📑 **双重账单对账表格**：自由切换「本地 SQLite 实时请求流水」与「Trae 官方云端计费流水 (Official Billing)」。

---

## ⚙️ 配置文件说明 (`config.json`)

服务支持在同目录下放置 `config.json`（可参考 `config.example.json`）：

```json
{
  "listen_addr": "127.0.0.1:9090",
  "log_level": "info",
  "api_key": "",
  "accounts": [
    { "name": "local_trae", "storage_path": "" },
    { "name": "backup_token", "token": "eyJhbGci..." }
  ],
  "protect": {
    "max_payload_bytes": 524288,
    "max_message_bytes": 65536,
    "filter_enabled": false,
    "max_concurrent": 4,
    "min_interval_ms": 0
  }
}
```

### 命令行常用参数

| 参数 | 默认值 | 作用说明 |
|---|---|---|
| `-listen` | `127.0.0.1:9090` | 指定服务监听 IP 与端口 |
| `-api-key` | `""` | 设置全局访问鉴权密钥 |
| `-allow-lan` | `false` | 允许局域网访问（启用时强制要求配置 `-api-key`） |
| `-log-level` | `info` | 日志级别（`debug` / `info` / `warn` / `error`） |
| `-config` | `config.json` | 自定义配置文件路径 |
| `-version` | | 打印当前软件构建版本并退出 |

---

## 🛠️ 自动化 CI/CD 流水线

本项目在 `.github/workflows/build.yml` 中建立了全自动化的生产级 GitHub Actions 流程：
1. **测试门禁 (Test Gate)**：对 PR 和 push 触发 Go 1.25 全套单元测试与前端 Vite 编译校验；
2. **多架构交叉编译矩阵 (Matrix Build)**：纯 Go 交叉编译 6 种平台架构制品并分别打成 `.zip` 与 `.tar.gz`；
3. **桌面端自动化打包**：在 Windows 运行机上通过 electron-builder 产出 NSIS Setup 与 Portable 双可执行文件；
4. **自动化 Release**：推送版本标签（如 `v1.0.0`）时，自动生成 `checksums.txt` SHA256 校验单并发布至 GitHub Releases。

---

## 📄 开源许可证与免责声明

- 本项目基于 [MIT 许可证](LICENSE) 开源发布。
- **免责声明**：本项目仅供个人技术研究、网络协议学习以及辅助个人日常代码编写使用。请勿用于商业化未经授权的二次分发或违反服务条款的用途。
