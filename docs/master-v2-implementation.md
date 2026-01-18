# Master V2 实施任务文档

> 创建时间: 2025-01-17
> 状态: 进行中

---

## 一、架构概览

```
atlhyper_master_v2/
│
├── agentsdk/          # Agent 通信层（HTTP Server）
├── processor/         # 数据处理层（去重清理）
├── datahub/           # 数据中心（内存，保留 30 分钟）
├── database/          # 数据库层（RDB 持久化）
├── service/           # 业务逻辑层
├── notifier/          # 通知发送
├── query/             # 抽象查询层
├── gateway/           # Web API 网关
├── config/            # 配置
└── master.go          # 启动入口
```

**I 型数据流：**
```
Agent → AgentSDK → Processor → DataHub → QueryLayer → Gateway → Web
```

---

## 二、实施任务清单

### 阶段 1：基础框架

| 任务 | 状态 | 说明 |
|------|------|------|
| 创建目录结构 | [x] | atlhyper_master_v2/ |
| config 层 | [x] | 配置定义和加载 |
| cmd/main.go | [x] | 启动入口 |

### 阶段 2：数据层

| 任务 | 状态 | 说明 |
|------|------|------|
| datahub/interfaces.go | [x] | DataHub 接口定义 |
| datahub/memory/ | [x] | MemoryHub 实现（快照、状态、指令队列）|
| database/interfaces.go | [x] | Database 接口定义 |
| database/repository/ | [x] | Repository 接口定义 |
| database/sqlite/ | [x] | SQLite 实现 |

### 阶段 3：Agent 通信

| 任务 | 状态 | 说明 |
|------|------|------|
| agentsdk/server.go | [x] | HTTP Server |
| agentsdk/snapshot.go | [x] | POST /agent/snapshot |
| agentsdk/heartbeat.go | [x] | POST /agent/heartbeat |
| agentsdk/command.go | [x] | GET /agent/commands（长轮询）|
| agentsdk/result.go | [x] | POST /agent/result |

### 阶段 4：处理层

| 任务 | 状态 | 说明 |
|------|------|------|
| processor/processor.go | [-] | 简化设计，处理逻辑集成到 agentsdk |
| processor/transformer.go | [-] | 简化设计，暂不需要 |

### 阶段 5：业务层

| 任务 | 状态 | 说明 |
|------|------|------|
| service/event_persist.go | [x] | Event 持久化服务 |
| service/notify.go | [x] | 通知服务 |
| notifier/ | [x] | Slack/Email 发送 |

### 阶段 6：查询和网关

| 任务 | 状态 | 说明 |
|------|------|------|
| query/engine.go | [-] | 简化设计，查询直接在 handler 中 |
| gateway/server.go | [x] | Gateway HTTP Server |
| gateway/middleware/ | [x] | Logging, CORS 中间件 |
| gateway/handler/ | [x] | API Handler (cluster, event, command, notify) |

### 阶段 7：集成

| 任务 | 状态 | 说明 |
|------|------|------|
| master.go | [x] | 主结构和启动 |
| 编译测试 | [x] | go build 通过 |
| Agent 对接测试 | [ ] | 与 Agent V2 联调 |

---

## 三、关键设计要点

### 3.1 DataHub 接口

```go
type DataHub interface {
    // 快照
    SetSnapshot(clusterID string, snapshot *ClusterSnapshot) error
    GetSnapshot(clusterID string) (*ClusterSnapshot, error)

    // Agent 状态
    UpdateHeartbeat(clusterID string) error
    GetAgentStatus(clusterID string) (*AgentStatus, error)
    ListAgents() ([]AgentInfo, error)

    // 指令队列
    EnqueueCommand(clusterID string, cmd *Command) error
    WaitCommand(clusterID string, timeout time.Duration) (*Command, error)
    AckCommand(cmdID string, result *CommandResult) error

    // 生命周期
    Start() error
    Stop() error
}
```

### 3.2 Event 持久化流程

```
快照到达 → Processor → DataHub → 触发 EventPersistService.Sync()
                                        ↓
                              从 DataHub 获取 Events
                                        ↓
                              UPSERT 到 RDB（基于 event_uid）
```

### 3.3 通知渠道判断

```go
func IsEffective(ch *NotifyChannel) bool {
    if !ch.Enabled { return false }
    switch ch.Type {
    case "slack": return ch.Config.WebhookURL != ""
    case "email": return ch.Config.SMTPHost != "" && len(ch.Config.ToAddresses) > 0
    }
    return false
}
```

---

## 四、参考文档

- 架构设计: `docs/master-v2-architecture.md`
- Agent V2 架构: `docs/已实施架构/agent-v2-architecture.md`

---

## 五、进度记录

| 日期 | 完成内容 |
|------|----------|
| 2025-01-17 | 创建实施文档 |
| 2025-01-17 | 完成 config 层 |
| 2025-01-17 | 完成 datahub 接口和 MemoryHub |
| 2025-01-17 | 完成 database 接口和 SQLite 实现 |
| 2025-01-17 | 完成 agentsdk（Agent 通信层）|
| 2025-01-17 | 完成 service 层（EventPersist, Notify）|
| 2025-01-17 | 完成 notifier（Slack/Email 发送）|
| 2025-01-17 | 完成 gateway（Web API）|
| 2025-01-17 | 编译通过，可启动运行 |
