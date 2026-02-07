# SLO OTel Master 设计书

## 概要

本文档描述 AtlHyper Master 处理 OTel SLO 数据的设计方案。Master 负责接收 Agent 上报的数据，存储到 SQLite，提供聚合计算和 API 服务。

---

## 1. 设计目标

### 1.1 功能目标

| 目标 | 说明 |
|------|------|
| 数据存储 | 接收 Agent 上报的 SLO 数据，持久化到 SQLite |
| 聚合计算 | 原始数据聚合为小时/天/周/月维度 |
| SLO 评估 | 计算可用性、延迟分位数、错误预算 |
| API 服务 | 提供前端所需的 SLO 查询 API |
| 状态变更 | 监测 SLO 状态变化并记录历史 |

### 1.2 数据流向

```
Agent ──gRPC──▶ Master ──▶ SQLite
                  │
                  ├──▶ 原始数据存储 (48h)
                  ├──▶ 小时聚合 (90d)
                  ├──▶ SLO 状态评估
                  └──▶ API 响应
```

### 1.3 文件夹结构

```
atlhyper_master_v2/
├── slo/                          # SLO 模块
│   ├── slo.go                    # 模块入口，初始化和启动
│   ├── config.go                 # SLO 配置结构
│   │
│   ├── receiver/                 # 数据接收
│   │   ├── receiver.go           # 接收器接口
│   │   └── grpc_receiver.go      # gRPC 数据接收实现
│   │
│   ├── repository/               # 数据访问层
│   │   ├── repository.go         # Repository 聚合
│   │   ├── services_repo.go      # slo_services 表操作
│   │   ├── edges_repo.go         # slo_edges 表操作
│   │   ├── routes_repo.go        # slo_ingress_routes 表操作
│   │   ├── snapshot_repo.go      # slo_metrics_snapshot 表操作
│   │   ├── golden_repo.go        # slo_golden_raw/hourly 表操作
│   │   ├── traefik_repo.go       # slo_traefik_raw 表操作
│   │   ├── traces_repo.go        # slo_traces/spans 表操作
│   │   ├── targets_repo.go       # slo_targets 表操作
│   │   └── status_repo.go        # slo_status_history 表操作
│   │
│   ├── aggregator/               # 聚合计算
│   │   ├── aggregator.go         # 聚合调度器
│   │   ├── hourly.go             # 小时聚合逻辑
│   │   ├── percentile.go         # 分位数计算
│   │   └── cleanup.go            # 数据清理
│   │
│   ├── evaluator/                # SLO 评估
│   │   ├── evaluator.go          # 评估器
│   │   ├── status.go             # 状态判断
│   │   └── budget.go             # 错误预算计算
│   │
│   └── api/                      # API 处理
│       ├── handler.go            # API 处理器注册
│       ├── services.go           # 服务列表 API
│       ├── topology.go           # 拓扑 API
│       ├── metrics.go            # 指标/延迟分布 API
│       ├── traces.go             # Trace API
│       ├── targets.go            # SLO 目标 API
│       └── status.go             # SLO 状态 API
│
├── database/
│   └── sqlite/
│       ├── migrations.go         # 数据库迁移（新增 SLO 表）
│       └── slo.go                # SLO 相关 SQL（重构）
│
├── gateway/
│   └── router.go                 # 路由（新增 /api/v2/slo/* 路由）
│
└── config/
    └── config.go                 # 配置（新增 SLO 配置项）
```

### 1.4 模型定义

```
model_v2/
├── slo.go                        # SLO 相关模型（重构）
│   ├── Service                   # 服务信息
│   ├── ServiceEdge               # 服务拓扑边
│   ├── IngressRoute              # 入口路由
│   ├── GoldenMetrics             # 黄金指标
│   ├── GoldenHourly              # 黄金指标聚合
│   ├── TraefikMetrics            # Traefik 指标
│   ├── Trace                     # Trace
│   ├── Span                      # Span
│   ├── SLOTarget                 # SLO 目标
│   ├── SLOStatus                 # SLO 状态
│   └── SLOStatusHistory          # 状态历史
```

---

## 2. 数据库设计

### 2.1 表结构总览

| 表名 | 用途 | 保留时间 |
|------|------|----------|
| `slo_services` | 服务注册 | 永久 |
| `slo_edges` | 服务拓扑边 | 永久 |
| `slo_ingress_routes` | 入口路由配置 | 永久 |
| `slo_metrics_snapshot` | 指标快照 | 实时覆盖 |
| `slo_golden_raw` | 黄金指标原始 | 48h |
| `slo_golden_hourly` | 黄金指标聚合 | 90d |
| `slo_traefik_raw` | Traefik 入口指标 | 48h |
| `slo_traces` | Trace 主表 | 7d |
| `slo_spans` | Span 详情 | 7d |
| `slo_targets` | 目标配置 | 永久 |
| `slo_status_history` | 状态历史 | 180d |

