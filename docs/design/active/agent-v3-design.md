# Agent V3 设计文档

## 背景

当前 Agent V2 通过 OTel Collector Prometheus 端点 (`/metrics`) 主动拉取 node_exporter 和 Linkerd/Traefik 指标，随快照推送给 Master。这种方式存在以下问题：

1. Agent 承担了大量指标解析和聚合逻辑（OTel Prometheus 文本解析、counter→rate 计算、Linkerd 管线）
2. 无法查询 APM Traces 和日志（V2 未对接）
3. OTel Collector 已将所有数据写入 ClickHouse，Agent 直接查 ClickHouse 更高效

**新架构核心思路**：Agent 精简为 **K8s 资源快照 + ClickHouse 按需查询**。

**实施策略**：在 `atlhyper_agent_v2/` 原地改造，不新建项目。

### 架构演进对比

| 方面 | Agent V2 | Agent V3 (原地改造) |
|------|---------|---------|
| K8s 资源 | API Server → 快照推送 | **不变** |
| 节点 CPU/Mem | Metrics Server + OTel node_exporter | **仅 Metrics Server**（K8s 级别） |
| 硬件指标 | OTel Prometheus scrape → 快照推送 | **ClickHouse 查询**（按需） |
| SLO | OTel Prometheus → 管线聚合 → 快照推送 | **ClickHouse 查询**（概览随快照缓存推送 + 按需详情） |
| APM | 无 | **ClickHouse 查询**（概览随快照缓存推送 + 按需详情） |
| 日志 | 无 | **ClickHouse 查询**（按需） |
| 数据模型 | model_v2 | **model_v3** |
| SDK | K8sClient + OTelClient + IngressClient + ReceiverClient | **K8sClient + ClickHouseClient** |
| 项目目录 | `atlhyper_agent_v2/` | **不变**（原地改造） |
| 入口 | `cmd/agent/` | **不变** |

### ClickHouse 数据源（5 张表）

| 表 | 行数量级 | 数据来源 | Agent 用途 |
|----|---------|---------|-----------|
| `otel_traces` | ~800 | OTel Java Agent (6 个 geass 服务) | APM 服务列表、拓扑、Trace 详情 |
| `otel_logs` | ~1,600 | OTel Java Agent | 日志搜索 |
| `otel_metrics_gauge` | ~490,000 | Linkerd + Node Exporter | 服务网格 SLO、节点硬件指标 |
| `otel_metrics_sum` | ~52,000 | Node Exporter + Traefik | CPU 时间、网络流量、Ingress 请求计数 |
| `otel_metrics_histogram` | ~5,200 | Traefik | 请求延迟分布 (P50/P99) |

### ClickHouse 连接方式

- **开发**: `kubectl port-forward svc/clickhouse 8123:8123 9000:9000 -n atlhyper`
- **生产**: 集群内 `http://clickhouse.atlhyper.svc:8123`

### Metrics Server 与 ClickHouse 指标的区别

| 来源 | 数据 | 用途 |
|------|------|------|
| **Metrics Server** (API Server) | Pod/Node 的 CPU/Memory 用量 | K8s 资源管理级别（快照随 K8s 资源推送） |
| **ClickHouse** (node_exporter) | 节点硬件详情：磁盘 IO、网络流量、温度、PSI、TCP 等 | 运维监控级别（按需查询） |

两者不可混淆，不可替代。

---

## 一、分层架构

```
Scheduler (调度层)
  ├─ SnapshotLoop (10s)     → SnapshotService → Gateway.PushSnapshot
  ├─ CommandLoop (ops)      → CommandService  → Gateway.ReportResult
  ├─ CommandLoop (ai)       → CommandService  → Gateway.ReportResult
  └─ HeartbeatLoop (30s)    → Gateway.Heartbeat
       │
       ↓
Service 层 (业务逻辑) — 接口不变，内部按职责拆分子模块
  ├─ SnapshotService (snapshot/)
  │    ├─ cluster.go          K8s 资源并发采集 (20 种)
  │    ├─ event.go            Event 采集
  │    ├─ metrics_server.go   Metrics Server (Node/Pod CPU/Mem)
  │    └─ otel_summary.go     ClickHouse OTel 概览缓存 (5min 刷新)
  └─ CommandService (command/)
       ├─ k8s_ops.go          固定指令 (Scale/Restart/Delete/Cordon...)
       ├─ k8s_dynamic.go      AI 动态指令 (GET 类)
       └─ ch_query.go         ClickHouse 查询指令 (Trace/Log/Metrics/SLO)
       │
       ↓
Repository 层 (数据访问) — 按功能模块平铺
  ├─ K8s Repos (k8s/):          20 个资源仓库 + Event + Metrics + Generic
  ├─ CH Summary (ch/summary.go): OTel 概览聚合查询                  [新增]
  └─ CH Query (ch/query/):       按需详情查询                       [新增]
       │                           ├─ trace.go    (otel_traces)
       │                           ├─ log.go      (otel_logs)
       │                           ├─ metrics.go  (otel_metrics_gauge/sum)
       │                           └─ slo.go      (Traefik + Linkerd)
       ↓
SDK 层 (外部系统集成)
  ├─ K8sClient:          API Server + Metrics Server (不变)
  └─ ClickHouseClient:   ClickHouse HTTP (port 8123)               [新增]
       │
       ↓
外部系统
  ├─ K8s API Server
  ├─ K8s Metrics Server
  ├─ ClickHouse (atlhyper 数据库, 5 张 OTel 表)
  └─ Master (Gateway)
```

