# AtlHyper - Claude 开发指南

> Claude Code 自动读取此文件。上下文切换时请先阅读。

---

## 语言

请始终使用**中文**回复。

---

## 项目概述

AtlHyper 是一个多集群 Kubernetes 监控和运维管理平台，采用 Master-Agent 架构。

**技术栈：** Go (后端) + React/Next.js (前端) + SQLite (存储) + 内存 MQ (消息队列)

---

## 目录结构

```
atlhyper/
├── atlhyper_master_v2/     # Master V2 (主服务)
│   ├── agentsdk/           #   Agent 通信层
│   ├── processor/          #   数据处理层
│   ├── datahub/            #   数据存储 (Store: 快照/Agent/Event)
│   ├── mq/                 #   消息队列 (CommandBus: 指令队列)
│   ├── database/           #   数据库层 (持久化)
│   ├── service/            #   统一服务层
│   │   ├── interfaces.go   #     Service 接口 (所有方法)
│   │   ├── factory.go      #     工厂函数 (组合 query + operations)
│   │   ├── query/          #     查询实现 (读取)
│   │   └── operations/     #     操作实现 (写入)
│   ├── gateway/            #   网关层 (HTTP API)
│   ├── config/             #   配置
│   └── master.go           #   启动入口
├── atlhyper_agent_v2/      # Agent V2 (集群代理)
├── atlhyper_web/           # Web 前端 (React/Next.js)
├── atlhyper_metrics/       # Metrics (未来)
├── _backup_v1/             # 旧版本备份 (禁止修改)
├── cmd/                    # 入口文件
├── model_v2/               # 共用模型
├── docs/                   # 文档
│   ├── architecture/       #   架构文档 (已实施)
│   ├── design/             #   设计方案 (待实施)
│   ├── guides/             #   开发指南
│   └── tasks/              #   任务追踪
├── go.mod                  # Go 模块
└── CLAUDE.md               # 本文件
```

---

## 架构规范 (重要)

### Master V2 分层架构

```
┌─────────────────────────────────────────────────────────────┐
│                         外部访问                             │
│                                                             │
│  Web/API ──> Gateway ──> Service (统一接口) ──> Store / MQ  │
│                            ├── query/       (读取实现)      │
│                            └── operations/  (写入实现)      │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                         内部处理                             │
│                                                             │
│  Agent ──> AgentSDK ──> Processor ──> Store                │
│                    └──> MQ (指令收发)                       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 层级职责

| 层级 | 职责 | 可调用 |
|------|------|--------|
| **Gateway** | HTTP Handler, 认证鉴权 | service.Query / service.Ops |
| **Service** | 统一业务接口 | - |
| **service/query** | 读取查询实现 | Store (读), MQ (读), Database |
| **service/operations** | 写入操作实现 | MQ (写) |
| **Processor** | 数据清洗, 转换 | Store (写) |
| **AgentSDK** | Agent 通信协议 | Processor, MQ |
| **Store (datahub/)** | 数据存储 (快照/Agent/Event) | - |
| **MQ (mq/)** | 消息队列 (指令队列) | - |
| **Database** | 持久化数据库 | - |

### 依赖规则 (必须遵守)

```
Gateway
   │
   └──> Service (统一接口)
           ├── query/        ──> Store (只读), MQ (只读), Database (只读)
           └── operations/   ──> MQ (只写)

AgentSDK
   │
   ├──> Processor
   │       └──> Store (只写)
   └──> MQ (WaitCommand, AckCommand)
