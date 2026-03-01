# APM 数据策略分析

> 按页面视图 + API 端点分析 APM 数据的存储与获取策略

---

## 一、页面结构与数据需求

### 页面: `/observe/apm/`

```
APM 页面（三层导航）
├── Level 1: 服务列表
│   ├── ServiceTopology（服务拓扑图：服务节点 + DB 节点 + 调用边）
│   ├── ServiceList（服务表格：名称、RPS、成功率、P99、错误数）
│   └── 自动刷新（30s）
│
├── Level 2: 服务详情
│   ├── ServiceTrendCharts（4 条趋势线: RPS/成功率/延迟/错误）
│   ├── TransactionsTable（操作/事务列表，可点击进入 Level 3）
│   ├── ErrorTracesList（最近失败 Trace）
│   ├── SlowTracesList（最慢 Trace）
│   ├── StatusCodeChart（HTTP 状态码分布饼图）
│   └── DBStatsTable（数据库调用统计）
│
└── Level 3: Trace 瀑布图
    ├── Span 树形展示（缩进 + 时间轴）
    ├── Span 详情（HTTP/DB 属性、K8s 元数据、Events）
    └── 关联日志链接（→ /observe/logs?traceId=xxx）
```

**关键特征**:
- Level 1 只需聚合统计 → 内存直读
- Level 2 需要时序趋势 + 操作级统计 → Concentrator + 内存
- Level 3 需要完整 Span 链路 → **必须查 ClickHouse**
- Trace ↔ Log 强关联

---

## 二、API 端点分析

### 2.1 Level 1: 服务列表（首屏加载）

| 端点 | Handler | 数据源 | 延迟 |
|------|---------|--------|------|
| `GET /traces/services` | `TracesServices()` | `OTelSnapshot.APMServices` | <10ms |
| `GET /traces/topology` | `TracesTopology()` | `OTelSnapshot.APMTopology` | <10ms |
| `GET /traces/operations` | `TracesOperations()` | `OTelSnapshot.APMOperations` | <10ms |

**首屏并发加载**:
```typescript
const [services, topology, operations] = await Promise.all([
    getAPMServices(clusterId),     // 快照直读
    getTopology(clusterId),         // 快照直读
    getOperations(clusterId),       // 快照直读
]);
// 总延迟 < 50ms，全部从内存
```

**非默认时间范围** (>15min):
```
GET /traces/services?time_range=1h
  → 快照无法满足 → 创建 Command(action="query_traces", sub_action="list_services")
  → Agent 查询 ClickHouse → 返回
  → 带 TTL 缓存
```

### 2.2 Level 2: 服务详情

| 端点 | 数据源 | 延迟 |
|------|--------|------|
| `GET /traces/services/{name}/series` | Concentrator `APMTimeSeries` (≤60min) 或 ClickHouse (>60min) | <50ms / 2-5s |
| `GET /traces?service=X` | `OTelSnapshot.RecentTraces` (15min) 或 ClickHouse | <10ms / 1-3s |
| `GET /traces/stats?sub_action=http_stats` | Command → ClickHouse | 1-3s |
| `GET /traces/stats?sub_action=db_stats` | Command → ClickHouse | 1-3s |

**服务时序趋势图**:
```
≤60min → OTelSnapshot.APMTimeSeries (Concentrator 预聚合, 1min 粒度)
>60min → Command → Agent → ClickHouse (自动调整桶大小)
```

### 2.3 Level 3: Trace 瀑布图

| 端点 | 数据源 | 延迟 |
|------|--------|------|
| `GET /traces/{traceId}` | **必须** Command → Agent → ClickHouse | 2-10s |

```sql
-- Agent 侧执行
SELECT Timestamp, TraceId, SpanId, ParentSpanId, SpanName, SpanKind,
       ServiceName, Duration, StatusCode, StatusMessage,
       SpanAttributes, ResourceAttributes, Events
FROM otel_traces
WHERE TraceId = ?
ORDER BY Timestamp
```

**这是唯一无法从内存获取的核心数据** — Trace 的完整 Span 链路无法预聚合。

### 2.4 Trace ↔ Log 关联

```
Level 3 中用户点击 Span → 想看关联日志
  → 前端导航到 /observe/logs?traceId=xxx&spanId=yyy
  → Logs 页面 Command 路径: SELECT * FROM otel_logs WHERE TraceId = ?
  → 无时间窗口限制（确保能找到）
```

---

## 三、当前存储现状

### OTelSnapshot 中的 APM 数据