### 2.2 完整 DDL

```sql
-- ==================== 1. 服务注册表 ====================
CREATE TABLE IF NOT EXISTS slo_services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    namespace TEXT NOT NULL,
    name TEXT NOT NULL,
    display_name TEXT,
    service_type TEXT DEFAULT 'service',  -- gateway/service/database/cache/external
    discovered_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    UNIQUE(cluster_id, namespace, name)
);
CREATE INDEX IF NOT EXISTS idx_slo_services_cluster ON slo_services(cluster_id);

-- ==================== 2. 服务拓扑边 ====================
CREATE TABLE IF NOT EXISTS slo_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    source_ns TEXT NOT NULL,
    source_name TEXT NOT NULL,
    target_ns TEXT NOT NULL,
    target_name TEXT NOT NULL,
    protocol TEXT DEFAULT 'http',
    discovered_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    UNIQUE(cluster_id, source_ns, source_name, target_ns, target_name)
);
CREATE INDEX IF NOT EXISTS idx_slo_edges_cluster ON slo_edges(cluster_id);
CREATE INDEX IF NOT EXISTS idx_slo_edges_source ON slo_edges(cluster_id, source_ns, source_name);
CREATE INDEX IF NOT EXISTS idx_slo_edges_target ON slo_edges(cluster_id, target_ns, target_name);

-- ==================== 3. 入口路由配置 ====================
CREATE TABLE IF NOT EXISTS slo_ingress_routes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    path_prefix TEXT NOT NULL DEFAULT '/',
    ingress_name TEXT NOT NULL,
    namespace TEXT NOT NULL,
    tls INTEGER NOT NULL DEFAULT 1,
    backend_ns TEXT NOT NULL,
    backend_svc TEXT NOT NULL,
    backend_port INTEGER NOT NULL,
    entrypoint TEXT DEFAULT 'websecure',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(cluster_id, domain, path_prefix)
);
CREATE INDEX IF NOT EXISTS idx_slo_routes_cluster ON slo_ingress_routes(cluster_id);
CREATE INDEX IF NOT EXISTS idx_slo_routes_domain ON slo_ingress_routes(cluster_id, domain);

-- ==================== 4. 指标快照 ====================
CREATE TABLE IF NOT EXISTS slo_metrics_snapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    labels_hash TEXT NOT NULL,
    labels_json TEXT NOT NULL,
    value REAL NOT NULL,
    prev_value REAL NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    UNIQUE(cluster_id, metric_name, labels_hash)
);
CREATE INDEX IF NOT EXISTS idx_slo_snapshot_cluster ON slo_metrics_snapshot(cluster_id);

-- ==================== 5. 黄金指标原始数据 ====================
CREATE TABLE IF NOT EXISTS slo_golden_raw (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    target_ns TEXT NOT NULL,
    target_name TEXT NOT NULL,
    source_ns TEXT,
    source_name TEXT,
    total_req INTEGER DEFAULT 0,
    success_req INTEGER DEFAULT 0,
    error_req INTEGER DEFAULT 0,
    latency_sum_ms REAL DEFAULT 0,
    latency_count INTEGER DEFAULT 0,
    -- Linkerd histogram buckets
    b_1ms INTEGER DEFAULT 0,
    b_2ms INTEGER DEFAULT 0,
    b_3ms INTEGER DEFAULT 0,
    b_4ms INTEGER DEFAULT 0,
    b_5ms INTEGER DEFAULT 0,
    b_10ms INTEGER DEFAULT 0,
    b_20ms INTEGER DEFAULT 0,
    b_50ms INTEGER DEFAULT 0,
    b_100ms INTEGER DEFAULT 0,
    b_200ms INTEGER DEFAULT 0,
    b_500ms INTEGER DEFAULT 0,
    b_1s INTEGER DEFAULT 0,
    b_2s INTEGER DEFAULT 0,
    b_5s INTEGER DEFAULT 0,
    b_10s INTEGER DEFAULT 0,
    b_inf INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_slo_golden_raw_ts ON slo_golden_raw(cluster_id, target_ns, target_name, timestamp);
CREATE INDEX IF NOT EXISTS idx_slo_golden_raw_time ON slo_golden_raw(timestamp);

-- ==================== 6. 黄金指标小时聚合 ====================
CREATE TABLE IF NOT EXISTS slo_golden_hourly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    hour_start TEXT NOT NULL,
    target_ns TEXT NOT NULL,
    target_name TEXT NOT NULL,
    total_req INTEGER DEFAULT 0,
    success_req INTEGER DEFAULT 0,
    error_req INTEGER DEFAULT 0,
    availability REAL NOT NULL,
    error_rate REAL NOT NULL,
    avg_rps REAL NOT NULL,
    p50_ms INTEGER DEFAULT 0,
    p95_ms INTEGER DEFAULT 0,
    p99_ms INTEGER DEFAULT 0,
    avg_ms INTEGER DEFAULT 0,
    -- buckets for recalc
    b_1ms INTEGER DEFAULT 0,
    b_2ms INTEGER DEFAULT 0,
    b_3ms INTEGER DEFAULT 0,
    b_4ms INTEGER DEFAULT 0,
    b_5ms INTEGER DEFAULT 0,
    b_10ms INTEGER DEFAULT 0,
    b_20ms INTEGER DEFAULT 0,
    b_50ms INTEGER DEFAULT 0,
    b_100ms INTEGER DEFAULT 0,
    b_200ms INTEGER DEFAULT 0,
    b_500ms INTEGER DEFAULT 0,
    b_1s INTEGER DEFAULT 0,
    b_2s INTEGER DEFAULT 0,
    b_5s INTEGER DEFAULT 0,
    b_10s INTEGER DEFAULT 0,
    b_inf INTEGER DEFAULT 0,
    sample_count INTEGER DEFAULT 0,
    created_at TEXT NOT NULL,
    UNIQUE(cluster_id, target_ns, target_name, hour_start)
);
CREATE INDEX IF NOT EXISTS idx_slo_golden_hourly_time ON slo_golden_hourly(hour_start);
CREATE INDEX IF NOT EXISTS idx_slo_golden_hourly_svc ON slo_golden_hourly(cluster_id, target_ns, target_name, hour_start);

-- ==================== 7. Traefik 入口指标 ====================
CREATE TABLE IF NOT EXISTS slo_traefik_raw (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    entrypoint TEXT NOT NULL,
    router TEXT,
    service TEXT,
    method TEXT,
    code TEXT,
    total_req INTEGER DEFAULT 0,
    b_5ms INTEGER DEFAULT 0,
    b_10ms INTEGER DEFAULT 0,
    b_25ms INTEGER DEFAULT 0,
    b_50ms INTEGER DEFAULT 0,
    b_100ms INTEGER DEFAULT 0,
    b_250ms INTEGER DEFAULT 0,
    b_500ms INTEGER DEFAULT 0,
    b_1s INTEGER DEFAULT 0,
    b_2500ms INTEGER DEFAULT 0,
    b_5s INTEGER DEFAULT 0,
    b_10s INTEGER DEFAULT 0,
    b_inf INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_slo_traefik_raw_ts ON slo_traefik_raw(cluster_id, service, timestamp);
CREATE INDEX IF NOT EXISTS idx_slo_traefik_raw_time ON slo_traefik_raw(timestamp);

-- ==================== 8. Trace 主表 ====================
CREATE TABLE IF NOT EXISTS slo_traces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    root_service TEXT,
    root_operation TEXT,
    start_time TEXT NOT NULL,
    duration_us INTEGER NOT NULL,
    span_count INTEGER NOT NULL,
    status TEXT DEFAULT 'ok',
    http_method TEXT,
    http_path TEXT,
    http_status INTEGER,
    created_at TEXT NOT NULL,
    UNIQUE(cluster_id, trace_id)
);
CREATE INDEX IF NOT EXISTS idx_slo_traces_time ON slo_traces(cluster_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_slo_traces_svc ON slo_traces(cluster_id, root_service, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_slo_traces_status ON slo_traces(cluster_id, status, start_time DESC);

-- ==================== 9. Span 表 ====================
CREATE TABLE IF NOT EXISTS slo_spans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    parent_span_id TEXT,
    service_name TEXT NOT NULL,
    operation_name TEXT NOT NULL,
    start_time TEXT NOT NULL,
    duration_us INTEGER NOT NULL,
    status TEXT DEFAULT 'ok',
    kind TEXT,
    attributes_json TEXT,
    UNIQUE(cluster_id, trace_id, span_id)
);
CREATE INDEX IF NOT EXISTS idx_slo_spans_trace ON slo_spans(cluster_id, trace_id);

-- ==================== 10. SLO 目标配置 ====================
CREATE TABLE IF NOT EXISTS slo_targets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    namespace TEXT NOT NULL,
    service_name TEXT NOT NULL,
    time_range TEXT NOT NULL,              -- 1d/7d/30d
    availability_target REAL DEFAULT 99.0,
    p95_latency_target INTEGER DEFAULT 500,
    error_rate_target REAL DEFAULT 1.0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(cluster_id, namespace, service_name, time_range)
);
CREATE INDEX IF NOT EXISTS idx_slo_targets_cluster ON slo_targets(cluster_id);
CREATE INDEX IF NOT EXISTS idx_slo_targets_svc ON slo_targets(cluster_id, namespace, service_name);

-- ==================== 11. SLO 状态历史 ====================
CREATE TABLE IF NOT EXISTS slo_status_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    namespace TEXT NOT NULL,
    service_name TEXT NOT NULL,
    time_range TEXT NOT NULL,
    old_status TEXT NOT NULL,
    new_status TEXT NOT NULL,
    availability REAL NOT NULL,
    p95_latency INTEGER NOT NULL,
    error_rate REAL NOT NULL,
    error_budget_remaining REAL NOT NULL,
    changed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_slo_history_svc ON slo_status_history(cluster_id, namespace, service_name);
CREATE INDEX IF NOT EXISTS idx_slo_history_time ON slo_status_history(changed_at DESC);
```

