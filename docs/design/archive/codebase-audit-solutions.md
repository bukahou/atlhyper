# 代码库审计整改方案

> 状态：活跃
> 创建：2026-03-01
> 依赖：[codebase-audit-design.md](./codebase-audit-design.md)（问题清单）
> 本文档：每个问题的具体解决方案

---

## 目录

- **Phase 0** — model_v2 → model_v3 迁移（阻塞性，必须先做）
- **Phase 1** — CLAUDE.md 更新（与 Phase 0 同步）
- **Phase 2** — Master 架构违规整改
- **Phase 3** — 前端规范整改
- **Phase 4** — 低优先级清理

---

## Phase 0：model_v2 → model_v3 迁移

### 背景

`model_v2/` 已废弃（代码已恢复仅作迁移参考）。Master V2 中 54 个文件仍 import `model_v2`，需全部迁移到 `model_v3`。Agent V2 已全量使用 `model_v3`，无需处理。

### 类型映射表

| model_v2 | model_v3 | 说明 |
|----------|----------|------|
| `model_v2.ClusterSnapshot` | `cluster.ClusterSnapshot` | `model_v3/cluster/snapshot.go`，字段名已变（见下方差异） |
| `model_v2.Pod` | `cluster.Pod` | `model_v3/cluster/pod.go` |
| `model_v2.PodSummary` | `cluster.PodSummary` | 同上 |
| `model_v2.PodStatus` | `cluster.PodStatus` | 同上 |
| `model_v2.PodContainerDetail` | `cluster.PodContainerDetail` | 同上 |
| `model_v2.Node` | `cluster.Node` | `model_v3/cluster/node.go` |
| `model_v2.NodeSummary` | `cluster.NodeSummary` | 同上 |
| `model_v2.Deployment` | `cluster.Deployment` | `model_v3/cluster/deployment.go` |
| `model_v2.DeploymentSummary` | `cluster.DeploymentSummary` | 同上 |
| `model_v2.Service` | `cluster.Service` | `model_v3/cluster/service.go` |
| `model_v2.ServiceSummary` | `cluster.ServiceSummary` | 同上 |
| `model_v2.Ingress` | `cluster.Ingress` | `model_v3/cluster/service.go` |
| `model_v2.IngressSummary` | `cluster.IngressSummary` | 同上 |
| `model_v2.IngressSpec` | `cluster.IngressSpec` | 同上 |
| `model_v2.Event` | `cluster.Event` | `model_v3/cluster/event.go` |
| `model_v2.Namespace` | `cluster.Namespace` | `model_v3/cluster/namespace.go` |
| `model_v2.StatefulSet` | `cluster.StatefulSet` | `model_v3/cluster/workload.go` |
| `model_v2.DaemonSet` | `cluster.DaemonSet` | 同上 |
| `model_v2.ReplicaSet` | `cluster.ReplicaSet` | 同上 |
| `model_v2.Job` | `cluster.Job` | `model_v3/cluster/job.go` |
| `model_v2.CronJob` | `cluster.CronJob` | 同上 |
| `model_v2.PersistentVolume` | `cluster.PersistentVolume` | `model_v3/cluster/storage.go` |
| `model_v2.PersistentVolumeClaim` | `cluster.PersistentVolumeClaim` | 同上 |
| `model_v2.NetworkPolicy` | `cluster.NetworkPolicy` | `model_v3/cluster/policy.go` |
| `model_v2.ResourceQuota` | `cluster.ResourceQuota` | 同上 |
| `model_v2.LimitRange` | `cluster.LimitRange` | 同上 |
| `model_v2.ServiceAccount` | `cluster.ServiceAccount` | 同上 |
| `model_v2.CommonMeta` | `cluster.CommonMeta` | `model_v3/cluster/common.go` |
| `model_v2.ResourceRef` | `cluster.ResourceRef` | 同上 |
| `model_v2.NodeMetrics` | `metrics.NodeMetrics` | `model_v3/metrics/node_metrics.go` |
| `model_v2.SLOSnapshot` | — | v3 中 SLO 数据已嵌入 `cluster.OTelSnapshot`（`SLOEdges` 字段） |
| `model_v2.ServiceEdge` | `slo.ServiceEdge` | `model_v3/slo/slo.go` |
| `model_v2.OverviewSummary` | `cluster.ClusterSummary` | `model_v3/cluster/snapshot.go` |
| `model_v2.AgentInfo` | `agent.AgentInfo` | `model_v3/agent/agent.go` |
| `model_v2.Command` | `command.Command` | `model_v3/command/command.go` |

