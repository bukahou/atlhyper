# AtlHyper - Claude 开发指南

> Claude Code 自动读取此文件。上下文切换时请先阅读。

## 语言

请始终使用**中文**回复。

---

## 目录

- **安全规范** — 开源仓库安全要求（最高优先级）
- **项目概述** — 技术栈与目录结构总览
- **一、全局开发规范** — 共用包、命名、目录结构、开发原则（所有模块适用）
- **二、Master V2 开发规范** — 分层架构、依赖规则、数据流、扩展指南
- **三、Agent V2 开发规范** — 分层架构、数据流、通信协议、设计原则
- **四、Web 前端开发规范** — 模块化、权限数据策略、国际化
- **五、运维规范** — Git 工作流、任务管理、docs 目录结构

---

## 安全规范（开源仓库，最高优先级）

本项目是**公开开源仓库**，所有代码、注释、提交历史均对外可见。安全问题一旦提交即永久暴露。

| 分类 | 禁止行为 |
|------|----------|
| **代码** | 禁止硬编码密钥、Token、密码、API Key，必须通过配置文件或环境变量注入 |
| **注释** | 禁止在注释、TODO、示例中出现真实密钥或敏感值（如 `// apiKey = "sk-xxx"`） |
| **日志** | 禁止日志输出密码、Token、Secret 内容，脱敏后方可打印 |
| **数据展示** | K8s Secret 内容必须脱敏展示，禁止明文返回前端 |
| **输入校验** | API 入口必须校验输入，防止 SQL 注入、命令注入 |
| **配置默认值** | 配置文件中的默认值禁止填入真实密钥、个人 URL（如 Slack Webhook URL）、邮件账号密码、Secret 等，使用空字符串或占位符（如 `""`, `"your-webhook-url"`） |
| **主动提醒** | 发现用户正在写入疑似敏感信息（密钥、真实 URL、账号密码）时，必须立即提醒 |
| **提交检查** | commit 前必须确认不包含 .env、credentials、私钥等敏感文件 |

---

## 项目概述

AtlHyper 是一个多集群 Kubernetes 监控和运维管理平台，采用 Master-Agent 架构。

**技术栈：** Go (后端) + React/Next.js (前端) + SQLite (存储) + 内存 MQ (消息队列)

### 目录结构总览

```
atlhyper/
├── atlhyper_master_v2/     # Master V2 (主服务) — 详见「二、Master V2 开发规范」
├── atlhyper_agent_v2/      # Agent V2 (集群代理) — 详见「三、Agent V2 开发规范」
├── atlhyper_web/           # Web 前端 (React/Next.js)
├── atlhyper_metrics_v2/    # Metrics V2 (节点指标采集)
├── common/                 # 共用工具包 (logger / crypto / gzip)
├── model_v2/               # 共用数据模型
├── cmd/                    # 各组件入口 main.go
├── docs/                   # 文档
├── go.mod                  # Go 模块 (全项目共用)
└── CLAUDE.md               # 本文件
```

---

## 一、全局开发规范

本节规范适用于所有模块（Master / Agent / Metrics / Web）。

### 1.1 共用包

Agent、Master、Metrics 三个组件共用：

| 包               | 用途                                                     |
| ---------------- | -------------------------------------------------------- |
| `common/logger`  | 统一日志模块                                             |
| `common/crypto`  | AES-256-GCM 加解密（API Key 加密存储）                   |
| `common/gzip.go` | gzip 压缩/解压工具（Agent ↔ Master 通信）                |
| `model_v2/`      | 共享数据结构（ClusterSnapshot、Command、K8s 资源模型等） |

### 1.2 命名规范

| 类型     | 规范       | 示例                                 |
| -------- | ---------- | ------------------------------------ |
| 包名     | 小写单词   | `service`, `repository`              |
| 文件名   | 小写下划线 | `command.go`, `event_persist.go`     |
| 统一接口 | 大写开头   | `Service` (在 interfaces.go 中)      |
| 子包实现 | 导出结构体 | `QueryService`, `CommandService`     |
| 内部实现 | 小写开头   | `serviceImpl` (factory 内部)         |
| 工厂函数 | `New` + 职责 + 类型 | `NewSnapshotService(...)`, `NewPodRepository(...)` — 禁止 `New()` 或 `NewService()` 等模糊命名，必须体现具体职责 |

### 1.3 标准文件命名