---

## 3. 架构设计

### 3.1 模块架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Master                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         SLO Module                                  │   │
│  │                                                                     │   │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │   │
│  │   │   DataReceiver  │  │   Aggregator    │  │   Evaluator     │   │   │
│  │   │                 │  │                 │  │                 │   │   │
│  │   │  接收 Agent     │  │  定时聚合      │  │  SLO 状态评估  │   │   │
│  │   │  上报数据       │  │  原始→小时     │  │  错误预算计算  │   │   │
│  │   └────────┬────────┘  └────────┬────────┘  └────────┬────────┘   │   │
│  │            │                    │                    │             │   │
│  │            ▼                    ▼                    ▼             │   │
│  │   ┌───────────────────────────────────────────────────────────┐   │   │
│  │   │                      Repository                           │   │   │
│  │   │                                                           │   │   │
│  │   │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐    │   │   │
│  │   │  │ Services │ │  Edges   │ │  Golden  │ │ Traefik  │    │   │   │
│  │   │  │   Repo   │ │   Repo   │ │   Repo   │ │   Repo   │    │   │   │
│  │   │  └──────────┘ └──────────┘ └──────────┘ └──────────┘    │   │   │
│  │   │  ┌──────────┐ ┌──────────┐ ┌──────────┐                 │   │   │
│  │   │  │  Traces  │ │ Targets  │ │  Status  │                 │   │   │
│  │   │  │   Repo   │ │   Repo   │ │   Repo   │                 │   │   │
│  │   │  └──────────┘ └──────────┘ └──────────┘                 │   │   │
│  │   └───────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         API Layer                                   │   │
│  │                                                                     │   │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │   │
│  │   │  /api/v2/slo/   │  │ /api/v2/slo/    │  │ /api/v2/slo/    │   │   │
│  │   │   services      │  │  topology       │  │  metrics        │   │   │
│  │   └─────────────────┘  └─────────────────┘  └─────────────────┘   │   │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │   │
│  │   │  /api/v2/slo/   │  │ /api/v2/slo/    │  │ /api/v2/slo/    │   │   │
│  │   │   traces        │  │  targets        │  │  status         │   │   │
│  │   └─────────────────┘  └─────────────────┘  └─────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│                                    │                                        │
│                                    ▼                                        │
│                               SQLite DB                                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 模块职责

