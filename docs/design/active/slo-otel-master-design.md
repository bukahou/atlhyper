# SLO OTel Master 设计书

## 概要

Master 端接收 Agent 上报的 `SLOSnapshot`，存储三层数据（服务网格 + 拓扑 + 入口），通过 API 提供给前端展示。

**数据可用性**: Agent 上报后 **立即可查** — raw 数据实时写入，API 直接查询。Aggregator 定时将 raw 预聚合为 hourly 用于历史查询和性能优化，但 API 始终有 raw 回退，不依赖聚合完成。

**两层存储**:
- `raw` 表: 每次 Agent 快照写入一行，保留 48h，API 实时查询
- `hourly` 表: Aggregator 定时聚合（含当前不完整小时），保留 90d，API 优先使用

**前置依赖**: Agent 端设计见 `slo-otel-agent-design.md`

**共享合约**: `model_v2/slo.go`（SLOSnapshot + ServiceMetrics + ServiceEdge + IngressMetrics + IngressRouteInfo）

---

## 1. 文件夹结构

现有文件标 `(现有)`，新增/修改标 `← NEW` 或 `← 修改`。

```
atlhyper_master_v2/
├── master.go                          (现有)  ← 修改: 新 Repository/Handler 依赖注入
│
├── agentsdk/
│   ├── server.go                      (现有)  ← 修改: 移除 /agent/slo 路由
│   ├── snapshot.go                    (现有)  不动: POST /agent/snapshot 快照接收
│   ├── slo.go                         (现有)  ← 删除: 旧模式专用，不再需要
│   ├── heartbeat.go                   (现有)  不动
│   ├── command.go                     (现有)  不动
│   ├── result.go                      (现有)  不动
│   └── types.go                       (现有)  不动
│
├── processor/                         (现有)  不动
│   └── processor.go                   处理快照 → 触发 OnSnapshotReceived 回调
│
├── service/
│   ├── interfaces.go                  (现有)  ← 修改: Query 接口新增 SLO 查询方法
│   ├── factory.go                     (现有)  ← 修改: QueryService 注入新 Repository
│   ├── sync/
│   │   ├── slo_persist.go             (现有)  ← 重写: 只保留 OTel 路径
│   │   ├── event_persist.go           (现有)  不动
│   │   └── metrics_persist.go         (现有)  不动
│   ├── operations/                    (现有)  不动
│   └── query/
│       ├── ...                        (现有)  不动
│       └── slo.go                     ← NEW   SLO 查询实现 (服务网格 + 域名增强)
│
├── slo/                               领域处理器（被 service 层调用 + 独立后台任务）
│   ├── interfaces.go                  ← NEW   对外接口定义 (SLOProcessor / SLOAggregator / SLOCleaner)
│   ├── processor.go                   (现有)  ← 重写: 删除旧 delta 逻辑(~280行)，新增 ProcessSLOSnapshot
│   ├── aggregator.go                  (现有)  ← 重写: 新增 service/edge 聚合，改写 ingress 聚合(bucket 格式变)
│   ├── cleaner.go                     (现有)  ← 小改: +2 新表清理，-snapshot 表清理
│   ├── calculator.go                  (现有)  ← 修改: 删 CalculateDelta/RawBuckets，改 Quantile 入参类型
│   └── status_checker.go             (现有)  ← 小改: 扩展支持 service 维度
│
├── database/
│   ├── interfaces.go                  (现有)  ← 修改: 新增 SLOServiceRepository + SLOEdgeRepository
│   ├── sync.go                        (现有)  不动
│   ├── factory.go                     (现有)  不动
│   ├── repo/
│   │   ├── slo.go                     (现有)  ← 修改: 实现新 Repository 接口
│   │   └── ...                        (现有)  不动
│   └── sqlite/
│       ├── slo.go                     (现有)  ← 重写: 移除 snapshot 表相关 SQL，适配新入口字段
│       ├── slo_service.go             ← NEW   服务网格表 SQL (service_raw + service_hourly)
│       ├── slo_edge.go                ← NEW   拓扑边表 SQL (edge_raw + edge_hourly)
│       ├── migrations.go              (现有)  ← 修改: 新增 4 张表 + 2 个 ALTER
│       └── ...                        (现有)  不动
│
├── gateway/
│   ├── server.go                      (现有)  不动
│   ├── routes.go                      (现有)  ← 修改: 注册 /api/v2/slo/mesh/* 路由
│   ├── handler/
│   │   ├── slo.go                     (现有)  ← 修改: DomainsV2 增强，改为依赖 service.Query
│   │   ├── slo_mesh.go                ← NEW   服务网格 API，依赖 service.Query (不直接访问 Database)
│   │   └── ...                        (现有)  不动
│   └── middleware/                     (现有)  不动
│
├── model/
│   └── slo.go                         (现有)  ← 修改: 新增 API 响应类型
│
├── datahub/                           (现有)  不动
├── mq/                                (现有)  不动
├── config/                            (现有)  不动
├── notifier/                          (现有)  不动
├── ai/                                (现有)  不动
└── tester/                            (现有)  不动
```

**共享模型（Agent ↔ Master 合约）**:

```
model_v2/
├── slo.go        (现有)  ← Agent 端重写: SLOSnapshot + ServiceMetrics
│                          + ServiceEdge + IngressMetrics
└── snapshot.go   (现有)  不动: SLOData *SLOSnapshot 字段已存在
```

### 变更统计

| 操作 | 文件数 | 文件 |
|------|--------|------|
| **新建** | 5 | `slo/interfaces.go`, `service/query/slo.go`, `sqlite/slo_service.go`, `sqlite/slo_edge.go`, `handler/slo_mesh.go` |
| **重写** | 4 | `processor.go`(delta逻辑→直接存储), `aggregator.go`(+service/edge聚合+bucket JSON化), `slo_persist.go`, `sqlite/slo.go`(入口表bucket JSON化+DROP snapshot) |
| **修改** | 8 | `service/interfaces.go`(+SLO查询), `service/factory.go`(注入新repo), `calculator.go`(统一bucket类型), `model/slo.go`(新增响应类型), `master.go`, `database/interfaces.go`, `repo/slo.go`, `migrations.go`(DROP重建入口表) |
| **小改** | 5 | `cleaner.go`, `status_checker.go`, `agentsdk/server.go`, `handler/slo.go`(改依赖service.Query), `routes.go` |
| **删除** | 1 | `agentsdk/slo.go` |
| 不动 | ~28 | 其余所有文件 |

---

