# Changelog / 变更日志

本项目的所有重要变更均记录在此文件中。
格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [语义化版本 2.0.0](https://semver.org/lang/zh-CN/)。

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