| 模块 | 职责 |
|------|------|
| `DataReceiver` | 接收 Agent gRPC 上报的 SLO 数据 |
| `Aggregator` | 定时将原始数据聚合为小时数据 |
| `Evaluator` | 评估 SLO 状态，计算错误预算 |
| `Repository` | 数据访问层，封装 SQL 操作 |
| `API Layer` | HTTP API 服务，响应前端请求 |

---

## 4. 数据接收

### 4.1 gRPC 处理

```go
// SLO 数据接收处理
func (s *SLOService) HandleSLOData(ctx context.Context, data *pb.SLOData) error {
    clusterId := data.ClusterId
    ts := time.Unix(data.Timestamp, 0)

    // 1. 更新服务列表
    if len(data.Services) > 0 {
        if err := s.repo.UpsertServices(clusterId, data.Services, ts); err != nil {
            log.Printf("[SLO] 更新服务失败: %v", err)
        }
    }

    // 2. 更新拓扑边
    if len(data.Edges) > 0 {
        if err := s.repo.UpsertEdges(clusterId, data.Edges, ts); err != nil {
            log.Printf("[SLO] 更新拓扑失败: %v", err)
        }
    }

    // 3. 存储黄金指标
    if len(data.GoldenMetrics) > 0 {
        if err := s.repo.InsertGoldenRaw(clusterId, data.GoldenMetrics, ts); err != nil {
            log.Printf("[SLO] 存储黄金指标失败: %v", err)
        }
    }

    // 4. 存储 Traefik 指标
    if len(data.TraefikMetrics) > 0 {
        if err := s.repo.InsertTraefikRaw(clusterId, data.TraefikMetrics, ts); err != nil {
            log.Printf("[SLO] 存储 Traefik 指标失败: %v", err)
        }
    }

    return nil
}
```

