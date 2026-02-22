# Observe 数据流优化：快照预推送 + Command 管道加速

## 背景

### 当前问题

所有 Observe API（Metrics/Traces/SLO）都通过 Command 机制实现：

```
Web → Master(创建 Command) → MQ → Agent(长轮询接收) → ClickHouse → Agent → Master → Web
```

每次 Web 请求都走完整链路，导致：

1. **高延迟**：每条 Command 需要 Agent 轮询周期（~1s 间隔） + ClickHouse 查询（~200ms） + HTTP 往返（~2ms）
2. **串行瓶颈**：Agent 每次只处理 1 条 Command，多条并发请求线性排队（N 条 → N × 1.2s）
3. **Dashboard 页面慢**：metrics 页面同时请求 summary + nodes = 2 条 Command → ~3s 才能加载

### 设计目标

- Dashboard 页面（列表/概览）**秒开**（从内存缓存读取，0 延迟）
- 详情/搜索页面（用户主动触发）延迟降至 **200-500ms**
- 不改变 Master-Agent 部署拓扑（Agent 在集群内，Master 在外部，支持多集群）

---

## 核心思路

**按 Web 页面用途将 Observe 端点分为两类：**

| 类型 | 数据特征 | 数据通道 | 延迟 |
|------|---------|---------|------|
| **Dashboard** | 可预测、全量概览、10s 更新即可 | Agent 定期采集 → 快照推送 → Master 内存读取 | ~0ms |
| **Detail** | 用户触发、带参数、单条查询 | Command 机制（优化后） | ~200-500ms |

---

## 端点分类

### Dashboard 端点（快照推送，共 8 个）

| # | 端点 | 当前 Action | 数据来源 |
|---|------|------------|---------|
| 1 | `GET /observe/metrics/summary` | query_metrics / get_summary | MetricsQueryRepo.GetMetricsSummary |
| 2 | `GET /observe/metrics/nodes` | query_metrics / list_all | MetricsQueryRepo.ListAllNodeMetrics |
| 3 | `GET /observe/traces/services` | query_traces / list_services | TraceQueryRepo.ListServices |
| 4 | `GET /observe/traces/topology` | query_traces / get_topology | TraceQueryRepo.GetTopology |
| 5 | `GET /observe/slo/summary` | query_slo / get_summary | SLOQueryRepo.GetSLOSummary |
| 6 | `GET /observe/slo/ingress` | query_slo / list_ingress | SLOQueryRepo.ListIngressSLO |
| 7 | `GET /observe/slo/services` | query_slo / list_service | SLOQueryRepo.ListServiceSLO |
| 8 | `GET /observe/slo/edges` | query_slo / list_edges | SLOQueryRepo.ListServiceEdges |

### Detail 端点（Command 按需查询，共 5 个）

| # | 端点 | 说明 | 触发场景 |
|---|------|------|---------|
| 1 | `GET /observe/metrics/nodes/{name}` | 单节点详情 | 点击节点 |
| 2 | `GET /observe/metrics/nodes/{name}/series` | 节点时序图 | 展开节点图表 |
| 3 | `GET /observe/traces` | Trace 列表（带筛选） | 搜索/过滤 |
| 4 | `GET /observe/traces/{id}` | Trace 瀑布图 | 点击某条 Trace |
| 5 | `POST /observe/logs/query` | 日志搜索 | 搜索日志 |
| 6 | `GET /observe/slo/timeseries` | SLO 时序图（单服务） | 点击某个服务 |

---

## 数据模型变更

### 扩展 OTelSummary → OTelSnapshot

**文件：** `model_v3/cluster/snapshot.go`

当前 `OTelSummary` 只包含标量摘要值。扩展为 `OTelSnapshot`，携带 Dashboard 所需的完整列表数据：