## 2. 调用链路

### 2.1 数据写入路径（Agent 上报 → 存储）

```
┌───────────────────────────────────────────────────────────────────┐
│  Agent                                                            │
│  scheduler → POST /agent/snapshot (ClusterSnapshot with SLOData)  │
└─────────────────────────────┬─────────────────────────────────────┘
                              │ HTTP
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│  agentsdk/snapshot.go                                             │
│  handleSnapshot()                                                 │
│      ├── 解析 ClusterSnapshot                                      │
│      ├── 存入 Store (datahub)                                      │
│      └── processor.ProcessSnapshot(clusterID, snapshot)            │
└─────────────────────────────┬─────────────────────────────────────┘
                              │ 触发回调 OnSnapshotReceived
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│  service/sync/slo_persist.go                                      │
│  SLOPersistService.Sync(clusterID)                                │
│      ├── store.GetSnapshot(clusterID)                              │
│      └── sloProcessor.ProcessSLOSnapshot(ctx, clusterID, sloData) │
└─────────────────────────────┬─────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│  slo/processor.go                                                 │
│  ProcessSLOSnapshot(ctx, clusterID, *SLOSnapshot)                 │
│      │                                                             │
│      ├── processServiceMetrics()  ── INSERT ──→ slo_service_raw    │
│      │   (遍历 Services[], 聚合状态码, 直接存储增量)                  │
│      │                                                             │
│      ├── processEdge()            ── INSERT ──→ slo_edge_raw       │
│      │   (遍历 Edges[], 直接存储增量)                                │
│      │                                                             │
│      ├── processIngressMetrics()  ── INSERT ──→ slo_metrics_raw    │
│      │   (遍历 Ingress[], 聚合 method 分布, 关联域名, 存储增量)       │
│      │                                                             │
│      └── processRoutes()          ── UPSERT ──→ slo_route_mapping  │
│          (遍历 Routes[], 更新域名 ↔ ServiceKey 映射)                 │
└───────────────────────────────────────────────────────────────────┘
```

### 2.2 定时预聚合路径（raw → hourly）

Aggregator 定时运行，聚合**上一个完整小时 + 当前不完整小时**，确保 hourly 数据始终是最新的。

```
┌───────────────────────────────────────────────────────────────────┐
│  slo/aggregator.go                                                │
│  Start() → 定时触发 (每 AggregateInterval)                         │
│      │                                                             │
│      └── aggregateAll()                                            │
│          │                                                         │
│          ├── aggregateServiceHour(hour)                  ← NEW     │
│          │   ├── SELECT FROM slo_service_raw WHERE ts ∈ [h, h+1h)  │
│          │   ├── 按 (namespace, name) 分组                          │
│          │   ├── SUM 请求/延迟/状态码/mTLS                           │
│          │   ├── calcPercentile(mergedBuckets, 0.50/0.95/0.99)     │
│          │   └── UPSERT INTO slo_service_hourly                    │
│          │                                                         │
│          ├── aggregateEdgeHour(hour)                     ← NEW     │
│          │   ├── SELECT FROM slo_edge_raw WHERE ts ∈ [h, h+1h)     │
│          │   ├── 按 (src_ns, src_name, dst_ns, dst_name) 分组       │
│          │   ├── SUM 请求/失败/延迟                                  │
│          │   └── UPSERT INTO slo_edge_hourly                       │
│          │                                                         │
│          └── aggregateIngressHour(hour)                  (现有扩展) │
│              ├── 原有逻辑不变                                       │
│              └── 新增 method 分布聚合                                │
└───────────────────────────────────────────────────────────────────┘
```

### 2.3 数据读取路径（API 查询）

**两层查询策略**: API 优先查 hourly（预计算好的 P50/P95/P99），无数据时回退到 raw 实时聚合。刚部署或 Aggregator 尚未运行时，raw 回退保证数据立即可见。

**分层约束**: Handler（Gateway 层）**禁止直接访问 Database**，必须通过 `service.Query` 接口。查询策略在 `service/query/slo.go` 中实现。

```
┌──── 查询策略（在 service/query/slo.go 中实现） ───────────────────┐
│                                                                    │
│  1. 查 hourly 表 (timeRange 对应的小时区间)                         │
│     ↓ 有数据 → 直接使用 (P50/P95/P99 已预计算)                     │
│     ↓ 无数据 → 回退                                                │
│  2. 查 raw 表 (同时间区间)                                          │
│     ↓ 有数据 → 实时聚合计算 (SUM + calcPercentile)                  │
│     ↓ 无数据 → 返回空                                              │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘

┌──── 服务网格 API ─────────────────────────────────────────────────┐
│                                                                    │
│  GET /api/v2/slo/mesh/topology                                     │
│      ↓                                                             │
│  gateway/handler/slo_mesh.go: MeshTopology()                       │
│      ↓ 调用 service.Query（不直接访问 Database）                    │
│  service/query/slo.go: GetMeshTopology()                           │
│      ├── repo.GetServiceHourly(clusterID, timeRange)               │
│      │   └── 回退: GetServiceRaw() → 实时聚合                      │
│      ├── repo.GetEdgeHourly(clusterID, timeRange)                  │
│      │   └── 回退: GetEdgeRaw() → 实时聚合                         │
│      ├── 计算衍生指标 (RPS, ErrorRate, Status)                      │
│      └── → ServiceMeshTopologyResponse { Nodes[], Edges[] }        │
│                                                                    │
│  GET /api/v2/slo/mesh/service/detail                               │
│      ↓                                                             │
│  gateway/handler/slo_mesh.go: ServiceDetail()                      │
│      ↓ 调用 service.Query                                          │
│  service/query/slo.go: GetServiceDetail()                          │
│      ├── repo.GetServiceHourly / GetServiceRaw (同策略)             │
│      ├── repo.GetEdgeHourly / GetEdgeRaw  ← 上下游                 │
│      └── → ServiceDetailResponse { 指标 + History[] + 上下游[] }    │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘

┌──── 域名 SLO API ─────────────────────────────────────────────────┐
│                                                                    │
│  GET /api/v2/slo/domains/v2                                        │
│      ↓                                                             │
│  gateway/handler/slo.go: DomainsV2()                               │
│      ↓ 调用 service.Query                                          │
│  service/query/slo.go: GetDomainsV2()                              │
│      ├── repo.GetHourlyMetrics / GetRawMetrics (同策略)             │
│      ├── 从 raw 查 latencyDistribution / requestBreakdown           │
│      ├── 从 route_mapping 匹配 backendServiceIDs                    │
│      └── → DomainSLOV2Response (+ 分布数据 + 关联服务)              │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### 2.4 定时清理路径

```
slo/cleaner.go: Start() → 每 CleanupInterval (1h)
    ├── repo.DeleteServiceRawBefore(now - 48h)      ← NEW
    ├── repo.DeleteEdgeRawBefore(now - 48h)          ← NEW
    ├── repo.DeleteRawMetricsBefore(now - 48h)       (现有)
    ├── repo.DeleteHourlyMetricsBefore(now - 90d)    (现有)
    └── repo.DeleteStatusHistoryBefore(now - 180d)   (现有)
