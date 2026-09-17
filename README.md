# trae-proxy

> 将 Trae CN 本地 AI 能力转换为标准 OpenAI / Anthropic / Codex 兼容 API 的轻量反向代理

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

`trae-proxy` 自动读取并解密本地 Trae CN IDE 的会话凭证，把底层大模型能力封装为本地 HTTP API（默认 `http://localhost:9090/v1`），供 Cherry Studio、Cline、Claude Code、Codex 等客户端接入。

---

## 核心特性

- **凭证自动装载**：AES-128-CBC 解密 `storage.json`（Magic 头校验），也支持 `config.json` 直填 token、环境变量三种来源；请求前检查过期、提前 5 分钟刷新，401 时熔断并轮换账号。
- **多协议**：`/v1/chat/completions`（OpenAI）、`/v1/messages`（Anthropic）、`/v1/responses`（Codex）。
- **思考链与工具调用**：`<think>` 跨分片切割；`delta.reasoning_content` / `delta.content` 分流；`tools` / `tool_choice` 双向透传。
- **防护**：Context Projector（栈追踪剥离 + 头尾截断 + 总预算缩减）、敏感词过滤（默认关）、并发/间隔限流。
- **模型注册表**：内置 16 个模型（含 MaxTokens / ContextWindow / 别名），启动后通过 `model_list` 动态合并上游预设模型。
- **可视化 WebUI 控制台**：单 exe 文件内置现代化暗黑主题管理仪表盘（零前端依赖），支持 Base URL 复制、API Key 生成与管理、21 个模型全景筛选、在线即时测试 Playground（支持思考链与流式渲染）以及实时请求与 Token 用量监控流水。
- **双重运行模式**：支持原生独立进程后台运行，亦可配合配套 Electron 桌面端 `traecn-tools` 实现系统托盘一键启停与可视化管理。

---

## 可视化管理控制台 (WebUI)

服务启动后，在浏览器直接打开根路径即可进入管理控制台：