### 层级职责

| 层级 | 职责 | 可调用 |
|------|------|--------|
| **Scheduler** | 定时任务编排、循环控制 | Service, Gateway |
| **Service** | 业务逻辑编排 | Repository |
| **Repository** | 数据访问、格式转换 | SDK |
| **SDK** | 外部系统连接 | K8s API, ClickHouse |
| **Gateway** | Agent ↔ Master 通信 | HTTP (Master) |

### 依赖规则

- 调用方向严格单向：Scheduler → Service → Repository → SDK
- 即使 Service 只是转发也不可跳层
- Gateway 独立于 Service 层，由 Scheduler 直接调用

---

## 二、目录结构

> **原地改造 V2**：在 `atlhyper_agent_v2/` 基础上修改，不新建项目。

```
atlhyper_agent_v2/                            # 原地改造，不新建目录
├── agent.go                                  # 启动入口 & 依赖注入（修改）
├── config/
│   ├── types.go                              # 配置结构体（新增 ClickHouse 配置）
│   ├── defaults.go                           # 默认值（修改）
│   └── loader.go                             # 加载器
├── sdk/
│   ├── interfaces.go                         # K8sClient, ClickHouseClient（新增 CH 接口）
│   ├── types.go                              # 共用类型
│   └── impl/
│       ├── k8s/                              # K8s API Server + Metrics Server（不变）
│       │   ├── client.go
│       │   ├── core.go
│       │   ├── apps.go
│       │   ├── batch.go
│       │   ├── networking.go
│       │   ├── metrics.go
│       │   ├── storage.go
│       │   ├── policy.go
│       │   └── generic.go
│       └── clickhouse/                       # ClickHouse HTTP 客户端 [新增]
│           └── client.go
├── repository/
│   ├── interfaces.go                         # 所有仓库接口定义（修改）
│   ├── k8s/                                  # K8s 资源仓库 → model_v3/cluster（converter 修改）
│   │   ├── converter.go                      # 转换目标从 model_v2 → model_v3
│   │   ├── pod.go
│   │   ├── node.go
│   │   ├── deployment.go
│   │   ├── workload.go
│   │   ├── service.go
│   │   ├── namespace.go
│   │   ├── event.go
│   │   ├── job.go
│   │   ├── storage.go
│   │   ├── policy.go
│   │   ├── metrics.go                        # Metrics Server 数据获取
│   │   └── generic.go
│   └── ch/                                   # ClickHouse 数据获取 [新增]
│       ├── summary.go                        # OTel 概览聚合查询 (OTelSummaryRepository)
│       └── query/                            # 按需详情查询 (独立文件夹)
│           ├── trace.go                      # Trace 查询
│           ├── log.go                        # 日志查询
│           ├── metrics.go                    # 节点指标查询
│           └── slo.go                        # SLO 详情查询
├── service/
│   ├── interfaces.go                         # SnapshotService + CommandService（不变）
│   ├── snapshot/                             # 快照推送 — 4 个子模块
│   │   ├── snapshot.go                       # 组合入口: Collect() 调用子模块
│   │   ├── cluster.go                        # 子模块 1: K8s 资源并发采集 (20 种)
│   │   ├── event.go                          # 子模块 2: Event 采集
│   │   ├── metrics_server.go                 # 子模块 3: Metrics Server (Node/Pod CPU/Mem)
│   │   └── otel_summary.go                   # 子模块 4: OTel 概览缓存 (5min 刷新)
│   └── command/                              # 指令下发 — 3 个子模块
│       ├── command.go                        # 路由入口: Execute() 按 Action 分发
│       ├── k8s_ops.go                        # 子模块 1: 固定指令 (Scale/Restart/Delete...)
│       ├── k8s_dynamic.go                    # 子模块 2: AI 动态指令 (GET 类)
│       └── ch_query.go                       # 子模块 3: ClickHouse 查询指令
├── scheduler/
│   └── scheduler.go                          # 4 条循环（修改）
├── gateway/
│   ├── interfaces.go
│   └── master_gateway.go                     # PushSnapshot（不变）
└── model/
    └── options.go
```

### V2 → V3 改造清单

#### 删除的组件

| V2 组件 | 原因 |
|---------|------|
| `sdk/impl/otel/` | 不再从 OTel Collector Prometheus 拉取 |
| `sdk/impl/ingress/` | 不再采集 IngressRoute CRD |
| `sdk/impl/receiver/` | 不再被动接收 metrics_v2 推送 |
| `repository/metrics/` | 被 `repository/ch/metrics.go` 替代 |
| `repository/slo/` | 被 `repository/ch/slo.go` 替代 |
| MetricsSyncLoop | 不再需要独立的指标同步循环 |