```

### 2.5 初始化链路（master.go）

```go
New() {
    // ... 已有初始化 ...

    // SLO 组件初始化
    sloServiceRepo := repo.NewSLOServiceRepository(db)   // ← NEW
    sloEdgeRepo := repo.NewSLOEdgeRepository(db)          // ← NEW
    sloProcessor := slo.NewProcessor(sloRepo, sloServiceRepo, sloEdgeRepo)
    sloAggregator := slo.NewAggregator(sloRepo, sloServiceRepo, sloEdgeRepo, cfg.SLO)
    sloCleaner := slo.NewCleaner(sloRepo, sloServiceRepo, sloEdgeRepo, cfg.SLO)
    sloPersist := sync.NewSLOPersistService(store, sloProcessor)

    // Processor 回调（快照处理完 → 触发 SLO 同步）
    processor := processor.New(Config{
        OnSnapshotReceived: func(clusterID string) {
            sloPersist.Sync(clusterID)
        },
    })

    // Service 层（QueryService 注入新 Repository）
    queryService := query.NewQueryService(store, bus, db, sloServiceRepo, sloEdgeRepo, sloRepo)
    opsService := operations.NewCommandService(bus)
    svc := service.NewService(queryService, opsService)

    // API Handler（依赖 Service 接口，禁止直接持有 Database Repository）
    sloMeshHandler := handler.NewSLOMeshHandler(svc)      // ← NEW: 依赖 service.Query
}

Run() {
    // ... 已有启动 ...
    sloAggregator.Start()
    sloCleaner.Start()
}
```

---

## 3. 架构变化

### 3.1 核心架构

| 特性 | 说明 |
|---|---|
| Agent 上报内容 | **增量值** — Agent 已算好 per-pod delta + service 聚合 |
| Master 计算 | **无需 delta** — 直接存储 Agent 增量，聚合为小时级 |
| 数据维度 | **三层**: service (服务网格) + edge (拓扑) + ingress (入口) |
| 拓扑数据 | ServiceEdge (Linkerd outbound) |
| mTLS 数据 | TLS 覆盖率 (Linkerd inbound tls 标签) |
| 服务网格指标 | per-service 黄金指标 (RPS / 延迟 / 错误率 / 状态码) |
| 入口指标 | per-serviceKey (Controller 无关，Agent 归一化) |

### 3.2 slo/ 模块定位

`slo/` 是 **领域处理器**（domain processor），不是独立的架构层。它的组件按调用方式分为两类：

| 组件 | 调用方式 | 调用者 | 可访问 |
|------|----------|--------|--------|
| `processor.go` | 被 Service 层调用 | `service/sync/slo_persist.go` | Database repo（写） |
| `aggregator.go` | 独立后台任务 | `master.go` 启动 | Database repo（读写） |
| `cleaner.go` | 独立后台任务 | `master.go` 启动 | Database repo（写） |
| `status_checker.go` | 独立后台任务 | `master.go` 启动 | Database repo（读写） |
| `calculator.go` | 纯函数 | 任何层 | 无外部依赖 |

**规则**：
- Gateway/Handler **禁止**直接调用 `slo/` 模块，必须经过 `service.Query`
- `slo/processor` 的写入通过 `service/sync/` 触发，属于 Service 层的内部实现
- 后台任务（aggregator/cleaner/status_checker）不在请求链路中，直接操作 Database 是合理的

### 3.3 数据接收流程

Agent 的 `SLOSnapshot` 嵌入 `ClusterSnapshot.SLOData`，随快照统一上报。

```
Agent POST /agent/snapshot
    ↓
agentsdk/snapshot.go: handleSnapshot()
    ↓
processor.ProcessSnapshot() → 回调 OnSnapshotReceived
    ↓
sync/slo_persist.go: Sync(clusterID)
    ↓ 提取 SLOSnapshot
slo/processor.go: ProcessSLOSnapshot(ctx, clusterID, snapshot)
    ├── processServiceMetrics()   → slo_service_raw 表
    ├── processEdges()            → slo_edge_raw 表
    ├── processIngressMetrics()   → slo_ingress_raw 表
    └── processRoutes()           → slo_route_mapping 表
```

**Processor 入口**：

```go
// ProcessSLOSnapshot 处理完整 SLO 快照
// Agent 发送的是增量值，Master 直接存储，无需 delta 计算
func (p *Processor) ProcessSLOSnapshot(
    ctx context.Context,
    clusterID string,
    snapshot *model_v2.SLOSnapshot,
) error {
    ts := time.Unix(snapshot.Timestamp, 0)

    // 1. 服务网格指标
    for _, svc := range snapshot.Services {
        p.processServiceMetrics(ctx, clusterID, ts, &svc)
    }

    // 2. 拓扑边
    for _, edge := range snapshot.Edges {
        p.processEdge(ctx, clusterID, ts, &edge)
    }

    // 3. 入口指标
    for _, ing := range snapshot.Ingress {
        p.processIngressMetrics(ctx, clusterID, ts, &ing)
    }

    // 4. 路由映射
    p.processRoutes(ctx, clusterID, snapshot.Routes)

    return nil
}
```

---

## 4. 数据库表结构

### 4.1 服务网格原始数据

```sql
-- 每次采集一行，保留 48 小时
CREATE TABLE slo_service_raw (
    id INTEGER PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    namespace TEXT NOT NULL,
    name TEXT NOT NULL,               -- workload name (deployment 等)
    timestamp TEXT NOT NULL,

    -- 请求汇总（从 ServiceMetrics.Requests[] 聚合）
    total_requests INTEGER NOT NULL DEFAULT 0,
    error_requests INTEGER NOT NULL DEFAULT 0,  -- classification=failure 的总和

    -- 状态码分组（按前缀归类）
    status_2xx INTEGER NOT NULL DEFAULT 0,
    status_3xx INTEGER NOT NULL DEFAULT 0,
    status_4xx INTEGER NOT NULL DEFAULT 0,
    status_5xx INTEGER NOT NULL DEFAULT 0,

    -- 延迟（毫秒，从 LatencySum/LatencyCount 直接存储）
    latency_sum REAL NOT NULL DEFAULT 0,
    latency_count INTEGER NOT NULL DEFAULT 0,

    -- 延迟直方图（JSON，从 LatencyBuckets 直接存储）
    -- 格式: {"1":10, "5":50, "100":200, ...}
    latency_buckets TEXT,

    -- mTLS 覆盖率
    tls_request_delta INTEGER NOT NULL DEFAULT 0,
    total_request_delta INTEGER NOT NULL DEFAULT 0,

    UNIQUE(cluster_id, namespace, name, timestamp)
);
CREATE INDEX idx_service_raw_cluster_ns_name_ts
    ON slo_service_raw(cluster_id, namespace, name, timestamp);