```go
OTelSnapshot {
    // Dashboard 列表（聚合统计）
    APMServices    []apm.APMService       // 服务级聚合: name, rps, successRate, p99Ms...
    APMTopology    *apm.Topology          // 拓扑图: nodes[] + edges[]
    APMOperations  []apm.OperationStats   // 操作级聚合: serviceName, operationName, rps...

    // 最近 Trace（首屏用）
    RecentTraces   []apm.TraceSummary     // 最近 ~50 个 Trace 摘要

    // 预聚合时序（Concentrator, 1min 粒度 × 60 点）
    APMTimeSeries  []APMServiceTimeSeries {
        ServiceName string
        Namespace   string
        Points      []APMTimePoint        // timestamp, rps, successRate, avgMs, p99Ms, errorCount
    }
}
```

### Ring Buffer 存储（90 份）

每 10 秒一份 → 90 份中每份都包含：

| 字段 | 单份大小估算 | 90 份总计 |
|------|------------|----------|
| `APMServices[]` | ~2KB (10 服务) | ~180KB |
| `APMTopology` | ~3KB | ~270KB |
| `APMOperations[]` | ~5KB (50 操作) | ~450KB |
| `RecentTraces[]` | ~10KB (50 条) | ~900KB |
| `APMTimeSeries[]` | ~30KB (10 服务 × 60 点) | ~2.7MB |

---

## 四、问题分析

### 4.1 Ring Buffer 中的冗余

| 字段 | 90 份是否必要 | 分析 |
|------|-------------|------|
| `APMServices[]` | **不需要** | 15min 聚合统计，只需最新 1 份 |
| `APMTopology` | **不需要** | 拓扑变化极慢，只需最新 1 份 |
| `APMOperations[]` | **不需要** | 操作级聚合，只需最新 1 份 |
| `RecentTraces[]` | **不需要** | 最近 Trace 摘要，只需最新 1 份 |
| `APMTimeSeries[]` | **不需要** | 每份都是独立完整的 60min 预聚合 |

**APM 的所有 Dashboard 端点都只读 `GetOTelSnapshot()`（最新 1 份），从不使用 Ring Buffer 历史。**

### 4.2 AIOps 引擎对 APM 数据的使用

当前 AIOps 引擎（`engine.go:72`）从 Ring Buffer 获取最近 30s 的 OTel 数据：

```go
timeline, _ := e.store.GetOTelTimeline(clusterID, time.Now().Add(-30*time.Second))
```

但它只使用 SLO 相关字段（`SLOServices`, `SLOIngress`），**不使用 APM 字段**。
根据 `aiops-otel-enhancement-design.md` 的计划，未来会增加 APM 指标提取，
但需要的仅是聚合值（如 `error_rate`, `avg_latency`），而不是完整的 `APMServices[]` 列表。

### 4.3 APM 聚合 vs 详情的清晰分界

```
聚合数据（内存可满足）:
  ├── APMServices — 服务级 RPS/成功率/延迟
  ├── APMTopology — 服务拓扑
  ├── APMOperations — 操作级统计
  ├── APMTimeSeries — 趋势图
  └── RecentTraces — 最近 Trace 摘要

详情数据（必须查 ClickHouse）:
  ├── TraceDetail — 完整 Span 链路（瀑布图）
  ├── HTTPStats — HTTP 状态码分布
  ├── DBStats — 数据库调用统计
  └── 关联日志 — 按 TraceId 查询 otel_logs
```

**APM 完全不需要在 Ring Buffer 中存全部数据。** 聚合数据只需最新 1 份，详情数据走 ClickHouse 实时查询。

---

## 五、优化结论

### APM 数据特征总结

| 特征 | 结论 |
|------|------|
| **首屏加载** | 聚合统计（APMServices + Topology + Operations）— 只需最新 1 份 |
| **趋势图** | Concentrator 预聚合（APMTimeSeries）— 只需最新 1 份 |
| **Trace 列表** | RecentTraces — 只需最新 1 份 |
| **Trace 详情** | **必须实时查 ClickHouse** — 无法预存 |
| **HTTP/DB 统计** | Command → ClickHouse |
| **Trace ↔ Log 关联** | Command → ClickHouse（按 TraceId 查 otel_logs） |
| **Ring Buffer 需求** | **完全不需要** |

### Ring Buffer 中 APM 的处理建议

```
当前:
  ClusterSnapshot.OTel (最新 1 份) → 全部 APM 聚合数据
  OTelRing[90] → 每份都含 APM 聚合数据 + 时序  ← 完全冗余

优化后:
  ClusterSnapshot.OTel (最新 1 份) → 全部 APM 聚合数据（不变）
  OTelRing[90] → 不存 APM 数据
```