#### 新增的组件

| 组件 | 用途 |
|------|------|
| `sdk/impl/clickhouse/client.go` | ClickHouse HTTP 客户端 |
| `repository/ch/summary.go` | OTel 概览聚合仓库 (OTelSummaryRepository) |
| `repository/ch/query/*.go` | 4 个 ClickHouse 按需查询仓库 |

#### 修改的组件

| 组件 | 变更内容 |
|------|---------|
| `repository/k8s/converter.go` | 转换目标从 model_v2 → model_v3 |
| `service/snapshot/` | 拆分为 4 个子模块（snapshot.go + cluster.go + event.go + metrics_server.go + otel_summary.go） |
| `service/command/` | 拆分为 3 个子模块（command.go + k8s_ops.go + k8s_dynamic.go + ch_query.go） |
| `scheduler/scheduler.go` | 从 V2 的循环简化为 4 条 |
| `agent.go` | 依赖注入更新（去 OTel，加 ClickHouse） |

---

## 三、SDK 层

### 3.1 K8sClient（从 V2 保留）

接口复用 V2，仍提供 K8s API Server 和 Metrics Server 访问。

**变更点**：
- 返回的原始 K8s 对象不变（`corev1.Pod`, `appsv1.Deployment` 等）
- 转换逻辑移至 Repository 的 converter.go，目标类型从 `model_v2` → `model_v3/cluster`

### 3.2 ClickHouseClient（新增）

```go
// sdk/interfaces.go
type ClickHouseClient interface {
    // 执行查询，返回结果行（通过回调扫描）
    Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)

    // 执行查询，扫描单行
    QueryRow(ctx context.Context, query string, args ...any) *sql.Row

    // 健康检查
    Ping(ctx context.Context) error

    // 关闭连接
    Close() error
}
```

**实现**：使用 `database/sql` + `github.com/ClickHouse/clickhouse-go/v2`（HTTP 协议）

```go
// sdk/impl/clickhouse/client.go
type client struct {
    db *sql.DB
}

func NewClient(endpoint, database string, timeout time.Duration) (sdk.ClickHouseClient, error) {
    dsn := fmt.Sprintf("clickhouse://%s/%s?dial_timeout=%s&read_timeout=%s",
        endpoint, database, timeout, timeout)
    db, err := sql.Open("clickhouse", dsn)
    // ...
}
```

**配置**：
```go
type ClickHouseConfig struct {
    Endpoint string        // 开发: "localhost:8123", 生产: "clickhouse.atlhyper.svc:8123"
    Database string        // "atlhyper"
    Timeout  time.Duration // 30s
}
```

---

## 四、Repository 层

### 4.1 K8s 仓库（从 V2 迁移，目标 model_v3）

**关键变更**：converter.go 转换目标从 `model_v2` → `model_v3/cluster`

model_v3 的结构更丰富：
- Pod: 完整 Spec/Status/Containers 详情（V2 只有摘要字段）
- Node: 完整 Conditions/Taints/Addresses（V2 扁平化）
- Deployment: 完整 Spec/Template/Status/Rollout（V2 只有列表字段）
- 部分资源使用 `model_v3.CommonMeta` 组合继承

**K8s 仓库的调用关系**：

| 目录/文件 | 职责 | 调用方 |
|-----------|------|--------|
| `k8s/` (12 文件) | 集群快照数据获取（20 种资源） | SnapshotService - cluster.go |
| `k8s/event.go` | Event 数据获取 | SnapshotService - event.go |
| `k8s/metrics.go` | Metrics Server 数据获取 | SnapshotService - metrics_server.go |
| `k8s/generic.go` | 通用写操作 & 动态查询 | CommandService - k8s_ops.go / k8s_dynamic.go |

k8s/ 目录结构不变，只是 Service 层按逻辑分组调用。

### 4.2 ClickHouse 仓库（新增）

Repository 层的 ClickHouse 仓库区分两类职责：**概览聚合**（随快照缓存推送）和**按需详情查询**（指令触发）。

#### 4.2.1 OTel 概览聚合仓库 (`ch/summary.go`)

单文件实现，因为聚合查询数量有限（~7 个方法），集中在一个文件便于维护。

```go
// repository/interfaces.go

// OTelSummaryRepository ClickHouse 概览聚合查询（快照 OTel 缓存用）
// 实现: ch/summary.go
type OTelSummaryRepository interface {
    ListAPMServices(ctx context.Context, tr model_v3.TimeRange) ([]apm.APMService, error)
    GetTopology(ctx context.Context, tr model_v3.TimeRange) (*apm.Topology, error)
    GetSLOSummary(ctx context.Context, tr model_v3.TimeRange) (*slo.SLOSummary, error)
    GetIngressSLOs(ctx context.Context, tr model_v3.TimeRange) ([]slo.IngressSLO, error)
    GetServiceSLOs(ctx context.Context, tr model_v3.TimeRange) ([]slo.ServiceSLO, error)
    GetServiceEdges(ctx context.Context, tr model_v3.TimeRange) ([]slo.ServiceEdge, error)
    GetNodeMetricsSummary(ctx context.Context) (*metrics.Summary, error)
}
```