### 4.2 服务 Upsert

```go
// 更新服务列表（存在则更新 last_seen_at，不存在则插入）
func (r *ServicesRepo) UpsertServices(clusterId string, services []*pb.ServiceInfo, ts time.Time) error {
    now := ts.Format(time.RFC3339)

    stmt, err := r.db.Prepare(`
        INSERT INTO slo_services (cluster_id, namespace, name, service_type, discovered_at, last_seen_at)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(cluster_id, namespace, name) DO UPDATE SET
            service_type = excluded.service_type,
            last_seen_at = excluded.last_seen_at
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, svc := range services {
        _, err := stmt.Exec(clusterId, svc.Namespace, svc.Name, svc.ServiceType, now, now)
        if err != nil {
            log.Printf("[SLO] Upsert service %s/%s 失败: %v", svc.Namespace, svc.Name, err)
        }
    }

    return nil
}
```

---

## 5. 数据聚合

### 5.1 聚合任务

```go
// 聚合调度器
type Aggregator struct {
    repo     *Repository
    interval time.Duration
}

func (a *Aggregator) Start() {
    // 每小时执行一次聚合
    ticker := time.NewTicker(a.interval)
    go func() {
        for range ticker.C {
            a.aggregateHourly()
            a.cleanup()
        }
    }()

    // 启动时立即执行一次
    go a.aggregateHourly()
}

// 小时聚合
func (a *Aggregator) aggregateHourly() {
    // 聚合上一个完整小时的数据
    hourStart := time.Now().Truncate(time.Hour).Add(-time.Hour)
    hourEnd := hourStart.Add(time.Hour)

    log.Printf("[SLO] 开始聚合 %s ~ %s", hourStart, hourEnd)

    // 按服务分组聚合
    services, err := a.repo.GetDistinctServicesInRange(hourStart, hourEnd)
    if err != nil {
        log.Printf("[SLO] 获取服务列表失败: %v", err)
        return
    }

    for _, svc := range services {
        if err := a.aggregateServiceHour(svc.ClusterId, svc.TargetNs, svc.TargetName, hourStart, hourEnd); err != nil {
            log.Printf("[SLO] 聚合 %s/%s 失败: %v", svc.TargetNs, svc.TargetName, err)
        }
    }

    log.Printf("[SLO] 聚合完成，处理 %d 个服务", len(services))
}
```

### 5.2 服务小时聚合

```go
func (a *Aggregator) aggregateServiceHour(clusterId, ns, name string, hourStart, hourEnd time.Time) error {
    // 查询该小时的原始数据
    raws, err := a.repo.GetGoldenRawInRange(clusterId, ns, name, hourStart, hourEnd)
    if err != nil {
        return err
    }

    if len(raws) == 0 {
        return nil
    }

    // 聚合计算
    hourly := &GoldenHourly{
        ClusterId:   clusterId,
        HourStart:   hourStart,
        TargetNs:    ns,
        TargetName:  name,
        SampleCount: len(raws),
    }

    // 累加
    for _, r := range raws {
        hourly.TotalReq += r.TotalReq
        hourly.SuccessReq += r.SuccessReq
        hourly.ErrorReq += r.ErrorReq

        // 累加 buckets
        hourly.B1ms += r.B1ms
        hourly.B2ms += r.B2ms
        // ... 其他 buckets
        hourly.BInf += r.BInf
    }

    // 计算可用性和错误率
    if hourly.TotalReq > 0 {
        hourly.Availability = float64(hourly.SuccessReq) / float64(hourly.TotalReq) * 100
        hourly.ErrorRate = float64(hourly.ErrorReq) / float64(hourly.TotalReq) * 100
    }

    // 计算平均 RPS
    hourly.AvgRPS = float64(hourly.TotalReq) / 3600.0

    // 计算延迟分位数
    hourly.P50ms = calculatePercentile(hourly, 50)
    hourly.P95ms = calculatePercentile(hourly, 95)
    hourly.P99ms = calculatePercentile(hourly, 99)

    // 写入聚合表
    return a.repo.UpsertGoldenHourly(hourly)
}
```

### 5.3 分位数计算

```go
// 从 Histogram buckets 计算分位数
func calculatePercentile(h *GoldenHourly, percentile int) int {
    total := h.B1ms + h.B2ms + h.B3ms + h.B4ms + h.B5ms +
             h.B10ms + h.B20ms + h.B50ms + h.B100ms + h.B200ms +
             h.B500ms + h.B1s + h.B2s + h.B5s + h.B10s + h.BInf

    if total == 0 {
        return 0
    }

    target := int64(float64(total) * float64(percentile) / 100.0)
    cumulative := int64(0)

    buckets := []struct {
        count int64
        le    int // 毫秒
    }{
        {h.B1ms, 1},
        {h.B2ms, 2},
        {h.B3ms, 3},
        {h.B4ms, 4},
        {h.B5ms, 5},
        {h.B10ms, 10},
        {h.B20ms, 20},
        {h.B50ms, 50},
        {h.B100ms, 100},
        {h.B200ms, 200},
        {h.B500ms, 500},
        {h.B1s, 1000},
        {h.B2s, 2000},
        {h.B5s, 5000},
        {h.B10s, 10000},
        {h.BInf, 100000}, // 假设 inf 为 100s
    }

    for _, b := range buckets {
        cumulative += b.count
        if cumulative >= target {
            return b.le
        }
    }

    return 100000 // 超出范围
}
```

### 5.4 数据清理

```go
func (a *Aggregator) cleanup() {
    now := time.Now()

    // 清理 48h 前的原始数据
    rawCutoff := now.Add(-48 * time.Hour)
    if n, err := a.repo.DeleteGoldenRawBefore(rawCutoff); err != nil {
        log.Printf("[SLO] 清理原始数据失败: %v", err)
    } else if n > 0 {
        log.Printf("[SLO] 清理 %d 条原始数据", n)
    }

    // 清理 90d 前的聚合数据
    hourlyCutoff := now.Add(-90 * 24 * time.Hour)
    if n, err := a.repo.DeleteGoldenHourlyBefore(hourlyCutoff); err != nil {
        log.Printf("[SLO] 清理聚合数据失败: %v", err)
    } else if n > 0 {
        log.Printf("[SLO] 清理 %d 条聚合数据", n)
    }

    // 清理 180d 前的状态历史
    historyCutoff := now.Add(-180 * 24 * time.Hour)
    if n, err := a.repo.DeleteStatusHistoryBefore(historyCutoff); err != nil {
        log.Printf("[SLO] 清理状态历史失败: %v", err)
    } else if n > 0 {
        log.Printf("[SLO] 清理 %d 条状态历史", n)
    }

    // 清理 7d 前的 Trace 数据
    traceCutoff := now.Add(-7 * 24 * time.Hour)
    if n, err := a.repo.DeleteTracesBefore(traceCutoff); err != nil {
        log.Printf("[SLO] 清理 Trace 数据失败: %v", err)
    } else if n > 0 {
        log.Printf("[SLO] 清理 %d 条 Trace", n)
    }
}
```

---

## 6. SLO 评估

### 6.1 评估器

```go
// SLO 状态评估
type Evaluator struct {
    repo *Repository
}