CREATE INDEX idx_service_raw_timestamp
    ON slo_service_raw(timestamp);
```

### 4.2 服务网格小时聚合

```sql
-- 保留 90 天
CREATE TABLE slo_service_hourly (
    id INTEGER PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    namespace TEXT NOT NULL,
    name TEXT NOT NULL,
    hour_start TEXT NOT NULL,

    -- 汇总
    total_requests INTEGER NOT NULL DEFAULT 0,
    error_requests INTEGER NOT NULL DEFAULT 0,
    availability REAL,                  -- (total - error) / total * 100

    -- 延迟百分位（从累积直方图插值计算）
    p50_latency_ms INTEGER,
    p95_latency_ms INTEGER,
    p99_latency_ms INTEGER,
    avg_latency_ms INTEGER,             -- latency_sum / latency_count
    avg_rps REAL,                       -- total_requests / 3600

    -- 状态码分组
    status_2xx INTEGER NOT NULL DEFAULT 0,
    status_3xx INTEGER NOT NULL DEFAULT 0,
    status_4xx INTEGER NOT NULL DEFAULT 0,
    status_5xx INTEGER NOT NULL DEFAULT 0,

    -- 延迟直方图（该小时累积，用于百分位计算）
    latency_buckets TEXT,

    -- mTLS
    mtls_percent REAL,                  -- tls 总和 / total 总和 * 100

    sample_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    UNIQUE(cluster_id, namespace, name, hour_start)
);
```

### 4.3 拓扑边原始数据

```sql
-- 每次采集一行，保留 48 小时
CREATE TABLE slo_edge_raw (
    id INTEGER PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    src_namespace TEXT NOT NULL,
    src_name TEXT NOT NULL,
    dst_namespace TEXT NOT NULL,
    dst_name TEXT NOT NULL,
    timestamp TEXT NOT NULL,

    request_delta INTEGER NOT NULL DEFAULT 0,
    failure_delta INTEGER NOT NULL DEFAULT 0,
    latency_sum REAL NOT NULL DEFAULT 0,      -- ms
    latency_count INTEGER NOT NULL DEFAULT 0,

    UNIQUE(cluster_id, src_namespace, src_name, dst_namespace, dst_name, timestamp)
);
CREATE INDEX idx_edge_raw_cluster_ts
    ON slo_edge_raw(cluster_id, timestamp);