#### 4.2.2 ClickHouse 按需查询仓库 (`ch/query/`)

独立文件夹，因为查询内容多、SQL 复杂，每个领域一个文件。

```go
// repository/interfaces.go

// TraceQueryRepository Trace 按需查询（CommandService 调用）
// 实现: ch/query/trace.go
type TraceQueryRepository interface {
    ListTraces(ctx context.Context, opts TraceQueryOpts) ([]apm.TraceSummary, int64, error)
    GetTraceDetail(ctx context.Context, traceId string) (*apm.TraceDetail, error)
}

// LogQueryRepository 日志按需查询（CommandService 调用）
// 实现: ch/query/log.go
type LogQueryRepository interface {
    Search(ctx context.Context, opts LogQueryOpts) (*log.QueryResult, error)
}

// MetricsQueryRepository 节点指标按需查询（CommandService 调用）
// 实现: ch/query/metrics.go
type MetricsQueryRepository interface {
    GetNodeHistory(ctx context.Context, nodeName string, metric string, tr model_v3.TimeRange) ([]metrics.Series, error)
}

// SLOQueryRepository SLO 详情按需查询（CommandService 调用）
// 实现: ch/query/slo.go
type SLOQueryRepository interface {
    GetTimeSeries(ctx context.Context, ns, name string, tr model_v3.TimeRange) (*slo.TimeSeries, error)
}
```

### 4.3 调用关系总览

```
SnapshotService.Collect()
  ├─ cluster.go        → PodRepo, NodeRepo, DeployRepo... (20 个 K8s 资源仓库)
  ├─ event.go          → EventRepo
  ├─ metrics_server.go → (K8sClient 的 ListNodeMetrics/ListPodMetrics)
  └─ otel_summary.go   → OTelSummaryRepo (ch/summary.go)

CommandService.Execute()
  ├─ k8s_ops.go        → GenericRepo (Scale/Restart/Delete...)
  ├─ k8s_dynamic.go    → GenericRepo (Dynamic GET)
  └─ ch_query.go       → TraceQueryRepo, LogQueryRepo, MetricsQueryRepo, SLOQueryRepo
                          (ch/query/*.go)
```

### 4.4 ClickHouse 查询 → 表映射

| 仓库 | 实现文件 | ClickHouse 表 | 关键查询维度 |
|------|---------|-------------|------------|
| OTelSummaryRepo | `ch/summary.go` | `otel_traces` + `otel_metrics_*` | 概览聚合（APM 服务列表、拓扑、SLO 摘要、节点指标摘要） |
| TraceQueryRepo | `ch/query/trace.go` | `otel_traces` | ServiceName, SpanKind, Duration, SpanAttributes |
| LogQueryRepo | `ch/query/log.go` | `otel_logs` | ServiceName, SeverityText, Body (LIKE), TraceId |
| MetricsQueryRepo | `ch/query/metrics.go` | `otel_metrics_gauge` + `otel_metrics_sum` | MetricName=`node_*`, ResourceAttributes[`net.host.name`] |
| SLOQueryRepo | `ch/query/slo.go` | `otel_metrics_sum` + `otel_metrics_histogram` + `otel_metrics_gauge` | Traefik: `traefik_*`; Linkerd: `response_*`/`request_*` |

### 4.5 关键 SQL 查询示例

**APM 服务列表**：
```sql
SELECT ServiceName,
       count() AS span_count,
       countIf(StatusCode = 'STATUS_CODE_ERROR') AS error_count,
       round(1 - error_count / span_count, 4) AS success_rate,
       round(avg(Duration) / 1e6, 2) AS avg_ms,
       round(quantile(0.50)(Duration) / 1e6, 2) AS p50_ms,
       round(quantile(0.99)(Duration) / 1e6, 2) AS p99_ms
FROM otel_traces
WHERE SpanKind = 'SPAN_KIND_SERVER'
  AND Timestamp >= now() - INTERVAL {timeRange}
GROUP BY ServiceName
```

**服务拓扑发现**：
```sql
SELECT t1.ServiceName AS source, t2.ServiceName AS target, count() AS call_count,
       round(avg(t2.Duration) / 1e6, 2) AS avg_ms
FROM otel_traces t1
JOIN otel_traces t2 ON t1.SpanId = t2.ParentSpanId AND t1.TraceId = t2.TraceId
WHERE t1.ServiceName != t2.ServiceName
GROUP BY source, target
```

**节点内存使用率**：
```sql
SELECT ResourceAttributes['net.host.name'] AS node,
       round((1 - argMax(a.Value, a.TimeUnix) / argMax(t.Value, t.TimeUnix)) * 100, 2) AS used_pct
FROM otel_metrics_gauge t
JOIN otel_metrics_gauge a USING (ResourceAttributes, TimeUnix)
WHERE t.MetricName = 'node_memory_MemTotal_bytes'
  AND a.MetricName = 'node_memory_MemAvailable_bytes'
  AND t.TimeUnix >= now() - INTERVAL 5 MINUTE
GROUP BY node
```

