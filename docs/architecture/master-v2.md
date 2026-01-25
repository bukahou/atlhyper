# Master V2 架构设计文档

> 状态: 设计中
> 创建时间: 2025-01-17
> 最后更新: 2025-01-17
> 版本: v0.3 (新增通知渠道配置 + Event 持久化设计)

---

## 一、设计背景

### 1.1 旧架构问题

旧 Master 采用 U 型架构：

```
Agent 上报 ──────┐          ┌────── Web 查询
                │          │
                ▼          ▼
              ┌──────────────┐
              │   Gateway    │  ← 入口和出口都在这里
              └──────────────┘
                     │
                     ▼
              混乱的内部逻辑
```

**问题：**
- Agent 上报和 Web 查询混在 Gateway 层
- 数据流向不清晰
- 难以扩展和维护

### 1.2 新架构目标

采用 I 型架构（自下而上）：

```
Agent
  ↓
AgentSDK (通信层)
  ↓
Processor (处理层)
  ↓
DataHub (数据中心) ← 可替换
  ↓
Query Layer (抽象查询层)
  ↓
Gateway (Web API)
  ↓
Web
```

**优势：**
- 数据流向清晰（单向向上）
- 各层职责分明
- DataHub 可替换（内存 → Redis → 第三方）
- 抽象查询层隔离底层实现

### 1.3 设计原则

1. **I 型数据流** - 自下而上，单向流动
2. **接口抽象** - DataHub 接口化，底层可替换
3. **职责分离** - 每层只做一件事
4. **解耦** - 上层不关心下层实现

---

## 二、整体架构

### 2.1 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                           Master V2                             │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                       Gateway                              │ │
│  │                   (REST API / WebSocket)                   │ │
│  │                                                            │ │
│  │  • Web API 入口                                            │ │
│  │  • 认证/鉴权                                               │ │
│  │  • 调用 Query Layer                                        │ │
│  └─────────────────────────────┬─────────────────────────────┘ │
│                                │                                │
│  ┌─────────────────────────────▼─────────────────────────────┐ │
│  │                     Query Layer                            │ │
│  │                (抽象查询层 - 类似 PromQL)                   │ │
│  │                                                            │ │
│  │  • DSL 解析和执行                                          │ │
│  │  • 查询转发给 DataHub                                      │ │
│  │  • 指令转发给 DataHub                                      │ │
│  └─────────────────────────────┬─────────────────────────────┘ │
│                                │                                │
│  ┌─────────────────────────────▼─────────────────────────────┐ │
│  │                   DataHub 接口（抽象）                      │ │
│  │  ┌─────────────────────────────────────────────────────┐  │ │
│  │  │                     实现层                           │  │ │
│  │  │   ┌─────────────┐   ┌─────────────┐                 │  │ │
│  │  │   │ MemoryHub   │   │  RedisHub   │   ...           │  │ │
│  │  │   │  (当前)     │   │   (后续)    │                 │  │ │
│  │  │   │             │   │             │                 │  │ │
│  │  │   │ • Snapshot  │   │ • Redis Hash│                 │  │ │
│  │  │   │ • AgentState│   │ • Redis Key │                 │  │ │
│  │  │   │ • 简易 MQ   │   │ • BLPOP     │                 │  │ │
│  │  │   └─────────────┘   └─────────────┘                 │  │ │
│  │  └─────────────────────────────────────────────────────┘  │ │
│  └─────────────────────────────┬─────────────────────────────┘ │
│                                ↑                                │
│  ┌─────────────────────────────┴─────────────────────────────┐ │
│  │                       Processor                            │ │
│  │                    (数据处理层)                             │ │
│  │                                                            │ │
│  │  • 数据清洗、转换                                          │ │
│  │  • 写入 DataHub                                            │ │
│  └─────────────────────────────┬─────────────────────────────┘ │
│                                ↑                                │
│  ┌─────────────────────────────┴─────────────────────────────┐ │
│  │                       AgentSDK                             │ │
│  │                    (Agent 通信层)                           │ │
│  │                                                            │ │
│  │  • HTTP Server 监听 Agent 请求                             │ │
│  │  • 接收快照、心跳、执行结果                                 │ │
│  │  • 长轮询下发指令                                          │ │
│  └─────────────────────────────┬─────────────────────────────┘ │
│                                │                                │
└────────────────────────────────┼────────────────────────────────┘
                                 │
                                 ▼
                           [ Agent V2 ]