### 需要特别注意的差异

**1. SLOSnapshot / SLOData**

model_v2 的 `ClusterSnapshot` 有独立的 `SLOData *SLOSnapshot` 字段，包含 `Edges []ServiceEdge`。model_v3 中 SLO 数据移到了 `OTelSnapshot.SLOEdges`。

影响文件：
- `aiops/correlator/builder.go` — 使用 `snap.SLOData.Edges`
- `aiops/correlator/builder_test.go` — 构造 `model_v2.SLOSnapshot`

迁移方案：`BuildFromSnapshot` 签名需增加 `otel *cluster.OTelSnapshot` 参数（与 AIOps OTel 融合设计一致），从 `otel.SLOEdges` 获取边数据。

**2. NodeMetrics**

model_v2 中 `ClusterSnapshot.NodeMetrics` 是 `map[string]*NodeMetrics`。model_v3 中节点指标在 `OTelSnapshot.MetricsNodes` 和 `OTelSnapshot.NodeMetricsSeries`。

影响文件：
- `aiops/baseline/extractor.go` — `extractNodeMetrics(snap)` 使用 `snap.NodeMetrics`
- `model/convert/node_metrics.go` — 转换函数

迁移方案：`extractNodeMetrics` 改为从 `otel.MetricsNodes` 或 `OTelSnapshot` 获取数据。

**3. OverviewSummary**

model_v2 有独立的 `OverviewSummary` 类型。model_v3 中为 `cluster.ClusterSummary`（字段可能不同）。

影响文件：
- `model/convert/overview.go`
- `service/query/overview.go`

### 迁移步骤（按依赖顺序）

#### Step 0-1：对比 model_v2 和 model_v3 的字段差异

逐个类型对比字段名、类型、JSON tag，生成差异清单。**必须在动手改代码之前完成**。

#### Step 0-2：迁移 datahub 层（3 个文件）

```
datahub/interfaces.go     — Store 接口的参数/返回类型
datahub/memory/store.go   — 内存实现
datahub/redis/store.go    — Redis 实现
```

这是最底层的依赖，改完后上层才能迁移。
- `import "AtlHyper/model_v2"` → `import "AtlHyper/model_v3/cluster"` 等
- `*model_v2.ClusterSnapshot` → `*cluster.ClusterSnapshot`
- 检查字段访问是否兼容

#### Step 0-3：迁移 processor + agentsdk（2 个文件）

```
processor/processor.go    — 处理 Agent 上报的快照
agentsdk/snapshot.go      — 接收快照
```

这两个文件接收 Agent 的数据，直接写入 datahub。

#### Step 0-4：迁移 service 层（3 个文件）

```
service/interfaces.go     — Query 接口方法签名中的 model_v2 类型
service/query/k8s.go      — K8s 查询实现
service/query/overview.go — 概览查询实现
```

#### Step 0-5：迁移 model/convert 层（38 个文件 — 最大批次）

这是机械性替换最多的部分：
```
model/convert/pod.go          + pod_test.go
model/convert/node.go         + node_test.go
model/convert/deployment.go   + deployment_test.go
model/convert/service.go      + service_test.go
model/convert/ingress.go      + ingress_test.go
model/convert/event.go        + event_test.go
model/convert/namespace.go    + namespace_test.go
model/convert/statefulset.go  + statefulset_test.go
model/convert/daemonset.go    + daemonset_test.go
model/convert/job.go          + job_test.go
model/convert/cronjob.go      + cronjob_test.go
model/convert/pv.go           + pv_test.go
model/convert/pvc.go          + pvc_test.go
model/convert/network_policy.go + network_policy_test.go
model/convert/resource_quota.go + resource_quota_test.go
model/convert/limit_range.go    + limit_range_test.go
model/convert/service_account.go + service_account_test.go
model/convert/node_metrics.go   + node_metrics_test.go
model/convert/overview.go       + overview_test.go
```

迁移模式统一：
```go
// 旧
import "AtlHyper/model_v2"
func PodItem(src *model_v2.Pod) model.PodItem { ... }

// 新
import "AtlHyper/model_v3/cluster"
func PodItem(src *cluster.Pod) model.PodItem { ... }
```