```

### 禁止行为

| 禁止 | 原因 |
|------|------|
| Gateway 直接访问 DataHub | 必须经过 Service 层 |
| Gateway 直接访问 Database | 必须经过 Service 层 |
| query/ 执行写入操作 | query 只负责读取 |
| Service 层处理 HTTP 请求 | HTTP 逻辑属于 Gateway |
| Processor 访问 Database | Processor 只写 DataHub |
| 跨层调用 | 只能调用相邻下层 |

### 数据流

```
上行 (快照):   Agent → AgentSDK → Processor → Store
下行 (指令):   MQ.WaitCommand() → AgentSDK → Agent
查询 (Web):    Gateway → Service (query/) → Store/MQ → 返回
写入 (Web):    Gateway → Service (operations/) → MQ
```

---

## 开发规范

### 接口规范

- `service/interfaces.go` 定义 `Query` (只读) + `Ops` (写入) + `Service` (组合) 接口
- 子包 (`query/`, `operations/`) 提供具体实现，导出结构体类型
- `service/factory.go` 通过工厂函数组合子包实现为统一接口
- Gateway Handler 按需依赖最小接口：查询用 `service.Query`，操作用 `service.Ops`

```go
// service/interfaces.go — 接口隔离
type Query interface {        // 只读查询 (13 个 handler 依赖)
    ListClusters(...)
    GetPods(...)
    GetCommandStatus(...)
    // ... 所有 Get/List 方法
}

type Ops interface {          // 写入操作 (OpsHandler 依赖)
    CreateCommand(...)
}

type Service interface {      // 组合接口 (master.go / Router 持有)
    Query
    Ops
}

// service/factory.go — 工厂函数
func New(q *query.QueryService, ops *operations.CommandService) Service { ... }

// service/query/impl.go — 读取实现
type QueryService struct { store datahub.Store; bus mq.Producer }

// service/operations/command.go — 写入实现
type CommandService struct { bus mq.Producer }
```

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写单词 | `service`, `repository` |
| 文件名 | 小写下划线 | `command.go`, `event_persist.go` |
| 统一接口 | 大写开头 | `Service` (在 service/interfaces.go) |
| 子包实现 | 导出结构体 | `QueryService`, `CommandService` |
| 内部实现 | 小写开头 | `serviceImpl` (factory 内部) |
| 工厂函数 | `New` 前缀 | `New(...)`, `NewCommandService(...)` |

### 标准文件命名 (重要)

| 文件名 | 职责 | 内容 |
|--------|------|------|
| `interfaces.go` | 接口定义 | 对外暴露的 interface 类型 |
| `factory.go` | 工厂函数 | `New...()` 构造函数 |
| `types.go` | 数据模型 | struct 定义、常量 |
| `errors.go` | 错误定义 | `var ErrXxx = errors.New(...)` |
| `{feature}.go` | 功能实现 | 具体业务逻辑 |

```
示例：notifier/
├── interfaces.go     # AlertManager 接口
├── types.go          # Alert, Message 等模型
├── errors.go         # ErrChannelNotFound 等
├── manager/
│   ├── manager.go    # AlertManager 实现 + NewAlertManager 工厂
│   └── ...
└── channel/
    ├── channel.go    # Channel 接口 (适配器接口)
    └── ...
```

**规则：看到文件名就知道里面是什么**

### 目录结构规范 (重要)

#### 核心原则：按职责分层，而非平铺

所有文件平铺在同一目录会导致无法快速区分职责。必须按照**职责**和**调用关系**组织子目录。

#### 错误示例

```
notifier/
├── interfaces.go   # 接口
├── alert.go        # 模型
├── errors.go       # 错误
├── dedup.go        # 核心组件 ─┐
├── buffer.go       # 核心组件  ├─ 混在一起，无法区分
├── limiter.go      # 核心组件 ─┘
├── manager.go      # 核心调度
├── slack.go        # 发送适配器 ─┐
└── email.go        # 发送适配器 ─┘ 与核心逻辑并列

