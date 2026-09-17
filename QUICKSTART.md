# 快速开始

## 1. 构建并启动

需要 Go 1.22+，本机已登录 Trae CN。

```powershell
go build -o trae-proxy.exe ./cmd/trae-proxy
.\trae-proxy.exe
```

默认监听 `:9090`，自动嗅探 `%APPDATA%\Trae CN\User\globalStorage\storage.json`（macOS / Linux 路径见下）。

```powershell
.\trae-proxy.exe -listen ":8080" -log-level debug -config config.json
```

## 2. 验证

```powershell
curl http://localhost:9090/health
curl http://localhost:9090/v1/models
```

```powershell
curl http://localhost:9090/v1/chat/completions `
  -H "Content-Type: application/json" `
  -d '{"model":"Seed-Code","messages":[{"role":"user","content":"hi"}],"stream":false}'
```

## 3. 客户端

| 项 | 值 |
|---|---|
| Base URL | `http://localhost:9090/v1` |
| API Key | 任意 |
| 推荐模型 | `Seed-Code`（别名 `doubao`） |

其它常用别名：`deepseek-r1` → DeepSeek-V4-Pro，`deepseek-chat` → DeepSeek-V4-Flash，`glm` → GLM-5.3，`qwen` → Qwen3.7-Plus，`kimi` → Kimi-K3，`minimax` → MiniMax-M3。完整表见 [README.md](README.md)。

- OpenAI 兼容客户端：Base URL 指到 `/v1`，走 `/v1/chat/completions`
- Claude Code / Anthropic SDK：指到 `/v1/messages`
- Codex：指到 `/v1/responses`

## 4. 多账号与防护

复制 `config.example.json` 为 `config.json`：

- `storage_path` 为空：自动嗅探本地 Trae
- `token`：直填 JWT
- `env_var`：从环境变量读 JWT（如 `TRAE_CN_TOKEN`）
- `protect`：载荷预算、敏感词（默认关）、并发/间隔限流

账号按池轮询；单账号熔断后换下一个。Token 过期前 5 分钟自动刷新，401 上报 stale。

## 5. 凭证位置

| 平台 | `storage.json` |
|---|---|
| Windows | `%APPDATA%\Trae CN\User\globalStorage\storage.json` |
| macOS | `~/Library/Application Support/Trae CN/User/globalStorage/storage.json` |
| Linux | `~/.config/Trae CN/User/globalStorage/storage.json` |

## 6. 说明

上游真实聊天接口为 `/api/ide/v1/llm_raw_chat`，请求体中消息经 AES-256-GCM 加密（详见 README「上游协议说明」）。4023 问题已解决：根因是请求形状（需 `model_name` + 加密 `message` + `get-svc` 头），与 aha 传输层无关。注意该端点只支持预设模型（`seed_m8`、`deepseek-V3`、`deepseek-R1` 等），自定义模型会返回 4023。