每个文件：替换 import → 替换类型引用 → 检查字段名差异 → 运行测试。

#### Step 0-6：迁移 aiops（4 个文件）

```
aiops/baseline/extractor.go      — extractNodeMetrics/extractPodMetrics 等
aiops/baseline/extractor_test.go  — 测试用例中构造 model_v2 类型
aiops/correlator/builder.go      — BuildFromSnapshot
aiops/correlator/builder_test.go  — 测试用例
```

注意：`extractNodeMetrics` 和 `BuildFromSnapshot` 的 SLO 边需要特殊处理（见上方差异说明）。

#### Step 0-7：迁移其他文件（4 个文件）

```
gateway/handler/node_metrics.go   — Handler 中的 model_v2 引用
notifier/enrich/interface.go      — 通知数据丰富接口
notifier/trigger/heartbeat.go     — 心跳触发器
slo/route_updater.go              — SLO 路由更新
```

#### Step 0-8：验证 + 删除 model_v2

```bash
go build ./atlhyper_master_v2/...
go test ./atlhyper_master_v2/...
# 全部通过后删除 model_v2/
rm -rf model_v2/
go build ./...  # 最终确认
```

---

## Phase 1：CLAUDE.md 更新

在 Phase 0 完成后，同步更新 CLAUDE.md 中所有过时描述。

### 1.1 需修改的章节清单

| 章节 | 修改内容 |
|------|---------|
| **项目概述 — 技术栈** | 加入 ClickHouse、Redis |
| **项目概述 — 目录结构** | `model_v2/` → `model_v3/` |
| **1.1 共用包** | `model_v2/` → `model_v3/`，说明 model_v3 按领域子包划分 |
| **1.5 DRY 原则** | `model_v2/` → `model_v3/` |
| **1.7 数据模型与转换层** | Convert 层引用改为 model_v3 |
| **2.2 Master 目录结构** | 补充 aiops/, notifier/, slo/, tester/, model/ |
| **2.3 层级职责表** | 补充 AIOps, Notifier, SLO 模块 |
| **2.7 接口规范** | Query 子接口拆分示例 |
| **2.9 扩展指南** | `database/repository/` → `database/repo/`; `sqlite/impl/` → `sqlite/` |
| **3.2 Agent 目录结构** | SDK: `IngressClient/ReceiverClient` → `ClickHouseClient`; Repo: `metrics/slo/` → `ch/`; 补充 concentrator/ |
| **3.4 Agent 数据流** | 删除被动接收型 SDK 描述 |
| **3.5 通信** | 快照内容更新（含 OTelSnapshot） |
| **3.6 初始化顺序** | 全部重写（含 ClickHouse + Concentrator） |
| **4.1 Web 目录结构** | `workbench/system/` → `observe/aiops/admin/settings/about/`; 补充 datasource/mock/theme/config/ |
| **参考文档表** | `api-reference.md` → `master-api-reference.md` + `web-api-reference.md` |

### 1.2 MEMORY.md 修正