// 评估服务 SLO 状态
func (e *Evaluator) Evaluate(clusterId, ns, name, timeRange string) (*SLOStatus, error) {
    // 获取目标配置
    target, err := e.repo.GetSLOTarget(clusterId, ns, name, timeRange)
    if err != nil {
        return nil, err
    }

    // 获取时间范围
    var duration time.Duration
    switch timeRange {
    case "1d":
        duration = 24 * time.Hour
    case "7d":
        duration = 7 * 24 * time.Hour
    case "30d":
        duration = 30 * 24 * time.Hour
    default:
        duration = 24 * time.Hour
    }

    endTime := time.Now()
    startTime := endTime.Add(-duration)

    // 查询聚合数据
    hourlyData, err := e.repo.GetGoldenHourlyInRange(clusterId, ns, name, startTime, endTime)
    if err != nil {
        return nil, err
    }

    // 计算汇总指标
    status := e.calculateStatus(hourlyData, target)

    return status, nil
}

func (e *Evaluator) calculateStatus(data []*GoldenHourly, target *SLOTarget) *SLOStatus {
    if len(data) == 0 {
        return &SLOStatus{Status: "unknown"}
    }

    var totalReq, successReq, errorReq int64
    var p95Sum, sampleCount int64

    for _, h := range data {
        totalReq += h.TotalReq
        successReq += h.SuccessReq
        errorReq += h.ErrorReq
        p95Sum += int64(h.P95ms)
        sampleCount++
    }

    status := &SLOStatus{
        TotalRequests: totalReq,
    }

    // 计算可用性
    if totalReq > 0 {
        status.Availability = float64(successReq) / float64(totalReq) * 100
        status.ErrorRate = float64(errorReq) / float64(totalReq) * 100
    }

    // 计算平均 P95
    if sampleCount > 0 {
        status.P95Latency = int(p95Sum / sampleCount)
    }

    // 计算错误预算
    // 错误预算 = (目标可用性 - 100%) 的允许范围
    // 剩余错误预算 = 1 - (实际错误率 / 允许错误率)
    allowedErrorRate := 100 - target.AvailabilityTarget
    if allowedErrorRate > 0 {
        usedBudget := status.ErrorRate / allowedErrorRate
        status.ErrorBudgetRemaining = (1 - usedBudget) * 100
        if status.ErrorBudgetRemaining < 0 {
            status.ErrorBudgetRemaining = 0
        }
    }

    // 判断状态
    status.Status = e.determineStatus(status, target)

    return status
}