### 需要保留的存储

```
最新 1 份 OTelSnapshot 中:
├── APMServices[]       → Level 1 服务列表
├── APMTopology         → Level 1 拓扑图
├── APMOperations[]     → Level 2 操作表
├── RecentTraces[]      → Level 2 Trace 列表
└── APMTimeSeries[]     → Level 2 趋势图（≤60min）

实时查询 ClickHouse:
├── TraceDetail         → Level 3 瀑布图
├── HTTPStats/DBStats   → Level 2 统计
└── 关联日志             → /observe/logs?traceId=xxx
```

### AIOps 集成不受影响

AIOps 需要的 APM 聚合指标（error_rate, avg_latency, rps）可以从最新 1 份 OTelSnapshot 中的 `APMServices[]` 提取，不需要 Ring Buffer 历史。

---

## 六、文件结构分析

### 6.1 当前 APM 文件分布

```
=== 前端 ===

atlhyper_web/src/
├── app/observe/apm/
│   ├── page.tsx                            # APM 主页面（三层导航 + 状态编排）
│   └── components/
│       ├── ServiceList.tsx                  # Level 1: 服务列表
│       ├── ServiceTopology.tsx              # Level 1: 服务拓扑图
│       ├── ServiceOverview.tsx              # Level 2: 服务详情概览（含 4 个子标签页）
│       ├── TransactionsTable.tsx            # Level 2: 操作/事务表
│       ├── DependenciesTable.tsx            # Level 2: 依赖表
│       ├── ServiceTrendCharts.tsx           # Level 2: 趋势图（RPS/成功率/延迟/错误）
│       ├── StatusCodeChart.tsx              # Level 2: HTTP 状态码分布
│       ├── DBStatsTable.tsx                 # Level 2: 数据库调用统计
│       ├── ErrorTracesList.tsx              # Level 2: 错误 Trace 列表
│       ├── SlowTracesList.tsx               # Level 2: 慢 Trace 列表
│       ├── TraceWaterfall.tsx               # Level 3: Trace 瀑布图
│       ├── ThroughputChart.tsx              # 吞吐量图表
│       ├── ErrorRateChart.tsx               # 错误率图表
│       ├── LatencyChart.tsx                 # 延迟图表
│       ├── LatencyDistribution.tsx          # 延迟分布直方图
│       ├── MiniSparkline.tsx                # 迷你火花线
│       ├── ImpactBar.tsx                    # 影响度条
│       └── SpanTypeChart.tsx                # Span 类型图表
├── types/model/apm.ts                       # APM TypeScript 类型定义
├── datasource/apm.ts                        # APM 数据源适配（TimeParams + API 调用）
├── api/observe.ts                           # ⚠️ 混合：APM + Logs + Metrics + SLO API 调用
├── mock/apm/
│   └── traces.ts                            # Mock Trace 数据
└── config/data-source.ts                    # 数据源开关

=== Master 后端 ===

atlhyper_master_v2/
├── gateway/handler/
│   ├── observe_apm.go                       # APM Handler（7 个端点）
│   └── observe.go                           # ⚠️ 共用基础：TTL 缓存 + executeQuery()
├── gateway/routes.go                        # 路由注册
└── service/query/otel.go                    # ⚠️ 混合：GetOTelSnapshot() 返回全部信号数据

=== Agent 后端 ===

atlhyper_agent_v2/
├── repository/
│   ├── interfaces.go                        # TraceQueryRepository 接口定义
│   └── ch/
│       ├── query/trace.go                   # ClickHouse Trace 查询实现（~718 行）
│       └── dashboard.go                     # ⚠️ 混合：OTelDashboardRepository（4 信号域）
└── service/snapshot/
    └── snapshot.go                          # ⚠️ 混合：getOTelSnapshot() 采集 4 信号域

=== 共享模型 ===

model_v3/apm/trace.go                        # APM 数据模型（Span, TraceSummary, TraceDetail...）
model_v3/cluster/snapshot.go                 # ⚠️ 混合：OTelSnapshot 包含 4 信号域字段
```

### 6.2 耦合问题

#### 问题一：`api/observe.ts` 混合 4 信号域 API

`atlhyper_web/src/api/observe.ts` 是前端的统一 Observe API 文件，混合了 APM/Logs/Metrics/SLO 的全部 API 调用（~335 行）。