```markdown
### 前端路由结构
- `/overview` — 总览
- `/observe/*` — 可观测性（metrics, logs, apm, slo）
- `/aiops/*` — AIOps（risk, incidents, topology, chat）
- `/cluster/*` — K8s 资源（section 分隔线区分核心/工作负载/存储/策略/告警）
- `/settings/*` — 设置（AI、通知）
- `/admin/*` — 管理（用户、角色、审计、指令、数据源）
```

---

## Phase 2：Master 架构违规整改

### 2.1 Gateway 跳层问题

**问题**：10 个 Handler 直接持有 `database.DB`，绕过 Service 层。

**方案**：将数据库访问逻辑下沉到 service 层。

#### 分批整改计划

**批次 A — CRUD 类 Handler（直接迁移，逻辑简单）**

| Handler | 现状 | 整改 |
|---------|------|------|
| `AuditHandler` | `h.db.Audit.*()` | 新增 `service.QueryAudit` 接口 + `query/audit.go` 实现 |
| `NotifyHandler` | `h.db.Notify.*()` | 新增 `service.OpsNotify` 接口 + `operations/notify.go` 实现 |
| `SettingsHandler` | `h.db.*()` | 新增 `service.OpsSettings` 接口 |
| `AIProviderHandler` | `h.db.*()` | 新增 `service.OpsAIProvider` 接口 |
| `UserHandler` | `h.db.User.*()` | 新增 `service.OpsUser` 接口 |

模式：
```go
// 旧 (handler 直接访问 DB)
type AuditHandler struct { db *database.DB }
func (h *AuditHandler) List(...) { h.db.Audit.List(...) }

// 新 (handler 通过 service 接口)
type AuditHandler struct { svc service.Query }
func (h *AuditHandler) List(...) { h.svc.ListAuditLogs(...) }
```

**批次 B — 有业务逻辑的 Handler**

| Handler | 复杂度 | 说明 |
|---------|--------|------|
| `EventHandler` | 中 | 同时从 DataHub（实时）和 Database（历史）读取，需要在 service 层合并 |
| `CommandHandler` | 中 | 读历史指令 + 轮询结果 |
| `SLOHandler` | 高 | 持有 `sloRepo`，涉及 targets/domains 复杂查询 |

**批次 C — Handler 持有 MQ Bus**

| Handler | 方案 |
|---------|------|
| `OpsHandler` | `bus.WaitCommandResult()` 逻辑封装到 `service/operations/` |
| `ObserveHandler` | 同上 |
| `NodeMetricsHandler` | 同上 |

**批次 D — AgentSDK 访问 Database**

`agentsdk/server.go` 持有 `CommandHistoryRepository`，用于写入指令执行历史。方案：通过 Processor 层中转，或直接在 AgentSDK 中保留（因为 agentsdk 本身就是特殊层级，不走 Service）。

**需确认**：AgentSDK 写入指令历史是否确实不适合走 Service 层？如果 AgentSDK → Processor → Database 路径可行，则更干净。

### 2.2 工厂函数命名统一

**方案**：批量重命名 15 个 `New()` 函数。

执行方式：按依赖从底向上改——先改被调用方，再改调用方。

```
1. datahub/memory/store.go:   New() → NewMemoryStore()
2. datahub/redis/store.go:    New() → NewRedisStore()
3. datahub/factory.go:        New() → NewStore()
4. mq/memory/bus.go:          New() → NewMemoryBus()
5. mq/redis/bus.go:           New() → NewRedisBus()
6. mq/factory.go:             New() → NewCommandBus()
7. database/factory.go:       New() → NewDatabase()
8. processor/processor.go:    New() → NewProcessor()
9. ai/llm/openai/client.go:   New() → NewOpenAIClient()
10. ai/llm/gemini/client.go:  New() → NewGeminiClient()
11. ai/llm/anthropic/client.go: New() → NewAnthropicClient()
12. ai/llm/factory.go:        New() → NewLLMClient()
13. service/query/impl.go:    New() → NewQueryService()
14. service/factory.go:       New() → NewService()
15. master.go:                New() → NewMaster()
```

每个改完后更新 `master.go` 中的调用点。最后 `go build` 验证。

### 2.3 统一 logger

**方案**：将 53 处 `log.Printf` 替换为 `common/logger`。

```go
// 旧
import "log"
log.Printf("xxx: %v", err)

// 新
import "AtlHyper/common/logger"
var log = logger.Module("ModuleName")
log.Error("xxx", "err", err)
// 或
log.Info("xxx", "key", value)
```

按文件逐个替换，模块名取包名或功能名（如 `logger.Module("Database")`、`logger.Module("Notifier")`）。

**注意**：`database/sqlite/migrations.go`（12 处）在启动阶段运行，需确认 logger 此时是否已初始化。如果未初始化，可在 migrations 完成后再切换到统一 logger。

### 2.4 handler/ 目录拆分

**方案**：将 ~50 个文件按域分子目录。

```
gateway/handler/
├── k8s/           ← pod, node, deployment, service, ingress, ...（~20 个资源）
├── observe/       ← observe.go, observe_apm.go, observe_logs.go, observe_metrics.go, ...
├── aiops/         ← aiops_*.go（risk, baseline, graph, incident, ai）
├── admin/         ← user.go, audit.go, command.go, settings.go, notify.go
├── slo/           ← slo.go, slo_domains.go, slo_targets.go, slo_latency.go, slo_mesh.go
├── overview.go    ← 留顶层（单文件）
└── helper.go      ← 留顶层（共用）
```

**注意**：拆分后所有 Handler 的 package 名会变（如 `package k8s`），需要在 `routes.go` 中更新引用。**建议与 Gateway 跳层整改一起做**，避免两次大改。

### 2.5 其他 Master 问题

| 问题 | 方案 |
|------|------|
| `notifier/interface.go` → `interfaces.go` | 直接重命名（3 处） |
| `database/interfaces.go` 814 行 | 拆分为 `types.go`（模型定义）+ `interfaces.go`（Repository/Dialect 接口） |
| `service/interfaces.go` 反向依赖 `operations` 子包 | 将 `CreateCommandRequest` 移到 `service/types.go` |
| `service/sync/` 不属于标准架构 | 评估是否归入 operations/ 或独立为定时任务模块 |
| `aiops/` 顶层实现文件 | `patterns.go`, `stats.go`, `store.go`, `timeline.go` 移到 `aiops/incident/` |

---

## Phase 3：前端规范整改

### 3.1 300 行拆分策略

**优先级排序**（按行数和使用频率）：

| 优先级 | 文件 | 行数 | 拆分方案 |
|--------|------|------|---------|
| 高 | `slo/MeshTab.tsx` | 895 | 拆分为 MeshServiceList + MeshHistogram + MeshStatusCodes |
| 高 | `navigation/Sidebar.tsx` | 699 | 拆分为 SidebarNav + SidebarGroup + NavItem |
| 高 | `apm/TraceWaterfall.tsx` | 691 | 拆分为 WaterfallChart + SpanDetail + SpanRow |
| 高 | `aiops/chat/page.tsx` | 534 | 拆 ChatSidebar + ChatMessages + ChatInput 到 components/ |
| 中 | `cluster/pod/page.tsx` | 473 | 提取 FilterBar 为 `components/common/FilterBar.tsx`（可复用） |
| 中 | `metrics/ClusterOverviewChart.tsx` | 475 | 拆分为子图表组件 |
| 中 | `ai/MessageBubble.tsx` | 468 | 拆分为 TextMessage + CommandMessage + ToolResults |
| 低 | `style-preview/page.tsx` | 2467 | 开发专用页面，可豁免或按展示区域拆分 |

**类型/翻译文件豁免**：`types/i18n.ts`（2206 行）、`locales/zh.ts`/`ja.ts`（2010 行）、`types/cluster.ts`（976 行）作为数据密集型定义文件，建议豁免 300 行限制。在 CLAUDE.md 中明确标注例外。

**系统性问题 — FilterInput/FilterSelect 内联**：12+ 个 cluster 子页面都在 page.tsx 内定义相同的过滤组件。

方案：提取为 `components/common/ListPageFilter.tsx` 共享组件。

### 3.2 i18n 硬编码补全

**范围**：~15 个组件缺失 i18n。

**分批执行**：

批次 A — 高频可见组件：
- `components/navigation/ThemeSwitcher.tsx` — 3 个字符串
- `components/auth/LoginDialog.tsx` — 1 个字符串
- `app/not-found.tsx` — 3 个字符串

批次 B — DetailModal 系列（模式统一，批量改）：
- `components/daemonset/DaemonSetDetailModal.tsx`
- `components/statefulset/StatefulSetDetailModal.tsx` + tabs
- `components/job/JobDetailModal.tsx`
- `components/cronjob/CronJobDetailModal.tsx`
- `components/ingress/IngressDetailModal.tsx`
- `components/service/ServiceDetailModal.tsx`

步骤：
1. `types/i18n.ts` 新增对应的 `XxxDetailTranslations` 接口
2. `locales/zh.ts` + `ja.ts` 添加翻译
3. 组件中替换硬编码为 `t.xxx.yyy`

批次 C — 工具组件：
- `components/pod/PodLogsViewer.tsx`
- `components/ai/MessageBubble.tsx`
- `app/settings/notif/components/TagInput.tsx` + `EmailCard.tsx`

### 3.3 大后端小前端整改

**问题**：15+ 组件用 `.reduce()` 做前端聚合统计。

**分两种情况处理**：

**情况 A — 后端 API 已有聚合数据，前端重复计算**

检查后端 Overview API 是否已返回这些统计。如果已有，前端删除 `.reduce()` 直接用 API 字段。

**情况 B — 后端 API 未提供聚合数据，需新增**

| 页面 | 需要的聚合字段 | 后端方案 |
|------|---------------|---------|
| K8s 资源列表页（statefulset/daemonset/job 等） | totalReplicas, totalReady 等 | 在 List API 响应中增加 `summary` 字段 |
| SLO 页 | avgP95, totalRPS, avgAvailability | 后端 SLO 概览 API 直接返回 |
| APM ServiceOverview | totalRequests, errorRate | 后端 APM 服务聚合 API 直接返回 |

**优先做情况 A（删代码）**，情况 B 可以后续与后端 API 扩展一起做。

### 3.4 组件可见性修复

**列表页面统一模式**：

```tsx
// 旧（违规）
{items.length > 0 && <Table>...</Table>}

// 新（合规）
{items.length > 0 ? (
  <Table>...</Table>
) : (
  <EmptyState message={t.common.noData} />
)}
```

涉及 12 个 cluster 子页面，模式一致，可批量替换。

**`return null` 组件**：逐个评估——

| 组件 | 处理 |
|------|------|
| GPUCard | 无 GPU 时显示"无 GPU 设备" |
| MiniSparkline | 无数据时显示占位虚线 |
| LogHistogram | 无数据时显示空状态 |
| LogDetail | 无选中项时显示提示文案（不算违规，选择状态组件） |
| Toast | 合理，保持 `return null` |
| ProbesDisplay | 无探针时显示"无探针配置" |

### 3.5 其他前端问题

| 问题 | 方案 |
|------|------|
| `api/cluster-resources.ts` 混合 8 种资源 | 拆分为 `job.ts`, `cronjob.ts`, `pv.ts`, `pvc.ts`, `network-policy.ts`, `resource-quota.ts`, `limit-range.ts`, `service-account.ts` |
| Mock 数据内联在 page.tsx | 移到 `mock/` 目录，通过 datasource 层调用 |
| `datasource/cluster.ts` 超 300 行 | 拆分后自然解决（跟随 API 文件拆分） |

---

## Phase 4：低优先级清理

### 4.1 Agent 死代码删除

直接删除：
- `repository/ch/query/log.go` — `scanFacets` 函数
- `repository/ch/query/helpers.go` — `computeRate` + `safeDiv` 函数及对应测试
- `model/options.go` — `ExecOptions`, `ScaleOptions`, `PatchOptions` 类型

### 4.2 Agent concentrator 接口化

- 新增 `concentrator/interfaces.go`，定义 `Concentrator` 接口
- `service/snapshot/` 持有接口而非具体类型

### 4.3 Master interface.go → interfaces.go 重命名

- `notifier/interface.go` → `notifier/interfaces.go`
- `notifier/channel/interface.go` → `notifier/channel/interfaces.go`
- `notifier/enrich/interface.go` → `notifier/enrich/interfaces.go`

---

## 执行顺序总览

```
Phase 0: model_v2 → model_v3 迁移
  ├── Step 0-1: 字段差异对比
  ├── Step 0-2: datahub 层 (3 文件)
  ├── Step 0-3: processor + agentsdk (2 文件)
  ├── Step 0-4: service 层 (3 文件)
  ├── Step 0-5: convert 层 (38 文件)
  ├── Step 0-6: aiops (4 文件)
  ├── Step 0-7: 其他 (4 文件)
  └── Step 0-8: 验证 + 删除 model_v2/
         ↓
Phase 1: CLAUDE.md + MEMORY.md 更新
         ↓
Phase 2: Master 架构整改（可与 Phase 3 并行）
  ├── 2.1 Gateway 跳层（批次 A→B→C→D）
  ├── 2.2 工厂函数命名
  ├── 2.3 统一 logger
  ├── 2.4 handler/ 目录拆分（与 2.1 合并执行）
  └── 2.5 其他
         ↓
Phase 3: 前端规范整改（可与 Phase 2 并行）
  ├── 3.1 300 行拆分
  ├── 3.2 i18n 补全
  ├── 3.3 大后端小前端
  ├── 3.4 组件可见性
  └── 3.5 其他
         ↓
Phase 4: 低优先级清理
```

---

## 验证方法

每个 Phase 完成后：

```bash
# 后端
go build ./...
go test ./atlhyper_master_v2/... -count=1
go test ./atlhyper_agent_v2/... -count=1

# 前端
cd atlhyper_web && npx tsc --noEmit && cd ..
```