**Traefik Ingress SLO**：
```sql
SELECT Attributes['service'] AS service_key,
       sum(Value) AS total_requests
FROM otel_metrics_sum
WHERE MetricName = 'traefik_service_requests_total'
  AND TimeUnix >= now() - INTERVAL {timeRange}
GROUP BY service_key
```

---

## 五、Service 层

### 5.0 接口暴露

`service/interfaces.go` 仍只暴露 2 个顶层接口，子模块是**内部实现拆分**，不暴露为独立接口：

```go
// service/interfaces.go
type SnapshotService interface {
    Collect(ctx context.Context) (*cluster.ClusterSnapshot, error)
}
type CommandService interface {
    Execute(ctx context.Context, cmd *command.Command) *model.Result
}
```

### 5.1 SnapshotService（4 个子模块）

| 文件 | 职责 |
|------|------|
| `snapshot/snapshot.go` | **组合入口**: `Collect()` 调用 4 个子模块，合并为 ClusterSnapshot |
| `snapshot/cluster.go` | 子模块 1: K8s 资源并发采集（20 种资源） |
| `snapshot/event.go` | 子模块 2: Event 采集 |
| `snapshot/metrics_server.go` | 子模块 3: Metrics Server 数据采集（Node/Pod CPU/Mem） |
| `snapshot/otel_summary.go` | 子模块 4: ClickHouse OTel 概览缓存（5min 刷新） |

`snapshot.go` 只做编排和合并，具体采集逻辑分散到各子模块文件。

#### 结构体与依赖

```go
// snapshot/snapshot.go — 组合入口
type snapshotService struct {
    // K8s repos (不变)
    podRepo, nodeRepo, deployRepo... repository.XxxRepository

    // CH 概览仓库 (新增)
    otelSummaryRepo repository.OTelSummaryRepository

    // OTel 概览缓存
    mu            sync.RWMutex
    otelCache     *cluster.OTelSummary
    otelCacheTime time.Time
    otelCacheTTL  time.Duration
}

func (s *snapshotService) Collect(ctx context.Context) (*cluster.ClusterSnapshot, error) {
    snap := &cluster.ClusterSnapshot{}

    // 1. K8s 资源并发采集 (cluster.go)
    s.collectClusterResources(ctx, snap)

    // 2. Event 采集 (event.go)
    s.collectEvents(ctx, snap)

    // 3. Metrics Server 采集 (metrics_server.go)
    s.collectMetricsServer(ctx, snap)

    // 4. OTel 概览缓存 (otel_summary.go)
    snap.OTel = s.getOTelSummary(ctx)

    // 5. 摘要计算
    snap.Summary = s.buildSummary(snap)

    return snap, nil
}
```

#### OTel 概览缓存机制 (`otel_summary.go`)

```
1. 检查 OTel 缓存：
   a. 若 time.Since(otelCacheTime) > 5min → 并发查询 ClickHouse 刷新缓存
   b. 若 ClickHouse 不可用 → 使用旧缓存（或 nil），不阻塞快照推送
   c. 若缓存未过期 → 直接使用现有缓存
2. 返回缓存数据
```

**关键设计原则**：
- K8s 数据始终实时（每 10s 采集）
- OTel 概览数据最多延迟 5min（缓存周期）
- ClickHouse 不可用不影响 K8s 快照推送（降级为 nil）
- 缓存刷新的 ClickHouse 查询有独立超时（30s），不阻塞快照循环

### 5.2 CommandService（3 个子模块）

| 文件 | 职责 |
|------|------|
| `command/command.go` | **路由入口**: `Execute()` 根据 Action 路由到 3 个子模块 |
| `command/k8s_ops.go` | 子模块 1: 固定指令（Web UI: Scale/Restart/Delete/Cordon 等） |
| `command/k8s_dynamic.go` | 子模块 2: 动态指令（AI 专用，仅 GET 类操作） |
| `command/ch_query.go` | 子模块 3: ClickHouse 查询指令（Trace/Log/Metrics/SLO 查询） |

#### 路由入口

```go
// command/command.go
type commandService struct {
    genericRepo      repository.GenericRepository
    traceQueryRepo   repository.TraceQueryRepository
    logQueryRepo     repository.LogQueryRepository
    metricsQueryRepo repository.MetricsQueryRepository
    sloQueryRepo     repository.SLOQueryRepository
}

func (s *commandService) Execute(ctx context.Context, cmd *command.Command) *model.Result {
    switch cmd.Action {
    // K8s 固定指令 → k8s_ops.go
    case ActionScale, ActionRestart, ActionDelete, ...:
        return s.executeK8sOps(ctx, cmd)
    // AI 动态指令 → k8s_dynamic.go
    case ActionDynamic:
        return s.executeK8sDynamic(ctx, cmd)
    // ClickHouse 查询 → ch_query.go
    case ActionQueryTraces, ActionQueryTraceDetail, ActionQueryLogs, ActionQueryMetrics, ActionQuerySLO:
        return s.executeCHQuery(ctx, cmd)
    default:
        return model.ErrorResult(...)
    }
}
```

