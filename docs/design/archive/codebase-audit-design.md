# 代码库规范审计报告与整改方案

> 状态：活跃
> 创建：2026-03-01
> 范围：CLAUDE.md 过时修正 + 代码规范违规整改 + MEMORY.md 修正

---

## 1. CLAUDE.md 过时问题（高优先级）

CLAUDE.md 是开发规范的唯一权威源，但大量描述已与实际代码严重脱节。以下逐项列出差异。

### 1.1 共享模型包：model_v2 → model_v3

**现状**：`model_v2/` 目录已不存在，`model_v3/` 已全面替代。

| 模块 | 实际使用 | 说明 |
|------|---------|------|
| Agent V2 | 100% `model_v3` | 零个文件 import model_v2 |
| Master V2（新模块） | `model_v3` | OTel、operations、MQ、aiops 等 |
| Master V2（旧模块） | 仍 `model_v2` | K8s datahub.Store、service.QueryK8s、convert 层 |

**CLAUDE.md 需修改的位置**：

- [ ] 项目概述 — 目录结构总览：`model_v2/` → `model_v3/`
- [ ] 1.1 共用包表：`model_v2/` → `model_v3/`
- [ ] 1.5 DRY 原则：`model_v2/` → `model_v3/`
- [ ] 1.7 数据模型与转换层规范：`model_v2/` → `model_v3/` — **需确认**：Master 中 model_v2 残留的迁移策略
- [ ] 二、Master V2 开发规范中所有 `model_v2` 引用
- [ ] 三、Agent V2 开发规范中所有 `model_v2` 引用

**待确认**：Master V2 中 54 个文件仍 import `model_v2`，是否有迁移计划？还是 model_v2/v3 长期共存？

### 1.2 技术栈描述不完整

**当前**：`Go (后端) + React/Next.js (前端) + SQLite (存储) + 内存 MQ (消息队列)`

**实际**：`Go (后端) + React/Next.js (前端) + SQLite (持久化) + ClickHouse (OTel 时序数据) + Redis (可选缓存/MQ) + 内存/Redis MQ`

- [ ] 项目概述 — 技术栈描述需更新
- [ ] ClickHouse 是 Agent 的关键依赖（`sdk/impl/clickhouse/`），用于 APM/Log/Metrics/SLO 时序查询
- [ ] Redis 已作为 DataHub 和 MQ 的可选后端（`datahub/redis/`、`mq/redis/`）

### 1.3 Agent V2 SDK 接口描述过时

**CLAUDE.md 描述**：

```
sdk/interfaces.go  ← K8sClient / IngressClient / ReceiverClient
sdk/impl/
├── k8s/
├── ingress/
└── receiver/
```

**实际代码**：

```
sdk/interfaces.go  ← K8sClient / ClickHouseClient（仅两个）
sdk/impl/
├── k8s/
└── clickhouse/
```

`IngressClient` 和 `ReceiverClient` 已不存在，被 `ClickHouseClient` 替代。

- [ ] 3.2 目录结构 — SDK 描述全部更新
- [ ] 3.4 数据流 — 被动接收型 SDK（Receiver）描述删除
- [ ] 3.5 Agent ↔ Master 通信 — 快照内容描述更新

### 1.4 Agent V2 Repository 描述过时

**CLAUDE.md 描述**：

```
repository/
├── k8s/        (21 个仓库)
├── metrics/    (指标仓库)
└── slo/        (SLO 仓库)
```

**实际代码**：

```
repository/
├── interfaces.go   ← 20+ K8s 接口 + 6 个 ClickHouse 查询接口 + 4 个 Dashboard 子接口
├── k8s/            (20 个 K8s 资源仓库)
└── ch/             (ClickHouse 查询仓库)
    ├── summary/    (OTelSummaryRepository)
    ├── dashboard/  (OTelDashboardRepository)
    └── query/      (Trace/Log/Metrics/SLO 查询)
```

`metrics/` 和 `slo/` 已不存在。

- [ ] 3.2 目录结构 — Repository 描述更新
- [ ] 新增 concentrator/ 模块说明

### 1.5 Agent V2 初始化顺序过时

**CLAUDE.md 描述**：