| 文件名          | 职责     | 内容                           |
| --------------- | -------- | ------------------------------ |
| `interfaces.go` | 接口定义 | 对外暴露的 interface 类型      |
| `factory.go`    | 工厂函数 | `New...()` 构造函数            |
| `types.go`      | 数据模型 | struct 定义、常量              |
| `errors.go`     | 错误定义 | `var ErrXxx = errors.New(...)` |
| `{feature}.go`  | 功能实现 | 具体业务逻辑                   |

**规则：看到文件名就知道里面是什么。**

### 1.4 目录结构规范

#### 核心原则：接口在顶层，实现在子目录，按职责分层而非平铺

每一层遵循相同模式：**顶层只放 `interfaces.go`（接口定义），实现放在按职责划分的子目录中。**

#### 错误示例（以 Agent SDK 层为例）

```
sdk/
├── interfaces.go          # 接口
├── types.go               # 类型
├── client.go              # K8s 客户端 ─┐
├── core.go                # K8s Core API  │
├── apps.go                # K8s Apps API  ├─ 所有实现平铺，无法区分职责
├── batch.go               # K8s Batch API │
├── networking.go          # K8s 网络 API  │
├── metrics.go             # K8s Metrics  ─┘
├── ingress_client.go      # Ingress 客户端 ─┐ 不同 SDK 类型混在一起
└── receiver_server.go     # 接收服务器 ─────┘
```

#### 正确示例（当前 Agent SDK 层）

```
sdk/
├── interfaces.go              # 接口定义（K8sClient / IngressClient / ReceiverClient）
├── types.go                   # 共用类型
└── impl/                      # 实现按职责分子目录
    ├── k8s/                   # K8sClient 实现（主动拉取 K8s API）
    │   ├── client.go
    │   ├── core.go
    │   ├── apps.go
    │   ├── batch.go
    │   ├── networking.go
    │   └── metrics.go
    ├── ingress/               # IngressClient 实现（主动拉取 Ingress Controller）
    │   └── client.go
    └── receiver/              # ReceiverClient 实现（被动接收，内置缓存）
        └── server.go
```

同样的模式适用于 Repository 层和 Service 层：

```
repository/                        service/
├── interfaces.go                  ├── interfaces.go
├── k8s/        (21 个仓库)        ├── snapshot/    (快照服务)
├── metrics/    (指标仓库)          └── command/     (指令服务)
└── slo/        (SLO 仓库)             ├── command.go
                                       └── summary.go
```

#### 检查清单

- [ ] 看目录结构就能理解模块架构
- [ ] 顶层只有 `interfaces.go` + `types.go` 等定义文件
- [ ] 实现按职责分布在子目录中
- [ ] 对外接口放在包根目录
- [ ] 没有超过 5-7 个平铺文件

### 1.5 通用开发原则

#### SOLID 原则

| 原则             | 说明                        | 项目实例                                                                 |
| ---------------- | --------------------------- | ------------------------------------------------------------------------ |
| **S - 单一职责** | 每个模块/结构体只负责一件事 | agentsdk=传输, processor=处理, query=读取, operations=写入               |
| **O - 开闭原则** | 对扩展开放，对修改关闭      | 新增读取方法只需在 query/ 加实现 + interfaces.go 加签名                  |
| **L - 里氏替换** | 子类型可替换父类型          | QueryService 和 CommandService 都满足 Service 接口                       |
| **I - 接口隔离** | 不强迫依赖不需要的方法      | service.Query（只读）/ service.Ops（只写），Handler 按需依赖最小接口     |
| **D - 依赖倒置** | 依赖抽象而非具体实现        | Gateway 依赖 service.Service 接口，不知道 query/operations 的存在        |

#### 其他原则

| 原则           | 实践要求                                  |
| -------------- | ----------------------------------------- |
| **DRY**        | 跨项目共用的工具函数放 `common/`（如日志、加密、压缩）；跨项目共用的数据结构放 `model_v2/`（如 ClusterSnapshot、Command） |
| **KISS**       | 优先简单直接的实现，避免过度抽象          |
| **YAGNI**      | 不为假设的未来需求编写代码                |
| **关注点分离** | 传输/业务/存储严格分层                    |
| **最小知识**   | 模块只了解直接协作者                      |

#### 代码实践

- **面向接口编程**：上层持有接口类型而非具体类型
- **禁止跳层调用**：只能调用相邻下层
- **不保留死代码**：无引用的文件/函数直接删除
- **优先组合而非继承**：通过 struct embedding 和接口组合实现复用
- **错误处理**：使用 `fmt.Errorf("xxx: %w", err)` 包装错误，保留链路
- **Context 传递**：所有跨层调用传递 `context.Context`，用于超时和取消
- **并发安全**：共享状态使用 `sync.RWMutex`，channel 优先于锁