```

### 4.4 拓扑边小时聚合

```sql
-- 保留 90 天
CREATE TABLE slo_edge_hourly (
    id INTEGER PRIMARY KEY,
    cluster_id TEXT NOT NULL,
    src_namespace TEXT NOT NULL,
    src_name TEXT NOT NULL,
    dst_namespace TEXT NOT NULL,
    dst_name TEXT NOT NULL,
    hour_start TEXT NOT NULL,

    total_requests INTEGER NOT NULL DEFAULT 0,
    error_requests INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms INTEGER,
    avg_rps REAL,
    error_rate REAL,                    -- error / total * 100

    sample_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    UNIQUE(cluster_id, src_namespace, src_name, dst_namespace, dst_name, hour_start)
);
```

### 4.5 入口原始数据

重写 `slo_metrics_raw`，**统一 bucket 格式为 JSON**（与 slo_service_raw 一致），删除 12 列固定 bucket：

```sql
-- 重写后的 slo_metrics_raw（旧表 DROP + 重建）
CREATE TABLE slo_metrics_raw (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    host TEXT NOT NULL,
    timestamp TEXT NOT NULL,

    total_requests INTEGER NOT NULL DEFAULT 0,
    error_requests INTEGER NOT NULL DEFAULT 0,

    -- 延迟（统一格式：sum/count + JSON bucket）
    latency_sum REAL NOT NULL DEFAULT 0,          -- ms（旧: sum_latency_ms INTEGER）
    latency_count INTEGER NOT NULL DEFAULT 0,     -- 新增
    latency_buckets TEXT,                          -- JSON（旧: 12 列 bucket_5ms~bucket_inf）

    -- HTTP method 分布
    method_get INTEGER DEFAULT 0,
    method_post INTEGER DEFAULT 0,
    method_put INTEGER DEFAULT 0,
    method_delete INTEGER DEFAULT 0,
    method_other INTEGER DEFAULT 0,

    -- 域名/路径
    domain TEXT,
    path_prefix TEXT DEFAULT '/',

    is_missing INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_slo_raw_cluster_host_ts ON slo_metrics_raw(cluster_id, host, timestamp);
CREATE INDEX idx_slo_raw_timestamp ON slo_metrics_raw(timestamp);
CREATE INDEX idx_slo_raw_domain ON slo_metrics_raw(cluster_id, domain, path_prefix, timestamp);
```

入口小时聚合 `slo_metrics_hourly` 同步重写：

```sql
CREATE TABLE slo_metrics_hourly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    host TEXT NOT NULL,
    hour_start TEXT NOT NULL,

    total_requests INTEGER NOT NULL DEFAULT 0,
    error_requests INTEGER NOT NULL DEFAULT 0,
    availability REAL NOT NULL,

    p50_latency_ms INTEGER NOT NULL DEFAULT 0,
    p95_latency_ms INTEGER NOT NULL DEFAULT 0,
    p99_latency_ms INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms INTEGER NOT NULL DEFAULT 0,
    avg_rps REAL NOT NULL,

    -- 延迟直方图（统一 JSON）
    latency_buckets TEXT,                          -- 旧: 12 列 bucket_5ms~bucket_inf

    -- HTTP method 分布
    method_get INTEGER DEFAULT 0,
    method_post INTEGER DEFAULT 0,
    method_put INTEGER DEFAULT 0,
    method_delete INTEGER DEFAULT 0,
    method_other INTEGER DEFAULT 0,

    -- 域名/路径
    domain TEXT,
    path_prefix TEXT DEFAULT '/',

    sample_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    UNIQUE(cluster_id, host, hour_start)
);
CREATE INDEX idx_slo_hourly_hour ON slo_metrics_hourly(hour_start);
CREATE INDEX idx_slo_hourly_domain ON slo_metrics_hourly(cluster_id, domain, path_prefix, hour_start);
```

**统一 bucket 格式的好处**: calculator/aggregator/processor/handler 全部只需处理 `map[string]int64` 一种格式，无需区分入口/服务网格。

### 4.6 路由映射表

沿用现有 `slo_route_mapping`，适配标准化 ServiceKey：

```sql
-- 不变: 表结构
-- 变化: service_key 列存储标准化格式 "namespace-service-port"（不含 @kubernetes 后缀）
```

### 4.7 SLO 目标表

扩展支持服务网格层（service 维度）：

```sql
-- 在现有 slo_targets 基础上，新增可选字段区分维度
ALTER TABLE slo_targets ADD COLUMN target_type TEXT DEFAULT 'ingress';
-- target_type: 'ingress' (域名维度) | 'service' (服务维度)
-- host 列: ingress 模式存 domain, service 模式存 "namespace/name"
```

### 4.8 表分类总览

| 表 | 操作 | 说明 |
|---|---|---|
| `slo_service_raw` | **新建** | 服务网格原始数据 (48h) |
| `slo_service_hourly` | **新建** | 服务网格小时聚合 (90d) |
| `slo_edge_raw` | **新建** | 拓扑边原始数据 (48h) |
| `slo_edge_hourly` | **新建** | 拓扑边小时聚合 (90d) |
| `slo_metrics_raw` | **重写** | 12列bucket→JSON + latency_count + method列（DROP旧表重建） |
| `slo_metrics_hourly` | **重写** | 12列bucket→JSON + method列（DROP旧表重建） |
| `slo_route_mapping` | 不变 | ServiceKey 标准化 |
| `slo_targets` | **修改** | +1 列 (target_type) |
| `slo_status_history` | 不变 | 状态变更历史 |
| `ingress_counter_snapshot` | **删除** | 不再需要 (Agent 端计算增量) |
| `ingress_histogram_snapshot` | **删除** | 不再需要 (Agent 端计算增量) |

---

## 5. Processor

### 5.1 processServiceMetrics

```go
// processServiceMetrics 存储服务网格指标到 slo_service_raw
func (p *Processor) processServiceMetrics(
    ctx context.Context,
    clusterID string,
    ts time.Time,
    svc *model_v2.ServiceMetrics,
) error {
    // 1. 从 Requests[] 聚合状态码分组
    var totalReqs, errorReqs int64
    var s2xx, s3xx, s4xx, s5xx int64
    for _, r := range svc.Requests {
        totalReqs += r.Delta
        if r.Classification == "failure" {
            errorReqs += r.Delta
        }
        switch {
        case strings.HasPrefix(r.StatusCode, "2"): s2xx += r.Delta
        case strings.HasPrefix(r.StatusCode, "3"): s3xx += r.Delta
        case strings.HasPrefix(r.StatusCode, "4"): s4xx += r.Delta
        case strings.HasPrefix(r.StatusCode, "5"): s5xx += r.Delta
        }
    }

    // 2. 直接插入 raw 表（Agent 已算好增量，无需 delta 计算）
    raw := &database.SLOServiceRaw{
        ClusterID:         clusterID,
        Namespace:         svc.Namespace,
        Name:              svc.Name,
        Timestamp:         ts,
        TotalRequests:     totalReqs,
        ErrorRequests:     errorReqs,
        Status2xx:         s2xx,
        Status3xx:         s3xx,
        Status4xx:         s4xx,
        Status5xx:         s5xx,
        LatencySum:        svc.LatencySum,
        LatencyCount:      svc.LatencyCount,
        LatencyBuckets:    marshalBuckets(svc.LatencyBuckets),
        TLSRequestDelta:   svc.TLSRequestDelta,
        TotalRequestDelta: svc.TotalRequestDelta,
    }
    return p.repo.InsertServiceRaw(ctx, raw)
}
```

### 5.2 processEdge

```go
// processEdge 存储拓扑边到 slo_edge_raw
func (p *Processor) processEdge(
    ctx context.Context,
    clusterID string,
    ts time.Time,
    edge *model_v2.ServiceEdge,
) error {
    raw := &database.SLOEdgeRaw{
        ClusterID:    clusterID,
        SrcNamespace: edge.SrcNamespace,
        SrcName:      edge.SrcName,
        DstNamespace: edge.DstNamespace,
        DstName:      edge.DstName,
        Timestamp:    ts,
        RequestDelta: edge.RequestDelta,
        FailureDelta: edge.FailureDelta,
        LatencySum:   edge.LatencySum,
        LatencyCount: edge.LatencyCount,
    }
    return p.repo.InsertEdgeRaw(ctx, raw)
}
```

### 5.3 processIngressMetrics

```go
// processIngressMetrics 存储入口指标到 slo_metrics_raw
// Agent 已算好增量，直接插入
func (p *Processor) processIngressMetrics(
    ctx context.Context,
    clusterID string,
    ts time.Time,
    ing *model_v2.IngressMetrics,
) error {
    // 1. 从 Requests[] 聚合
    var totalReqs, errorReqs int64
    methodCounts := map[string]int64{}  // GET/POST/PUT/DELETE/OTHER
    for _, r := range ing.Requests {
        totalReqs += r.Delta
        code, _ := strconv.Atoi(r.Code)
        if code >= 500 {
            errorReqs += r.Delta
        }
        methodCounts[r.Method] += r.Delta
    }

    // 2. 直接存储（bucket 统一为 JSON，无需列映射）
    raw := &database.SLOMetricsRaw{
        ClusterID:      clusterID,
        Host:           ing.ServiceKey,
        Timestamp:      ts,
        TotalRequests:  totalReqs,
        ErrorRequests:  errorReqs,
        LatencySum:     ing.LatencySum,
        LatencyCount:   ing.LatencyCount,
        LatencyBuckets: ing.LatencyBuckets,   // map[string]int64 → JSON TEXT
        MethodGet:      methodCounts["GET"],
        MethodPost:     methodCounts["POST"],
        MethodPut:      methodCounts["PUT"],
        MethodDelete:   methodCounts["DELETE"],
        MethodOther:    methodCounts["OTHER"] + methodCounts["PATCH"] + methodCounts["HEAD"],
    }

    // 3. 关联域名（从 route_mapping 查）
    mapping, _ := p.repo.GetRouteMappingByServiceKey(ctx, clusterID, ing.ServiceKey)
    if mapping != nil {
        raw.Domain = mapping.Domain
        raw.PathPrefix = mapping.PathPrefix
    }

    return p.repo.InsertRawMetrics(ctx, raw)
}
```

### 5.4 processRoutes

```go
// processRoutes 更新路由映射（不变）
func (p *Processor) processRoutes(
    ctx context.Context,
    clusterID string,
    routes []model_v2.IngressRouteInfo,
) {
    for _, route := range routes {
        p.repo.UpsertRouteMapping(ctx, &database.SLORouteMapping{
            ClusterID:   clusterID,
            Domain:      route.Domain,
            PathPrefix:  route.PathPrefix,
            ServiceKey:  route.ServiceKey,  // 标准化: "namespace-service-port"
            ServiceName: route.ServiceName,
            ServicePort: route.ServicePort,
            TLS:         route.TLS,
            // ...
        })
    }
}
```

---

## 6. Aggregator

定时预聚合 raw → hourly。每次运行聚合**上一个完整小时**和**当前不完整小时**，确保 hourly 始终是最新的。Aggregator 是性能优化，不是数据可见性的前置条件。

### 6.1 服务网格聚合

```go
// aggregateServiceHour 聚合服务指标
func (a *Aggregator) aggregateServiceHour(ctx context.Context, hour time.Time) {
    start := hour.Truncate(time.Hour)
    end := start.Add(time.Hour)

    // 1. 查询该小时内所有 service raw 数据
    raws, _ := a.repo.GetServiceRawByTimeRange(ctx, clusterID, start, end)

    // 2. 按 (namespace, name) 分组
    groups := groupServiceRaws(raws)

    for key, rows := range groups {
        var totalReqs, errorReqs int64
        var latencySum float64
        var latencyCount int64
        var tlsReqs, totalReqsDelta int64
        mergedBuckets := map[string]int64{}
        var s2xx, s3xx, s4xx, s5xx int64

        for _, row := range rows {
            totalReqs += row.TotalRequests
            errorReqs += row.ErrorRequests
            latencySum += row.LatencySum
            latencyCount += row.LatencyCount
            tlsReqs += row.TLSRequestDelta
            totalReqsDelta += row.TotalRequestDelta
            s2xx += row.Status2xx
            s3xx += row.Status3xx
            s4xx += row.Status4xx
            s5xx += row.Status5xx
            // 合并直方图桶
            mergeBuckets(mergedBuckets, row.LatencyBuckets)
        }

        // 3. 计算衍生指标
        hourly := &database.SLOServiceHourly{
            Namespace:      key.Namespace,
            Name:           key.Name,
            HourStart:      start,
            TotalRequests:  totalReqs,
            ErrorRequests:  errorReqs,
            Availability:   calcAvailability(totalReqs, errorReqs),
            P50LatencyMs:   calcPercentile(mergedBuckets, 0.50),
            P95LatencyMs:   calcPercentile(mergedBuckets, 0.95),
            P99LatencyMs:   calcPercentile(mergedBuckets, 0.99),
            AvgLatencyMs:   calcAvgLatency(latencySum, latencyCount),
            AvgRPS:         float64(totalReqs) / 3600.0,
            MtlsPercent:    calcMtlsPercent(tlsReqs, totalReqsDelta),
            LatencyBuckets: marshalBuckets(mergedBuckets),
            SampleCount:    len(rows),
        }
        a.repo.UpsertServiceHourly(ctx, hourly)
    }
}
```

### 6.2 拓扑聚合

```go
// aggregateEdgeHour 聚合拓扑边
func (a *Aggregator) aggregateEdgeHour(ctx context.Context, hour time.Time) {
    // 同理：按 (src_ns, src_name, dst_ns, dst_name) 分组
    // 计算:
    //   total_requests = sum(request_delta)
    //   error_requests = sum(failure_delta)
    //   avg_latency_ms = sum(latency_sum) / sum(latency_count)
    //   avg_rps = total_requests / 3600
    //   error_rate = error_requests / total_requests * 100
}
```

### 6.3 入口聚合

`aggregateIngressHour` 按 `(cluster_id, host)` 分组聚合 `slo_metrics_raw` → `slo_metrics_hourly`，含 method 分布聚合。

### 6.4 百分位计算

```go
// calcPercentile 从累积直方图插值计算百分位
// buckets: {"1":10, "5":50, "100":200, "+Inf":250}
// percentile: 0.50, 0.95, 0.99
func calcPercentile(buckets map[string]int64, percentile float64) int64 {
    // 1. 排序 bucket 边界
    // 2. 计算目标 count = total * percentile
    // 3. 找到目标 count 落在的 bucket 区间 [lower, upper]
    // 4. 线性插值:
    //    result = lower + (upper - lower) * (target - countAtLower) / (countAtUpper - countAtLower)
}
```

---

## 7. API 端点

**分层约束**: 所有 Handler 依赖 `service.Query` 接口，**禁止直接持有 Database Repository**。查询策略（hourly 优先 → raw 回退）在 `service/query/slo.go` 中实现。

### 7.0 Service 层接口新增

```go
// service/interfaces.go — Query 接口新增 SLO 查询方法
type Query interface {
    // ... 现有方法 ...

    // SLO 服务网格
    GetMeshTopology(ctx context.Context, clusterID, timeRange string) (*model.ServiceMeshTopologyResponse, error)
    GetServiceDetail(ctx context.Context, clusterID, namespace, name, timeRange string) (*model.ServiceDetailResponse, error)

    // SLO 域名（增强）
    GetDomainsV2(ctx context.Context, clusterID, timeRange string) ([]model.ServiceSLOEnhanced, error)
}
```

```go
// service/query/slo.go — 查询实现（含 hourly 优先 → raw 回退策略）
func (q *QueryService) GetMeshTopology(ctx context.Context, clusterID, timeRange string) (*model.ServiceMeshTopologyResponse, error) {
    // 1. 查 slo_service_hourly → 回退 slo_service_raw → 实时聚合
    // 2. 查 slo_edge_hourly → 回退 slo_edge_raw
    // 3. 计算衍生指标 (RPS, ErrorRate, Status)
    // 4. 组装响应
}