在现有 K8s 操作指令基础上，新增 ClickHouse 查询指令：

```go
// model_v3/command/command.go 新增 Action 常量
const (
    // 现有 K8s 操作（保留 V2 指令体系）
    ActionScale       = "scale"
    ActionRestart     = "restart"
    ActionDelete      = "delete"
    ActionDeletePod   = "delete_pod"
    ActionExec        = "exec"
    ActionCordon      = "cordon"
    ActionUncordon    = "uncordon"
    ActionDrain       = "drain"
    ActionUpdateImage = "update_image"
    ActionGetLogs     = "get_logs"
    ActionGetConfigMap = "get_configmap"
    ActionGetSecret   = "get_secret"
    ActionDynamic     = "dynamic"

    // 新增 ClickHouse 查询 [V3]
    ActionQueryTraces      = "query_traces"
    ActionQueryTraceDetail = "query_trace_detail"
    ActionQueryLogs        = "query_logs"
    ActionQueryMetrics     = "query_metrics"
    ActionQuerySLO         = "query_slo"
)
```

**查询指令数据流**：

```
用户在前端点击 "查看 Trace 详情"
  → Master 创建 Command{Action: "query_trace_detail", Params: {"traceId": "25db41ab..."}}
  → Agent CommandLoop 拉取指令
  → CommandService.Execute()
    → CH TraceRepository.GetTraceDetail("25db41ab...")
    → 返回 Result{Output: JSON(TraceDetail)}
  → Gateway.ReportResult() → Master 收到 → 前端展示
```

---

## 六、数据模型变更

### 6.1 OTelSummary（新增到 `model_v3/cluster/snapshot.go`）

OTel 概览数据随快照推送，Master 缓存用于前端展示。

```go
// model_v3/cluster/snapshot.go

// OTelSummary Agent 从 ClickHouse 查询的 OTel 概览数据（5min 缓存刷新）
//
// 随 ClusterSnapshot 一起推送给 Master。K8s 数据实时（10s），OTel 概览最多延迟 5min。
type OTelSummary struct {
    FetchedAt    time.Time           `json:"fetchedAt"`
    SLO          *slo.SLOSummary     `json:"slo,omitempty"`
    IngressSLOs  []slo.IngressSLO    `json:"ingressSLOs,omitempty"`
    ServiceSLOs  []slo.ServiceSLO    `json:"serviceSLOs,omitempty"`
    ServiceEdges []slo.ServiceEdge   `json:"serviceEdges,omitempty"`
    APMServices  []apm.APMService    `json:"apmServices,omitempty"`
    Topology     *apm.Topology       `json:"topology,omitempty"`
    NodeMetrics  *metrics.Summary    `json:"nodeMetrics,omitempty"`
}
```

### 6.2 ClusterSnapshot 新增字段

```go
// model_v3/cluster/snapshot.go

// ClusterSnapshot Agent 采集的完整集群状态
//
// K8s 资源每 10s 实时采集推送。OTel 概览数据从 ClickHouse 查询，
// 5min 缓存刷新后随快照推送给 Master。
type ClusterSnapshot struct {
    // ... 现有 K8s 资源字段不变 ...

    // OTel 概览（从 ClickHouse 缓存，5min 刷新）
    OTel *OTelSummary `json:"otel,omitempty"`

    // 摘要
    Summary ClusterSummary `json:"summary"`
}
```

---

## 七、Scheduler

```go
func (s *Scheduler) Start(ctx context.Context) {
    go s.snapshotLoop(ctx)          // 10s: K8s 快照 + OTel 概览缓存推送
    go s.commandLoop(ctx, "ops")    // 长轮询: 系统操作指令
    go s.commandLoop(ctx, "ai")     // 长轮询: AI 查询指令
    go s.heartbeatLoop(ctx)         // 30s: 心跳
}
```

| 循环 | 间隔 | 超时 | 数据源 | 目标 |
|------|------|------|--------|------|
| snapshotLoop | 10s | 10s | K8s API + Metrics Server + ClickHouse (缓存) | `POST /agent/snapshot` |
| commandLoop (ops) | 长轮询 60s | 30s | Master | `POST /agent/result` |
| commandLoop (ai) | 长轮询 60s | 30s | Master | `POST /agent/result` |
| heartbeatLoop | 30s | 10s | — | `POST /agent/heartbeat` |

**与 V2 的区别**：
- 移除 MetricsSyncLoop（每 15s 从 OTel 拉取 node_exporter 指标）
- snapshotLoop 内部新增 OTel 概览缓存刷新（5min 周期，惰性触发）

---

## 八、Gateway

### Agent ↔ Master 端点

| 方法 | 端点 | 方向 | 用途 |
|------|------|------|------|
| POST | `/agent/snapshot` | Agent → Master | K8s 快照 + OTel 概览推送 (Gzip) |
| GET | `/agent/commands` | Agent ← Master | 指令长轮询 |
| POST | `/agent/result` | Agent → Master | 指令结果上报 |
| POST | `/agent/heartbeat` | Agent → Master | 心跳 |