**影响**:
- 修改 APM API 时需要在一个包含全部信号域的大文件中操作
- 前端组件通过 `datasource/apm.ts` 间接调用，不直接导入此文件

**注**: 这属于组织问题而非功能耦合，因为各信号域的 API 函数之间无相互依赖。

**解决方案**: 将 `api/observe.ts` 中的 Traces 相关函数拆分到 `api/apm.ts`：

```
迁出函数:
  getTracesList()          → api/apm.ts
  getTracesServices()      → api/apm.ts
  getTracesTopology()      → api/apm.ts
  getTracesOperations()    → api/apm.ts
  getAPMServiceSeries()    → api/apm.ts
  getTracesHTTPStats()     → api/apm.ts
  getTracesDBStats()       → api/apm.ts
  getTraceDetail()         → api/apm.ts

保留在 observe.ts:
  Logs/Metrics/SLO 各自的函数（后续各自拆分）

更新导入:
  datasource/apm.ts 的 import 从 "@/api/observe" → "@/api/apm"
```

#### 问题二：Agent `snapshot.go` 混合 4 信号域采集

`service/snapshot/snapshot.go`（997 行）的 `getOTelSnapshot()` 方法内部并发采集 4 个信号域的数据：

```go
// snapshot.go 中的 OTel 采集（4 信号域混在一起）
func (s *snapshotService) getOTelSnapshot(...) {
    // 并发采集 8 个数据源:
    go dashboardRepo.ListAPMServices()      // APM
    go dashboardRepo.GetAPMTopology()        // APM
    go dashboardRepo.ListAPMOperations()     // APM
    go otelSummaryRepo.GetMetricsSummary()   // Metrics
    go dashboardRepo.ListAllNodeMetrics()    // Metrics
    go otelSummaryRepo.GetSLOSummary()       // SLO
    go dashboardRepo.ListServiceSLO()        // SLO
    go getRecentLogs()                       // Logs
}
```

**影响**:
- 无法独立管理或升级某个信号域的采集逻辑
- 文件过大（997 行），违反单文件 300 行限制

**解决方案**: 将 `getOTelSnapshot()` 中的 OTel 采集逻辑提取到 `otel_collector.go`：

```go
// service/snapshot/otel_collector.go（新增）

type otelCollector struct {
    summaryRepo   repository.OTelSummaryRepository
    dashboardRepo repository.OTelDashboardRepository
    logRepo       repository.LogQueryRepository
    conc          *concentrator.Concentrator
}

func newOTelCollector(
    summary repository.OTelSummaryRepository,
    dashboard repository.OTelDashboardRepository,
    logRepo repository.LogQueryRepository,
    conc *concentrator.Concentrator,
) *otelCollector {
    return &otelCollector{summaryRepo: summary, dashboardRepo: dashboard, logRepo: logRepo, conc: conc}
}

// CollectSummary 采集标量摘要（TTL = 5min）
func (c *otelCollector) CollectSummary(ctx context.Context) (summaryFields, error) { ... }

// CollectDashboard 采集 Dashboard 列表数据（TTL = 30s）— 12 个并发 goroutine
func (c *otelCollector) CollectDashboard(ctx context.Context) (dashboardFields, error) { ... }

// CollectTimeSeries 将 Dashboard 数据注入 Concentrator 输出预聚合时序
func (c *otelCollector) CollectTimeSeries(snapshot *cluster.OTelSnapshot) { ... }
```

```go
// snapshot.go 变更 — getOTelSnapshot() 简化为编排调用

func (s *snapshotService) getOTelSnapshot(ctx context.Context) *cluster.OTelSnapshot {
    snapshot := &cluster.OTelSnapshot{}

    // 标量摘要
    if !s.otelCollector.SummaryCacheFresh() {
        s.otelCollector.CollectSummary(ctx, snapshot)
    }
    // Dashboard 列表
    if !s.otelCollector.DashboardCacheFresh() {
        s.otelCollector.CollectDashboard(ctx, snapshot)
    }
    // Concentrator 时序
    s.otelCollector.CollectTimeSeries(snapshot)
    // SLO 窗口
    s.sloCollector.CollectWindows(ctx, snapshot)

    return snapshot
}
```

**效果**: `snapshot.go` 从 997 行降至 ~500 行，OTel 采集逻辑（~300 行）移至独立文件。

#### 问题三：`OTelDashboardRepository` 接口混合 4 信号域

`repository/interfaces.go` 中的 `OTelDashboardRepository` 定义了 10+ 个方法，涵盖 APM/Metrics/SLO/Logs 四个信号域的 Dashboard 数据：