```
1. SDK 层 — K8sClient, ReceiverClient
2. Gateway 层
3. Repository — K8s repos, MetricsRepository(ReceiverClient), SLORepository(IngressClient)
4. Service — SnapshotService, CommandService
5. Scheduler
```

**实际顺序**：

```
1. SDK 层 — K8sClient（必选）
2. Gateway 层 — MasterGateway
3. Repository — K8s repos（21 个）
3.1 ClickHouse 客户端（可选）→ OTelSummary, Trace, Log, Metrics, SLO, Dashboard 6 个仓库
3.2 Concentrator（预聚合时序）
4. SnapshotService(k8s repos + otelSummary + dashboard + concentrator)
   CommandService(pod + generic + trace + log + metrics + slo repos)
5. Scheduler
```

- [ ] 3.6 初始化顺序 — 完全重写

### 1.6 Master V2 目录结构缺失模块

**CLAUDE.md 列出**：agentsdk, processor, datahub, mq, database, service, gateway, ai, config

**实际还有但未记录的**：

| 目录 | 用途 |
|------|------|
| `aiops/` | AIOps 引擎（依赖图、基线检测、风险评分、状态机、事件管理、AI 增强） |
| `notifier/` | 告警通知（Slack/Email 渠道、事件触发器、模板渲染） |
| `slo/` | SLO 模块（计算器、路由更新器） |
| `tester/` | 测试模块 |
| `model/` | Master 自有 API 响应模型 + Convert 层 |

- [ ] 2.2 目录结构 — 补充 aiops, notifier, slo, tester, model
- [ ] 2.3 层级职责表 — 补充新模块的层级和依赖规则

### 1.7 Master Service 接口示例过旧

**CLAUDE.md 示例**：

```go
type Query interface { ListClusters(...); GetPods(...); GetCommandStatus(...) }
type Ops interface { CreateCommand(...) }
```

**实际**：Query 已拆分为 5 个子接口：

```go
type QueryK8s interface { ... }
type QueryOTel interface { ... }
type QuerySLO interface { ... }
type QueryAIOps interface { ... }
type QueryOverview interface { ... }
type Query interface { QueryK8s; QueryOTel; QuerySLO; QueryAIOps; QueryOverview }
```

- [ ] 2.7 接口规范 — 更新示例代码

### 1.8 Web 前端目录过时

**CLAUDE.md 列出**：`overview/`, `cluster/`, `workbench/`, `system/`, `style-preview/`

**实际**（`workbench/` 和 `system/` 已不存在）：

```
app/
├── about/          # 关于页
├── admin/          # 管理（用户、角色、审计、指令、数据源）
├── aiops/          # AIOps（风险、事件、拓扑、AI Chat）
├── cluster/        # K8s 资源（17 个子路由）
├── observe/        # 可观测性（metrics, logs, apm, slo, landing page）
├── overview/       # 总览
├── settings/       # 设置（AI、通知）
└── style-preview/  # 样式预览（开发专用）
```

新增但未记录的前端目录：`datasource/`, `mock/`, `theme/`, `config/`

- [ ] 4.1 目录结构 — 全部更新

### 1.9 参考文档路径错误

| CLAUDE.md 引用 | 实际 |
|----------------|------|
| `docs/static/reference/api-reference.md` | **不存在**，实际为 `master-api-reference.md` + `web-api-reference.md` |

- [ ] 参考文档表 — 修正路径

### 1.10 Database 目录结构描述偏差

**CLAUDE.md 描述**：`database/repository/` + `database/sqlite/impl/`

**实际**：`database/repo/` + `database/sqlite/`（无 impl 子目录）

- [ ] 2.9 扩展指南 — 修正目录名

---

## 2. 代码规范违规（按模块分组）

### 2.1 Master V2 — Gateway 跳层问题（高严重性）

**规则**：Gateway 只能通过 Service 层访问数据，禁止直接访问 DataHub 或 Database。

**现状：10 个 Handler 直接持有 `database.DB`，绕过 Service 层**：

| Handler | 文件 | 直接访问 |
|---------|------|---------|
| `EventHandler` | `handler/event.go` | `h.db.Event.ListByCluster()` |
| `CommandHandler` | `handler/command.go` | `h.db.Command.List()` |
| `NotifyHandler` | `handler/notify.go` | `h.db.Notify.*()` |
| `AuditHandler` | `handler/audit.go` | `h.db.Audit.*()` |
| `SettingsHandler` | `handler/settings.go` | `h.db.*()` |
| `AIProviderHandler` | `handler/ai_provider.go` | `h.db.*()` |
| `SLOHandler` | `handler/slo.go` | `sloRepo database.SLORepository` |
| `UserHandler` | `handler/user.go` | `userRepo database.UserRepository` |