func (e *Evaluator) determineStatus(status *SLOStatus, target *SLOTarget) string {
    // 状态判断逻辑
    // healthy: 所有指标都在目标内，错误预算 > 50%
    // warning: 接近目标边界，错误预算 20-50%
    // critical: 超出目标，错误预算 < 20%

    if status.Availability < target.AvailabilityTarget ||
       status.P95Latency > target.P95LatencyTarget ||
       status.ErrorRate > target.ErrorRateTarget {
        return "critical"
    }

    if status.ErrorBudgetRemaining < 50 {
        return "warning"
    }

    return "healthy"
}
```

---

## 7. API 设计

### 7.1 API 列表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v2/slo/services` | 获取服务列表 |
| GET | `/api/v2/slo/topology` | 获取服务拓扑 |
| GET | `/api/v2/slo/services/{ns}/{name}/metrics` | 获取服务指标 |
| GET | `/api/v2/slo/services/{ns}/{name}/latency-distribution` | 获取延迟分布 |
| GET | `/api/v2/slo/traces` | 获取 Trace 列表 |
| GET | `/api/v2/slo/traces/{traceId}` | 获取 Trace 详情 |
| GET | `/api/v2/slo/targets` | 获取 SLO 目标 |
| PUT | `/api/v2/slo/targets` | 更新 SLO 目标 |
| GET | `/api/v2/slo/status/{ns}/{name}` | 获取 SLO 状态 |

### 7.2 API 详细设计

#### 7.2.1 获取服务列表

```
GET /api/v2/slo/services?cluster_id={clusterId}

Response:
{
  "services": [
    {
      "namespace": "atlantis",
      "name": "atlantis",
      "display_name": "Atlantis",
      "service_type": "service",
      "last_seen_at": "2026-02-06T10:00:00Z",
      "status": {
        "availability": 99.95,
        "p95_latency": 125,
        "error_rate": 0.05,
        "status": "healthy"
      }
    }
  ]
}
```

#### 7.2.2 获取服务拓扑

```
GET /api/v2/slo/topology?cluster_id={clusterId}&time_range=15m

Response:
{
  "nodes": [
    {
      "id": "kube-system/traefik",
      "namespace": "kube-system",
      "name": "traefik",
      "service_type": "gateway",
      "metrics": {
        "rps": 1234,
        "success_rate": 99.9,
        "p95_latency": 45
      }
    }
  ],
  "edges": [
    {
      "source": "kube-system/traefik",
      "target": "atlantis/atlantis",
      "metrics": {
        "rps": 500,
        "success_rate": 99.8,
        "p95_latency": 32
      }
    }
  ]
}
```

#### 7.2.3 获取延迟分布

```
GET /api/v2/slo/services/{ns}/{name}/latency-distribution?time_range=15m

Response:
{
  "service": "atlantis/atlantis",
  "time_range": "15m",
  "total_requests": 12345,
  "p50": 25,
  "p95": 125,
  "p99": 280,
  "distribution": [
    {"le": 1, "count": 100},
    {"le": 2, "count": 250},
    {"le": 5, "count": 1200},
    {"le": 10, "count": 3500},
    {"le": 20, "count": 4200},
    {"le": 50, "count": 2100},
    {"le": 100, "count": 800},
    {"le": 200, "count": 150},
    {"le": 500, "count": 40},
    {"le": 1000, "count": 5}
  ]
}
```

#### 7.2.4 获取 Trace 列表

```
GET /api/v2/slo/traces?cluster_id={clusterId}&service={ns/name}&time_range=15m&limit=50

Response:
{
  "traces": [
    {
      "trace_id": "abc123...",
      "root_service": "traefik",
      "root_operation": "HTTP GET /api/users",
      "start_time": "2026-02-06T10:00:00Z",
      "duration_us": 125000,
      "span_count": 7,
      "status": "ok",
      "http_method": "GET",
      "http_path": "/api/users",
      "http_status": 200
    }
  ],
  "total": 1234
}
```

#### 7.2.5 获取 Trace 详情