**通信原则**: Agent 主动连接 Master，Master 不能连接 Agent。

**与 V2 的区别**: 端点不变，`/agent/snapshot` 的 payload 中新增 `otel` 字段。

---

## 九、配置

```go
type AppConfig struct {
    Log        LogConfig
    Agent      AgentConfig        // ClusterID
    Master     MasterConfig       // URL
    Kubernetes KubernetesConfig   // KubeConfig (空=InCluster)
    ClickHouse ClickHouseConfig   // [新增]
    Scheduler  SchedulerConfig
    Timeout    TimeoutConfig
}

type ClickHouseConfig struct {
    Endpoint string        `env:"AGENT_CLICKHOUSE_ENDPOINT"` // "clickhouse.atlhyper.svc:8123"
    Database string        `env:"AGENT_CLICKHOUSE_DATABASE"` // "atlhyper"
    Timeout  time.Duration `env:"AGENT_CLICKHOUSE_TIMEOUT"`  // 30s
}

type SchedulerConfig struct {
    SnapshotInterval    time.Duration // 10s
    OTelCacheTTL        time.Duration // 5min (OTel 概览缓存过期时间)
    CommandPollInterval time.Duration // 1s
    HeartbeatInterval   time.Duration // 30s
}
```

**环境变量** (K8s ConfigMap 新增)：
```yaml
AGENT_CLICKHOUSE_ENDPOINT: "clickhouse.atlhyper.svc:8123"
AGENT_CLICKHOUSE_DATABASE: "atlhyper"
```

---

## 十、数据流

### 10.1 定期推送: K8s 快照 + OTel 概览 (每 10s)

```
API Server ──→ K8s Repos ──→ SnapshotService ──→ ClusterSnapshot (K8s 实时 + OTel 缓存)
                  ↑           ├─ cluster.go          ↓
Metrics Server ──┘           ├─ event.go       Gateway.PushSnapshot → Master
                              ├─ metrics_server.go
                              └─ otel_summary.go
                                    ↑
ClickHouse ──→ OTelSummaryRepo (ch/summary.go) ──→ OTel 缓存 (5min 刷新)
  ├─ otel_traces   → ListAPMServices/GetTopology
  ├─ otel_metrics  → GetSLOSummary/GetIngressSLOs
  └─ otel_metrics  → GetNodeMetricsSummary
```

**内容**: 20 种 K8s 资源 + Events + Metrics Server 的 Pod/Node CPU/Mem + OTel 概览缓存

**时序特点**:
- K8s 数据：每 10s 实时采集
- OTel 概览：5min 缓存，惰性刷新（在 snapshotLoop 中检查过期后刷新）
- ClickHouse 不可用时降级为 nil，不阻塞 K8s 快照

### 10.2 按需查询: 指令 (事件驱动)

```
Master ──→ Gateway.PollCommands ──→ CommandService.Execute()
  ├─ k8s_ops.go:     Scale/Restart/Delete/Cordon...  ──→ GenericRepo ──→ K8s API
  ├─ k8s_dynamic.go: AI Dynamic GET                  ──→ GenericRepo ──→ K8s API
  └─ ch_query.go:    QueryTraces/QueryLogs/...       ──→ CH QueryRepos (ch/query/*.go)
                                                              ↓                ↓
                                                         ClickHouse    Gateway.ReportResult → Master
```

**内容**: 用户点击查看详情 → Master 下发指令 → Agent 从 ClickHouse 查询后返回

---

## 十一、初始化顺序

```go
// agent.go
func NewAgent(cfg *config.AppConfig) (*Agent, error) {
    // 1. SDK
    k8sClient := k8s.NewClient(cfg.Kubernetes.KubeConfig)
    chClient  := clickhouse.NewClient(cfg.ClickHouse.Endpoint, cfg.ClickHouse.Database, cfg.ClickHouse.Timeout)

    // 2. Gateway
    gateway := gateway.NewMasterGateway(cfg.Master.URL, cfg.Agent.ClusterID, cfg.Timeout.HTTPClient)

    // 3. Repository — K8s repos (20 个)
    podRepo       := k8srepo.NewPodRepository(k8sClient)
    nodeRepo      := k8srepo.NewNodeRepository(k8sClient)
    deployRepo    := k8srepo.NewDeploymentRepository(k8sClient)
    // ... 其他 17 个 K8s repo
    genericRepo   := k8srepo.NewGenericRepository(k8sClient)

    // 3b. Repository — CH 概览仓库 (1 个)
    otelSummaryRepo := chrepo.NewOTelSummaryRepository(chClient)

    // 3c. Repository — CH 按需查询仓库 (4 个)
    traceQueryRepo   := chquery.NewTraceQueryRepository(chClient)
    logQueryRepo     := chquery.NewLogQueryRepository(chClient)
    metricsQueryRepo := chquery.NewMetricsQueryRepository(chClient)
    sloQueryRepo     := chquery.NewSLOQueryRepository(chClient)

    // 4. Service
    snapshotSvc := snapshot.NewSnapshotService(
        podRepo, nodeRepo, deployRepo, ...,    // K8s repos
        otelSummaryRepo,                       // CH 概览仓库
        cfg.Scheduler.OTelCacheTTL,            // 缓存过期时间
    )
    commandSvc := command.NewCommandService(
        genericRepo,                           // K8s 操作
        traceQueryRepo, logQueryRepo,          // CH 按需查询
        metricsQueryRepo, sloQueryRepo,
    )

    // 5. Scheduler
    sched := scheduler.NewScheduler(snapshotSvc, commandSvc, gateway, cfg.Scheduler)

    return &Agent{scheduler: sched, chClient: chClient}, nil
}
```