**另外 3 个 Handler 直接持有 MQ Bus**：

| Handler | 文件 | 问题 |
|---------|------|------|
| `OpsHandler` | `handler/ops.go` | `bus mq.Producer` |
| `ObserveHandler` | `handler/observe.go` | `bus mq.Producer` |
| `NodeMetricsHandler` | `handler/node_metrics.go` | `bus mq.Producer` |

**AgentSDK 也直接访问 Database**：`agentsdk/server.go` 持有 `CommandHistoryRepository`。

**整改方案**：将这些数据访问逻辑下沉到 `service/query/` 或 `service/operations/`，Handler 只通过 Service 接口调用。

- [ ] 评估整改优先级（哪些 Handler 改动量最大）
- [ ] 是否分批整改？先改高频使用的 Handler？

### 2.2 Master V2 — 工厂函数命名违规（中严重性）

**规则**：工厂函数必须用 `New` + 职责 + 类型，禁止模糊的 `New()`。

**15 处违规**：

| 文件 | 当前 | 应改为 |
|------|------|--------|
| `service/factory.go` | `New(q, ops)` | `NewService(q, ops)` |
| `service/query/impl.go` | `New(store, bus)` | `NewQueryService(store, bus)` |
| `database/factory.go` | `New(cfg, dialect)` | `NewDatabase(cfg, dialect)` |
| `datahub/factory.go` | `New(cfg)` | `NewStore(cfg)` |
| `mq/factory.go` | `New(cfg)` | `NewCommandBus(cfg)` |
| `processor/processor.go` | `New(cfg)` | `NewProcessor(cfg)` |
| `master.go` | `New()` | `NewMaster()` |
| `ai/llm/factory.go` | `New(cfg)` | `NewLLMClient(cfg)` |
| `ai/llm/openai/client.go` | `New(...)` | `NewOpenAIClient(...)` |
| `ai/llm/gemini/client.go` | `New(...)` | `NewGeminiClient(...)` |
| `ai/llm/anthropic/client.go` | `New(...)` | `NewAnthropicClient(...)` |
| `mq/memory/bus.go` | `New()` | `NewMemoryBus()` |
| `mq/redis/bus.go` | `New(cfg)` | `NewRedisBus(cfg)` |
| `datahub/redis/store.go` | `New(cfg)` | `NewRedisStore(cfg)` |
| `datahub/memory/store.go` | `New(...)` | `NewMemoryStore(...)` |

- [ ] 是否统一整改？还是只在新代码中遵循？
- [ ] 整改会影响 `master.go` 的初始化代码和所有调用点

### 2.3 Master V2 — 统一日志模块违规（中严重性）

**规则**：使用 `common/logger.Module("xxx")`，禁止标准库 `log.Printf`。

**53 处违规**，分布在 16 个文件：

| 文件 | 违规数 |
|------|--------|
| `database/sqlite/migrations.go` | 12 |
| `ai/llm/anthropic/client.go` | 8 |
| `notifier/manager.go` | 6 |
| `notifier/trigger/heartbeat.go` | 5 |
| `notifier/trigger/event.go` | 5 |
| `database/sync.go` | 4 |
| `database/repo/ai_provider.go` | 4 |
| 其他 9 个文件 | 各 1-2 |

- [ ] 是否统一整改？
- [ ] 部分文件（如 migrations.go）在启动阶段运行，此时 logger 可能未初始化——需确认

### 2.4 Master V2 — 目录平铺过多（中严重性）

| 目录 | 文件数 | 限制 | 建议 |
|------|--------|------|------|
| `gateway/handler/` | ~50 | 5-7 | 按域分子目录：`k8s/`, `observe/`, `aiops/`, `admin/` |
| `model/` | ~23 | 5-7 | 按资源类型分子目录 |
| `model/convert/` | ~39 | 5-7 | 按资源类型分子目录 |
| `database/repo/` | 17 | 5-7 | 按功能域分组 |
| `database/sqlite/` | ~16 | 5-7 | 按功能域分组 |