```

### 2.2 数据流

```
上行（快照）:  Agent → AgentSDK → Processor → DataHub
下行（指令）:  DataHub.WaitCommand() → AgentSDK → Agent
查询（Web）:   Gateway → QueryLayer → DataHub → 返回
下发（Web）:   Gateway → QueryLayer → DataHub.EnqueueCommand()
```

---

## 三、各层详细设计

### 3.1 AgentSDK（通信层）

**职责：**
- HTTP Server 监听 Agent 请求
- 协议解析（Gzip 解压、JSON 反序列化）
- 将数据传递给 Processor
- 从 DataHub 获取指令下发给 Agent

**对外接口（Agent 调用）：**

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /agent/snapshot | 接收集群快照 |
| POST | /agent/heartbeat | 接收心跳 |
| POST | /agent/result | 接收执行结果 |
| GET | /agent/commands | 长轮询下发指令 |

**内部流程：**

```
接收快照:
  HTTP Request → 解压 Gzip → 反序列化 JSON → 调用 Processor

下发指令:
  Agent 请求 → 调用 DataHub.WaitCommand() → 阻塞等待 → 返回指令
```

### 3.2 Processor（处理层）

**职责：**
- 数据清洗、标准化
- 数据转换
- 写入 DataHub

**处理流程：**

```
原始快照
    ↓
基础校验 (ClusterID、时间戳)
    ↓
数据标准化 (时间格式、资源单位)
    ↓
写入 DataHub.SetSnapshot()
```

**特点：**
- 无状态
- 可扩展处理管道
- 数据只在这里清洗一次

### 3.3 DataHub（数据中心）

**职责：**
- 内存/外部存储的统一抽象
- 快照存储
- Agent 状态管理
- 指令队列（简易 MQ）

**接口定义：**

```
DataHub 接口
│
├── 快照管理
│   ├── SetSnapshot(clusterID, snapshot) error
│   ├── GetSnapshot(clusterID) (*ClusterSnapshot, error)
│   └── QueryResources(clusterID, filter) ([]Resource, error)
│
├── Agent 状态
│   ├── UpdateHeartbeat(clusterID) error
│   ├── GetAgentStatus(clusterID) (*AgentStatus, error)
│   └── ListAgents() ([]AgentInfo, error)
│
├── 指令队列（简易 MQ）
│   ├── EnqueueCommand(clusterID, cmd) error      ← 入队
│   ├── WaitCommand(clusterID, timeout) (*Command, error)  ← 长轮询
│   ├── AckCommand(cmdID, result) error           ← 确认完成
│   ├── GetCommandStatus(cmdID) (*CommandStatus, error)
│   └── ListCommands(clusterID, filter) ([]Command, error)
│
└── 生命周期
    ├── Start() error
    └── Stop() error
```

**MemoryHub 内部结构：**

```
┌─────────────────────────────────────────────────┐
│                  MemoryHub                      │
│                                                 │
│   ┌─────────────────────────────────────────┐  │
│   │          SnapshotStore                  │  │
│   │   map[clusterID]*ClusterSnapshot        │  │
│   └─────────────────────────────────────────┘  │
│                                                 │
│   ┌─────────────────────────────────────────┐  │
│   │          AgentState                     │  │
│   │   map[clusterID]*AgentInfo              │  │
│   │     • LastHeartbeat                     │  │
│   │     • Status (online/offline)           │  │
│   └─────────────────────────────────────────┘  │
│                                                 │
│   ┌─────────────────────────────────────────┐  │
│   │       CommandQueue (简易 MQ)             │  │
│   │   map[clusterID]*Queue                  │  │
│   │     • commands []Command                │  │
│   │     • waiting chan (长轮询通知)          │  │
│   │     • mutex (并发保护)                   │  │
│   └─────────────────────────────────────────┘  │
│                                                 │
└─────────────────────────────────────────────────┘
```

**简易 MQ 工作原理：**

```
生产者（Web/AI）                消费者（Agent）
       │                              │
       │── EnqueueCommand() ─────────>│
       │                              │
       │                    ┌─────────┴─────────┐
       │                    │   CommandQueue    │
       │                    │                   │
       │                    │  cmd-1  cmd-2 ... │
       │                    │                   │
       │                    │   waiting chan ───┼──> 通知
       │                    └───────────────────┘
       │                              │
       │                              │<── WaitCommand()
       │                              │    (阻塞等待)
       │                              │
       │                              │<── 收到通知，取出指令