```go
// OTelSnapshot Agent 从 ClickHouse 定期采集的可观测性数据
// Dashboard 端点直接从此结构读取，无需 Command 中继
type OTelSnapshot struct {
    // ===== 原有摘要字段（保持不变）=====

    // APM 服务概览
    TotalServices   int     `json:"totalServices"`
    HealthyServices int     `json:"healthyServices"`
    TotalRPS        float64 `json:"totalRps"`
    AvgSuccessRate  float64 `json:"avgSuccessRate"`
    AvgP99Ms        float64 `json:"avgP99Ms"`

    // SLO 概览
    IngressServices int     `json:"ingressServices"`
    IngressAvgRPS   float64 `json:"ingressAvgRps"`
    MeshServices    int     `json:"meshServices"`
    MeshAvgMTLS     float64 `json:"meshAvgMtls"`

    // 基础设施指标概览
    MonitoredNodes int     `json:"monitoredNodes"`
    AvgCPUPct      float64 `json:"avgCpuPct"`
    AvgMemPct      float64 `json:"avgMemPct"`
    MaxCPUPct      float64 `json:"maxCpuPct"`
    MaxMemPct      float64 `json:"maxMemPct"`

    // ===== 新增 Dashboard 列表数据 =====

    // Metrics Dashboard
    MetricsSummary *metrics.Summary      `json:"metricsSummary,omitempty"`
    MetricsNodes   []metrics.NodeMetrics `json:"metricsNodes,omitempty"`

    // APM Dashboard
    APMServices []apm.APMService `json:"apmServices,omitempty"`
    APMTopology *apm.Topology    `json:"apmTopology,omitempty"`

    // SLO Dashboard
    SLOSummary  *slo.SLOSummary  `json:"sloSummary,omitempty"`
    SLOIngress  []slo.IngressSLO `json:"sloIngress,omitempty"`
    SLOServices []slo.ServiceSLO `json:"sloServices,omitempty"`
    SLOEdges    []slo.ServiceEdge `json:"sloEdges,omitempty"`
}
```

> **命名变更**：`OTelSummary` → `OTelSnapshot`，反映其已从"摘要"扩展为"完整快照"。
> 原有字段全部保留，只新增 8 个列表字段。`ClusterSnapshot.OTel` 字段类型同步更新。

### 快照大小预估

| 数据 | 预估大小（JSON） | 压缩后 |
|------|----------------|--------|
| 原有摘要 | ~500B | - |
| MetricsNodes（6 节点） | ~15KB | ~3KB |
| APMServices（10 服务） | ~2KB | ~500B |
| APMTopology（10 节点 + 15 边） | ~3KB | ~800B |
| SLO 三个列表 | ~5KB | ~1KB |
| **新增合计** | **~25KB** | **~5KB** |

当前 K8s 快照约 200-500KB（压缩后 30-80KB），新增 5KB 可忽略不计。

---

## Agent 变更

### 1. 新增 OTel Dashboard 仓库接口

**文件：** `atlhyper_agent_v2/repository/interfaces.go`

```go
// OTelDashboardRepository ClickHouse Dashboard 数据采集
// 查询结果随快照推送到 Master，供 Dashboard 页面直读
type OTelDashboardRepository interface {
    // Metrics
    GetMetricsSummary(ctx context.Context) (*metrics.Summary, error)
    ListAllNodeMetrics(ctx context.Context) ([]metrics.NodeMetrics, error)

    // APM
    ListAPMServices(ctx context.Context) ([]apm.APMService, error)
    GetAPMTopology(ctx context.Context) (*apm.Topology, error)

    // SLO
    GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error)
    ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error)
    ListServiceSLO(ctx context.Context, since time.Duration) ([]slo.ServiceSLO, error)
    ListServiceEdges(ctx context.Context, since time.Duration) ([]slo.ServiceEdge, error)
}
```

> **实现复用**：直接组合现有的 `MetricsQueryRepository`、`TraceQueryRepository`、`SLOQueryRepository`，不写新查询。

### 2. 实现 Dashboard 仓库

**文件：** `atlhyper_agent_v2/repository/ch/dashboard.go`

