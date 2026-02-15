# AtlHyper - Claude 开发指南

> Claude Code 自动读取此文件。上下文切换时请先阅读。

## 语言

请始终使用**中文**回复。

---

## 目录

- **安全规范** — 开源仓库安全要求（最高优先级）
- **项目概述** — 技术栈与目录结构总览
- **一、全局开发规范** — 共用包、命名、目录结构、开发原则、TDD 流程、数据模型/转换层、大后端小前端（所有模块适用）
- **二、Master V2 开发规范** — 分层架构、依赖规则、数据流、扩展指南
- **三、Agent V2 开发规范** — 分层架构、数据流、通信协议、设计原则
- **四、Web 前端开发规范** — 模块化、权限数据策略、国际化、API 文件规范
- **五、运维规范** — Git 工作流、任务生命周期、设计文档工作流、docs 目录结构

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
├── common/                 # 共用工具包 (logger / crypto / gzip)
├── model_v2/               # 共用数据模型
├── cmd/                    # 各组件入口 main.go
├── docs/                   # 文档（设计/任务/参考）
├── go.mod                  # Go 模块 (全项目共用)
└── CLAUDE.md               # 本文件
```

---

## 一、全局开发规范

本节规范适用于所有模块（Master / Agent / Web）。

### 1.1 共用包

Agent、Master 两个组件共用：

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

### 1.6 TDD 开发流程（大规模设计/重构必须遵循）

单文件 bug 修复、UI 微调等小改动不强制 TDD。但涉及**跨模块设计、新功能实现、架构重构**时，必须遵循以下流程：

#### 核心原则：先写测试，再写实现，测试即规格

设计文档定义「做什么」，测试用例定义「怎样算做对了」。没有测试的设计只是意图，通过测试的代码才是成果。

#### 流程

```
1. 设计文档 → 2. 测试用例 → 3. 运行测试（全红）→ 4. 最小实现 → 5. 运行测试（全绿）→ 6. 重构
```

| 阶段 | 产出 | 要求 |
|------|------|------|
| **设计** | `docs/design/active/*.md` | 明确接口签名、数据流、边界条件 |
| **写测试** | `*_test.go` / `*.test.ts` | 覆盖正常路径 + 边界 + 错误场景，测试必须能独立运行 |
| **红灯确认** | 全部 FAIL | 证明测试有效（不是永远通过的空测试） |
| **最小实现** | 功能代码 | 只写让测试通过的最少代码，不做多余设计 |
| **绿灯确认** | 全部 PASS | 每完成一个函数/模块立即运行测试 |
| **重构** | 优化后仍全绿 | 测试守护下安全重构，不破坏行为 |

#### 适用范围

| 场景 | 是否强制 TDD |
|------|-------------|
| 新模块/新功能（跨 3+ 文件） | **必须** |
| 架构重构 / 数据模型变更 | **必须** |
| 设计文档驱动的任务 | **必须** |
| 单文件 bug 修复 | 建议（补回归测试） |
| UI 样式调整 / 文案修改 | 不需要 |

#### Go 后端测试规范

- 测试文件与实现文件同目录：`converter.go` → `converter_test.go`
- 使用表驱动测试（table-driven tests）
- Mock 外部依赖（SDK / DB），不依赖真实集群
- 测试函数命名：`Test<函数名>_<场景>`（如 `TestConvert_EmptyInput`）

#### 前端测试规范

- 数据转换 / 工具函数：单元测试
- API 层 transform：快照测试或断言测试
- 组件：仅测试复杂交互逻辑，不测纯展示

#### 检查清单

- [ ] 设计文档中已列出关键接口和边界条件
- [ ] 测试用例在实现代码之前编写
- [ ] 红灯阶段确认所有测试 FAIL（测试本身有效）
- [ ] 实现完成后所有测试 PASS
- [ ] 重构后测试仍然全绿

### 1.7 数据模型与转换层规范

#### JSON Tag 统一为 camelCase

| 包 | 用途 | JSON Tag | 示例 |
|----|------|---------|------|
| `model_v2/` | Agent/Master 共享模型 | **camelCase** | `nodeName`, `createdAt`, `ownerKind` |
| `master_v2/model/` | Master API 响应模型 | **camelCase** | `cpuText`, `memoryText`, `startTime` |

**全栈统一 camelCase**：前端 TypeScript 直接使用后端返回的 JSON 字段名，无需转换。

#### Convert 层（model_v2 → model 转换）

路径：`atlhyper_master_v2/model/convert/`

**职责：** 将 Agent 上报的 `model_v2` 结构转换为前端可直接渲染的 `model` 扁平结构。

**命名模式：**

| 函数类型 | 命名 | 示例 |
|---------|------|------|
| 列表项 | `ResourceItem(src)` | `PodItem(src *model_v2.Pod) model.PodItem` |
| 批量转换 | `ResourceItems(src)` | `PodItems(src []model_v2.Pod) []model.PodItem` |
| 详情 | `ResourceDetail(src)` | `PodDetail(src *model_v2.Pod) model.PodDetail` |

**标准实现：**
```go
func PodItem(src *model_v2.Pod) model.PodItem {
    return model.PodItem{
        Name:      src.Summary.Name,
        Age:       src.Summary.Age,        // 后端已计算
        StartTime: src.Summary.CreatedAt.Format(timeFormat), // 后端格式化
    }
}
func PodItems(src []model_v2.Pod) []model.PodItem {
    if src == nil { return []model.PodItem{} } // 返回空切片而非 nil
    result := make([]model.PodItem, len(src))
    for i := range src { result[i] = PodItem(&src[i]) }
    return result
}
```

**职责边界：**
- **Convert 做**：字段映射、时间格式化、单位转换、派生字段计算
- **Convert 不做**：业务逻辑、数据过滤、聚合统计（属于 Service 层）
- **辅助函数** 放 `helpers.go`：`formatTime()`, `formatAge()` 等

#### Master Handler 标准模式

路径：`atlhyper_master_v2/gateway/handler/`

```go
type PodHandler struct {
    svc service.Query  // 持有最小接口
}
func NewPodHandler(svc service.Query) *PodHandler { ... }
```

**List 端点（6 步模式）：**
1. 检查 HTTP 方法 → 2. 提取必需参数 → 3. 构建查询选项 → 4. 调用 Service → 5. 调用 Convert → 6. 统一响应

**统一 JSON 响应格式：**
```json
{ "message": "获取成功", "data": [...], "total": 10 }
```

使用 `writeJSON(w, statusCode, data)` 和 `writeError(w, statusCode, msg)` 统一输出。

### 1.8 大后端小前端架构原则

#### 核心思想：后端输出即前端展示，前端不做数据加工

API 返回的数据必须是前端可以直接渲染的最终形态。前端只负责「调用 API → 绑定数据 → 渲染 UI」，不承担业务逻辑和数据处理。

#### 后端职责（数据加工全部在此完成）

| 职责 | 说明 | 示例 |
|------|------|------|
| **字段转换** | model_v2 → model 的映射、重命名、扁平化 | `CreatedAt time.Time` → `"createdAt": "2025-01-01T00:00:00Z"` |
| **计算派生字段** | 基于原始数据计算出展示所需的值 | 根据 `CreatedAt` 计算 `age: "3d"` |
| **格式化** | 时间格式化、单位换算、状态文本映射 | `time.Time` → ISO 8601 字符串 |
| **过滤与排序** | 按条件筛选、排序 | namespace 过滤在 service/query 层完成 |
| **聚合统计** | 汇总计数、分组统计 | 如需要，在 API 响应中直接返回统计数据 |

#### 前端职责（仅限 UI 层面）

| 允许 | 禁止 |
|------|------|
| 调用 API 获取数据 | 对 API 返回数据做二次计算/转换 |
| 将数据绑定到组件渲染 | 在前端重新聚合、分组、排序原始数据 |
| UI 状态管理（loading、error、分页） | 实现本应在后端完成的业务逻辑 |
| 纯展示层的格式化（如 i18n 文本映射） | 解析、拆分、重组后端返回的字段 |
| 客户端筛选（已加载数据的即时过滤） | 从多个 API 拉取数据后在前端 join/merge |

#### 判断标准

> **如果删掉前端某段逻辑后数据无法正确展示，说明这段逻辑应该移到后端。**

前端代码中不应出现：
- `data.map(item => ({ ...item, computedField: someCalculation(item) }))` — 派生字段应由后端计算
- `data.reduce(...)` 用于业务统计 — 统计应由后端 API 提供
- 手动拼接多个 API 响应的数据 — 后端应提供聚合接口

前端代码中允许：
- `items.filter(item => item.name.includes(searchText))` — 已加载数据的客户端即时搜索
- `StatusBadge` 组件根据 `status` 值映射颜色 — 纯 UI 展示逻辑
- i18n 翻译键映射 — 国际化是前端职责

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

**类型驱动流程（新增翻译时）：**

```
1. types/i18n.ts 定义接口（如 PodTranslations）
2. Translations 接口中添加字段（如 pod: PodTranslations）
3. zh.ts / ja.ts 实现翻译
4. 组件中 const { t } = useI18n(); 使用 t.pod.xxx
```

**按模块分组**：`nav`（导航）、`common`（通用）、`pod`/`node`（按页面）。新增页面模块时先定义独立的 `XxxTranslations` 接口再组合进 `Translations`。

### 4.5 前端 API 文件规范

路径：`atlhyper_web/src/api/`

**文件结构（每个文件按此顺序）：**
```
1. import 语句
2. 查询参数类型（interface XxxParams）
3. 响应类型（interface XxxResponse）
4. API 方法（export function getXxxList / getXxxDetail）
5. 概览聚合（可选，已加载数据的客户端统计）
```

**命名约定：**
- 文件名：`pod.ts`、`node.ts`（与资源对应）
- 列表函数：`getXxxList(params)`
- 详情函数：`getXxxDetail(params)`
- 路径参数：`encodeURIComponent()` 编码

---

## 五、运维规范

### 5.1 Git 工作流

- **每次代码修改后必须提交 git** (本地 commit)
- **禁止自动 push 到 GitHub** — 只有用户明确要求时才能 push
- commit message 格式: 简洁描述变更内容，禁止添加 `Co-Authored-By` 等 AI 标识
- 开发时频繁 commit (安全网)，保留零散 commit 方便溯源

### 5.2 任务管理

#### 核心文件

| 文件 | 用途 |
|------|------|
| `docs/tasks/active/tracker.md` | 当前所有待办和进行中的任务 |
| `docs/tasks/archive/{name}-tasks.md` | 已完成任务的归档记录 |
| `docs/design/active/*.md` | 当前需要执行的设计文档 |
| `docs/design/archive/*.md` | 已完成的设计文档归档 |

#### 完整生命周期

```
需求 → 设计文档 → 任务拆分 → 执行 → 验证 → 归档
```

| 阶段 | 操作 | 产出 |
|------|------|------|
| **需求分析** | 理解目标、评估范围 | 初步方案 |
| **设计** | 编写设计文档 | `docs/design/active/{name}-design.md` |
| **任务拆分** | 从设计文档拆分可执行任务 | tracker.md 中新增任务组 |
| **执行** | 逐个完成任务，每完成一个更新 tracker | 代码 + git commit |
| **验证** | 构建通过 + 测试通过 | `go build` / `next build` |
| **归档** | 设计文档 active → archive，任务标记 ✅ | 清理 tracker.md |

#### tracker.md 格式约定

```markdown
## 功能名称 — 状态

> 原设计文档: [链接](../../design/active/xxx.md)

- Phase 1: 描述 — ✅ 完成
- Phase 2: 描述 — 🔄 进行中
  - 子任务 A ✅
  - 子任务 B（进行中）
- Phase 3: 描述 — 待办
```

状态标记：`✅ 完成` / `🔄 进行中` / 无标记=待办

#### 多阶段项目管理

大型任务拆分为 Phase，每个 Phase 独立可验证：

```
Phase 1: 数据模型 → Phase 2: 后端实现 → Phase 3: 前端对接
```

- 每个 Phase 完成后 commit + 更新 tracker
- Phase 间可并行的部分标注依赖关系
- 一个 Phase 完成≠整体完成，不要过早归档

#### 上下文保持（Claude Code 特有）

Claude Code 对话有上下文限制，切换对话时所有内存丢失。**必须确保：**

- **设计文档自包含**：包含完整的接口定义、数据流、文件清单，不依赖对话记忆
- **tracker 实时更新**：每完成一个子任务立即更新状态，新对话可以从 tracker 恢复进度
- **MEMORY.md 记录关键决策**：跨对话共享的设计决策写入 `~/.claude/projects/*/memory/MEMORY.md`
- **commit message 可溯源**：描述清楚做了什么，方便新对话通过 `git log` 理解历史

### 5.3 设计文档工作流

#### 生命周期

```
构思 → docs/design/future/   （未来想法，临时记录）
执行 → docs/design/active/   （当前正在实施的设计）
完成 → docs/design/archive/  （已完成，仅供参考）
```

#### 设计文档内容规范

一个好的设计文档至少包含：

| 章节 | 内容 |
|------|------|
| **背景** | 为什么做这件事 |
| **数据模型** | 新增/修改的 struct 定义（含 JSON tag） |
| **接口定义** | 新增/修改的函数签名 |
| **数据流** | 数据从哪来、经过哪些层、到哪去 |
| **文件变更清单** | 需要新增/修改的文件列表 |
| **验证方法** | 如何确认实现正确（测试命令、预期输出） |

#### 文档命名规范

所有 docs 下的文件使用 **kebab-case**：

- 设计文档：`{功能名}-{子模块}-{类型}.md`
  - 示例：`slo-otel-agent-design.md`、`node-metrics-phase1-infra.md`
- 任务文档：`{功能名}-tasks.md`
  - 示例：`node-metrics-tasks.md`、`slo-otel-tasks.md`
- 同一功能的文档使用**相同前缀**（如 `node-metrics-*`）

### 5.4 docs 目录结构

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