```

**后续可替换为：**

| 组件 | MemoryHub | RedisHub |
|------|-----------|----------|
| SnapshotStore | map | Redis Hash |
| AgentState | map + 定时检查 | Redis Key + TTL |
| CommandQueue | chan | Redis List + BLPOP |

### 3.4 Query Layer（抽象查询层）

**职责：**
- 提供统一的查询 DSL
- 隔离上层和 DataHub 实现
- 解析查询语句，执行并返回结果

**查询示例（DSL 伪代码）：**

```
pods(cluster="prod", namespace="default", status="Running")
nodes(cluster="prod").where(cpu_usage > 80%)
deployments(cluster="prod").count()
events(cluster="prod", last="5m", type="Warning")
```

**功能：**
- 解析查询语句
- 从 DataHub 获取数据
- 过滤、聚合、排序
- 返回结构化结果

**指令转发：**
- 接收操作请求
- 调用 DataHub.EnqueueCommand()
- 返回指令 ID

### 3.5 Gateway（网关层）

**职责：**
- Web API 入口
- 认证/鉴权
- 调用 Query Layer

**对外接口（Web 调用）：**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v2/query | 统一查询入口（DSL）|
| POST | /api/v2/command | 下发指令 |
| GET | /api/v2/clusters | 集群列表 |
| GET | /api/v2/clusters/{id}/status | 集群状态 |
| WS | /api/v2/subscribe | 订阅变化（可选）|

---

## 四、指令下发流程

### 4.1 完整时序

```
Web                     Master V2                           Agent
 │                          │                                 │
 │── POST /api/v2/command ─>│                                 │
 │                          │                                 │
 │                    Gateway                                 │
 │                      ↓                                     │
 │                    Query Layer                             │
 │                      ↓                                     │
 │                    DataHub.EnqueueCommand()                │
 │                      │                                     │
 │<── 202 Accepted ─────│                                     │
 │   (返回 cmdID)        │                                     │
 │                      │                                     │
 │                      │<──── GET /agent/commands ───────────│
 │                      │             (长轮询)                 │
 │                      │                                     │
 │                    AgentSDK                                │
 │                      ↓                                     │
 │                    DataHub.WaitCommand()                   │
 │                      │ (阻塞等待)                           │
 │                      │                                     │
 │                      │ (有指令，返回)                       │
 │                      │──────── Command ───────────────────>│
 │                      │                                     │
 │                      │                                     ├─> 执行
 │                      │                                     │
 │                      │<──── POST /agent/result ────────────│
 │                      │                                     │
 │                    AgentSDK                                │
 │                      ↓                                     │
 │                    DataHub.AckCommand()                    │
 │                      │                                     │
 │                    (更新指令状态)                           │
```

### 4.2 长轮询实现

```
WaitCommand(clusterID, timeout):
    1. 获取锁
    2. 检查队列
       - 有指令 → 取出返回
       - 无指令 → 释放锁，进入等待
    3. 等待:
       select {
           case <-waiting:   // 新指令信号
               → 取指令返回
           case <-timeout:   // 超时
               → 返回 nil
       }
```

---

## 五、目录结构

```
atlhyper_master_v2/
│
├── agentsdk/                   # Agent 通信层
│   ├── server.go               # HTTP Server 启动
│   ├── snapshot.go             # POST /agent/snapshot
│   ├── heartbeat.go            # POST /agent/heartbeat
│   ├── command.go              # GET /agent/commands (长轮询)
│   ├── result.go               # POST /agent/result
│   └── types.go                # 协议类型定义
│
├── processor/                  # 数据处理层
│   ├── processor.go            # 处理入口
│   └── transformer.go          # 数据转换
│
├── datahub/                    # 数据中心（实时数据）
│   ├── interfaces.go           # DataHub 接口定义
│   ├── memory/                 # 内存实现（当前）
│   │   ├── hub.go              # MemoryHub 主结构
│   │   ├── snapshot.go         # 快照存储
│   │   ├── agent.go            # Agent 状态
│   │   └── queue.go            # 简易 MQ（指令队列）
│   └── redis/                  # Redis 实现（后续）
│       └── ...
│
├── database/                   # 数据库层（持久化数据）
│   ├── interfaces.go           # Database 接口定义
│   ├── repository/             # Repository 接口
│   │   ├── user.go
│   │   ├── audit.go
│   │   ├── command_history.go
│   │   └── ...
│   ├── sqlite/                 # SQLite 实现（当前）
│   │   ├── db.go
│   │   ├── migrations.go
│   │   └── impl/
│   └── mysql/                  # MySQL 实现（后续）
│       └── ...
│
├── query/                      # 抽象查询层
│   ├── engine.go               # 查询引擎
│   ├── parser.go               # DSL 解析
│   └── executor.go             # 执行器
│
├── gateway/                    # 网关层
│   ├── router.go               # 路由注册
│   ├── middleware/             # 中间件
│   │   ├── auth.go
│   │   └── logging.go
│   └── handler/
│       ├── query.go            # GET /api/v2/query
│       └── command.go          # POST /api/v2/command
│
├── config/                     # 配置
│   ├── types.go
│   ├── defaults.go
│   └── loader.go
│
└── master.go                   # 启动入口
```

---

## 六、配置项

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| MASTER_HTTP_PORT | HTTP 端口 | 8080 |
| MASTER_AGENT_PORT | Agent 通信端口 | 8081 |
| MASTER_COMMAND_POLL_TIMEOUT | 长轮询超时 | 60s |
| MASTER_HEARTBEAT_TIMEOUT | 心跳超时阈值 | 45s |
| MASTER_SNAPSHOT_RETENTION | 快照保留数量 | 10 |
| MASTER_DATAHUB_TYPE | DataHub 类型 | memory |
| MASTER_REDIS_ADDR | Redis 地址（RedisHub 时需要）| - |

---

## 七、扩展性设计

### 7.1 替换 DataHub 实现

```go
// 开发环境：内存
hub := memory.NewDataHub()