- [ ] handler/ 拆分优先级最高（日常开发最频繁接触）
- [ ] 是否现在整改还是等 Gateway 跳层问题一起解决？

### 2.5 Master V2 — 其他问题

| 问题 | 文件 | 说明 |
|------|------|------|
| `interfaces.go` 命名为 `interface.go`（单数） | `notifier/interface.go` 等 3 处 | 应统一为复数 `interfaces.go` |
| `database/interfaces.go` 814 行 | 混合了模型 + 接口 + Dialect | 应拆分为 `types.go` + `interfaces.go` |
| `service/interfaces.go` 反向依赖子包 | import `service/operations` | `CreateCommandRequest` 应定义在 service/ 根部 |
| `service/sync/` 不属于标准架构 | `event_persist.go` | query/operations 之外的第三个子包 |
| `aiops/` 顶层混放实现文件 | `patterns.go`, `stats.go`, `store.go`, `timeline.go` | 应移到子目录 |

### 2.6 Agent V2 — 死代码（低严重性）

| 文件 | 函数/类型 | 说明 |
|------|-----------|------|
| `repository/ch/query/log.go:402` | `scanFacets` | 无生产代码调用 |
| `repository/ch/query/helpers.go:50` | `computeRate` | 仅测试调用 |
| `repository/ch/query/helpers.go:142` | `safeDiv` | 仅测试调用 |
| `model/options.go:54-106` | `ExecOptions`, `ScaleOptions`, `PatchOptions` | 无任何引用 |

- [ ] 直接删除

### 2.7 Agent V2 — 其他问题

| 问题 | 文件 | 说明 |
|------|------|------|
| `concentrator/` 缺 `interfaces.go` | `concentrator/concentrator.go` | snapshotService 直接持有 `*concentrator.Concentrator` 具体类型 |
| `converter.go` 1827 行 | `repository/k8s/converter.go` | 过大，但后端无明确行数限制 |

### 2.8 Web 前端 — 超过 300 行（高严重性，系统性）

**46 个文件超过 300 行限制。** Top 10：

| 文件 | 行数 |
|------|------|
| `style-preview/page.tsx` | 2467 |
| `components/slo/MeshTab.tsx` | 895 |
| `components/navigation/Sidebar.tsx` | 699 |
| `observe/apm/components/TraceWaterfall.tsx` | 691 |
| `aiops/chat/page.tsx` | 534 |
| `observe/metrics/components/ClusterOverviewChart.tsx` | 475 |
| `cluster/pod/page.tsx` | 473 |
| `components/ai/MessageBubble.tsx` | 468 |
| `components/daemonset/DaemonSetDetailModal.tsx` | 467 |
| `components/node/NodeDetailModal.tsx` | 460 |

- [ ] 类型/翻译文件（`types/i18n.ts` 2206 行、`locales/zh.ts` 2010 行）是否豁免 300 行限制？
- [ ] style-preview 是开发专用页面，是否豁免？
- [ ] 是否分批整改？先改最大的几个？

### 2.9 Web 前端 — i18n 硬编码（高严重性，系统性）

以下组件大量硬编码中文字符串，未使用 `useI18n()`：

- `components/daemonset/DaemonSetDetailModal.tsx`
- `components/statefulset/StatefulSetDetailModal.tsx` + tabs
- `components/pod/PodLogsViewer.tsx`
- `components/ai/MessageBubble.tsx`
- `components/navigation/ThemeSwitcher.tsx`
- `components/auth/LoginDialog.tsx`
- `app/not-found.tsx`
- `app/settings/notif/components/TagInput.tsx`、`EmailCard.tsx`

page.tsx 级别普遍已使用 i18n，**但 components/ 下的 DetailModal 系列和工具组件普遍缺失**。

- [ ] 是否统一补全？
- [ ] 优先级：用户直接可见的组件 > 内部工具组件

### 2.10 Web 前端 — 大后端小前端违规（中严重性）

**15+ 个组件在前端使用 `.reduce()` 做聚合统计**：