func (q *QueryService) GetServiceDetail(ctx context.Context, clusterID, namespace, name, timeRange string) (*model.ServiceDetailResponse, error) {
    // 1. 查服务指标（同 hourly → raw 策略）
    // 2. 查上下游 edge 数据
    // 3. 查历史趋势（hourly 数据点）
    // 4. 组装响应
}

func (q *QueryService) GetDomainsV2(ctx context.Context, clusterID, timeRange string) ([]model.ServiceSLOEnhanced, error) {
    // 1. 查 slo_metrics_hourly → 回退 slo_metrics_raw
    // 2. 从 raw 查 latencyDistribution / requestBreakdown
    // 3. 从 route_mapping 匹配 backendServiceIDs
    // 4. 组装增强响应
}
```

### 7.1 新增：服务网格 API

#### Handler 定义

```go
// handler/slo_mesh.go — 依赖 service.Query，不直接访问 Database
type SLOMeshHandler struct {
    query service.Query
}

func NewSLOMeshHandler(query service.Query) *SLOMeshHandler {
    return &SLOMeshHandler{query: query}
}
```

#### GET /api/v2/slo/mesh/topology

返回服务拓扑（节点 + 边），供前端 `ServiceTopologyView` 渲染。

```go
// 请求参数
// cluster_id: 集群 ID
// time_range: "1h" | "6h" | "24h"（拓扑展示的时间窗口）