```go
type dashboardRepository struct {
    metricsRepo MetricsQueryRepository
    traceRepo   TraceQueryRepository
    sloRepo     SLOQueryRepository
}

func NewDashboardRepository(
    m MetricsQueryRepository, t TraceQueryRepository, s SLOQueryRepository,
) OTelDashboardRepository {
    return &dashboardRepository{metricsRepo: m, traceRepo: t, sloRepo: s}
}

// 每个方法直接委托给对应的查询仓库
func (r *dashboardRepository) GetMetricsSummary(ctx context.Context) (*metrics.Summary, error) {
    return r.metricsRepo.GetMetricsSummary(ctx)
}
// ... 其余方法同理
```

### 3. 扩展快照采集

**文件：** `atlhyper_agent_v2/service/snapshot/snapshot.go`

在 `snapshotService` 中注入 `OTelDashboardRepository`，重构 `getOTelSummary()` 为 `getOTelSnapshot()`：

```go
func (s *snapshotService) getOTelSnapshot(ctx context.Context) *cluster.OTelSnapshot {
    // TTL 缓存检查（复用现有逻辑）
    if time.Since(s.otelCacheTime) < s.otelCacheTTL && s.otelCache != nil {
        return s.otelCache
    }

    snapshot := &cluster.OTelSnapshot{}

    // 并发采集所有 Dashboard 数据（8 个查询并行）
    var wg sync.WaitGroup
    var mu sync.Mutex
    hasError := false

    collect := func(name string, fn func()) {
        wg.Add(1)
        go func() {
            defer wg.Done()
            fn()
        }()
    }

    // Metrics
    collect("MetricsSummary", func() {
        if v, err := s.dashboardRepo.GetMetricsSummary(ctx); err != nil {
            log.Warn("OTel MetricsSummary 采集失败", "err", err)
            mu.Lock(); hasError = true; mu.Unlock()
        } else {
            mu.Lock(); snapshot.MetricsSummary = v; mu.Unlock()
            // 同时填充原有摘要字段
            snapshot.MonitoredNodes = v.TotalNodes
            snapshot.AvgCPUPct = v.AvgCpuPct
            snapshot.AvgMemPct = v.AvgMemPct
            snapshot.MaxCPUPct = v.MaxCpuPct
            snapshot.MaxMemPct = v.MaxMemPct
        }
    })
    collect("MetricsNodes", func() { /* ListAllNodeMetrics → snapshot.MetricsNodes */ })
    collect("APMServices", func() { /* ListAPMServices → snapshot.APMServices + 填充摘要 */ })
    collect("APMTopology", func() { /* GetAPMTopology → snapshot.APMTopology */ })
    collect("SLOSummary", func() { /* GetSLOSummary → snapshot.SLOSummary */ })
    collect("SLOIngress", func() { /* ListIngressSLO → snapshot.SLOIngress + 填充摘要 */ })
    collect("SLOServices", func() { /* ListServiceSLO → snapshot.SLOServices */ })
    collect("SLOEdges", func() { /* ListServiceEdges → snapshot.SLOEdges */ })

    wg.Wait()

    // 全部失败时保留旧缓存
    if hasError && s.otelCache != nil {
        // 部分成功：合并新数据到旧缓存
        // 全部失败：返回旧缓存
    }

    s.otelCache = snapshot
    s.otelCacheTime = time.Now()
    return snapshot
}
```

> **关键设计**：8 个查询全部并行执行。ClickHouse 可以很好地处理并发查询，
> 总耗时取决于最慢的单个查询（通常是 MetricsNodes ~500ms），而非总和。

### 4. 依赖注入

**文件：** `atlhyper_agent_v2/agent.go`

```go
// 现有的 query repos（Command 按需查询仍然使用）
metricsQueryRepo := chquery.NewMetricsQueryRepository(chClient, k8sClient)
traceQueryRepo := chquery.NewTraceQueryRepository(chClient)
sloQueryRepo := chquery.NewSLOQueryRepository(chClient)

// 新增：Dashboard 仓库（组合现有 repos，不创建新查询）
dashboardRepo := ch.NewDashboardRepository(metricsQueryRepo, traceQueryRepo, sloQueryRepo)

// 注入到 SnapshotService
snapshotSvc := snapshot.NewSnapshotService(
    // ... 现有参数 ...
    snapshot.WithDashboard(dashboardRepo),  // 新增
)
```