```go
type OTelDashboardRepository interface {
    ListAPMServices(...)        // APM
    GetAPMTopology(...)         // APM
    ListAPMOperations(...)      // APM
    ListRecentTraces(...)       // APM
    ListAllNodeMetrics(...)     // Metrics
    GetMetricsSummary(...)      // Metrics
    ListServiceSLO(...)         // SLO
    GetSLOSummary(...)          // SLO
    GetLogsSummary(...)         // Logs
    ListRecentLogs(...)         // Logs
}
```

**影响**:
- 单一实现 (`dashboard.go`) 承担全部 4 信号域的聚合查询
- 新增信号域字段时必须修改共用接口

**解决方案**: 用接口组合替代单一大接口，保持实现不变：

```go
// repository/interfaces.go — 拆分为 4 个子接口 + 1 个组合接口

type APMDashboardRepository interface {
    ListAPMServices(ctx context.Context) ([]apm.APMService, error)
    GetAPMTopology(ctx context.Context) (*apm.Topology, error)
    ListAPMOperations(ctx context.Context) ([]apm.OperationStats, error)
    ListRecentTraces(ctx context.Context, limit int) ([]apm.TraceSummary, error)
}

type MetricsDashboardRepository interface {
    GetMetricsSummary(ctx context.Context) (*metrics.Summary, error)
    ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error)
}

type SLODashboardRepository interface {
    GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error)
    ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
    ListIngressSLOPrevious(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
    GetIngressSLOHistory(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error)
    ListServiceSLO(ctx context.Context, since time.Duration) ([]slo.ServiceSLO, error)
    ListServiceEdges(ctx context.Context, since time.Duration) ([]slo.ServiceEdge, error)
}

type LogsDashboardRepository interface {
    GetLogsSummary(ctx context.Context) (*log.Summary, error)
    ListRecentLogs(ctx context.Context, limit int) ([]log.Entry, error)
}

// OTelDashboardRepository 组合接口 — 保持向后兼容
type OTelDashboardRepository interface {
    APMDashboardRepository
    MetricsDashboardRepository
    SLODashboardRepository
    LogsDashboardRepository
}
```

**效果**:
- `dashboard.go` 实现不变（仍满足组合接口）
- `otel_collector.go` 可按需依赖最小子接口（如只注入 `APMDashboardRepository`）
- 新增信号域只需新增子接口 + 扩展组合接口

### 6.3 理想文件结构（整理后）

```
=== 前端（已达标，无需修改） ===

atlhyper_web/src/
├── app/observe/apm/
│   ├── page.tsx                             # 页面
│   └── components/*.tsx                     # 组件（18 个）
├── types/model/apm.ts                       # 类型
├── datasource/apm.ts                        # 数据源
└── mock/apm/*.ts                            # Mock

=== Master 后端（已基本达标） ===

atlhyper_master_v2/
├── gateway/handler/
│   ├── observe_apm.go                       # APM Handler（独立文件 ✅）
│   └── observe.go                           # 共用基础（可接受）
└── service/query/otel.go                    # OTel 快照查询

=== Agent 后端（需拆分 snapshot.go） ===

atlhyper_agent_v2/
├── repository/
│   ├── interfaces.go                        # 保持 TraceQueryRepository 独立
│   └── ch/query/trace.go                    # ClickHouse 实现（独立 ✅）
└── service/snapshot/
    ├── snapshot.go                          # 通用快照编排（调用各信号域采集器）
    └── otel_collector.go                    # ← 新增：OTel 采集逻辑（或按信号域再细分）
```

### 6.4 整理检查清单

| 检查项 | 当前 | 目标 |
|--------|------|------|
| 前端 APM 页面/组件是否独立 | ✅ 已隔离 | 无需修改 |
| 前端 APM 数据源/类型是否独立 | ✅ `datasource/apm.ts` + `types/model/apm.ts` | 无需修改 |
| 前端 API 调用是否独立 | ❌ 混在 `api/observe.ts` 中 | 可拆分为 `api/apm.ts`（优先级低） |
| Master Handler 是否独立 | ✅ `observe_apm.go` 独立文件 | 无需修改 |
| Agent 查询层是否独立 | ✅ `ch/query/trace.go` 独立 | 无需修改 |
| Agent 快照采集是否独立 | ❌ 混在 `snapshot.go` 中 | 拆分 OTel 采集逻辑 |
| 共享模型是否独立 | ✅ `model_v3/apm/` 独立 | 无需修改 |