// 生产环境：Redis（改一行）
hub := redis.NewDataHub(redisClient)

// 启动 Master
master := NewMaster(hub, ...)
```

上层代码（Gateway、QueryLayer、Processor、AgentSDK）完全不变。

### 7.2 新增资源类型

1. model/ 新增资源结构
2. Processor 处理新资源
3. DataHub 存储新资源
4. Query Layer 支持新资源查询

### 7.3 新增指令类型

1. model/command.go 新增指令定义
2. Agent V2 实现对应执行逻辑
3. DataHub 指令队列自动支持

---

## 八、与 Agent V2 的对接

### 8.1 共享 Model

```
atlhyper/
├── shared/
│   └── model/                  # 共享数据模型
│       ├── snapshot.go         # ClusterSnapshot
│       ├── command.go          # Command, Result
│       └── resources.go        # Pod, Node, Deployment...
│
├── atlhyper_agent_v2/          # Agent 引用 shared/model
└── atlhyper_master_v2/         # Master 引用 shared/model
```

### 8.2 通信协议

| Agent 接口 | Master 对应 |
|------------|-------------|
| POST /agent/snapshot | AgentSDK.HandleSnapshot() |
| POST /agent/heartbeat | AgentSDK.HandleHeartbeat() |
| GET /agent/commands | AgentSDK.HandleCommands() |
| POST /agent/result | AgentSDK.HandleResult() |

---

## 九、RDB 数据库设计

### 9.1 存储职责划分

```
┌─────────────────────────────────────────────────────────────┐
│                      DataHub (内存)                         │
│                                                             │
│  • 集群实时快照 (ClusterSnapshot)                           │
│  • Agent 在线状态                                           │
│  • 指令队列 (等待执行)                                       │
│                                                             │
│  特点: 实时、易失、重启丢失                                  │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ 需要持久化的数据
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      RDB (SQLite/MySQL)                     │
│                                                             │
│  • 用户管理                                                 │
│  • Token 管理                                               │
│  • 集群注册信息                                             │
│  • 操作审计日志                                             │
│  • 指令历史记录                                             │
│  • AI 操作日志                                              │
│  • 告警配置                                                 │
│  • 系统设置                                                 │
│                                                             │
│  特点: 持久、可查询、重启保留                                │
└─────────────────────────────────────────────────────────────┘
```

### 9.2 表结构设计

#### 9.2.1 用户与认证

**users - 用户表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 用户 ID |
| username | TEXT UNIQUE | 用户名 |
| password_hash | TEXT | 密码哈希 |
| display_name | TEXT | 显示名称 |
| email | TEXT | 邮箱 |
| role | INTEGER | 角色: 1=Admin, 2=Operator, 3=Viewer |
| status | INTEGER | 状态: 1=Active, 0=Disabled |
| created_at | TEXT | 创建时间 |
| updated_at | TEXT | 更新时间 |
| last_login_at | TEXT | 最后登录时间 |
| last_login_ip | TEXT | 最后登录 IP |

**tokens - Token 表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | Token ID |
| user_id | INTEGER FK | 关联用户 |
| token_hash | TEXT UNIQUE | Token 哈希值 |
| name | TEXT | Token 名称 |
| type | INTEGER | 类型: 1=Session, 2=API Key |
| permissions | TEXT | 权限范围（JSON）|
| expires_at | TEXT | 过期时间 |
| last_used_at | TEXT | 最后使用时间 |
| created_at | TEXT | 创建时间 |

#### 9.2.2 集群管理

**clusters - 集群注册表**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 集群 ID |
| cluster_uid | TEXT UNIQUE | 集群 UID（来自 kube-system）|
| name | TEXT | 集群显示名称 |
| description | TEXT | 描述 |
| environment | TEXT | 环境: prod/staging/dev |
| created_at | TEXT | 注册时间 |
| updated_at | TEXT | 更新时间 |

#### 9.2.3 审计日志

**audit_logs - 操作审计日志**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 日志 ID |
| timestamp | TEXT | 时间戳 |
| user_id | INTEGER | 用户 ID（0=系统/匿名）|
| username | TEXT | 用户名 |
| role | INTEGER | 用户角色 |
| source | TEXT | 来源: web/api/ai |
| action | TEXT | 操作类型 |
| resource | TEXT | 资源路径 |
| method | TEXT | HTTP 方法 |
| request_body | TEXT | 请求体（脱敏）|
| status_code | INTEGER | 响应状态码 |
| success | INTEGER | 是否成功: 1/0 |
| error_message | TEXT | 错误信息 |
| ip | TEXT | 客户端 IP |
| user_agent | TEXT | User-Agent |
| duration_ms | INTEGER | 耗时（毫秒）|

**security_events - 安全事件（非法操作记录）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 事件 ID |
| timestamp | TEXT | 时间戳 |
| event_type | TEXT | 类型: invalid_token/permission_denied/rate_limit |
| severity | TEXT | 严重程度: low/medium/high/critical |
| ip | TEXT | 来源 IP |
| user_id | INTEGER | 用户 ID（如有）|
| token_id | INTEGER | Token ID（如有）|
| resource | TEXT | 访问的资源 |
| details | TEXT | 详细信息（JSON）|
| resolved | INTEGER | 是否已处理: 1/0 |
| resolved_at | TEXT | 处理时间 |
| resolved_by | INTEGER | 处理人 |

#### 9.2.4 指令历史

**command_history - 指令操作历史**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 记录 ID |
| command_id | TEXT UNIQUE | 指令唯一 ID |
| cluster_id | TEXT | 目标集群 |
| source | TEXT | 来源: web/ai |
| user_id | INTEGER | 操作用户 |
| action | TEXT | 操作类型: scale/restart/delete_pod |
| target_kind | TEXT | 目标资源类型 |
| target_namespace | TEXT | 目标命名空间 |
| target_name | TEXT | 目标资源名 |
| params | TEXT | 参数（JSON）|
| status | TEXT | 状态: pending/running/success/failed/timeout |
| result | TEXT | 执行结果（JSON）|
| error_message | TEXT | 错误信息 |
| created_at | TEXT | 创建时间 |
| started_at | TEXT | 开始执行时间 |
| finished_at | TEXT | 完成时间 |
| duration_ms | INTEGER | 执行耗时 |

#### 9.2.5 AI 操作日志

**ai_sessions - AI 会话**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 会话 ID |
| session_id | TEXT UNIQUE | 会话唯一标识 |
| user_id | INTEGER | 用户 ID |
| cluster_id | TEXT | 关联集群 |
| title | TEXT | 会话标题 |
| status | TEXT | 状态: active/closed |
| created_at | TEXT | 创建时间 |
| updated_at | TEXT | 更新时间 |
| closed_at | TEXT | 关闭时间 |

**ai_messages - AI 对话消息**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 消息 ID |
| session_id | TEXT FK | 会话 ID |
| role | TEXT | 角色: user/assistant/system |
| content | TEXT | 消息内容 |
| tokens_used | INTEGER | 消耗 Token 数 |
| created_at | TEXT | 创建时间 |

**ai_tool_calls - AI 工具调用**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 调用 ID |
| session_id | TEXT FK | 会话 ID |
| message_id | INTEGER FK | 关联消息 |
| tool_name | TEXT | 工具名称 |
| tool_input | TEXT | 输入参数（JSON）|
| tool_output | TEXT | 输出结果（JSON）|
| success | INTEGER | 是否成功 |
| error_message | TEXT | 错误信息 |
| duration_ms | INTEGER | 耗时 |
| created_at | TEXT | 调用时间 |

#### 9.2.6 告警配置

**alert_rules - 告警规则**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 规则 ID |
| name | TEXT | 规则名称 |
| description | TEXT | 描述 |
| cluster_id | TEXT | 目标集群（空=全部）|
| resource_type | TEXT | 资源类型 |
| condition | TEXT | 条件表达式（JSON）|
| severity | TEXT | 严重程度: info/warning/critical |
| enabled | INTEGER | 是否启用: 1/0 |
| notify_channels | TEXT | 通知渠道 ID 列表（JSON）|
| cooldown_sec | INTEGER | 冷却时间（秒）|
| created_at | TEXT | 创建时间 |
| updated_at | TEXT | 更新时间 |
| created_by | INTEGER | 创建人 |

**alert_history - 告警历史**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 告警 ID |
| rule_id | INTEGER FK | 规则 ID |
| cluster_id | TEXT | 集群 |
| severity | TEXT | 严重程度 |
| title | TEXT | 告警标题 |
| message | TEXT | 告警内容 |
| resource_kind | TEXT | 资源类型 |
| resource_namespace | TEXT | 命名空间 |
| resource_name | TEXT | 资源名 |
| status | TEXT | 状态: firing/resolved |
| fired_at | TEXT | 触发时间 |
| resolved_at | TEXT | 恢复时间 |
| notified | INTEGER | 是否已通知 |

**notify_channels - 通知渠道**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 渠道 ID |
| type | TEXT UNIQUE | 类型: slack/email（一个类型一条记录）|
| name | TEXT | 渠道显示名称 |
| enabled | INTEGER | 是否启用: 1/0（默认 0）|
| config | TEXT | 配置（JSON，敏感字段加密存储）|
| created_at | TEXT | 创建时间 |
| updated_at | TEXT | 更新时间 |

**notify_channels.config JSON 结构**

```json
// Slack
{
  "webhook_url": "https://hooks.slack.com/services/xxx"
}

