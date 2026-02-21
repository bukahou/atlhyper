# 统一可观测性架构设计 — OTel + ClickHouse

> **文档类型**: 中心架构设计（Central Design Philosophy）
> **影响范围**: Agent V2 / Master V2 / Web 前端 / OTel Collector / 基础设施
> **状态**: 设计中

---

## 目录

1. [背景与动机](#1-背景与动机)
2. [现有架构分析](#2-现有架构分析)
3. [新架构总览](#3-新架构总览)
4. [ClickHouse 表设计](#4-clickhouse-表设计)
5. [OTel Collector 改造](#5-otel-collector-改造)
6. [Agent V2 改造](#6-agent-v2-改造)
7. [Master V2 改造](#7-master-v2-改造)
8. [Web 前端改造](#8-web-前端改造)
9. [数据关联查询模型](#9-数据关联查询模型)
10. [迁移路径](#10-迁移路径)
11. [后续详细设计清单](#11-后续详细设计清单)

---

## 1. 背景与动机

### 1.1 数据孤岛问题

当前 AtlHyper 的可观测性数据分散在多个独立系统中：

| 数据类型 | 当前存储位置 | 问题 |
|---------|-------------|------|
| **Traces** | Jaeger (Badger) | 独立存储，无法与其他信号关联 |
| **硬件指标** | Agent 内存 → Master 内存快照 | 无历史趋势，重启即丢失 |
| **SLO 指标** | Agent delta 计算 → Master SQLite | 仅 raw + hourly 两层聚合，查询灵活性差 |
| **K8s Events** | Agent 快照 → Master 内存 | 无法与 trace/metrics 交叉查询 |
| **K8s 资源状态** | Agent 快照 → Master 内存 | 适合实时展示，无历史回溯 |

**核心矛盾**: 这些数据本质上是强因果关联的（一个请求延迟高 → 对应 trace 的 span 慢 → 对应 Pod 的 CPU 打满 → 对应 Node 的 IO 等待高），但分散的存储使得跨信号关联分析不可能。

### 1.2 现有架构的瓶颈

1. **Agent 做了太多计算**: rate 计算、delta 计算、per-pod 聚合、service 聚合 → Agent 代码膨胀，逻辑复杂
2. **Master 内存快照无历史**: NodeMetrics 和 SLO 数据只有最新一帧，无法查询 "过去 1 小时的 CPU 趋势"
3. **Jaeger 是 trace 专用**: 不支持 metrics/logs，无法 JOIN 查询
4. **数据管线多且碎片化**: OTel → Prometheus :8889 → Agent scrape → delta → Master SQLite，链路过长

### 1.3 目标

将 **ClickHouse** 作为统一的可观测性数据湖，实现：

- **All-in-One 存储**: Traces / Metrics / Logs / K8s Events 统一存入 ClickHouse
- **跨信号关联**: 通过 `trace_id`, `service_name`, `pod_name`, `node_name`, `cluster_id` 等公共维度 SQL JOIN
- **历史回溯**: 所有数据天然保留历史，支持任意时间范围查询
- **计算下沉**: rate/delta/聚合/百分位数 全部由 ClickHouse SQL 完成，Agent 不再做数据加工
- **Agent 瘦身**: Agent 只保留 K8s 状态快照 + 指令执行，删除 ~40-50% 采集代码

---

## 2. 现有架构分析

### 2.1 数据流（现状）

```
┌─────────────────────────── 采集层 ───────────────────────────┐
│                                                               │
│  应用 SDK ──OTLP──> OTel Collector ──OTLP──> Jaeger (Badger) │
│                          │                                    │
│  Linkerd Prometheus ─────┤                                    │
│  Traefik Metrics ────────┤── scrape ──> Prometheus :8889      │
│  node_exporter ──────────┘                                    │
│                                                               │
└───────────────────────────────────────────────────────────────┘
                               │
                          Agent scrape :8889
                               │
                    ┌──────────┴──────────┐
                    │ Agent V2            │
                    │  - rate 计算        │
                    │  - delta 计算       │
                    │  - per-pod 聚合     │
                    │  - K8s API 采集     │
                    └──────────┬──────────┘
                               │
                     ClusterSnapshot (gzip)
                               │
                    ┌──────────┴──────────┐
                    │ Master V2           │
                    │  - 内存快照 Store   │
                    │  - SLO SQLite 持久化│
                    │  - 聚合/查询        │
                    └──────────┬──────────┘
                               │
                          HTTP JSON API
                               │
                    ┌──────────┴──────────┐
                    │ Web 前端            │
                    └─────────────────────┘
```

### 2.2 Agent 组件盘点（当前）

| 组件 | 职责 | 新架构去留 |
|------|------|-----------|
| **K8sClient** | 采集 20 种 K8s 资源 | **保留** — OTel 不提供完整资源状态 |
| **OTelClient** | scrape OTel :8889 的 Prometheus 指标 | **删除** — ClickHouse 直存 |
| **IngressClient** | 采集 IngressRoute CRD 路由映射 | **删除** — 路由信息可在 Master 查 K8s API |
| **ReceiverClient** | HTTP Server 被动接收 node_exporter 推送 | **删除** — node_exporter 走 OTel → ClickHouse |
| **MetricsRepository** | rate 计算 node_exporter → NodeMetricsSnapshot | **删除** — rate 由 ClickHouse SQL 计算 |
| **SLORepository** | delta 计算 + 聚合 Linkerd/Traefik 指标 | **删除** — delta/聚合由 ClickHouse SQL 完成 |
| **21 个 K8s Repo** | K8s 资源 List/Get | **保留** |
| **GenericRepository** | 写操作 (scale/restart/delete) + 动态查询 | **保留** |
| **SnapshotService** | 采集集群快照 (K8s + NodeMetrics + SLO) | **简化** — 只采集 K8s 资源 |
| **CommandService** | 执行 Master 下发指令 | **保留** |
| **MetricsSyncLoop** | 15s 周期 OTel scrape | **删除** |

### 2.3 Master 组件盘点（当前）

| 组件 | 职责 | 新架构影响 |
|------|------|-----------|
| **DataHub (内存)** | 快照存储、Agent 状态、Event 查询 | K8s 快照保留，NodeMetrics/SLO 数据改查 ClickHouse |
| **SLO Processor** | 写入 raw 表 | **删除** — OTel 直写 ClickHouse |
| **SLO Aggregator** | raw → hourly 聚合 | **删除** — ClickHouse Materialized View 替代 |
| **SLO Cleaner** | 过期数据清理 | **删除** — ClickHouse TTL 替代 |
| **SLO Calculator** | 纯函数计算 | 部分保留（查询时计算） |
| **SQLite Database** | SLO raw/hourly 表, Event 持久化, 设置等 | SLO 表删除，设置等业务表保留 |
| **Service Query** | 所有只读查询 | **扩展** — 新增 ClickHouse 查询方法 |

---

## 3. 新架构总览

### 3.1 架构图

```
┌─────────────────────────── 采集层 (OTel 生态) ──────────────────────────┐
│                                                                         │
│  应用 SDK ──OTLP──┐                                                    │
│                    │                                                    │
│  Linkerd ──scrape──┤                                                    │
│  Traefik ──scrape──┼──> OTel Collector ──clickhouse exporter──> ClickHouse │
│  node_exporter ────┤                                                    │
│                    │                                                    │
│  (未来: 应用日志) ─┘                                                    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────── Agent V2 (瘦身后) ───────────┐
│                                          │
│  K8sClient ──> K8s Repos ──> SnapshotSvc │
│                                          │
│  GenericRepo ──> CommandSvc              │
│                                          │
│  调度: SnapshotLoop + CommandLoop        │
│        + HeartbeatLoop                   │
│  (删除: MetricsSyncLoop)                 │
│                                          │
└──────────────┬───────────────────────────┘
               │
     ClusterSnapshot (仅 K8s 资源)
               │
┌──────────────┴───────────────────────────┐
│          Master V2 (增强后)              │
│                                          │
│  DataHub (内存)  ── K8s 快照 + Agent 状态│
│  ClickHouseClient ── 查询可观测性数据    │
│  SQLite ── 业务数据 (设置/用户/审计)     │
│                                          │
│  Service Query:                          │
│    - K8s 资源 → DataHub                  │
│    - Traces/Metrics/SLO → ClickHouse     │
│    - 跨信号关联 → ClickHouse SQL JOIN    │
│                                          │
│  (删除: SLO Processor/Aggregator/Cleaner)│
│                                          │
└──────────────┬───────────────────────────┘
               │
          HTTP JSON API
               │
┌──────────────┴───────────────────────────┐
│          Web 前端 (增强后)               │
│                                          │
│  现有: K8s 资源管理 / Overview           │
│  增强: APM (真实 trace 数据)             │
│  增强: SLO (ClickHouse 历史查询)         │
│  增强: 节点指标 (ClickHouse 历史查询)    │
│  新增: 统一搜索 / 跨信号关联视图        │
│                                          │
└──────────────────────────────────────────┘
```

### 3.2 核心原则

| 原则 | 说明 |
|------|------|
| **采集与存储分离** | OTel Collector 负责采集和路由，ClickHouse 负责存储，互不耦合 |
| **计算下沉** | rate / delta / percentile / 聚合全部由 ClickHouse SQL 在查询时完成 |
| **Agent 最小化** | Agent 只做 OTel 生态无法替代的事：K8s API 全量资源快照 + 指令执行 |
| **公共维度关联** | 所有数据表共享 `cluster_id`, `service_name`, `pod_name`, `node_name`, `timestamp` 等关联键 |
| **查询时聚合** | 不预计算固定粒度聚合，利用 ClickHouse 列式存储的聚合性能，按需查询 |
| **TTL 自动过期** | ClickHouse 表级 TTL 替代手动清理任务 |

### 3.3 数据分层

```
┌─────────────────────────────────────────────────────┐
│ 实时层 (Agent → Master 内存)                         │
│   K8s 资源状态快照 (Pods/Nodes/Deployments/...)     │
│   Agent 心跳和连接状态                               │
│   特点: 最新一帧，重启后由 Agent 重新推送即恢复      │
└─────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────┐
│ 历史层 (OTel → ClickHouse)                           │
│   Traces (spans)                                    │
│   Metrics (node_exporter / Linkerd / Traefik)       │
│   K8s Events (Agent 推送 → Master 写入 ClickHouse)  │
│   (未来: 应用日志)                                   │
│   特点: 时序存储，支持任意时间范围查询               │
└─────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────┐
│ 业务层 (SQLite)                                      │
│   用户设置 / 通知配置 / 审计日志 / AI 对话历史       │
│   特点: 低频读写，关系型数据                         │
└─────────────────────────────────────────────────────┘
```

---

## 4. ClickHouse 表设计

### 4.1 公共维度（所有表共享的关联键）

```sql
-- 每张表都包含以下字段的子集，用于跨表 JOIN
cluster_id    String      -- 多集群隔离
timestamp     DateTime64  -- 纳秒级时间戳
service_name  String      -- 服务名 (namespace/name)
pod_name      String      -- Pod 名称
node_name     String      -- 节点名称
trace_id      String      -- Trace ID (仅 traces/logs)
```

### 4.2 Traces 表 (otel_traces)

使用 [clickhouse-exporter](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/clickhouseexporter) 的标准 schema，与 OTel 生态兼容。

```sql
CREATE TABLE otel_traces (
    Timestamp       DateTime64(9),
    TraceId         String,
    SpanId          String,
    ParentSpanId    String,
    TraceState      String,
    SpanName        LowCardinality(String),
    SpanKind        LowCardinality(String),
    ServiceName     LowCardinality(String),
    Duration        Int64,              -- 纳秒
    StatusCode      LowCardinality(String),
    StatusMessage   String,

    -- 资源属性 (展开为列，便于过滤)
    ResourceAttributes  Map(LowCardinality(String), String),
    -- Span 属性
    SpanAttributes      Map(LowCardinality(String), String),
    -- Events
    Events Nested (
        Timestamp DateTime64(9),
        Name      LowCardinality(String),
        Attributes Map(LowCardinality(String), String)
    ),
    -- Links
    Links Nested (
        TraceId    String,
        SpanId     String,
        TraceState String,
        Attributes Map(LowCardinality(String), String)
    ),

    -- AtlHyper 扩展关联字段 (通过 OTel resource processor 注入)
    cluster_id  LowCardinality(String) DEFAULT ResourceAttributes['cluster.name'],
    pod_name    LowCardinality(String) DEFAULT ResourceAttributes['k8s.pod.name'],
    node_name   LowCardinality(String) DEFAULT ResourceAttributes['k8s.node.name'],

    INDEX idx_trace_id TraceId TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_service ServiceName TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_duration Duration TYPE minmax GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, SpanName, toUnixTimestamp(Timestamp))
TTL toDateTime(Timestamp) + INTERVAL 30 DAY
```

### 4.3 Metrics 表 (otel_metrics)

OTel ClickHouse exporter 自动创建 gauge / sum / histogram 等子表。以下列出主要表结构。

**Gauge 指标** (CPU 使用率、内存使用量、磁盘空间等瞬时值):

```sql
CREATE TABLE otel_metrics_gauge (
    ResourceAttributes  Map(LowCardinality(String), String),
    ResourceSchemaUrl   String,
    ScopeName           String,
    ScopeAttributes     Map(LowCardinality(String), String),
    MetricName          LowCardinality(String),
    AggregationTemporality Int32,
    StartTimeUnix       DateTime64(9),
    TimeUnix            DateTime64(9),
    Value               Float64,
    Attributes          Map(LowCardinality(String), String),
    -- 展开的关联字段
    cluster_id          LowCardinality(String),
    node_name           LowCardinality(String),
    instance            LowCardinality(String)
)
ENGINE = MergeTree()
PARTITION BY toDate(TimeUnix)
ORDER BY (MetricName, Attributes, toUnixTimestamp(TimeUnix))
TTL toDateTime(TimeUnix) + INTERVAL 90 DAY
```

**Sum 指标** (计数器: request_total, bytes_total 等累积值):

```sql
CREATE TABLE otel_metrics_sum (
    -- 同 gauge 基础字段 ...
    MetricName          LowCardinality(String),
    Value               Float64,
    IsMonotonic         Bool,
    TimeUnix            DateTime64(9),
    StartTimeUnix       DateTime64(9),
    Attributes          Map(LowCardinality(String), String),
    ResourceAttributes  Map(LowCardinality(String), String)
)
ENGINE = MergeTree()
PARTITION BY toDate(TimeUnix)
ORDER BY (MetricName, Attributes, toUnixTimestamp(TimeUnix))
TTL toDateTime(TimeUnix) + INTERVAL 90 DAY
```

**Histogram 指标** (延迟分布: response_latency_ms_bucket):

```sql
CREATE TABLE otel_metrics_histogram (
    MetricName          LowCardinality(String),
    TimeUnix            DateTime64(9),
    Count               UInt64,
    Sum                 Float64,
    BucketCounts        Array(UInt64),
    ExplicitBounds      Array(Float64),
    Attributes          Map(LowCardinality(String), String),
    ResourceAttributes  Map(LowCardinality(String), String)
)
ENGINE = MergeTree()
PARTITION BY toDate(TimeUnix)
ORDER BY (MetricName, Attributes, toUnixTimestamp(TimeUnix))
TTL toDateTime(TimeUnix) + INTERVAL 90 DAY
```

### 4.4 K8s Events 表 (k8s_events)

K8s Events 不走 OTel，由 Master 接收 Agent 快照后写入 ClickHouse（增量写入，非覆盖）。

```sql
CREATE TABLE k8s_events (
    cluster_id          LowCardinality(String),
    timestamp           DateTime64(6),      -- Event lastTimestamp
    first_timestamp     DateTime64(6),
    kind                LowCardinality(String),  -- 关联资源类型
    namespace           LowCardinality(String),
    name                String,             -- 关联资源名
    reason              LowCardinality(String),
    message             String,
    type                LowCardinality(String),  -- Normal / Warning
    source_component    LowCardinality(String),
    count               UInt32,
    -- 关联字段
    pod_name            LowCardinality(String),  -- 从 involvedObject 提取
    node_name           LowCardinality(String),

    INDEX idx_reason reason TYPE set(100) GRANULARITY 1,
    INDEX idx_type type TYPE set(10) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (cluster_id, namespace, timestamp)
TTL toDateTime(timestamp) + INTERVAL 30 DAY
```

### 4.5 日志表 (otel_logs) — 未来扩展

```sql
CREATE TABLE otel_logs (
    Timestamp           DateTime64(9),
    TraceId             String,
    SpanId              String,
    SeverityText        LowCardinality(String),
    SeverityNumber      Int32,
    ServiceName         LowCardinality(String),
    Body                String,
    ResourceAttributes  Map(LowCardinality(String), String),
    LogAttributes       Map(LowCardinality(String), String),
    -- 关联字段
    cluster_id          LowCardinality(String),
    pod_name            LowCardinality(String),
    node_name           LowCardinality(String),

    INDEX idx_trace_id TraceId TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_severity SeverityText TYPE set(20) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toDate(Timestamp)
ORDER BY (ServiceName, toUnixTimestamp(Timestamp))
TTL toDateTime(Timestamp) + INTERVAL 30 DAY
```

### 4.6 Materialized View（可选预聚合）

ClickHouse 允许按需创建 Materialized View，在数据写入时自动聚合。如果查询性能不足，可添加：

```sql
-- 示例: 服务级请求速率 5min 聚合
CREATE MATERIALIZED VIEW slo_service_5min
ENGINE = SummingMergeTree()
PARTITION BY toDate(window_start)
ORDER BY (cluster_id, service_name, window_start)
AS SELECT
    ResourceAttributes['cluster.name'] AS cluster_id,
    Attributes['service'] AS service_name,
    toStartOfFiveMinutes(TimeUnix) AS window_start,
    sum(Value) AS total_requests,
    count() AS sample_count
FROM otel_metrics_sum
WHERE MetricName = 'response_total'
GROUP BY cluster_id, service_name, window_start;
```

**原则: 先用原始表 + 查询时聚合，性能不足时再加 Materialized View。**

---

## 5. OTel Collector 改造

### 5.1 新增 ClickHouse Exporter

```yaml
exporters:
  # 删除 Jaeger exporter
  # otlp/jaeger:  ← 删除

  # 新增 ClickHouse exporter (traces + metrics)
  clickhouse:
    endpoint: tcp://clickhouse.atlhyper.svc:9000
    database: atlhyper
    # Traces 表配置
    traces_table_name: otel_traces
    # Metrics 表配置
    metrics_table_name: otel_metrics
    # 日志表配置 (未来启用)
    # logs_table_name: otel_logs
    ttl: 720h   # 30 天
    create_schema: true
    timeout: 5s
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s

  # 保留 Prometheus exporter (供 Grafana 等外部工具使用，可选)
  prometheus:
    endpoint: 0.0.0.0:8889
    namespace: otel
```

### 5.2 新增 Resource Processor（注入关联属性）

```yaml
processors:
  # 保留现有 memory_limiter
  memory_limiter:
    check_interval: 1s
    limit_mib: 900
    spike_limit_mib: 200

  # 增强现有 resource processor — 注入更多关联属性
  resource:
    attributes:
      - key: cluster.name
        value: zgmf-x10a
        action: upsert

  # 新增: K8s 属性处理器 (自动注入 pod/node/namespace)
  k8sattributes:
    auth_type: "serviceAccount"
    passthrough: false
    extract:
      metadata:
        - k8s.pod.name
        - k8s.pod.uid
        - k8s.namespace.name
        - k8s.node.name
        - k8s.deployment.name
    pod_association:
      - sources:
          - from: resource_attribute
            name: k8s.pod.ip
```

### 5.3 更新 Pipeline

```yaml
service:
  pipelines:
    # Traces → ClickHouse (替代 Jaeger)
    traces:
      receivers: [otlp]
      processors: [memory_limiter, k8sattributes, resource]
      exporters: [clickhouse]

    # Metrics → ClickHouse + Prometheus (双写)
    metrics:
      receivers: [otlp, prometheus]
      processors: [memory_limiter, resource]
      exporters: [clickhouse, prometheus]

    # Logs → ClickHouse (未来启用)
    # logs:
    #   receivers: [otlp]
    #   processors: [memory_limiter, k8sattributes, resource]
    #   exporters: [clickhouse]
```

### 5.4 Receiver 配置不变

现有的 `receivers` 部分（OTLP、Prometheus scrape for Linkerd/Traefik/node_exporter）保持不变。
采集源不变，只改变输出目标。

---

## 6. Agent V2 改造

### 6.1 变更总结

```
删除的组件 (~40-50% 代码):
├── sdk/interfaces.go
│   ├── OTelClient          ← 删除 (不再 scrape OTel :8889)
│   ├── IngressClient       ← 删除 (路由信息改由 Master 查询)
│   └── ReceiverClient      ← 删除 (不再被动接收 metrics)
├── sdk/impl/
│   ├── ingress/            ← 整目录删除
│   └── receiver/           ← 整目录删除
├── repository/
│   ├── metrics/            ← 整目录删除 (rate 计算交给 ClickHouse)
│   └── slo/                ← 整目录删除 (delta/聚合交给 ClickHouse)
└── scheduler/
    └── runMetricsSyncLoop  ← 删除

保留的组件:
├── sdk/interfaces.go
│   └── K8sClient           ← 保留 (K8s API 采集)
├── sdk/impl/k8s/           ← 保留
├── repository/
│   ├── k8s/ (21 个 repo)   ← 保留
│   └── GenericRepository   ← 保留
├── service/
│   ├── SnapshotService     ← 简化 (仅 K8s 资源，不含 metrics/SLO)
│   └── CommandService      ← 保留
└── scheduler/
    ├── SnapshotLoop        ← 保留
    ├── CommandLoop (ops)   ← 保留
    ├── CommandLoop (ai)    ← 保留
    └── HeartbeatLoop       ← 保留
```

### 6.2 ClusterSnapshot 瘦身

```go
// model_v2/snapshot.go — 改造后

type ClusterSnapshot struct {
    ClusterID string    `json:"cluster_id"`
    FetchedAt time.Time `json:"fetched_at"`

    // K8s 资源 (保留)
    Pods         []Pod         `json:"pods"`
    Deployments  []Deployment  `json:"deployments"`
    // ... 其他 K8s 资源不变 ...
    Nodes  []Node  `json:"nodes"`
    Events []Event `json:"events"`

    // 删除以下字段:
    // NodeMetrics map[string]*NodeMetricsSnapshot  ← 删除 (查 ClickHouse)
    // SLOData     *SLOSnapshot                     ← 删除 (查 ClickHouse)

    Summary ClusterSummary `json:"summary"`
}
```

### 6.3 Scheduler 简化

```go
// scheduler.go — 改造后

func (s *Scheduler) Start(ctx context.Context) error {
    s.ctx, s.cancel = context.WithCancel(ctx)

    s.wg.Add(4)
    go s.runSnapshotLoop()     // K8s 资源快照 (不含 metrics/SLO)
    go s.runCommandLoop("ops") // 系统操作指令
    go s.runCommandLoop("ai")  // AI 查询指令
    go s.runHeartbeatLoop()    // 心跳

    // 删除: runMetricsSyncLoop (不再需要)

    return nil
}
```

### 6.4 agent.go 初始化简化

```go
// agent.go — 改造后的初始化

func New(cfg *config.Config) (*Agent, error) {
    // 1. SDK 层 — 仅保留 K8sClient
    k8sClient, err := k8simpl.NewK8sClient(kubeConfig)
    // 删除: otelClient, ingressClient, receiverClient

    // 2. Gateway
    masterGw := gateway.NewMasterGateway(...)

    // 3. Repository — 仅保留 K8s repos + GenericRepo
    podRepo := k8srepo.NewPodRepository(k8sClient)
    nodeRepo := k8srepo.NewNodeRepository(k8sClient)
    // ... 其他 K8s repos ...
    genericRepo := k8srepo.NewGenericRepository(k8sClient)
    // 删除: metricsRepo, sloRepo

    // 4. Service
    snapshotSvc := snapshotsvc.NewSnapshotService(/* 仅 K8s repos */)
    commandSvc := commandsvc.NewCommandService(podRepo, genericRepo)

    // 5. Scheduler — 不传 metricsRepo
    sched := scheduler.New(config, snapshotSvc, commandSvc, masterGw, nil)

    return &Agent{...}, nil
}
```

---

## 7. Master V2 改造

### 7.1 新增 ClickHouse 客户端层

```
atlhyper_master_v2/
├── clickhouse/                    ← 新增
│   ├── interfaces.go              ←   ClickHouseClient 接口
│   ├── client.go                  ←   连接管理
│   ├── traces.go                  ←   Trace 查询方法
│   ├── metrics.go                 ←   Metrics 查询方法
│   └── events.go                  ←   Events 写入/查询
├── service/
│   ├── query/
│   │   ├── k8s.go                 ←   保留 (查 DataHub)
│   │   ├── traces.go              ←   新增 (查 ClickHouse)
│   │   ├── metrics.go             ←   新增 (查 ClickHouse)
│   │   └── slo.go                 ←   重写 (查 ClickHouse 替代 SQLite)
│   └── ...
├── slo/                           ←   大幅简化或删除
│   ├── processor.go               ←   删除 (OTel 直写 ClickHouse)
│   ├── aggregator.go              ←   删除 (Materialized View 替代)
│   └── cleaner.go                 ←   删除 (TTL 替代)
```

### 7.2 ClickHouseClient 接口

```go
// clickhouse/interfaces.go

type ClickHouseClient interface {
    // ==================== Traces ====================

    // QueryTraces 查询 trace 列表
    QueryTraces(ctx context.Context, opts TraceQueryOpts) ([]TraceSummary, int, error)
    // GetTraceDetail 获取单个 trace 的所有 spans
    GetTraceDetail(ctx context.Context, traceID string) (*TraceDetail, error)
    // GetServiceList 获取服务列表及统计
    GetServiceList(ctx context.Context, clusterID string, timeRange TimeRange) ([]ServiceInfo, error)
    // GetServiceTopology 获取服务调用拓扑
    GetServiceTopology(ctx context.Context, clusterID string, timeRange TimeRange) (*TopologyData, error)

    // ==================== Metrics ====================

    // QueryNodeMetrics 查询节点指标 (rate 在 SQL 中计算)
    QueryNodeMetrics(ctx context.Context, clusterID string, opts MetricsQueryOpts) (map[string]*NodeMetricsResult, error)
    // QueryNodeMetricsHistory 查询节点指标历史趋势
    QueryNodeMetricsHistory(ctx context.Context, nodeName string, timeRange TimeRange, interval string) ([]MetricsDataPoint, error)

    // ==================== SLO ====================

    // QueryServiceSLO 查询服务 SLO 指标 (替代 SQLite 的 raw/hourly)
    QueryServiceSLO(ctx context.Context, clusterID string, opts SLOQueryOpts) (*SLOResult, error)
    // QueryMeshTopology 查询服务网格拓扑
    QueryMeshTopology(ctx context.Context, clusterID string, timeRange TimeRange) (*MeshTopologyResult, error)
    // QueryServiceDetail 查询单个服务的 SLO 详情
    QueryServiceDetail(ctx context.Context, clusterID, namespace, name string, timeRange TimeRange) (*ServiceDetailResult, error)

    // ==================== Events ====================

    // WriteEvents 写入 K8s Events (来自 Agent 快照)
    WriteEvents(ctx context.Context, clusterID string, events []model_v2.Event) error
    // QueryEvents 查询 K8s Events
    QueryEvents(ctx context.Context, clusterID string, opts EventQueryOpts) ([]EventResult, error)

    // ==================== 跨信号关联 ====================

    // CorrelateTraceToMetrics 通过 trace_id 关联到对应时间窗口的 pod/node 指标
    CorrelateTraceToMetrics(ctx context.Context, traceID string) (*CorrelationResult, error)

    // ==================== 生命周期 ====================
    Ping(ctx context.Context) error
    Close() error
}
```

### 7.3 Service Query 扩展

```go
// service/interfaces.go — 新增方法

type Query interface {
    // ... 保留现有 K8s 查询方法 ...

    // ==================== APM (新增, 替代 mock) ====================

    GetTraceServices(ctx context.Context, clusterID string, timeRange string) ([]model.ServiceInfo, error)
    QueryTraces(ctx context.Context, clusterID string, opts model.TraceQueryOpts) ([]model.TraceSummary, int, error)
    GetTraceDetail(ctx context.Context, traceID string) (*model.TraceDetail, error)
    GetServiceTopology(ctx context.Context, clusterID string, timeRange string) (*model.TopologyData, error)

    // ==================== 节点指标 (新增, 替代内存快照) ====================

    // GetNodeMetricsCurrent 获取节点当前指标 (最近 1 分钟)
    GetNodeMetricsCurrent(ctx context.Context, clusterID string) (map[string]*model.NodeMetricsResult, error)
    // GetNodeMetricsHistory 获取节点指标历史 (任意时间范围)
    GetNodeMetricsHistory(ctx context.Context, clusterID, nodeName, timeRange string) ([]model.MetricsPoint, error)

    // ==================== SLO (重写, 查 ClickHouse) ====================

    // GetMeshTopology — 签名保持，内部改查 ClickHouse
    // GetServiceDetail — 签名保持，内部改查 ClickHouse

    // ==================== 跨信号关联 (新增) ====================

    GetTraceCorrelation(ctx context.Context, traceID string) (*model.CorrelationResult, error)
}
```

### 7.4 删除的组件

```
删除:
├── slo/processor.go       → OTel 直写 ClickHouse，Master 不参与写入
├── slo/aggregator.go      → ClickHouse Materialized View 替代
├── slo/cleaner.go         → ClickHouse TTL 替代
├── database/sqlite/ 中:
│   ├── slo_raw 表操作     → ClickHouse
│   └── slo_hourly 表操作  → ClickHouse
└── processor/ 中:
    └── SLO 相关处理逻辑   → 不再需要
```

### 7.5 DataHub 简化

DataHub (内存 Store) 保留，但职责收窄为：
- **K8s 快照存储** — SetSnapshot / GetSnapshot
- **Agent 状态** — UpdateHeartbeat / GetAgentStatus / ListAgents

删除 DataHub 中的 Event 查询（改查 ClickHouse）。

### 7.6 Processor 改造

Processor 保留对 K8s 快照的处理，新增：
- 从快照中提取 K8s Events，增量写入 ClickHouse
- 不再处理 SLO 数据（SLO 从 OTel → ClickHouse 自动入库）

```go
// processor — 改造后

type Processor struct {
    store      datahub.Store
    chClient   clickhouse.ClickHouseClient  // 新增
}

func (p *Processor) ProcessSnapshot(ctx context.Context, clusterID string, snapshot *model_v2.ClusterSnapshot) error {
    // 1. 写入内存 Store (保留)
    p.store.SetSnapshot(clusterID, snapshot)

    // 2. 增量写入 K8s Events 到 ClickHouse (新增)
    if len(snapshot.Events) > 0 {
        p.chClient.WriteEvents(ctx, clusterID, snapshot.Events)
    }

    return nil
}
```

### 7.7 master.go 初始化变更

```go
// master.go — 改造后

func main() {
    // 1. 基础设施
    store := datahub.New(...)
    bus := mq.New(...)
    db := database.New(...)               // SQLite (仅业务数据)
    chClient := clickhouse.New(cfg.CH)    // 新增: ClickHouse 客户端

    // 2. Processor (新增 chClient 依赖)
    proc := processor.New(store, chClient)

    // 3. Service
    querySvc := query.NewQueryService(store, bus, db, chClient)  // 新增 chClient
    opsSvc := operations.NewCommandService(bus)
    svc := service.New(querySvc, opsSvc)

    // 4. AgentSDK
    agentSDK := agentsdk.New(proc, bus)

    // 5. Gateway
    gw := gateway.New(svc, bus)

    // 删除: SLO Processor, SLO Aggregator, SLO Cleaner 的初始化和启动
}
```

---

## 8. Web 前端改造

### 8.1 APM 页面 — Mock → 真实 API

当前 APM 页面使用 `apm-mock.ts` 的模拟数据。新架构下改为调用真实 API：

```typescript
// api/apm.ts — 改造后 (调用真实 Master API)

// 替代 mockGetTraceServices
export async function getTraceServices(params: {
  cluster_id: string;
  time_range: string;
}): Promise<ServiceInfo[]> {
  const { data } = await request.get('/api/v2/apm/services', { params });
  return data;
}

// 替代 mockQueryTraces
export async function queryTraces(params: TraceQueryParams): Promise<{
  traces: TraceSummary[];
  total: number;
}> {
  const { data } = await request.get('/api/v2/apm/traces', { params });
  return data;
}

// 替代 mockGetTraceDetail
export async function getTraceDetail(traceId: string): Promise<TraceDetail> {
  const { data } = await request.get(`/api/v2/apm/traces/${encodeURIComponent(traceId)}`);
  return data;
}

// 替代 mockGetServiceTopology
export async function getServiceTopology(params: {
  cluster_id: string;
  time_range: string;
}): Promise<ServiceTopologyData> {
  const { data } = await request.get('/api/v2/apm/topology', { params });
  return data;
}
```

### 8.2 节点指标 — 新增历史趋势

当前节点指标只有最新一帧。新架构支持历史查询：

```typescript
// api/node-metrics.ts — 新增历史查询 API

export async function getNodeMetricsHistory(params: {
  cluster_id: string;
  node_name: string;
  time_range: string;  // "1h" | "6h" | "24h" | "7d"
  interval: string;    // "1m" | "5m" | "1h"
}): Promise<MetricsDataPoint[]> {
  const { data } = await request.get('/api/v2/metrics/node/history', { params });
  return data;
}
```

### 8.3 SLO 页面 — SQLite → ClickHouse API

API 签名不变，只是后端从 SQLite 改为查 ClickHouse。前端无需修改现有代码，但可增强：

- **时间范围更灵活**: 现在限于 raw (48h) + hourly (90d)，改为任意时间粒度
- **查询更快**: ClickHouse 列式聚合比 SQLite 快 10-100x

### 8.4 新增跨信号关联视图（未来）

```
Trace 详情页 → 点击 span → 查看对应 Pod 的 CPU/Memory 趋势
                         → 查看对应 Node 的 Events
                         → 查看对应时间窗口的日志
```

---

## 9. 数据关联查询模型

### 9.1 关联键矩阵

| 数据源 | cluster_id | service_name | pod_name | node_name | trace_id | timestamp |
|--------|:---:|:---:|:---:|:---:|:---:|:---:|
| Traces (spans) | Y | Y | Y* | Y* | Y | Y |
| Metrics (node_exporter) | Y | - | - | Y | - | Y |
| Metrics (Linkerd) | Y | Y | Y | - | - | Y |
| Metrics (Traefik) | Y | Y | - | - | - | Y |
| K8s Events | Y | - | Y* | Y* | - | Y |
| Logs (未来) | Y | Y | Y | Y* | Y* | Y |

`Y*` = 通过 k8sattributes processor 自动注入

### 9.2 典型关联查询示例

**查询: 某服务在某时段的完整画像**

```sql
-- 1. 服务的请求量和错误率 (Linkerd metrics)
SELECT
    toStartOfMinute(TimeUnix) AS minute,
    sumIf(Value, Attributes['classification'] = 'success') AS success_count,
    sumIf(Value, Attributes['classification'] = 'failure') AS failure_count
FROM otel_metrics_sum
WHERE MetricName = 'response_total'
  AND Attributes['deployment'] = 'geass-gateway'
  AND TimeUnix BETWEEN '2025-01-01 00:00:00' AND '2025-01-01 01:00:00'
GROUP BY minute
ORDER BY minute;

-- 2. 同时段该服务 Pod 所在 Node 的 CPU 使用率
SELECT
    toStartOfMinute(TimeUnix) AS minute,
    ResourceAttributes['instance'] AS node,
    avg(Value) AS cpu_usage
FROM otel_metrics_gauge
WHERE MetricName IN ('node_cpu_usage_percent')  -- 或通过 rate 计算
  AND ResourceAttributes['instance'] IN (
      SELECT DISTINCT ResourceAttributes['k8s.node.name']
      FROM otel_traces
      WHERE ServiceName = 'geass-gateway'
        AND Timestamp BETWEEN ...
  )
GROUP BY minute, node;

-- 3. 同时段的慢 trace
SELECT TraceId, Duration / 1000000 AS duration_ms, SpanName
FROM otel_traces
WHERE ServiceName = 'geass-gateway'
  AND Duration > 1000000000  -- > 1s
  AND Timestamp BETWEEN ...
ORDER BY Duration DESC
LIMIT 10;
```

**查询: 从 Trace ID 追踪到基础设施**

```sql
-- 1. 获取 trace 涉及的所有 pod 和 node
SELECT DISTINCT
    SpanAttributes['k8s.pod.name'] AS pod,
    SpanAttributes['k8s.node.name'] AS node
FROM otel_traces
WHERE TraceId = 'abc123...'
  AND SpanAttributes['k8s.pod.name'] != '';

-- 2. 查询这些 node 在 trace 时间窗口前后的指标
-- (用 trace 的时间戳 ± 5min 作为窗口)
```

---

## 10. 迁移路径

### Phase 0: 基础设施部署

1. 部署 ClickHouse 单节点（或 ClickHouse Keeper 集群）
2. 创建数据库和表
3. OTel Collector 添加 ClickHouse exporter（双写: ClickHouse + 原有 Jaeger/Prometheus）
4. 验证数据正确写入 ClickHouse

**产出**: ClickHouse 开始积累数据，现有系统不受影响

### Phase 1: Master V2 — 新增 ClickHouse 查询能力

1. 实现 `clickhouse/` 包（ClickHouseClient 接口 + 实现）
2. Service Query 新增 APM / 节点指标历史 / 跨信号关联方法
3. Gateway 新增 APM API 端点
4. 验证: APM API 返回真实 ClickHouse 数据

**产出**: Master 具备从 ClickHouse 查询的能力，现有 SLO 管线仍运行

### Phase 2: Web 前端 — APM Mock → 真实数据

1. `api/apm.ts` 替换 mock 调用为真实 API
2. 删除 `api/apm-mock.ts` 和 `api/apm-mock-data.json`
3. 节点指标页面增加历史趋势图

**产出**: APM 页面展示真实数据

### Phase 3: SLO 管线迁移

1. Master Service Query 的 SLO 方法改查 ClickHouse
2. 验证 SLO 页面数据一致
3. 删除 Master 的 SLO Processor / Aggregator / Cleaner
4. 删除 SQLite 中的 SLO 相关表

**产出**: SLO 数据完全由 ClickHouse 承载

### Phase 4: Agent V2 瘦身

1. 从 ClusterSnapshot 中移除 NodeMetrics 和 SLOData 字段
2. 删除 Agent 的 OTelClient / IngressClient / ReceiverClient
3. 删除 Agent 的 MetricsRepository / SLORepository
4. 删除 Agent 的 MetricsSyncLoop
5. 简化 SnapshotService（不再采集 metrics/SLO）

**产出**: Agent 代码量减少 ~40-50%

### Phase 5: 清理与优化

1. 删除 Jaeger 部署
2. OTel Collector 移除 Jaeger exporter（只保留 ClickHouse）
3. 按需添加 ClickHouse Materialized View 优化查询性能
4. 前端添加跨信号关联视图

**产出**: 架构完全迁移，Jaeger 退役

---

## 11. 后续详细设计清单

本文档是中心设计思想，每个 Phase 需要独立的详细设计文档：

| 详细设计 | 对应 Phase | 主要内容 |
|---------|-----------|---------|
| `clickhouse-infra-design.md` | Phase 0 | ClickHouse 部署方案、表创建 DDL、OTel Collector 配置 |
| `clickhouse-master-query-design.md` | Phase 1 | ClickHouseClient 接口实现、SQL 查询模板、数据模型 |
| `apm-real-data-design.md` | Phase 1-2 | APM API 端点设计、前端 API 切换方案 |
| `slo-clickhouse-migration-design.md` | Phase 3 | SLO 查询 SQL、SQLite → ClickHouse 数据迁移 |
| `agent-simplification-design.md` | Phase 4 | Agent 删除清单、ClusterSnapshot 精简、测试方案 |
| `cross-signal-correlation-design.md` | Phase 5 | 关联查询 API、前端交互设计 |

---

## 附录

### A. ClickHouse vs 其他方案对比

| 维度 | ClickHouse | OpenSearch | Grafana Stack |
|------|-----------|------------|---------------|
| **聚合性能** | 极快（列式存储） | 中等 | 快（但分散） |
| **SQL 支持** | 原生 SQL | DSL 为主 | 各组件各自查询语言 |
| **跨信号 JOIN** | SQL JOIN | 不支持 | 不支持 |
| **存储效率** | 高（列式压缩） | 中（倒排索引） | 各组件独立 |
| **资源占用** | 中等 (2-4GB) | 高 (4-8GB+) | 高（多组件叠加） |
| **OTel 兼容** | 官方 exporter | 社区 exporter | 各组件独立对接 |
| **全文搜索** | 弱 | 强 | 中（Loki） |

**选择 ClickHouse 的原因**: 核心需求是跨信号聚合关联（SQL JOIN），而非全文搜索。ClickHouse 的列式存储在聚合查询场景下性能远超 OpenSearch，资源占用更低，且有官方 OTel exporter 支持。

### B. 资源估算

| 组件 | 当前资源 | 新增资源 |
|------|---------|---------|
| OTel Collector | 512Mi-1Gi | 不变 |
| Jaeger | 512Mi-1Gi (Badger) | **删除** |
| ClickHouse | - | **新增** 2-4Gi RAM, 50-100Gi Disk |
| Agent | ~200Mi | 减少至 ~120Mi (删除 metrics 采集) |
| Master | ~300Mi | +50Mi (ClickHouse 连接池) |

**净资源变化**: 删除 Jaeger (~1Gi) + 新增 ClickHouse (~3Gi) ≈ 净增 ~2Gi RAM