---

## 二、Master V2 开发规范

### 2.1 分层架构

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

### 2.2 目录结构

```
atlhyper_master_v2/
├── agentsdk/           # Agent 通信层
├── processor/          # 数据处理层
├── datahub/            # 数据存储 (Store: 快照/Agent/Event)
├── mq/                 # 消息队列 (CommandBus: 指令队列)
├── database/           # 数据库层 (持久化)
├── service/            # 统一服务层
│   ├── interfaces.go   #   Service 接口 (所有方法)
│   ├── factory.go      #   工厂函数 (组合 query + operations)
│   ├── query/          #   查询实现 (读取)
│   └── operations/     #   操作实现 (写入)
├── gateway/            # 网关层 (HTTP API)
├── ai/                 # AI 模块
├── config/             # 配置
└── master.go           # 启动入口
```

### 2.3 层级职责

| 层级                   | 职责                        | 可调用                        |
| ---------------------- | --------------------------- | ----------------------------- |
| **Gateway**            | HTTP Handler, 认证鉴权      | service.Query / service.Ops   |
| **Service**            | 统一业务接口                | -                             |
| **service/query**      | 读取查询实现                | Store (读), MQ (读), Database |
| **service/operations** | 写入操作实现                | MQ (写)                       |
| **Processor**          | 数据清洗, 转换              | Store (写)                    |
| **AgentSDK**           | Agent 通信协议              | Processor, MQ                 |
| **Store (datahub/)**   | 数据存储 (快照/Agent/Event) | -                             |
| **MQ (mq/)**           | 消息队列 (指令队列)         | -                             |
| **Database**           | 持久化数据库                | -                             |

### 2.4 依赖规则

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

### 2.5 禁止行为

| 禁止                      | 原因                   |
| ------------------------- | ---------------------- |
| Gateway 直接访问 DataHub  | 必须经过 Service 层    |
| Gateway 直接访问 Database | 必须经过 Service 层    |
| query/ 执行写入操作       | query 只负责读取       |
| Service 层处理 HTTP 请求  | HTTP 逻辑属于 Gateway  |
| Processor 访问 Database   | Processor 只写 DataHub |
| 跨层调用                  | 只能调用相邻下层       |

### 2.6 数据流

```
上行 (快照):   Agent → AgentSDK → Processor → Store
下行 (指令):   MQ.WaitCommand() → AgentSDK → Agent
查询 (Web):    Gateway → Service (query/) → Store/MQ → 返回
写入 (Web):    Gateway → Service (operations/) → MQ
```

### 2.7 接口规范

```go
// service/interfaces.go — 接口隔离
type Query interface {        // 只读查询
    ListClusters(...)
    GetPods(...)
    GetCommandStatus(...)
}

type Ops interface {          // 写入操作
    CreateCommand(...)
}

type Service interface {      // 组合接口 (master.go / Router 持有)
    Query
    Ops
}

// service/factory.go — 工厂函数
func NewService(q *query.QueryService, ops *operations.CommandService) Service { ... }

// service/query/impl.go — 读取实现
type QueryService struct { store datahub.Store; bus mq.Producer }

// service/operations/command.go — 写入实现
type CommandService struct { bus mq.Producer }
```

Gateway Handler 按需依赖最小接口：查询用 `service.Query`，操作用 `service.Ops`。

### 2.8 初始化顺序

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

### 2.9 扩展指南

#### 新增 API 端点

```
1. gateway/handler/ 新增 Handler (依赖 service.Service)
2. gateway/routes.go 注册路由
3. 如需新数据: service/query/ 新增读取方法 或 service/operations/ 新增写入方法
4. service/interfaces.go 添加新方法签名
```

#### 新增业务功能

```
1. 读取功能: service/query/ 新增方法 → service/interfaces.go 添加签名
2. 写入功能: service/operations/ 新增方法 → service/interfaces.go 添加签名
3. gateway/handler/ 通过 service.Service 调用新方法
```

#### 新增数据存储

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

#### 新增模块

```
atlhyper_master_v2/
└── ai/                       # 示例：AI 模块
    ├── interfaces.go         #   对外接口 (AIService)
    ├── service.go            #   实现
    └── llm/                  #   LLM 客户端抽象（外部依赖封装）
        ├── interfaces.go     #     LLMClient 接口
        └── gemini.go         #     Gemini 实现
```