| 页面/组件 | 前端计算内容 |
|-----------|-------------|
| `cluster/statefulset/page.tsx` | totalReplicas, totalReady |
| `cluster/daemonset/page.tsx` | totalDesired, totalReady |
| `cluster/job/page.tsx` | totalActive, totalSucceeded, totalFailed |
| `observe/slo/page.tsx` | avgP95, totalRPS, avgAvailability |
| `observe/apm/components/ServiceOverview.tsx` | totalRequests, avgLatencyMs, errorRate |
| `observe/apm/components/ServiceTopology.tsx` | totalCalls, avgP99 |
| `observe/metrics/components/NetworkCard.tsx` | totalRxPS, totalTxPS |
| `observe/metrics/components/DiskCard.tsx` | totalReadPS, totalWritePS |
| `components/slo/MeshTab.tsx` | totalStatusRequests |

这些汇总统计应由后端 API 直接返回。

- [ ] 后端是否已有这些聚合字段？还是需要新增 API？
- [ ] 优先整改哪些？

### 2.11 Web 前端 — 组件可见性违规（中严重性）

**12 个列表页面使用 `{items.length > 0 && <Table>}` 隐藏空列表**：

statefulset, quota, job, netpol, pvc, pv, cronjob, sa, limit, daemonset, ServiceList, slo/page

**8 个组件在无数据时 `return null`**：

GPUCard, MiniSparkline, ProbesDisplay, LatencyTab, MeshTab, LogHistogram, LogDetail, Toast（可接受）

- [ ] 列表页面应统一显示空状态提示
- [ ] 部分 `return null` 是合理的（Toast、Modal），需逐个评估

### 2.12 Web 前端 — 其他问题

| 问题 | 说明 |
|------|------|
| `api/cluster-resources.ts` 混合 8 种资源（404 行） | 应拆分为独立文件 |
| page.tsx 内联组件（FilterInput 等） | 应提取为共享组件 |
| Mock 数据内联在 page.tsx（chat、audit） | 应通过 datasource 层 |

---

## 3. MEMORY.md 过时修正

### 3.1 前端路由记录过时

**当前 MEMORY.md**：

```
/monitoring/*  — 监控相关（指标、日志）
/settings/*    — 设置（AI、通知）
/admin/*       — 管理（用户、角色、审计）
/cluster/*     — K8s 资源
```

**实际路由**：

```
/overview      — 总览
/observe/*     — 可观测性（metrics, logs, apm, slo）
/aiops/*       — AIOps（risk, incidents, topology, chat）
/cluster/*     — K8s 资源
/settings/*    — 设置（AI、通知）
/admin/*       — 管理（用户、角色、审计、指令、数据源）
```

- [ ] 修正 `/monitoring/*` → `/observe/*`
- [ ] 补充 `/aiops/*`、`/overview`

---

## 4. 整改优先级建议

| 优先级 | 任务 | 理由 |
|--------|------|------|
| **P0** | 更新 CLAUDE.md 过时描述 | 开发规范是所有开发的基础，过时描述会误导 |
| **P0** | 修正 MEMORY.md 路由 | 快速修复，1 分钟 |
| **P1** | Gateway 跳层问题评估 | 最严重的架构违规，但改动量大，需要设计方案 |
| **P1** | 前端 i18n 硬编码补全 | 影响国际化功能完整性 |
| **P2** | 工厂函数命名统一 | 批量重命名，影响面广但机械性强 |
| **P2** | 前端 300 行拆分 | 系统性问题，逐步改善 |
| **P2** | 前端大后端小前端整改 | 需要后端配合新增聚合字段 |
| **P3** | 统一 logger | 简单替换但文件数多 |
| **P3** | handler/ 目录拆分 | 建议与 Gateway 跳层整改一起做 |
| **P3** | Agent 死代码清理 | 简单删除 |
| **P3** | 前端组件可见性修复 | 逐个组件改 |

---

## 5. 合规确认（做得好的部分）

以下规范经审计确认完全合规，值得保持：

- Agent 层级依赖（Service → Repository → SDK）严格遵守
- Agent 面向接口编程（agent.go 持有接口类型）
- Agent Context 传递 / 错误包装
- Master AIOps 模块（目录结构、接口隔离、测试覆盖）
- 所有 Handler 统一使用 `writeJSON` / `writeError`
- 前端 datasource 层 mock/API 切换架构
- 前端 page.tsx 级别 i18n 覆盖
- Git 工作流和设计文档生命周期管理