// Email (SMTP)
{
  "smtp_host": "smtp.gmail.com",
  "smtp_port": 587,
  "smtp_user": "alerts@example.com",
  "smtp_password": "xxx",           // 加密存储
  "smtp_tls": true,
  "from_address": "alerts@example.com",
  "to_addresses": ["a@example.com", "b@example.com"]  // 支持多个
}
```

**Service 层有效性判断**

```go
// NotifyService.IsEffective 判断渠道是否真正生效
func (s *NotifyService) IsEffective(ch *NotifyChannel) bool {
    if !ch.Enabled {
        return false
    }
    switch ch.Type {
    case "slack":
        return ch.Config.WebhookURL != ""
    case "email":
        return ch.Config.SMTPHost != "" && len(ch.Config.ToAddresses) > 0
    }
    return false
}
```

**配置逻辑说明**

- 配置存储和读取都在 DB
- Service 层只负责判断是否生效
- 默认 `enabled=0`，即使 `enabled=1`，配置无效也不生效
- Web 端操作配置，最终生效需要 `enabled=1` 且配置有效

#### 9.2.7 集群 Event 持久化

**cluster_events - 集群事件表（核心表）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 事件 ID |
| event_uid | TEXT UNIQUE | K8s Event UID |
| cluster_id | TEXT | 集群 ID |
| namespace | TEXT | 命名空间 |
| name | TEXT | Event 名称 |
| type | TEXT | 类型: Normal/Warning |
| reason | TEXT | 原因: Scheduled/Pulled/Created/Started/Killing... |
| message | TEXT | 详细消息 |
| source_component | TEXT | 来源组件: kubelet/scheduler/controller... |
| source_host | TEXT | 来源主机 |
| involved_kind | TEXT | 关联资源类型: Pod/Node/Deployment... |
| involved_name | TEXT | 关联资源名 |
| involved_namespace | TEXT | 关联资源命名空间 |
| involved_uid | TEXT | 关联资源 UID |
| first_timestamp | TEXT | 首次发生时间 |
| last_timestamp | TEXT | 最后发生时间 |
| count | INTEGER | 发生次数 |
| created_at | TEXT | 入库时间 |

**索引设计**

```sql
-- 按集群+时间范围查询
CREATE INDEX idx_events_cluster_time ON cluster_events(cluster_id, last_timestamp DESC);