新模块规则:

- 必须定义 `interfaces.go` 暴露接口
- 只能依赖 Service/Database 层
- 禁止直接访问 DataHub
- Gateway 通过接口调用新模块

---

## 三、Agent V2 开发规范

### 3.1 分层架构

```
SDK → Repository → Service → Scheduler
                                 ↓
                              Gateway → Master
```

依赖方向严格单向，调用链固定为 Service → Repository → SDK，**即使 Service 只是转发也不可跳层**。

### 3.2 目录结构

```
atlhyper_agent_v2/
├── sdk/
│   ├── interfaces.go              ← K8sClient / IngressClient / ReceiverClient 接口
│   └── impl/
│       ├── k8s/                   ← K8sClient 实现（主动拉取 K8s API）
│       ├── ingress/               ← IngressClient 实现（主动拉取 Ingress Controller）
│       └── receiver/              ← ReceiverClient 实现（被动接收外部推送，内置缓存）
├── repository/
│   ├── interfaces.go              ← 三类仓库接口（K8s / Metrics / SLO）
│   ├── k8s/                       ← 21 个 K8s 资源仓库实现
│   ├── metrics/                   ← 节点指标仓库（从 ReceiverClient 拉取）
│   └── slo/                       ← SLO 指标仓库（从 IngressClient 采集）
├── service/
│   ├── interfaces.go              ← SnapshotService / CommandService 接口
│   ├── snapshot/                  ← 快照采集服务实现
│   └── command/                   ← 指令执行服务实现
├── scheduler/                     ← 调度层（定时任务编排）
├── gateway/                       ← 通信层（Agent ↔ Master HTTP 通信）
├── config/                        ← 配置
├── model/                         ← Agent 内部模型
└── agent.go                       ← 入口：依赖注入与生命周期管理
```

### 3.3 组织规范

子包导入父包获取接口类型，构造函数返回父包接口：

```go
// repository/k8s/pod.go
package k8s
import "AtlHyper/atlhyper_agent_v2/repository"
func NewPodRepository(client sdk.K8sClient) repository.PodRepository { ... }
```

`agent.go` 中用别名区分同名子包：

```go
k8srepo "AtlHyper/atlhyper_agent_v2/repository/k8s"
metricsrepo "AtlHyper/atlhyper_agent_v2/repository/metrics"
snapshotsvc "AtlHyper/atlhyper_agent_v2/service/snapshot"
commandsvc "AtlHyper/atlhyper_agent_v2/service/command"
```

### 3.4 数据流

- **主动拉取型 SDK**（K8s / Ingress）：Repository 调用 SDK 获取数据
- **被动接收型 SDK**（Receiver）：SDK 接收并缓存数据，Repository 按需拉取
  ```
  外部推送 → ReceiverClient(SDK, 内存缓存) → MetricsRepository(拉取) → SnapshotService
  ```
- 禁止 SDK 层直接依赖 Repository 层

### 3.5 Agent ↔ Master 通信

- 3 条数据通道：Snapshot（推送）、Command Poll（拉取）、Heartbeat（推送）
- 快照统一打包为 `ClusterSnapshot`（含 K8s 资源 + NodeMetrics + SLO），单次 HTTP POST 发送
- 压缩/解压使用 `common.GzipBytes()` / `common.MaybeGunzipReaderAuto()`

### 3.6 初始化顺序

```go
1. SDK 层     — K8sClient, ReceiverClient
2. Gateway 层 — MasterGateway
3. Repository — K8s repos, MetricsRepository(ReceiverClient), SLORepository(IngressClient)
4. Service    — SnapshotService(所有 repos), CommandService(pod + generic repos)
5. Scheduler  — 调度器(services, gateway)
```

### 3.7 设计原则

- 面向接口编程：`agent.go` 中持有接口类型而非具体类型
- 调用链固定：Service → Repository → SDK，即使 Service 只是转发也不可跳层
- 不保留死代码：无引用的文件/函数直接删除

---

## 四、Web 前端开发规范

### 4.1 目录结构

