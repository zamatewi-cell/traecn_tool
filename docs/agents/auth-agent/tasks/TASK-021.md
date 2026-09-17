# Task: 实现账号池管理

**ID**: TASK-021  
**优先级**: Normal  
**分配给**: @Auth-Agent  
**依赖**: TASK-020  
**截止日期**: 2026-03-19  
**状态**: ? Pending  

---

## 描述

实现多账号池管理系统，包括账号存储、健康检查、轮换策略和防封号机制。

---

## 验收标准

- [ ] 账号加密存储（文件/数据库）
- [ ] 账号健康检查机制
- [ ] 智能轮换策略
- [ ] 使用统计和限流
- [ ] 防封号策略（请求间隔、频率控制）

---

## 账号状态机

```
┌─────────┐
│ Active  │
└────┬────┘
     │ 开始使用
     ▼
┌─────────┐
│ InUse   │──────┐
└────┬────┘      │ 使用完成
     │ 触发限流   ▼
     │      ┌──────────┐
     │      │ Cooldown │
     │      └────┬─────┘
     │           │ 冷却结束
     └───────────┘
     
     │ 连续错误
     ▼
┌──────────┐
│ Disabled │
└──────────┘
```

---

## 轮换策略

### 策略 1: 轮询 (Round-Robin)

```go
func (ap *AccountPool) GetNextAccount() *Account {
    ap.mu.Lock()
    defer ap.mu.Unlock()
    
    // 简单轮询
    account := ap.accounts[ap.index]
    ap.index = (ap.index + 1) % len(ap.accounts)
    return account
}
```

### 策略 2: 最少使用 (Least-Used)

```go
func (ap *AccountPool) GetLeastUsedAccount() *Account {
    // 选择使用次数最少的账号
}
```

### 策略 3: 权重随机 (Weighted-Random)

```go
func (ap *AccountPool) GetWeightedRandomAccount() *Account {
    // 根据账号权重（成功率、响应速度）随机选择
}
```

---

## 防封号策略

1. **请求间隔**: 每个账号请求间隔 >= 3 秒
2. **频率限制**: 每账号每分钟 <= 20 请求
3. **错误熔断**: 连续 3 次错误后暂停使用
4. **时间模拟**: 模拟人类使用时间分布
5. **IP 轮换**: 多 IP 出口（可选）

---

## 依赖关系

**上游**: 
- TASK-020 (Token 管理器)

**下游**:
- TASK-030 (Queue-Agent: 基于账号池调度)

---

## 交付物

1. **账号池实现** (`internal/auth/pool.go`)
2. **存储后端** (`internal/auth/storage_file.go`)
3. **轮换策略** (`internal/auth/strategy.go`)
4. **配置模板** (`config.example.json` 多账号示例)

---

**创建时间**: 2026-03-15  
**验收人**: PM-Agent