问题：所有文件平铺，无法一眼区分职责层次
```

#### 正确示例

```
notifier/
├── interfaces.go       # 对外接口 (AlertManager interface)
├── types.go            # 数据模型 (Alert, Message)
├── errors.go           # 错误定义
│
├── manager/            # 核心调度逻辑 (内聚)
│   ├── manager.go      #   AlertManager 实现 + NewAlertManager 工厂
│   ├── dedup.go        #   去重组件
│   ├── buffer.go       #   聚合组件
│   └── limiter.go      #   限流组件
│
└── channel/            # 发送适配器 (可插拔)
    ├── channel.go      #   Channel 接口
    ├── slack.go        #   Slack 实现
    └── email.go        #   Email 实现
```

#### 设计模式：端口-适配器架构 (Hexagonal Architecture)

```
              ┌─────────────────────────────────────┐
              │           Core Domain               │
              │  ┌───────────────────────────────┐  │
  Port ───────┼──│      manager/                 │──┼─────── Port
 (入口)       │  │  核心业务逻辑 (纯粹，无依赖)  │  │       (出口)
              │  └───────────────────────────────┘  │
              └─────────────────────────────────────┘
                   ↑                          ↓
              ┌─────────┐              ┌──────────────┐
              │ Adapter │              │   Adapter    │
              │ (调用方) │              │  (外部依赖)  │
              └─────────┘              └──────────────┘
```

#### 分层规则

| 层级 | 位置 | 职责 | 示例 |
|------|------|------|------|
| **接口层** | 包根目录 | 对外暴露的接口和类型 | `interfaces.go`, `types.go`, `errors.go` |
| **核心层** | `core/` 或 `manager/` | 业务逻辑 + 工厂函数 | 去重、聚合、限流、NewXxx() |
| **适配层** | `adapter/` 或 `channel/` | 外部依赖的封装 | Slack API, SMTP |

#### 检查清单

- [ ] 看目录结构就能理解模块架构
- [ ] 核心逻辑在同一子目录内聚
- [ ] 外部依赖（API、数据库）独立为适配器
- [ ] 对外接口放在包根目录
- [ ] 没有超过 5-7 个平铺文件

### 通用开发原则

#### SOLID 原则

| 原则 | 说明 | 本项目体现 |
|------|------|-----------|
| **S - 单一职责** | 每个模块/结构体只负责一件事 | agentsdk=传输, processor=处理, query=读取, operations=写入 |
| **O - 开闭原则** | 对扩展开放，对修改关闭 | 新增读取方法只需在 query/ 加实现 + interfaces.go 加签名 |
| **L - 里氏替换** | 子类型可替换父类型 | QueryService 和 CommandService 都满足 Service 接口 |
| **I - 接口隔离** | 不强迫依赖不需要的方法 | service.Query/Ops, mq.Producer/Consumer, Processor |
| **D - 依赖倒置** | 依赖抽象而非具体实现 | Gateway 依赖 service.Service 接口，不知道 query/operations 的存在 |

#### 其他原则

| 原则 | 说明 | 实践要求 |
|------|------|---------|
| **DRY** | Don't Repeat Yourself | 公共逻辑提取到共用包 (model_v2, datahub 接口) |
| **KISS** | Keep It Simple, Stupid | 优先简单直接的实现，避免过度抽象 |
| **YAGNI** | You Aren't Gonna Need It | 不为假设的未来需求编写代码 |
| **关注点分离** | 不同关注点放在不同模块 | 传输/业务/存储 严格分层 |
| **最小知识** | 模块只了解直接协作者 | Handler 不知道 DataHub 的存在 |

#### 代码实践

- **避免过度工程**：只做当前需要的事，不添加"以防万一"的抽象
- **优先组合而非继承**：通过 struct embedding 和接口组合实现复用
- **错误处理**：使用 `fmt.Errorf("xxx: %w", err)` 包装错误，保留链路
- **Context 传递**：所有跨层调用传递 `context.Context`，用于超时和取消
- **并发安全**：共享状态使用 `sync.RWMutex`，channel 优先于锁

### 初始化顺序

依赖注入必须按以下顺序:

```go
1. Store (datahub.New)          // 数据存储
2. Bus (mq.New)                 // 消息队列
3. Database                     // 持久化数据库
4. Processor                    // 数据处理 (依赖 Store)
5. query.QueryService           // 读取实现 (依赖 Store, Bus, Database)
6. operations.CommandService    // 写入实现 (依赖 Bus)
7. service.New(query, ops)      // 组合为统一 Service
8. AgentSDK                     // Agent 通信 (依赖 Processor, Bus)
9. Gateway                      // Web API (依赖 Service, Bus)
```

---

## 扩展指南

### 新增 API 端点

```
1. gateway/handler/ 新增 Handler (依赖 service.Service)
2. gateway/routes.go 注册路由
3. 如需新数据: service/query/ 新增读取方法 或 service/operations/ 新增写入方法
4. service/interfaces.go 添加新方法签名
```

### 新增业务功能

```
1. 读取功能: service/query/ 新增方法 → service/interfaces.go 添加签名
2. 写入功能: service/operations/ 新增方法 → service/interfaces.go 添加签名
3. gateway/handler/ 通过 service.Service 调用新方法
```

### 新增数据存储

```
实时数据 (易失):
  → datahub/interfaces.go 新增 Store 方法
  → datahub/memory/store.go 实现