```
atlhyper_web/src/
├── api/                # API 模块（按资源拆分，每个文件对应一类接口）
│   ├── request.ts      #   Axios 封装（拦截器、Token 注入）
│   ├── pod.ts          #   Pod 相关 API
│   ├── node.ts         #   Node 相关 API
│   └── ...
├── app/                # 页面路由（Next.js App Router）
│   ├── overview/       #   总览页
│   ├── cluster/        #   集群详情页
│   ├── workbench/      #   工作台（AI Chat 等）
│   ├── system/         #   系统管理
│   └── style-preview/  #   样式设计实验页（开发专用）
├── components/         # UI 组件（按功能模块分目录）
│   ├── common/         #   通用组件
│   ├── pod/            #   Pod 相关组件
│   ├── ai/             #   AI Chat 组件
│   └── ...
├── i18n/               # 国际化
│   ├── locales/        #   语言文件（zh.ts / ja.ts）
│   ├── context.tsx     #   i18n Context
│   └── index.ts        #   导出
├── hooks/              # 自定义 Hooks
├── store/              # 状态管理
├── types/              # TypeScript 类型定义
├── lib/                # 工具函数
└── utils/              # 通用工具
```

### 4.2 模块化规范

- **单文件禁止超过 300 行**：超过时必须按职责拆分为子模块
- **组件按功能分目录**：`components/pod/`、`components/ai/`，禁止所有组件平铺在 components/ 根目录
- **页面拆分**：页面 `page.tsx` 只做布局和状态编排，具体 UI 抽取为独立组件
- **API 按资源拆分**：每个文件对应一类资源（`pod.ts`、`node.ts`），禁止所有 API 堆在一个文件

### 4.3 权限与数据策略

后端 API 分为公开接口和鉴权接口：

| API 权限等级 | 登录前 | 登录后 |
|-------------|--------|--------|
| **Public** | 直接调用真实 API | 直接调用真实 API |
| **Viewer / Operator / Admin** | 使用 mock 数据展示 | 调用真实 API |

- 未登录时，非公开接口必须使用 mock 数据进行虚假展示，禁止调用真实 API
- 登录后切换为真实数据，mock 数据仅作为降级展示

### 4.4 国际化（i18n）

- **所有用户可见文本**必须通过 `i18n/locales/` 中的语言文件管理，禁止在代码中内联硬编码
- 支持语言：中文（`zh.ts`）、日文（`ja.ts`）
- **例外**：`style-preview` 页面不需要国际化（开发专用样式实验平台）

---

## 五、运维规范

### 5.1 Git 工作流

- **每次代码修改后必须提交 git** (本地 commit)
- **禁止自动 push 到 GitHub** — 只有用户明确要求时才能 push
- commit message 格式: 简洁描述变更内容，禁止添加 `Co-Authored-By` 等 AI 标识
- 开发时频繁 commit (安全网)，保留零散 commit 方便溯源

### 5.2 任务管理

- 当前任务: `docs/tasks/active/tracker.md`
- **每次开始工作前**: 先读取 tracker.md，了解当前进度
- **执行中**: 更新任务状态为 "进行中"
- **完成后**: 将任务归档到 `docs/tasks/archive/`，保持 tracker 只有待办和进行中的任务
- **上下文切换时**: 任务细节会丢失，因此所有关键信息必须写入任务文件

### 5.3 docs 目录

docs/ 目录结构:

```
docs/
├── README.md                  # 文档索引入口
├── static/                    # 不常更新 — 项目资料
│   ├── readme/                #   项目介绍（中文、日文）
│   ├── img/                   #   项目截图
│   └── reference/             #   API 参考、设计决策文档
├── design/                    # 经常更新 — 设计文档
│   ├── active/                #   当前需要执行的设计
│   ├── future/                #   未来想法、临时记录
│   └── archive/               #   已完成的设计
└── tasks/                     # 经常更新 — 任务管理
    ├── active/tracker.md      #   当前任务追踪
    ├── future/                #   未来待规划的任务
    └── archive/               #   已完成的任务记录
```

---

## 参考文档

| 文档              | 路径                                        |
| ----------------- | ------------------------------------------- |
| 文档索引          | `docs/README.md`                            |
| Master V2 设计决策 | `docs/static/reference/master-v2-design.md` |
| Agent V2 设计决策  | `docs/static/reference/agent-v2-design.md`  |
| API 参考          | `docs/static/reference/api-reference.md`    |
| 当前任务追踪      | `docs/tasks/active/tracker.md`              |

---

## 注意事项

1. **共用 `go.mod`** - 使用根目录的 go.mod
2. **依赖注入** - 所有模块通过接口交互
3. **国际化** - Web 前端支持中文/日文
4. **架构约束** - 严格遵守层级调用规则