-- 按关联资源查询（故障排查核心）
CREATE INDEX idx_events_involved ON cluster_events(cluster_id, involved_kind, involved_namespace, involved_name);

-- 按类型过滤（快速筛选 Warning）
CREATE INDEX idx_events_type ON cluster_events(cluster_id, type, last_timestamp DESC);

-- 按原因分类
CREATE INDEX idx_events_reason ON cluster_events(cluster_id, reason, last_timestamp DESC);
```

**event_resource_snapshot - 资源关联快照（可选）**

> 用于保存事件发生时关联资源的状态快照，便于事后分析

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INTEGER PK | 记录 ID |
| event_id | INTEGER FK | 关联事件 ID |
| resource_kind | TEXT | 资源类型 |
| resource_name | TEXT | 资源名 |
| resource_namespace | TEXT | 命名空间 |
| resource_uid | TEXT | 资源 UID |
| snapshot_data | TEXT | 资源状态快照（JSON）|
| created_at | TEXT | 创建时间 |

**Event 持久化设计**

```
┌─────────────────────────────────────────────────────────────────┐
│                    Event 持久化流程                              │
│                                                                 │
│  Agent V2                                                       │
│     │                                                           │
│     │── 快照（含 Events）                                       │
│     ▼                                                           │
│  AgentSDK 接收                                                  │
│     │                                                           │
│     ▼                                                           │
│  Processor 去重清理                                             │
│     │                                                           │
│     ▼                                                           │
│  DataHub（保留最近 30 分钟）                                    │
│     │                                                           │
│     │── 快照到达时触发 ──────────────────────────────┐          │
│     │                                                │          │
│     ▼                                                ▼          │
│  EventPersistService.Sync(clusterID)                            │
│     │                                                           │
│     ├── 1. 从 DataHub 获取当前集群所有 Events                   │
│     │                                                           │
│     ├── 2. 批量 UPSERT 到 RDB                                   │
│     │      • 存在 → 更新 count, last_timestamp                  │
│     │      • 不存在 → 插入新记录                                │
│     │                                                           │
│     └── 3. 完成（无状态，幂等）                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**UPSERT 语句（SQLite）**