// 响应
type ServiceMeshTopologyResponse struct {
    Nodes []ServiceNodeResponse `json:"nodes"`
    Edges []ServiceEdgeResponse `json:"edges"`
}

type ServiceNodeResponse struct {
    ID        string  `json:"id"`         // "namespace/name"
    Name      string  `json:"name"`
    Namespace string  `json:"namespace"`

    // 黄金指标（从 hourly 或 raw 实时聚合）
    RPS        float64 `json:"rps"`
    AvgLatency float64 `json:"avg_latency"`  // ms
    P50Latency float64 `json:"p50_latency"`
    P95Latency float64 `json:"p95_latency"`
    P99Latency float64 `json:"p99_latency"`
    ErrorRate  float64 `json:"error_rate"`   // %

    Status      string  `json:"status"`       // healthy/warning/critical
    MtlsPercent float64 `json:"mtls_percent"` // %

    // 详细分布
    LatencyDistribution []LatencyBucketResponse    `json:"latency_distribution"`
    StatusCodeBreakdown []StatusCodeBreakdownResponse `json:"status_code_breakdown"`

    TotalRequests int64 `json:"total_requests"`
}

type ServiceEdgeResponse struct {
    Source     string  `json:"source"`      // "namespace/name"
    Target     string  `json:"target"`
    RPS        float64 `json:"rps"`
    AvgLatency float64 `json:"avg_latency"` // ms
    ErrorRate  float64 `json:"error_rate"`  // %
}

type LatencyBucketResponse struct {
    Le    float64 `json:"le"`    // 上界 (ms)
    Count int64   `json:"count"`
}

type StatusCodeBreakdownResponse struct {
    Code  string `json:"code"`  // "2xx", "3xx", "4xx", "5xx"
    Count int64  `json:"count"`
}
```

**Handler 调用**：

```go
func (h *SLOMeshHandler) MeshTopology(w http.ResponseWriter, r *http.Request) {
    // Handler 只做参数解析和响应序列化，查询逻辑在 service/query/slo.go
    resp, err := h.query.GetMeshTopology(ctx, clusterID, timeRange)
    // ...
}
```

#### GET /api/v2/slo/mesh/service/detail

返回单个服务的详细指标 + 历史。

```go
// 请求参数
// cluster_id, namespace, name, time_range

// 响应
type ServiceDetailResponse struct {
    ServiceNodeResponse              // 嵌入基本指标

    // 历史数据点（用于趋势图）
    History []ServiceHistoryPoint `json:"history"`

    // 上游/下游服务（从 edge 数据）
    Upstreams  []ServiceEdgeResponse `json:"upstreams"`
    Downstreams []ServiceEdgeResponse `json:"downstreams"`
}

type ServiceHistoryPoint struct {
    Timestamp    string  `json:"timestamp"`
    RPS          float64 `json:"rps"`
    P95Latency   float64 `json:"p95_latency"`
    ErrorRate    float64 `json:"error_rate"`
    Availability float64 `json:"availability"`
    MtlsPercent  float64 `json:"mtls_percent"`
}
```

### 7.2 适配：域名 SLO API

`/api/v2/slo/domains/v2` 接口不变，数据来源为 `slo_metrics_raw` + `slo_metrics_hourly`。

#### GET /api/v2/slo/domains/v2（增强）

响应中新增详细分布字段：

```go
// 在现有 DomainSLOV2 / ServiceSLO 基础上，新增:
type ServiceSLOEnhanced struct {
    ServiceSLO  // 嵌入现有字段

    // 新增详细分布
    LatencyDistribution []LatencyBucketResponse       `json:"latency_distribution,omitempty"`
    RequestBreakdown    []RequestBreakdownResponse     `json:"request_breakdown,omitempty"`
    StatusCodeBreakdown []StatusCodeBreakdownResponse  `json:"status_code_breakdown,omitempty"`

    // 关联的服务网格节点 ID（用于前端 backendServices 跳转）
    BackendServiceIDs []string `json:"backend_service_ids,omitempty"`
}

