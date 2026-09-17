# 任务卡片模板

---

## Task: [任务类型] 任务名称

**ID**: TASK-XXX  
**优先级**: High/Normal/Low  
**分配给**: @Agent-Name  
**依赖**: 无 / TASK-XXX  
**截止日期**: YYYY-MM-DD  
**状态**: ? Pending / ? In Progress / ? Review / ? Done / ? Blocked  

### 描述

详细说明任务目标和交付物

### 验收标准

- [ ] 标准 1
- [ ] 标准 2
- [ ] 标准 3

### 技术提示

- 提示 1
- 提示 2

### 工作文件

- `path/to/file1.go`
- `path/to/file2.md`

### 进度记录

**YYYY-MM-DD**:
- 进度：XX%
- 完成：具体完成的内容
- 问题：遇到的问题

---

## 使用指南

### 创建任务

1. 复制上方模板
2. 填写所有字段
3. 保存到对应 Agent 的 `tasks/` 目录
4. 在事件日志中通知相关 Agent

### 更新状态

- ? **Pending**: 等待开始
- ? **In Progress**: 进行中
- ? **Review**: 完成待审查
- ? **Done**: 已完成
- ? **Blocked**: 被阻塞

### 任务分配

**PM-Agent 职责**:
- 创建任务卡片
- 分配给合适的 Agent
- 追踪进度
- 验收交付物

**Agent 职责**:
- 接收任务
- 更新进度
- 交付成果
- 请求协助 (如需要)

---

## 任务编号规则

- **TASK-001 ~ TASK-009**: Protocol-Agent (协议逆向)
- **TASK-010 ~ TASK-019**: API-Agent (API 开发)
- **TASK-020 ~ TASK-029**: Auth-Agent (认证管理)
- **TASK-030 ~ TASK-039**: Queue-Agent (队列管理)
- **TASK-040 ~ TASK-049**: Test-Agent (测试 QA)

---

## 优先级定义

- **High**: 关键路径任务，优先处理
- **Normal**: 普通任务，按顺序处理
- **Low**: 优化类任务，空闲时处理

---

**模板版本**: v1.0  
**维护者**: PM-Agent