---

## Master 变更

### 1. 扩展 DataHub 存储

当前 `SetSnapshot` / `GetSnapshot` 已经存储完整快照（含 OTel 字段）。
**无需修改 DataHub 接口** — OTel 数据自然随快照存入内存。

### 2. 扩展 Query Service

**文件：** `atlhyper_master_v2/service/query/impl.go`

新增读取 OTel Dashboard 数据的方法：

```go
// GetOTelSnapshot 获取集群的 OTel Dashboard 数据
func (q *QueryService) GetOTelSnapshot(clusterID string) (*cluster.OTelSnapshot, error) {
    snapshot, err := q.store.GetSnapshot(clusterID)
    if err != nil || snapshot == nil {
        return nil, err
    }
    return snapshot.OTel, nil  // 直接返回快照中的 OTel 数据
}
```

**文件：** `atlhyper_master_v2/service/interfaces.go`

```go
type Query interface {
    // ... 现有方法 ...
    GetOTelSnapshot(clusterID string) (*cluster.OTelSnapshot, error)  // 新增
}
```

### 3. 改造 Observe Handler

**文件：** `atlhyper_master_v2/gateway/handler/observe.go`

Dashboard 端点从 datahub 读取，Detail 端点保持 Command 机制：

```go
type ObserveHandler struct {
    svc   service.Service  // 改为 Service（需要 Query + Ops）
    bus   mq.Producer
    cache *observeCache
}

// ===== Dashboard 端点：从快照读取 =====

func (h *ObserveHandler) MetricsSummary(w http.ResponseWriter, r *http.Request) {
    clusterID, ok := requireClusterID(r)
    if !ok { writeError(w, http.StatusBadRequest, "cluster_id is required"); return }

    otel, err := h.svc.GetOTelSnapshot(clusterID)
    if err != nil || otel == nil || otel.MetricsSummary == nil {
        writeError(w, http.StatusNotFound, "数据尚未就绪")
        return
    }

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "message": "获取成功",
        "data":    otel.MetricsSummary,
    })
}

func (h *ObserveHandler) MetricsNodes(w http.ResponseWriter, r *http.Request) {
    // 同理：读取 otel.MetricsNodes
}

func (h *ObserveHandler) TracesServices(w http.ResponseWriter, r *http.Request) {
    // 同理：读取 otel.APMServices
}

// ... 其余 Dashboard 端点类似

// ===== Detail 端点：保持 Command 机制（不变）=====

func (h *ObserveHandler) TracesDetail(w http.ResponseWriter, r *http.Request) {
    // 保持原有 executeQuery 逻辑
    h.executeQuery(w, r, clusterID, command.ActionQueryTraceDetail, params, 30*time.Second)
}

func (h *ObserveHandler) LogsQuery(w http.ResponseWriter, r *http.Request) {
    // 保持原有 executeQuery 逻辑
}
```

---

## Command 管道优化（Detail 端点加速）

### 1. 减小轮询间隔

**文件：** `atlhyper_agent_v2/config/defaults.go`

```go
"AGENT_COMMAND_POLL_INTERVAL": "100ms",  // 1s → 100ms
```

### 2. 排空队列

**文件：** `atlhyper_agent_v2/scheduler/scheduler.go`

当前每次 poll 拿到 1 条指令后就等 1s。改为：拿到指令后立即继续 poll，直到队列为空才等待：

```go
func (s *Scheduler) runCommandLoop(topic string) {
    for {
        select {
        case <-s.ctx.Done():
            return
        default:
            hadCommands := s.pollAndExecuteCommands(topic)
            if hadCommands {
                // 有指令时立即继续轮询（排空队列）
                continue
            }
            // 无指令时等待后再轮询
            select {
            case <-s.ctx.Done():
                return
            case <-time.After(s.config.CommandPollInterval):
            }
        }
    }
}

func (s *Scheduler) pollAndExecuteCommands(topic string) bool {
    // ... 原有逻辑 ...
    return len(commands) > 0
}
```

