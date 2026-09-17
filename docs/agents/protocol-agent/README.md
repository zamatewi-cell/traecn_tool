# Protocol Reverse Engineer Agent 工作区

## 角色定位

**Protocol Agent** - 协议逆向工程师，负责破解 Trae CN 通信协议

---

## 当前任务

### ? TASK-001: 完成认证流程逆向

**状态**: ? In Progress  
**优先级**: High  
**截止日期**: 2026-03-16  
**进度**: 60%

#### 任务详情

分析 Trae CN 的完整认证机制:

1. **Token 生成逻辑**
   - [x] 定位 Token 生成函数入口
   - [x] 提取关键参数
   - [ ] 还原生成算法
   - [ ] 编写可执行脚本

2. **认证头信息**
   - [x] 识别所有认证相关 Header
   - [x] 分析字段含义
   - [ ] 验证字段生成规则

3. **刷新机制**
   - [ ] 分析 Token 过期处理
   - [ ] 提取刷新接口
   - [ ] 测试刷新流程

#### 工作文件

- `analysis/auth-flow.md` - 认证流程分析
- `scripts/find_token.js` - Token 函数定位脚本
- `captured/auth-requests.json` - 抓包数据

---

### ? TASK-002: SSE 流式响应格式分析

**状态**: ? Pending  
**优先级**: High  
**截止日期**: 2026-03-17  
**依赖**: TASK-001

#### 任务详情

1. **SSE 事件格式**
   - [ ] 提取事件类型
   - [ ] 分析数据结构
   - [ ] 识别流式分块规则

2. **完成和错误处理**
   - [ ] 分析 [DONE] 标记
   - [ ] 错误格式提取
   - [ ] 重连机制分析

---

## 技术栈

### 工具

- **抓包**: mitmproxy, Wireshark
- **调试**: Chrome DevTools, Node.js debugger
- **反混淆**: de4js, javascript-obfuscator
- **脚本**: Python, JavaScript

### 环境

```bash
# mitmproxy 安装
pip install mitmproxy

# Node.js 调试
node --inspect-brk cueMain.js
```

---

## 工作目录结构

```
docs/agents/protocol-agent/
├── README.md              # 本文件
├── tasks/                 # 任务卡片
│   ├── TASK-001.md
│   └── TASK-002.md
├── analysis/              # 分析文档
│   ├── auth-flow.md
│   ├── endpoints.md
│   └── encryption.md
├── scripts/               # 分析脚本
│   ├── mitm_proxy.js
│   ├── find_token.js
│   └── probe_endpoints.js
├── captured/              # 抓包数据
│   ├── auth-requests.json
│   └── chat-flow.json
└── daily-log.md           # 工作日志
```

---

## 分析进度

### 2026-03-12: 环境搭建 ?

**完成**:
- 安装 mitmproxy
- 配置 HTTPS 抓包
- 捕获首次请求

**问题**:
- 无

---

### 2026-03-13: 基础端点探测 ?

**完成**:
- 探测主要 API 端点
- 识别认证端点
- 分析请求方法

**发现**:
```
POST /api/chat          # 聊天接口
POST /api/builder       # Builder 模式
GET  /api/models        # 模型列表
```

---

### 2026-03-14: 认证流程分析 ?

**进行中**:
- Token 生成函数定位
- 关键参数提取

**初步发现**:
```javascript
// 疑似 Token 生成函数
function generateToken(userId, timestamp) {
    // TODO: 还原算法
}
```

**下一步**:
- 深入分析函数内部逻辑
- 提取种子参数

---

## 技术难点

### 难点 1: 代码混淆

**问题**: cueMain.js 经过重度混淆

**解决方案**:
1. 使用 de4js 初步反混淆
2. 手动还原控制流
3. 关键函数提取

**进度**: 50%

---

### 难点 2: 动态参数

**问题**: 请求参数包含动态生成的签名

**假设**:
- 可能基于时间戳
- 可能基于请求内容哈希
- 可能包含随机盐

**验证计划**:
1. 固定时间戳测试
2. 相同请求对比
3. 定位签名生成函数

---

## 知识库贡献

### 已输出

1. **端点列表** → `docs/knowledge-base/protocol/endpoints.md`
   ```markdown
   ## 认证相关
   - POST /api/auth/token    # 获取 Token
   - POST /api/auth/refresh  # 刷新 Token
   
   ## 业务相关
   - POST /api/chat          # 聊天
   - POST /api/builder       # Builder
   ```

2. **认证流程** → `docs/knowledge-base/protocol/auth-flow.md`
   ```mermaid
   sequenceDiagram
       Client->>Server: 1. 请求 Token
       Server->>Client: 2. 返回 Token
       Client->>Server: 3. 带 Token 请求
       Server->>Client: 4. 返回结果
   ```

---

## 依赖关系

### 需要协助

**请求**: @Auth-Agent

**内容**:
> 完成认证流程逆向后，需要你实现 Token 管理器
> 
> 交付物:
> - Token 生成函数
> - 刷新逻辑
> - 错误处理

**时间**: 2026-03-17 前

---

### 阻塞中

**无** - 当前任务可独立推进

---

## 资源需求

### 测试账号

- **需求**: 3 个 Trae CN 账号
- **用途**: 逆向分析
- **状态**: ? 已批准 (PM-Agent 2026-03-15)

### 计算资源

- **本地调试**: 足够
- **无需服务器**

---

## 风险评估

| 风险 | 影响 | 概率 | 应对 |
|------|------|------|------|
| 混淆过于复杂 | 高 | 中 | 预留额外时间 |
| 需要动态调试 | 中 | 高 | 学习 VS Code 调试 |
| 协议变更 | 高 | 低 | 持续监控更新 |

---

## 工作日志

### 2026-03-15

**工作时间**: 6h

**完成**:
- ? 定位 Token 生成函数所在文件
- ? 提取 3 个关键参数
- ? 初步分析签名算法

**问题**:
- 控制流混淆严重，需要手动还原

**明日计划**:
- 还原 Token 生成算法
- 编写测试脚本验证

---

### 2026-03-14

**工作时间**: 5h

**完成**:
- ? 搭建 mitmproxy 环境
- ? 捕获认证流程请求
- ? 识别认证头信息

**发现**:
```
X-Token: eyJhbGciOiJIUzI1NiIs...
X-Timestamp: 1710345600
X-Signature: a1b2c3d4e5f6...
```

**下一步**:
- 分析 X-Signature 生成逻辑

---

### 2026-03-13

**工作时间**: 4h

**完成**:
- ? 端点探测脚本
- ? 识别主要 API 端点

**输出**:
- `scripts/probe_endpoints.js`

---

### 2026-03-12

**工作时间**: 3h

**完成**:
- ? 项目启动
- ? 工具安装
- ? 环境配置

---

## 联系信息

- **工作目录**: `docs/agents/protocol-agent/`
- **日志**: `docs/agents/protocol-agent/daily-log.md`
- **分析文档**: `docs/agents/protocol-agent/analysis/`
- **脚本**: `docs/agents/protocol-agent/scripts/`

---

**最后更新**: 2026-03-15 20:00  
**下次更新**: 2026-03-16 09:00  
**当前状态**: ? Busy (TASK-001)