👉 **[http://localhost:9090/](http://localhost:9090/)** 或 **[http://localhost:9090/dashboard](http://localhost:9090/dashboard)**

- **请求配置一键复制**：展示标准 Base URL（`http://127.0.0.1:9090/v1`）与 API Key 配置；
- **API Key 生成器**：支持一键生成随机高安全密钥，或填入自定义密钥接入；
- **模型全景矩阵**：实时拉取并展示 21 个模型（16 个新内置模型 + 5 个预设模型）的参数、上下文上限与双通道类型；
- **在线 Playground**：无需打开外部客户端，直接在控制台中与 `DeepSeek-V4.1-Flash`、`Seed-Code` 等模型发起对话测试，实时查看思考链（Reasoning Process）与首字延迟；
- **用量与日志流**：实时统计会话数、估算 Token 消耗、平均耗时与详细请求历史表格。

---

## 快速开始

```powershell
go build -o trae-proxy.exe ./cmd/trae-proxy
.\trae-proxy.exe
```

可选：`-listen ":8080"`、`-log-level debug`、`-config config.json`。

客户端填入：

| 配置项 | 内容 |
|---|---|
| **Base URL** | `http://localhost:9090/v1` |
| **API Key** | 任意（默认不校验） |
| **Model** | `Seed-Code` / `doubao` / `deepseek-r1` / `glm` 等 |

```powershell
curl http://localhost:9090/health
curl http://localhost:9090/v1/models
```

更完整的接入示例见 [QUICKSTART.md](QUICKSTART.md)。

---

## 内置模型

客户端可用 **Model ID** 或 **别名**。转发时会映射到上游真实 `model_name`（`UpstreamID`）；未验证映射的模型不瞎编，回落默认 `seed_m8`。启动后远程刷新会合并上游 `is_preset` 预设模型（llm_raw_chat 只服务这类模型），以 `/v1/models` 为准。

| 厂商 | Model ID | 别名 | 上游 model_name | MaxTokens | Context |
|---|---|---|---|---|---|
| ByteDance | `Seed-Evolving` | | （远程刷新后填入） | 16000 | 256K |
| ByteDance | `Seed-2.1-Pro-0915` | | （远程刷新后填入） | 16000 | 256K |
| ByteDance | `Seed-2.1-Turbo` | | （远程刷新后填入） | 16000 | 256K |
| ByteDance | `Seed-Code` | `doubao`, `seed` | `seed_m8` | 16000 | 256K |
| Zhipu | `GLM-5.3-Flash` | | （远程刷新后填入） | 8192 | 128K |
| Zhipu | `GLM-5.3` | `glm-5`, `glm` | （远程刷新后填入） | 16384 | 128K |
| Zhipu | `GLM-5.2` | | （远程刷新后填入） | 16384 | 128K |
| DeepSeek | `DeepSeek-V4.1-Flash` | | （远程刷新后填入） | 16384 | 128K |
| DeepSeek | `DeepSeek-V4-Flash` | `deepseek-v4`, `deepseek-chat` | `deepseek-V3` | 8192 | 128K |
| DeepSeek | `DeepSeek-V4-Pro` | `deepseek-r1`, `deepseek-reasoner` | `deepseek-R1` | 16384 | 128K |
| Moonshot | `Kimi-K3` | `kimi` | （远程刷新后填入） | 16384 | 256K |
| Moonshot | `Kimi-K2.8-Preview` | | （远程刷新后填入） | 16384 | 256K |
| MiniMax | `MiniMax-M3` | `minimax` | （远程刷新后填入） | 16384 | 1M |
| Alibaba | `Qwen3.8-Flash` | | （远程刷新后填入） | 8192 | 128K |
| Alibaba | `Qwen3.8-Max` | | （远程刷新后填入） | 16384 | 256K |
| Alibaba | `Qwen3.7-Plus` | `qwen` | （远程刷新后填入） | 16384 | 128K |

默认模型：`Seed-Code`。

---

## 配置

复制 `config.example.json` 为 `config.json`（与可执行文件同目录，或用 `-config` 指定）：

```json
{
  "listen_addr": ":9090",
  "log_level": "info",
  "accounts": [
    { "name": "local_trae", "storage_path": "" },
    { "name": "backup_token", "token": "eyJhbGci..." },
    { "name": "from_env", "env_var": "TRAE_CN_TOKEN" }
  ],
  "protect": {
    "max_payload_bytes": 524288,
    "max_message_bytes": 65536,
    "filter_enabled": false,
    "filter_replacements": { "敏感词A": "替代词A" },
    "max_concurrent": 4,
    "min_interval_ms": 0
  }
}
```

| 字段 | 说明 |
|---|---|
| `accounts[].storage_path` | Trae `storage.json` 路径；空则自动嗅探 Windows / macOS / Linux 三平台默认位置 |
| `accounts[].token` | 直接提供 JWT |
| `accounts[].env_var` | 从环境变量读取 JWT |
| `protect.max_payload_bytes` | 消息列表总预算（0=不限），超出则头尾截断 |
| `protect.max_message_bytes` | 单条消息上限 |
| `protect.filter_enabled` | 敏感词平滑，默认关 |
| `protect.max_concurrent` | 上游并发上限 |
| `protect.min_interval_ms` | 上游请求最小间隔 |

无 `config.json` 时使用默认配置并自动探测本地 Trae CN 凭证。

---

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查 |
| GET | `/v1/models` | 模型列表（含元数据） |
| POST | `/v1/chat/completions` | OpenAI Chat Completions（流式/非流式） |
| POST | `/v1/messages` | Anthropic Messages |
| POST | `/v1/responses` | Codex Responses |
| GET | `/v1/queue/status` | 上游排队状态 |
| GET | `/v1/accounts` | 账号池状态 |

---

## 构建与测试

```bash
go test ./internal/...
go build -o trae-proxy.exe ./cmd/trae-proxy
```

---

## 上游协议说明

真实上游聊天端点是 `/api/ide/v1/llm_raw_chat`（旧 `chat_completion` 已 404）。请求体为 `{"model_name": ..., "message": <密文>}`：`message` 是消息数组（parts 形状）经 AES-256-GCM 加密后的 base64——密钥硬编码于客户端，key 前 8 字节与随机 pin 异或，请求时间戳作 AAD，并需携带 `get-svc: 1`、`X-Request-Pin`、`X-Requested-At` 头（4023 `MODEL_NOT_EXISTED` 的根因就是旧版请求形状不对，已打通）。

该端点只服务 `is_preset: true` 的预设模型（实测 `seed_m8` / `Doubao_1_5_thinking_pro` / `deepseek-R1` / `deepseek-V3` / `deepseek-V3-0324` 五个）；用户自定义模型（`client_connect` / 自带 ak 的条目）无论如何都返回 4023，不在支持范围。

仅供个人技术研究。基于 [MIT License](LICENSE)。