```sql
INSERT INTO cluster_events (event_uid, cluster_id, ..., count, last_timestamp)
VALUES (?, ?, ..., ?, ?)
ON CONFLICT(event_uid) DO UPDATE SET
    count = excluded.count,
    last_timestamp = excluded.last_timestamp;
```

**设计优点**

| 特点 | 说明 |
|------|------|
| 简单 | 不需要维护同步位置、版本号等状态 |
| 可靠 | UPSERT 天然幂等，重复写入无副作用 |
| 无状态 | Master 重启不丢失同步进度 |
| 性能可控 | DataHub 限制 30 分钟，每次同步量有限 |

**Event 关联查询设计**

核心目标：不用到集群中查看 Event，直接在平台上关联查看

```
场景 1: 查看某个 Pod 的事件
────────────────────────────────
SELECT * FROM cluster_events
WHERE cluster_id = ?
  AND involved_kind = 'Pod'
  AND involved_namespace = ?
  AND involved_name = ?
ORDER BY last_timestamp DESC;

场景 2: 查看 Deployment 及其所有 Pod 的事件
────────────────────────────────
-- 先获取 Deployment 关联的 ReplicaSet
-- 再获取 ReplicaSet 下的所有 Pod UID
-- 最后查询所有相关事件

SELECT * FROM cluster_events
WHERE cluster_id = ?
  AND (
    (involved_kind = 'Deployment' AND involved_name = ?)
    OR
    (involved_kind = 'ReplicaSet' AND involved_name LIKE 'deployment-name-%')
    OR
    (involved_kind = 'Pod' AND involved_uid IN (...))
  )
ORDER BY last_timestamp DESC;

场景 3: 故障排查 - 按时间线查看所有 Warning 事件
────────────────────────────────
SELECT * FROM cluster_events
WHERE cluster_id = ?
  AND type = 'Warning'
  AND last_timestamp BETWEEN ? AND ?
ORDER BY last_timestamp DESC;

场景 4: 某个 Node 上的所有事件
────────────────────────────────
SELECT * FROM cluster_events
WHERE cluster_id = ?
  AND (
    (involved_kind = 'Node' AND involved_name = ?)
    OR
    source_host = ?
  )
ORDER BY last_timestamp DESC;
```

**Event 保留策略**

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| MASTER_EVENT_RETENTION_DAYS | Event 保留天数 | 30 |
| MASTER_EVENT_MAX_COUNT | 单集群最大事件数 | 100000 |
| MASTER_EVENT_CLEANUP_INTERVAL | 清理检查间隔 | 1h |

**Event 清理逻辑**

```
定时任务 (每小时):
  1. 删除超过 RETENTION_DAYS 的事件
  2. 如果单集群事件数超过 MAX_COUNT
     → 删除最早的事件，保留最新 MAX_COUNT 条
```

#### 9.2.8 系统设置

**settings - 系统设置**

| 字段 | 类型 | 说明 |
|------|------|------|
| key | TEXT PK | 设置键 |
| value | TEXT | 设置值（JSON）|
| description | TEXT | 描述 |
| updated_at | TEXT | 更新时间 |
| updated_by | INTEGER | 更新人 |

### 9.3 表关系图

```
users
  │
  ├──< tokens (1:N)
  │
  ├──< audit_logs (1:N)
  │
  ├──< command_history (1:N)
  │
  ├──< ai_sessions (1:N)
  │       │
  │       ├──< ai_messages (1:N)
  │       │
  │       └──< ai_tool_calls (1:N)
  │
  ├──< alert_rules.created_by (1:N)
  │
  └──< notify_channels.created_by (1:N)

clusters
  │
  ├──< command_history (1:N)
  │
  ├──< ai_sessions (1:N)
  │
  ├──< alert_history (1:N)
  │
  └──< cluster_events (1:N)           ← 核心：Event 持久化
          │
          └──< event_resource_snapshot (1:N)  ← 可选：资源快照

alert_rules
  │
  └──< alert_history (1:N)

notify_channels
  │
  └──< alert_rules.notify_channels (M:N via JSON)
```

### 9.4 数据库抽象接口