```
GET /api/v2/slo/traces/{traceId}

Response:
{
  "trace_id": "abc123...",
  "root_service": "traefik",
  "start_time": "2026-02-06T10:00:00Z",
  "duration_us": 125000,
  "spans": [
    {
      "span_id": "span1",
      "parent_span_id": null,
      "service_name": "traefik",
      "operation_name": "HTTP GET /api/users",
      "start_time": "2026-02-06T10:00:00.000Z",
      "duration_us": 125000,
      "status": "ok",
      "kind": "server"
    },
    {
      "span_id": "span2",
      "parent_span_id": "span1",
      "service_name": "atlantis",
      "operation_name": "handleGetUsers",
      "start_time": "2026-02-06T10:00:00.002Z",
      "duration_us": 120000,
      "status": "ok",
      "kind": "server"
    }
  ]
}
```

---

## 8. 配置设计

### 8.1 Master 配置

```yaml
# config.yaml
slo:
  enabled: true

  # 数据库
  database:
    path: "./data/slo.db"

  # 聚合配置
  aggregation:
    # 聚合间隔（每小时）
    interval: 1h

    # 原始数据保留时间
    raw_retention: 48h

    # 聚合数据保留时间
    hourly_retention: 90d

    # 状态历史保留时间
    history_retention: 180d

    # Trace 保留时间
    trace_retention: 7d

  # 评估配置
  evaluation:
    # 评估间隔
    interval: 1m

    # 默认 SLO 目标
    default_targets:
      availability: 99.0
      p95_latency: 500
      error_rate: 1.0
```

---

## 9. 迁移计划

### 9.1 从旧表迁移

```sql
-- 旧表数据迁移脚本

-- 1. 迁移 slo_route_mapping → slo_ingress_routes
INSERT INTO slo_ingress_routes (cluster_id, domain, path_prefix, ingress_name, namespace, tls, backend_ns, backend_svc, backend_port, created_at, updated_at)
SELECT
    cluster_id,
    domain,
    path_prefix,
    ingress_name,
    namespace,
    tls,
    namespace AS backend_ns,
    service_name AS backend_svc,
    service_port AS backend_port,
    created_at,
    updated_at
FROM slo_route_mapping;

-- 2. 迁移 slo_targets (旧) → slo_targets (新)
-- 需要将 host 维度转换为 service 维度
INSERT INTO slo_targets (cluster_id, namespace, service_name, time_range, availability_target, p95_latency_target, created_at, updated_at)
SELECT DISTINCT
    t.cluster_id,
    r.namespace,
    r.service_name AS service_name,
    t.time_range,
    t.availability_target,
    t.p95_latency_target,
    t.created_at,
    t.updated_at
FROM slo_targets_old t
JOIN slo_route_mapping r ON t.cluster_id = r.cluster_id AND t.host = r.domain;
```

### 9.2 清理旧表

```sql
-- 数据迁移验证后执行
DROP TABLE IF EXISTS ingress_counter_snapshot;
DROP TABLE IF EXISTS ingress_histogram_snapshot;
DROP TABLE IF EXISTS slo_metrics_raw;
DROP TABLE IF EXISTS slo_metrics_hourly;
DROP TABLE IF EXISTS slo_route_mapping;
DROP TABLE IF EXISTS slo_targets_old;
DROP TABLE IF EXISTS slo_status_history_old;
```

---

## 10. 实现计划

### 10.1 阶段一：数据库重构

- [ ] 创建新表结构
- [ ] 实现 Repository 层
- [ ] 数据迁移脚本

### 10.2 阶段二：数据接收

- [ ] 扩展 gRPC 协议支持 SLO 数据
- [ ] 实现 DataReceiver
- [ ] 服务发现和拓扑更新

### 10.3 阶段三：聚合与评估

- [ ] 实现 Aggregator
- [ ] 实现 Evaluator
- [ ] 定时任务调度

### 10.4 阶段四：API 服务

- [ ] 实现服务列表 API
- [ ] 实现拓扑 API
- [ ] 实现指标/延迟分布 API
- [ ] 实现 Trace API
- [ ] 实现 SLO 配置 API

### 10.5 阶段五：清理与优化

- [ ] 数据清理任务
- [ ] 性能优化
- [ ] 单元测试

---

## 11. 附录

### 11.1 SLO 状态定义

| 状态 | 说明 | 条件 |
|------|------|------|
| `healthy` | 健康 | 所有指标达标，错误预算 > 50% |
| `warning` | 警告 | 接近目标边界，错误预算 20-50% |
| `critical` | 严重 | 超出目标，错误预算 < 20% |
| `unknown` | 未知 | 数据不足 |

### 11.2 错误预算计算

```
允许错误率 = 100% - 可用性目标
已用错误率 = 实际错误率
错误预算剩余 = (1 - 已用错误率 / 允许错误率) × 100%

示例:
- 可用性目标: 99.9%
- 允许错误率: 0.1%
- 实际错误率: 0.05%
- 错误预算剩余: (1 - 0.05/0.1) × 100% = 50%
```