> **效果**：Detail 查询延迟从 ~2s 降至 ~200-500ms（Agent 空闲时长轮询立即返回 + ClickHouse 查询）。

---

## 数据流对比

### 变更前

```
所有 Observe 请求:
Web → Master → Command → MQ → Agent 轮询 → ClickHouse → Agent → Master → Web
                                  ↑ 1s 间隔
                          延迟: 2-3s（串行）
```

### 变更后

```
Dashboard 请求 (8 个端点):
Web → Master → DataHub 内存读取 → Web
                延迟: ~0ms

Detail 请求 (6 个端点):
Web → Master → Command → MQ → Agent 轮询 → ClickHouse → Agent → Master → Web
                                  ↑ 100ms 间隔 + 排空队列
                          延迟: ~200-500ms
```

---

## 实施计划

### Phase 1: Command 管道优化（立即生效）

修改 2 个文件，所有 Observe 端点立即受益。

| 文件 | 变更 |
|------|------|
| `atlhyper_agent_v2/config/defaults.go` | `CommandPollInterval` 1s → 100ms |
| `atlhyper_agent_v2/scheduler/scheduler.go` | 排空队列逻辑 |

### Phase 2: 数据模型 + Agent 采集

| 文件 | 变更 |
|------|------|
| `model_v3/cluster/snapshot.go` | `OTelSummary` → `OTelSnapshot`，新增 8 个列表字段 |
| `atlhyper_agent_v2/repository/interfaces.go` | 新增 `OTelDashboardRepository` 接口 |
| `atlhyper_agent_v2/repository/ch/dashboard.go` | 实现（委托现有 repos） |
| `atlhyper_agent_v2/service/snapshot/snapshot.go` | `getOTelSummary` → `getOTelSnapshot`，并发采集 8 类数据 |
| `atlhyper_agent_v2/agent.go` | 依赖注入 DashboardRepository |

### Phase 3: Master 读取 + Handler 改造

| 文件 | 变更 |
|------|------|
| `atlhyper_master_v2/service/interfaces.go` | 新增 `GetOTelSnapshot` 方法签名 |
| `atlhyper_master_v2/service/query/impl.go` | 实现 `GetOTelSnapshot` |
| `atlhyper_master_v2/gateway/handler/observe.go` | 8 个 Dashboard 端点改为从快照读取 |

### Phase 4: 清理

| 文件 | 变更 |
|------|------|
| `atlhyper_master_v2/gateway/handler/observe.go` | 移除 Dashboard 端点的 `executeQuery` 调用和相关缓存 |
| 测试 | 验证所有 14 个 Observe 端点 |

---

## 验证方法

### 功能验证

```bash
# Phase 1 后：Detail 端点延迟降低
time curl -s 'http://localhost:8080/api/v2/observe/traces/xxx?cluster_id=ZGFX-X10A'
# 期望: < 1s

# Phase 3 后：Dashboard 端点秒开
time curl -s 'http://localhost:8080/api/v2/observe/metrics/summary?cluster_id=ZGFX-X10A'
# 期望: < 50ms

time curl -s 'http://localhost:8080/api/v2/observe/metrics/nodes?cluster_id=ZGFX-X10A'
# 期望: < 50ms
```

### 数据一致性验证

Dashboard 端点返回的数据结构必须与变更前完全一致（前端无需任何修改）。

### 快照大小验证

```bash
# 监控快照传输大小变化
# 变更前/后对比，预期增长 < 10KB（压缩后）
```

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| ClickHouse 不可用时 Dashboard 数据为空 | 复用现有容错：查询失败保留旧缓存 |
| 快照采集超时（8 个并发 CH 查询） | 每个查询独立超时 + 部分失败不影响其他 |
| 快照体积膨胀 | 预估新增 ~5KB（压缩后），可忽略 |
| 模型版本兼容 | `OTelSnapshot` 使用 `omitempty`，旧 Agent 不发送新字段不影响 Master |