```go
// Database 数据库接口
type Database interface {
    // 用户
    UserRepository() UserRepository
    TokenRepository() TokenRepository

    // 集群
    ClusterRepository() ClusterRepository

    // 审计
    AuditRepository() AuditRepository
    SecurityEventRepository() SecurityEventRepository

    // 指令
    CommandHistoryRepository() CommandHistoryRepository

    // AI
    AISessionRepository() AISessionRepository

    // 告警
    AlertRuleRepository() AlertRuleRepository
    AlertHistoryRepository() AlertHistoryRepository
    NotifyChannelRepository() NotifyChannelRepository

    // 集群 Event（核心）
    ClusterEventRepository() ClusterEventRepository

    // 设置
    SettingsRepository() SettingsRepository

    // 生命周期
    Migrate() error
    Close() error
}

// ClusterEventRepository Event 持久化接口
type ClusterEventRepository interface {
    // 插入/更新事件（基于 event_uid 去重）
    Upsert(ctx context.Context, event *ClusterEvent) error
    UpsertBatch(ctx context.Context, events []*ClusterEvent) error

    // 按集群查询
    ListByCluster(ctx context.Context, clusterID string, opts EventQueryOpts) ([]*ClusterEvent, error)

    // 按关联资源查询（故障排查核心）
    ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*ClusterEvent, error)

    // 按类型查询
    ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*ClusterEvent, error)

    // 清理过期事件
    DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error)
    DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error)

    // 统计
    CountByCluster(ctx context.Context, clusterID string) (int64, error)
}

// NotifyChannelRepository 通知渠道接口
type NotifyChannelRepository interface {
    Create(ctx context.Context, channel *NotifyChannel) error
    Update(ctx context.Context, channel *NotifyChannel) error
    Delete(ctx context.Context, id int64) error

    GetByID(ctx context.Context, id int64) (*NotifyChannel, error)
    List(ctx context.Context) ([]*NotifyChannel, error)
    ListEnabled(ctx context.Context) ([]*NotifyChannel, error)

    // 更新测试状态
    UpdateTestStatus(ctx context.Context, id int64, status, message string) error
}
```

### 9.5 目录结构

```
atlhyper_master_v2/
│
├── database/                       # 数据库层
│   ├── interfaces.go               # Database 接口定义
│   ├── repository/                 # Repository 接口
│   │   ├── user.go
│   │   ├── token.go
│   │   ├── cluster.go
│   │   ├── audit.go
│   │   ├── command_history.go
│   │   ├── ai_session.go
│   │   ├── alert.go
│   │   ├── notify_channel.go       # 通知渠道
│   │   ├── cluster_event.go        # ← Event 持久化（核心）
│   │   └── settings.go
│   │
│   ├── sqlite/                     # SQLite 实现（当前）
│   │   ├── db.go
│   │   ├── migrations.go
│   │   └── impl/
│   │       ├── user.go
│   │       ├── token.go
│   │       ├── cluster_event.go    # ← Event 持久化实现
│   │       └── ...
│   │
│   └── mysql/                      # MySQL 实现（后续）
│       └── ...
│
├── notifier/                       # 通知服务
│   ├── interfaces.go               # Notifier 接口
│   ├── slack.go                    # Slack 通知
│   ├── email.go                    # Email (SMTP) 通知
│   ├── webhook.go                  # 通用 Webhook
│   ├── dingtalk.go                 # 钉钉通知
│   └── feishu.go                   # 飞书通知
```

### 9.6 数据库切换

```go
// 开发环境：SQLite
db := sqlite.New("data/master.db")

// 生产环境：MySQL（改一行）
db := mysql.New(mysqlDSN)

// 传入 Master
master := NewMaster(datahub, db, ...)
```

### 9.7 数据库配置项

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| MASTER_DB_TYPE | 数据库类型 | sqlite |
| MASTER_DB_PATH | SQLite 路径 | data/master.db |
| MASTER_DB_DSN | MySQL/PG 连接串 | - |
| MASTER_DB_MAX_CONNS | 最大连接数 | 10 |
| MASTER_EVENT_RETENTION_DAYS | Event 保留天数 | 30 |
| MASTER_EVENT_MAX_COUNT | 单集群最大事件数 | 100000 |
| MASTER_EVENT_CLEANUP_INTERVAL | Event 清理间隔 | 1h |

---

## 十、待定事项

- [ ] Query Layer DSL 语法设计
- [ ] 是否需要 WebSocket 推送变化
- [ ] 快照存储策略（保留多少份）
- [ ] 多 Master 实例支持（需要 Redis）
- [ ] AI 模块对接方式
- [x] RDB 数据库设计
- [x] 通知渠道配置设计 (Slack/Email/Webhook/钉钉/飞书)
- [x] 集群 Event 持久化设计（核心功能）

---

## 十一、参考

- Agent V2 架构设计（已实施）
- Elastic APM 架构模式
- PromQL / GraphQL 查询设计