---

## 十二、开发计划（分阶段）

### Phase 1: 基础设施 (SDK + Config + model_v3 变更)

| 任务 | 文件 |
|------|------|
| 新增 OTelSummary 到 ClusterSnapshot | `model_v3/cluster/snapshot.go` |
| ClickHouseClient SDK | `sdk/interfaces.go`, `sdk/impl/clickhouse/client.go` |
| 配置体系更新 | `config/types.go`, `defaults.go` |
| 删除废弃 SDK | 删除 `sdk/impl/otel/`, `sdk/impl/ingress/`, `sdk/impl/receiver/` |
| 删除废弃 Repository | 删除 `repository/metrics/`, `repository/slo/` |

### Phase 2: K8s 快照迁移 (converter → model_v3 + Service 拆分)

| 任务 | 文件 |
|------|------|
| K8s Repository converter 迁移 | `repository/k8s/converter.go` → model_v3 |
| SnapshotService 拆分为 4 个子模块 | `service/snapshot/snapshot.go` + `cluster.go` + `event.go` + `metrics_server.go` (先不含 OTel) |
| CommandService 拆分为 3 个子模块 | `service/command/command.go` + `k8s_ops.go` + `k8s_dynamic.go` (先不含 CH) |
| Scheduler 简化 | `scheduler/scheduler.go` (4 条循环) |
| **验证** | Agent 启动 → 推送 K8s 快照 → Master 收到 |

### Phase 3: ClickHouse 集成 (CH Repository + OTel 缓存)

| 任务 | 文件 |
|------|------|
| OTelSummaryRepository | `repository/ch/summary.go` |
| SnapshotService OTel 缓存 | `service/snapshot/otel_summary.go` (加入缓存机制) |
| **验证** | `port-forward clickhouse` → Agent 查询成功 → 快照含 OTel 概览 |

### Phase 4: 指令扩展 (CommandService + CH 查询)

| 任务 | 文件 |
|------|------|
| TraceQueryRepository | `repository/ch/query/trace.go` |
| LogQueryRepository | `repository/ch/query/log.go` |
| MetricsQueryRepository | `repository/ch/query/metrics.go` |
| SLOQueryRepository | `repository/ch/query/slo.go` |
| CommandService CH 查询子模块 | `service/command/ch_query.go` |
| 新增 Action 常量 | `model_v3/command/command.go` |
| **验证** | Master 下发 `query_trace_detail` → Agent 返回结果 |

---

## 十三、验证方法

```bash
# 1. ClickHouse 连通性
kubectl port-forward svc/clickhouse 8123:8123 -n atlhyper
curl "http://localhost:8123/?query=SELECT+1"

# 2. Agent 构建
go build ./cmd/agent/...

# 3. Agent 启动 (本地)
go run cmd/agent/main.go --kubeconfig=$HOME/.kube/config

# 4. 快照验证: 检查 Master 日志
# 看到 "Received snapshot from zgmf-x10a"
# 快照 JSON 中包含 "otel" 字段

# 5. OTel 缓存验证: 检查 Agent 日志
# 首次启动后 ~10s: "OTel cache refreshed: 6 APM services, 2 ingress SLO, ..."
# 后续 5min 内: "Using cached OTel summary (age: 2m30s)"
# 5min 后: "OTel cache expired, refreshing..."

# 6. ClickHouse 降级验证: 停止 port-forward 后
# Agent 日志: "ClickHouse unavailable, using stale OTel cache"
# 快照仍正常推送，otel 字段为旧缓存或 nil

# 7. 指令验证: 通过 Master API 下发查询
curl -X POST http://localhost:8080/api/v3/commands \
  -d '{"action":"query_traces","clusterId":"zgmf-x10a","params":{"timeRange":"1h"}}'
```

---

## 依赖

| 包 | 用途 | 状态 |
|----|------|------|
| `github.com/ClickHouse/clickhouse-go/v2` | ClickHouse Go 驱动 (HTTP) | **新增** |
| `k8s.io/client-go` | K8s 客户端 | 已有 |
| `k8s.io/metrics` | Metrics Server | 已有 |
| `model_v3/` | 共享数据模型 | 已有 |

---

## 参考文档

| 文档 | 路径 |
|------|------|
| ClickHouse 数据参考 | `docs/design/active/clickhouse-otel-data-reference.md` |
| model_v3 数据模型 | `model_v3/` |
| Agent V2 (改造基础) | `atlhyper_agent_v2/` |
| K8s 部署配置 | `Config/zgmf-x10a/k8s-configs/atlhyper/` |