type RequestBreakdownResponse struct {
    Method     string `json:"method"`      // GET, POST, PUT, DELETE
    Count      int64  `json:"count"`
    ErrorCount int64  `json:"error_count"`
}
```

**BackendServiceIDs 计算**: 从 `slo_route_mapping` 的 `service_name` + `namespace` 匹配 `slo_service_raw` 中的服务节点。

### 7.3 API 端点总览

| 端点 | 操作 | 说明 |
|---|---|---|
| `GET /api/v2/slo/mesh/topology` | **新增** | 服务拓扑（节点+边） |
| `GET /api/v2/slo/mesh/service/detail` | **新增** | 单服务详情+历史+上下游 |
| `GET /api/v2/slo/domains/v2` | **增强** | 域名 SLO +分布数据+关联服务 |
| `GET /api/v2/slo/domains/detail` | **增强** | 域名详情 +延迟分布+请求分布 |
| `GET /api/v2/slo/domains/history` | 不变 | 域名历史数据 |
| `GET /api/v2/slo/targets` | 不变 | SLO 目标 |
| `PUT /api/v2/slo/targets` | **适配** | 支持 target_type 区分服务/入口 |

---

## 8. 数据保留与清理

| 数据类型 | 保留时间 | 表 |
|---|---|---|
| 服务网格原始数据 | 48 小时 | `slo_service_raw` |
| 服务网格小时聚合 | 90 天 | `slo_service_hourly` |
| 拓扑边原始数据 | 48 小时 | `slo_edge_raw` |
| 拓扑边小时聚合 | 90 天 | `slo_edge_hourly` |
| 入口原始数据 | 48 小时 | `slo_metrics_raw` |
| 入口小时聚合 | 90 天 | `slo_metrics_hourly` |
| 状态变更历史 | 180 天 | `slo_status_history` |
| 路由映射 | 永久 | `slo_route_mapping` |
| SLO 目标 | 永久 | `slo_targets` |

沿用现有 `Cleaner` 的定时清理机制（每小时执行），新增对 `slo_service_raw`、`slo_edge_raw` 的清理。

---

## 9. 文件变更清单

### 新建

| 文件 | 说明 |
|---|---|
| `slo/interfaces.go` | SLO 领域处理器对外接口（SLOProcessor / SLOAggregator / SLOCleaner） |
| `service/query/slo.go` | SLO 查询实现（服务网格 + 域名增强，含 hourly→raw 回退策略） |
| `database/sqlite/slo_service.go` | 服务网格表 SQL Dialect（service_raw + service_hourly） |
| `database/sqlite/slo_edge.go` | 拓扑边表 SQL Dialect（edge_raw + edge_hourly） |
| `gateway/handler/slo_mesh.go` | 服务网格 API Handler（依赖 service.Query，不直接访问 Database） |

### 重写

| 文件 | 旧代码 | 新代码 | 变更说明 |
|---|---|---|---|
| `slo/processor.go` | ~368 行（ProcessIngressMetrics + snapshot delta 计算 ~280 行） | ~150 行（ProcessSLOSnapshot + processServiceMetrics + processEdge + processIngressMetrics） | 移除全部 snapshot-based delta 逻辑（Agent 已计算增量），只做直接写入 raw 表 |
| `slo/aggregator.go` | ~219 行（仅 aggregateHostHour，12 列固定 bucket，秒/float64） | ~350 行（aggregateServiceHour + aggregateEdgeHour + aggregateIngressHour，JSON map bucket，ms/string） | 3 条聚合路径 + bucket 格式从 12 列固定列改为 JSON map |
| `service/sync/slo_persist.go` | 含旧模式分支 | 只保留 OTel 路径 | 移除旧模式分支 |
| `database/sqlite/slo.go` | 含 snapshot 表 SQL + 12 列 bucket | 移除 snapshot SQL，bucket 改为 JSON 列 | 适配新表结构 |

### 修改

| 文件 | 变更 |
|---|---|
| `slo/calculator.go` | 删除 `CalculateDelta()`（Agent 处理）、`RawBuckets` 结构体、`BucketsFromRaw()`/`BucketsToRaw()`/`MergeBuckets()`（12 列格式废弃）；`CalculateQuantile()` 入参统一为 `map[string]int64`（毫秒），入口/服务网格共用 |
| `slo/cleaner.go` | +2 处：新增 `slo_service_raw` + `slo_edge_raw` 清理；移除 snapshot 表清理 |
| `slo/status_checker.go` | 小改：扩展支持 service 维度的状态检测（现有只支持 host 维度） |
| `database/sqlite/migrations.go` | 新增 4 张表 + DROP 重建 slo_metrics_raw/hourly（bucket JSON 化）+ ALTER slo_targets + DROP 2 张 snapshot 表 |
| `database/interfaces.go` | 新增 SLOServiceRepository + SLOEdgeRepository 接口；移除 snapshot 相关接口 |
| `database/repo/slo.go` | 实现新 Repository 接口；移除 snapshot 实现 |
| `service/interfaces.go` | Query 接口新增 SLO 查询方法（GetMeshTopology / GetServiceDetail / GetDomainsV2） |
| `service/factory.go` | QueryService 注入新 Repository（sloServiceRepo, sloEdgeRepo） |
| `gateway/handler/slo.go` | DomainsV2 增强，改为依赖 service.Query（不再直接调 repo）|
| `gateway/routes.go` | 注册新路由 `/api/v2/slo/mesh/*` |
| `master.go` | 依赖注入新 Repository + Handler + QueryService 扩展 |
| `model/slo.go` | 新增 API 响应类型（ServiceNodeResponse 等）|
| `agentsdk/server.go` | 移除 `/agent/slo` 路由注册 |

### 删除

| 文件 | 说明 |
|---|---|
| `agentsdk/slo.go` | 旧模式专用 HTTP Handler，不再需要 |

### 保留不动

| 文件 | 说明 |
|---|---|
| `agentsdk/snapshot.go` | 快照接收不变 |

---

## 10. 实现阶段

```
Phase 1: 数据库 + 清理旧代码
  - migrations: 新增 4 张表 + 2 个 ALTER + DROP 2 张 snapshot 表
  - SQL Dialect: 新建 slo_service.go, slo_edge.go; 重写 slo.go (移除 snapshot)
  - Repository: 新增接口 + 实现; 移除 snapshot 相关接口
  - 新建 slo/interfaces.go (SLOProcessor / SLOAggregator / SLOCleaner 接口)
  - 删除 agentsdk/slo.go; server.go 移除路由注册

Phase 2: Processor + Sync
  - 重写 processor.go: ProcessSLOSnapshot 入口
  - processServiceMetrics / processEdge / processIngressMetrics / processRoutes
  - 重写 slo_persist.go: 只保留 SLOData 路径
  - 单元测试

Phase 3: Aggregator + Cleaner
  - aggregateServiceHour + aggregateEdgeHour + aggregateIngressHour
  - Cleaner 新增 service_raw + edge_raw 清理
  - 单元测试

Phase 4: Service 层 + API + 集成
  - service/interfaces.go 新增 SLO 查询方法
  - 新建 service/query/slo.go (查询实现 + hourly→raw 回退策略)
  - service/factory.go 注入新 Repository
  - 新建 handler/slo_mesh.go (依赖 service.Query)
  - handler/slo.go DomainsV2 增强，改为依赖 service.Query
  - 路由注册 + master.go 依赖注入
  - 端到端测试: Agent → Master → Service → API → 验证前端字段
```