消息队列:
  → mq/interfaces.go 新增 CommandBus 方法
  → mq/memory/bus.go 实现

持久化数据:
  → database/repository/ 新增 Repository 接口
  → database/sqlite/impl/ 实现
```

### 新增模块 (如 AI)

```
atlhyper_master_v2/
└── ai/
    ├── interfaces.go     # 对外接口 (AIService)
    ├── service.go        # 实现
    └── llm/              # LLM 客户端抽象
        ├── interfaces.go # LLMClient 接口
        └── gemini.go     # Gemini 实现
```

新模块规则:
- 必须定义 `interfaces.go` 暴露接口
- 只能依赖 Service/Database 层
- 禁止直接访问 DataHub
- Gateway 通过接口调用新模块

---

## 当前状态

- **Master V2**: 已实施，正在使用
- **Agent V2**: 已实施，正在使用
- **Web**: 已实施，i18n 完成
- **AI 功能**: 设计中 (见 `docs/design/ai-system.md`)

---

## 参考文档

| 文档 | 路径 |
|------|------|
| 文档索引 | `docs/README.md` |
| Master V2 架构 | `docs/architecture/master-v2.md` |
| Agent V2 架构 | `docs/architecture/agent-v2.md` |
| AI 系统设计 | `docs/design/ai-system.md` |
| 开发指南 | `docs/guides/claude-dev-guide.md` |

---

## 任务管理

- 任务文件: `docs/tasks/tracker.md`
- **每次开始工作前**: 先读取任务文件，了解当前进度
- **执行中**: 更新任务状态为 "进行中"
- **完成后**: 将任务归档到 `docs/tasks/archive/`，保持 tracker 只有待办和进行中的任务
- **上下文切换时**: 任务细节会丢失，因此所有关键信息必须写入任务文件

---

## Git 工作流

- **每次代码修改后必须提交 git** (本地 commit)
- **禁止自动 push 到 GitHub** — 只有用户明确要求时才能 push
- commit message 格式: 简洁描述变更内容
- 开发时频繁 commit (安全网)，push 前可 squash 合并

---

## docs 目录

Claude Code 不会自动读取 docs/ 目录。重要信息:
- `docs/tasks/tracker.md` — 当前任务追踪 (每次开始工作先读)
- `docs/tasks/archive/` — 已完成任务归档
- `docs/architecture/` — 已实施的架构文档
- `docs/design/` — 待实施的设计方案

---

## 注意事项

1. **禁止修改 `_backup_v1/`** - 旧版本备份
2. **共用 `go.mod`** - 使用根目录的 go.mod
3. **依赖注入** - 所有模块通过接口交互
4. **国际化** - Web 前端支持中文/日文
5. **架构约束** - 严格遵守层级调用规则
