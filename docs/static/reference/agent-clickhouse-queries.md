# Agent ClickHouse 查询参考

> 基于 `atlhyper_agent_v2/repository/ch/` 和 `atlhyper_agent_v2/service/command/ch_query.go` 源码整理。
> 记录所有 Agent 执行的 ClickHouse SQL 查询、输入参数和数据流。

---

## 目录

0. [Agent 数据采集全景](#0-agent-数据采集全景)
1. [数据流总览](#1-数据流总览)
2. [OTel 表结构](#2-otel-表结构)
3. [Traces 查询（4 个）](#3-traces-查询)
4. [Logs 查询（3 个并行）](#4-logs-查询)
5. [Metrics 查询（20+ 个子查询）](#5-metrics-查询)
6. [SLO 查询（5 个）](#6-slo-查询)
7. [Summary 查询（3 个，快照用）](#7-summary-查询)
8. [已修复问题](#8-已修复问题)

---

## 0. Agent 数据采集全景

Agent 有两种数据通道，分别服务于不同场景：

### 快照上报（定时推送，K8s API + ClickHouse Summary）

Agent 定时采集集群状态，打包为 `ClusterSnapshot` 推送给 Master。数据存在 Master 内存中供前端即时查询。

| 数据 | 来源 | 采集方式 | 用途 |
|------|------|---------|------|
| **20 种 K8s 资源** | K8s API Server | 并发拉取 | 集群资源列表/详情 |
| Pods（含 CPU/内存） | API Server + metrics-server | 拉取 | Pod 列表、资源使用率 |
| Nodes（含 CPU/内存） | API Server + metrics-server | 拉取 | 节点列表、资源使用率 |
| Deployments | API Server | 拉取 | 工作负载管理 |
| StatefulSets | API Server | 拉取 | 有状态应用 |
| DaemonSets | API Server | 拉取 | 守护进程 |
| ReplicaSets | API Server | 拉取 | 副本集 |
| Services | API Server | 拉取 | 服务发现 |
| Ingresses | API Server | 拉取 | 入口路由 |
| ConfigMaps | API Server | 拉取 | 配置管理 |
| Secrets（脱敏） | API Server | 拉取 | 密钥管理 |
| Namespaces | API Server | 拉取 | 命名空间 |
| Events | API Server | 拉取 | 事件流 |
| Jobs / CronJobs | API Server | 拉取 | 批处理任务 |
| PV / PVC | API Server | 拉取 | 持久化存储 |
| ResourceQuotas | API Server | 拉取 | 资源配额 |
| LimitRanges | API Server | 拉取 | 资源限制 |
| NetworkPolicies | API Server | 拉取 | 网络策略 |
| ServiceAccounts | API Server | 拉取 | 服务账户 |
| **OTel 概览（3 个聚合查询）** | ClickHouse | 定时查询（带缓存） | 首页仪表盘摘要 |
| APM Summary | ClickHouse `otel_traces` | 聚合 | 服务数、健康率、RPS、P99 |
| SLO Summary | ClickHouse `otel_metrics_*` | 聚合 | Ingress 服务数、Mesh 服务数、mTLS 率 |
| Metrics Summary | ClickHouse `otel_metrics_*` | 聚合 | 节点数、平均/最大 CPU、平均/最大内存 |

### 按需查询（Web → Master → Agent → ClickHouse）

用户在前端操作时，通过 Command 机制实时查询 ClickHouse，结果 JSON 透传回前端。

| 数据 | Action | 来源 | 用途 |
|------|--------|------|------|
| **Traces（4 个子查询）** | `query_traces` | ClickHouse `otel_traces` | APM 页面 |
| Trace 列表 | list_traces | otel_traces | Trace 搜索 |
| Trace 详情 | query_trace_detail | otel_traces | Span 瀑布图 |
| 服务列表 | list_services | otel_traces | 服务性能排行 |
| 服务拓扑 | get_topology | otel_traces | 调用关系图 |
| **Logs（3 个并行查询）** | `query_logs` | ClickHouse `otel_logs` | 日志浏览器 |
| 日志条目 + 总数 + Facets | — | otel_logs | Kibana 风格搜索 |
| **Metrics（4 个子操作）** | `query_metrics` | ClickHouse `otel_metrics_*` | 节点监控 |
| 集群指标概览 | get_summary | otel_metrics_* | 仪表盘概览 |
| 所有节点指标 | list_all | otel_metrics_* | 节点列表 |
| 单节点指标（10 并行） | get_node | otel_metrics_* | 节点详情 |
| 单指标时序 | get_series | otel_metrics_* | 趋势图 |
| **SLO（5 个子操作）** | `query_slo` | ClickHouse `otel_metrics_*` | SLO 页面 |
| Ingress SLO | list_ingress | otel_metrics_sum + histogram | Traefik 入口 SLO |
| Service SLO | list_service | otel_metrics_gauge | Linkerd 网格 SLO |
| 服务间拓扑 | list_edges | otel_metrics_gauge | 网格调用关系 |
| SLO 时序 | get_time_series | otel_metrics_gauge | SLO 趋势图 |
| SLO 摘要 | get_summary | 并行调用上述 | 仪表盘统计 |
| **K8s 操作指令** | 各 action | K8s API Server | 运维操作 |
| 扩缩容 / 重启 / 更新镜像 | scale / restart / update_image | API Server | Deployment 管理 |
| 删除资源 | delete | API Server | 资源删除 |
| 封锁/解封节点 | cordon / uncordon | API Server | 节点调度 |
| Pod 日志 | get_logs | API Server | 日志查看 |
| ConfigMap/Secret 数据 | get_configmap / get_secret | API Server | 配置查看 |
| AI 动态查询 | dynamic | API Server | AI 只读查询 |

### 关键区别

| 维度 | 快照上报 | 按需查询 |
|------|---------|---------|
| **触发方式** | 定时（Scheduler 周期调度） | 用户操作（Command 机制） |
| **数据存储** | Master 内存（DataHub） | 不存储，JSON 透传 |
| **K8s 资源** | 20 种资源完整列表 | 单资源操作/查询 |
| **ClickHouse** | 3 个聚合 Summary（带 TTL 缓存） | ~42 个 SQL（按需实时查询） |
| **延迟** | 秒级（快照间隔） | 实时（30s 超时） |

---

## 1. 数据流总览

```
Web 前端
  │ GET /api/v2/observe/*
  ↓
Master (observe.go)
  │ 构建 Command{Action, Params}
  │ → MQ.EnqueueCommand()
  │ → MQ.WaitCommandResult(30s)
  ↓
Agent (ch_query.go)
  │ 解析 Command.Action + Params
  │ 调用对应 Repository 方法
  ↓
ClickHouse (otel_* 表)
  │ 执行 SQL
  ↓
结果 JSON 原路透传回前端
```

**Command Action 映射：**

| Action | ch_query.go handler | Repository |
|--------|-------------------|------------|
| `query_traces` | `handleQueryTraces` | `TraceQueryRepository` |
| `query_trace_detail` | `handleQueryTraceDetail` | `TraceQueryRepository` |
| `query_logs` | `handleQueryLogs` | `LogQueryRepository` |
| `query_metrics` | `handleQueryMetrics` | `MetricsQueryRepository` |
| `query_slo` | `handleQuerySLO` | `SLOQueryRepository` |

---

## 2. OTel 表结构

| 表名 | 类型 | 关键字段 | 用途 |
|------|------|---------|------|
| `otel_traces` | Span | TraceId, SpanId, ParentSpanId, ServiceName, SpanName, SpanKind, Duration, StatusCode, Timestamp | APM Traces |
| `otel_logs` | Log | Timestamp, ServiceName, Body, SeverityText, ScopeName, TraceId | 日志 |
| `otel_metrics_gauge` | Gauge | MetricName, Value, TimeUnix, ResourceAttributes, Attributes | 瞬时值指标 |
| `otel_metrics_sum` | Counter | MetricName, Value, TimeUnix, ResourceAttributes, Attributes | 累积值指标（需计算 rate） |
| `otel_metrics_histogram` | Histogram | MetricName, ExplicitBounds, BucketCounts, TimeUnix | 分布指标 |

**公共字段约定：**
- 节点 IP: `ResourceAttributes['net.host.name']`
- 时间筛选: `TimeUnix >= now() - INTERVAL N MINUTE/SECOND`

---

## 3. Traces 查询

### 3.1 ListTraces — Trace 列表

**触发路径：** `GET /api/v2/observe/traces?cluster_id=X&service=Y&limit=50`

**Command 参数：**
```json
{
  "sub_action": "list_traces",
  "service": "frontend",          // 可选，服务名过滤
  "min_duration_ms": 100,         // 可选，最小耗时 (ms)
  "limit": 50,                    // 可选，默认 50，最大 200
  "since": "5m"                   // 可选，默认 5 分钟
}
```

**SQL：**
```sql
SELECT TraceId,
       min(Timestamp)                                          AS ts,
       argMinIf(ServiceName, Timestamp, ParentSpanId = '')     AS rootSvc,
       argMinIf(SpanName,    Timestamp, ParentSpanId = '')     AS rootOp,
       max(Duration) / 1e6                                     AS durationMs,
       count()                                                 AS spanCount,
       count(DISTINCT ServiceName)                             AS serviceCount,
       countIf(StatusCode = 'STATUS_CODE_ERROR') > 0           AS hasError
FROM otel_traces
WHERE Timestamp >= now() - INTERVAL {since_seconds} SECOND
  [AND ServiceName = '{service}']
  [AND Duration >= {min_duration_ms * 1e6}]
GROUP BY TraceId
ORDER BY ts DESC
LIMIT {limit}
```

---

### 3.2 GetTraceDetail — Trace 详情

**触发路径：** `GET /api/v2/observe/traces/{traceId}?cluster_id=X`

**Command 参数：**
```json
{
  "trace_id": "abc123def456..."    // 必需
}
```

**SQL：**
```sql
SELECT Timestamp, TraceId, SpanId, ParentSpanId, SpanName, SpanKind,
       ServiceName, Duration, StatusCode, StatusMessage,
       SpanAttributes, ResourceAttributes,
       Events.Timestamp, Events.Name, Events.Attributes
FROM otel_traces
WHERE TraceId = '{trace_id}'
ORDER BY Timestamp
```

---

### 3.3 ListServices — 服务列表（APM 聚合统计）

**触发路径：** `GET /api/v2/observe/traces/services?cluster_id=X`

**Command 参数：**
```json
{
  "sub_action": "list_services"
}
```

**SQL：**
```sql
SELECT ServiceName,
       ResourceAttributes['service.namespace']                AS ns,
       count()                                                AS spanCount,
       countIf(StatusCode = 'STATUS_CODE_ERROR')              AS errorCount,
       1 - countIf(StatusCode = 'STATUS_CODE_ERROR') / count() AS successRate,
       avg(Duration) / 1e6                                    AS avgMs,
       quantile(0.50)(Duration) / 1e6                         AS p50Ms,
       quantile(0.99)(Duration) / 1e6                         AS p99Ms,
       count() / 300                                          AS rps
FROM otel_traces
WHERE SpanKind = 'SPAN_KIND_SERVER'
  AND Timestamp >= now() - INTERVAL 5 MINUTE
GROUP BY ServiceName, ns
```

---

### 3.4 GetTopology — 服务拓扑图

**触发路径：** `GET /api/v2/observe/traces/topology?cluster_id=X`

**Command 参数：**
```json
{
  "sub_action": "get_topology"
}
```

**SQL 1 — 拓扑节点（各服务指标）：**
```sql
SELECT ServiceName,
       ResourceAttributes['service.namespace']                AS ns,
       count() / 300                                          AS rps,
       1 - countIf(StatusCode = 'STATUS_CODE_ERROR') / count() AS successRate,
       quantile(0.99)(Duration) / 1e6                         AS p99Ms
FROM otel_traces
WHERE SpanKind = 'SPAN_KIND_SERVER'
  AND Timestamp >= now() - INTERVAL 5 MINUTE
GROUP BY ServiceName, ns
```

**SQL 2 — 拓扑边（跨服务调用）：**
```sql
SELECT t1.ServiceName AS source, t2.ServiceName AS target,
       count()                                                AS callCount,
       avg(t2.Duration) / 1e6                                 AS avgMs,
       countIf(t2.StatusCode = 'STATUS_CODE_ERROR') / count() AS errorRate
FROM otel_traces t1
JOIN otel_traces t2 ON t1.SpanId = t2.ParentSpanId AND t1.TraceId = t2.TraceId
WHERE t1.ServiceName != t2.ServiceName
  AND t1.Timestamp >= now() - INTERVAL 5 MINUTE
  AND t2.Timestamp >= now() - INTERVAL 5 MINUTE
GROUP BY source, target
```

---

## 4. Logs 查询

### 4.1 QueryLogs — 日志查询（3 个并行 SQL）

**触发路径：** `POST /api/v2/observe/logs/query`

**Command 参数（POST body 透传）：**
```json
{
  "query": "error timeout",      // 可选，Body LIKE 搜索
  "service": "frontend",         // 可选，ServiceName 过滤
  "level": "ERROR",              // 可选，SeverityText 过滤
  "scope": "http.server",        // 可选，ScopeName 过滤
  "limit": 50,                   // 可选，默认 50，最大 500
  "offset": 0,                   // 可选，分页偏移
  "since": "15m"                 // 可选，默认 15 分钟
}
```

**SQL 1 — 日志条目：**
```sql
SELECT Timestamp, TraceId, SpanId, SeverityText, SeverityNumber,
       ServiceName, Body, ScopeName,
       LogAttributes, ResourceAttributes
FROM otel_logs
WHERE Timestamp >= now() - INTERVAL {since_seconds} SECOND
  [AND Body LIKE '%{query}%']
  [AND ServiceName = '{service}']
  [AND SeverityText = '{level}']
  [AND ScopeName = '{scope}']
ORDER BY Timestamp DESC
LIMIT {limit} OFFSET {offset}
```

**SQL 2 — 总数统计（并行）：**
```sql
SELECT count()
FROM otel_logs
WHERE Timestamp >= now() - INTERVAL {since_seconds} SECOND
  [AND Body LIKE '%{query}%']
  [AND ServiceName = '{service}']
  [AND SeverityText = '{level}']
  [AND ScopeName = '{scope}']
```

**SQL 3 — Facets 分面统计（并行，执行 3 次）：**
```sql
-- 分别对 ServiceName / SeverityText / ScopeName 执行
SELECT {column} AS value, count() AS cnt
FROM otel_logs
WHERE Timestamp >= now() - INTERVAL {since_seconds} SECOND
GROUP BY value
ORDER BY cnt DESC
LIMIT 50
```

---

## 5. Metrics 查询

### 5.1 GetMetricsSummary — 集群指标概览

**触发路径：** `GET /api/v2/observe/metrics/summary?cluster_id=X`

**Command 参数：**
```json
{
  "sub_action": "get_summary"
}
```

**实现：** 调用 `ListAllNodeMetrics()` 后在 Go 侧聚合计算 AvgCPU/AvgMem/MaxCPU/MaxMem/MaxTemp。

---

### 5.2 ListAllNodeMetrics — 所有节点指标

**触发路径：** `GET /api/v2/observe/metrics/nodes?cluster_id=X`

**Command 参数：**
```json
{
  "sub_action": "list_all"
}
```

**SQL — 获取活跃节点 IP：**
```sql
SELECT DISTINCT ResourceAttributes['net.host.name'] AS ip
FROM otel_metrics_gauge
WHERE MetricName = 'node_load1'
  AND TimeUnix >= now() - INTERVAL 5 MINUTE
```

然后对每个节点 IP 并行执行 `buildNodeMetrics`（10 个子查询并行），见下方 5.4。

---

### 5.3 GetNodeMetrics — 单节点指标

**触发路径：** `GET /api/v2/observe/metrics/nodes/{nodeName}?cluster_id=X`

**Command 参数：**
```json
{
  "sub_action": "get_node",
  "node_name": "k8s-worker-1"    // 必需
}
```

**实现：** NodeName → IP 映射后，调用 `buildNodeMetrics` 获取完整指标。

---

### 5.4 buildNodeMetrics — 单节点完整指标（10 个并行子查询）

每个子查询的输入参数都是 `ip`（节点 IP 地址）。

#### 5.4.1 fillCPU — CPU 指标

**SQL 1 — CPU mode 使用率（rate 计算）：**
```sql
SELECT Attributes['mode'] AS mode,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
       (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
FROM otel_metrics_sum
WHERE MetricName = 'node_cpu_seconds_total'
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL 5 MINUTE
GROUP BY Attributes['cpu'], mode
HAVING count() >= 2
```

**SQL 2 — CPU 核心数：**
```sql
SELECT count(DISTINCT Attributes['cpu'])
FROM otel_metrics_sum
WHERE MetricName = 'node_cpu_seconds_total'
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL 5 MINUTE
```

**SQL 3 — Load Average（gauge）：**
```sql
SELECT MetricName, argMax(Value, TimeUnix)
FROM otel_metrics_gauge
WHERE MetricName IN ('node_load1', 'node_load5', 'node_load15')
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL 2 MINUTE
GROUP BY MetricName
```

**产出字段：** UsagePct, UserPct, SystemPct, IOWaitPct, Cores, Load1, Load5, Load15

#### 5.4.2 fillMemory — 内存指标

```sql
SELECT MetricName, argMax(Value, TimeUnix)
FROM otel_metrics_gauge
WHERE MetricName IN (
  'node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes',
  'node_memory_MemFree_bytes', 'node_memory_Cached_bytes',
  'node_memory_Buffers_bytes', 'node_memory_SwapTotal_bytes',
  'node_memory_SwapFree_bytes'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 2 MINUTE
GROUP BY MetricName
```

**产出字段：** TotalBytes, AvailableBytes, FreeBytes, CachedBytes, BuffersBytes, SwapTotalBytes, SwapFreeBytes, UsagePct, SwapUsagePct

#### 5.4.3 fillDisks — 磁盘指标

**SQL 1 — 容量（gauge）：**
```sql
SELECT Attributes['device'] AS device,
       Attributes['mountpoint'] AS mp,
       Attributes['fstype'] AS fs,
       argMaxIf(Value, TimeUnix, MetricName='node_filesystem_size_bytes') AS total,
       argMaxIf(Value, TimeUnix, MetricName='node_filesystem_avail_bytes') AS avail
FROM otel_metrics_gauge
WHERE MetricName IN ('node_filesystem_size_bytes', 'node_filesystem_avail_bytes')
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL 2 MINUTE
  AND Attributes['fstype'] NOT IN ('tmpfs', 'devtmpfs', 'overlay', 'squashfs')
GROUP BY device, mp, fs
HAVING total > 0
```

**SQL 2 — IO rate（counter）：**
```sql
SELECT Attributes['device'] AS device, MetricName,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
       (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
FROM otel_metrics_sum
WHERE MetricName IN (
  'node_disk_read_bytes_total', 'node_disk_written_bytes_total',
  'node_disk_reads_completed_total', 'node_disk_writes_completed_total',
  'node_disk_io_time_seconds_total'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 5 MINUTE
GROUP BY device, MetricName
HAVING count() >= 2
```

**产出字段（每磁盘）：** Device, MountPoint, FSType, TotalBytes, AvailBytes, UsagePct, ReadBytesPerSec, WriteBytesPerSec, ReadIOPS, WriteIOPS, IOUtilPct

#### 5.4.4 fillNetworks — 网络指标

**SQL 1 — 状态（gauge）：**
```sql
SELECT Attributes['device'] AS iface,
       argMaxIf(Value, TimeUnix, MetricName='node_network_up') AS up,
       argMaxIf(Value, TimeUnix, MetricName='node_network_speed_bytes') AS speed,
       argMaxIf(Value, TimeUnix, MetricName='node_network_mtu_bytes') AS mtu
FROM otel_metrics_gauge
WHERE MetricName IN ('node_network_up', 'node_network_speed_bytes', 'node_network_mtu_bytes')
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL 2 MINUTE
GROUP BY iface
```

**SQL 2 — 吞吐量 rate（counter）：**
```sql
SELECT Attributes['device'] AS iface, MetricName,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
       (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
FROM otel_metrics_sum
WHERE MetricName IN (
  'node_network_receive_bytes_total', 'node_network_transmit_bytes_total',
  'node_network_receive_packets_total', 'node_network_transmit_packets_total',
  'node_network_receive_errs_total', 'node_network_transmit_errs_total',
  'node_network_receive_drop_total', 'node_network_transmit_drop_total'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 5 MINUTE
GROUP BY iface, MetricName
HAVING count() >= 2
```

**产出字段（每网卡）：** Interface, Up, SpeedBps, MTU, RxBytesPerSec, TxBytesPerSec, RxPktPerSec, TxPktPerSec, RxErrPerSec, TxErrPerSec, RxDropPerSec, TxDropPerSec

#### 5.4.5 fillTemperature — 温度

```sql
SELECT Attributes['chip'] AS chip, Attributes['sensor'] AS sensor,
       argMaxIf(Value, TimeUnix, MetricName='node_hwmon_temp_celsius') AS current,
       argMaxIf(Value, TimeUnix, MetricName='node_hwmon_temp_max_celsius') AS maxv,
       argMaxIf(Value, TimeUnix, MetricName='node_hwmon_temp_crit_celsius') AS crit
FROM otel_metrics_gauge
WHERE MetricName IN ('node_hwmon_temp_celsius', 'node_hwmon_temp_max_celsius', 'node_hwmon_temp_crit_celsius')
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL 2 MINUTE
GROUP BY chip, sensor
```

**产出字段：** CPUTempC, CPUMaxC, CPUCritC, Sensors[]

#### 5.4.6 fillPSI — Pressure Stall Info

```sql
SELECT MetricName,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
       (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
FROM otel_metrics_sum
WHERE MetricName IN (
  'node_pressure_cpu_waiting_seconds_total',
  'node_pressure_memory_waiting_seconds_total',
  'node_pressure_memory_stalled_seconds_total',
  'node_pressure_io_waiting_seconds_total',
  'node_pressure_io_stalled_seconds_total'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 5 MINUTE
GROUP BY MetricName
HAVING count() >= 2
```

**产出字段：** CPUSomePct, MemSomePct, MemFullPct, IOSomePct, IOFullPct

#### 5.4.7 fillTCP — TCP 连接状态

```sql
SELECT MetricName, argMax(Value, TimeUnix)
FROM otel_metrics_gauge
WHERE MetricName IN (
  'node_netstat_Tcp_CurrEstab',
  'node_sockstat_TCP_alloc',
  'node_sockstat_TCP_inuse',
  'node_sockstat_TCP_tw',
  'node_sockstat_sockets_used'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 2 MINUTE
GROUP BY MetricName
```

**产出字段：** CurrEstab, Alloc, InUse, TimeWait, SocketsUsed

#### 5.4.8 fillSystem — 系统资源

```sql
SELECT MetricName, argMax(Value, TimeUnix)
FROM otel_metrics_gauge
WHERE MetricName IN (
  'node_nf_conntrack_entries', 'node_nf_conntrack_entries_limit',
  'node_filefd_allocated', 'node_filefd_maximum',
  'node_entropy_available_bits'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 2 MINUTE
GROUP BY MetricName
```

**产出字段：** ConntrackEntries, ConntrackLimit, FilefdAllocated, FilefdMax, EntropyBits

#### 5.4.9 fillVMStat — VMStat + Softnet

vmstat 指标在 gauge 表（OTel collector 将其归类为 gauge），softnet 在 sum 表，分两次查询。

**SQL 1 — VMStat（gauge 表）：**
```sql
SELECT MetricName,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
       (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
FROM otel_metrics_gauge
WHERE MetricName IN (
  'node_vmstat_pgfault', 'node_vmstat_pgmajfault',
  'node_vmstat_pswpin', 'node_vmstat_pswpout'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 5 MINUTE
GROUP BY MetricName
HAVING count() >= 2
```

**SQL 2 — Softnet（sum 表）：**
```sql
SELECT MetricName,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
       (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
FROM otel_metrics_sum
WHERE MetricName IN (
  'node_softnet_dropped_total', 'node_softnet_times_squeezed_total'
)
AND ResourceAttributes['net.host.name'] = '{ip}'
AND TimeUnix >= now() - INTERVAL 5 MINUTE
GROUP BY MetricName
HAVING count() >= 2
```

**产出字段：** PgFaultPerSec, PgMajFaultPerSec, PswpInPerSec, PswpOutPerSec, DroppedPerSec, SqueezedPerSec

#### 5.4.10 fillSystemInfo — 系统信息

**SQL 1 — 启动时间：**
```sql
SELECT argMax(Value, TimeUnix)
FROM otel_metrics_gauge
WHERE MetricName = 'node_boot_time_seconds'
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL 2 MINUTE
```

**SQL 2 — 内核版本：**
```sql
SELECT Attributes['release']
FROM otel_metrics_gauge
WHERE MetricName = 'node_uname_info'
  AND ResourceAttributes['net.host.name'] = '{ip}'
ORDER BY TimeUnix DESC LIMIT 1
```

**产出字段：** Uptime (秒), Kernel (版本字符串)

---

### 5.5 GetNodeMetricsSeries — 单指标时序数据

**触发路径：** `GET /api/v2/observe/metrics/nodes/{name}/series?cluster_id=X&metric=node_load1&minutes=30`

**Command 参数：**
```json
{
  "sub_action": "get_series",
  "node_name": "k8s-worker-1",   // 必需
  "metric": "node_load1",        // 必需 — 指标名
  "since": "30m"                  // 可选，默认 30 分钟
}
```

**SQL — Gauge 类型（如 node_load1）：**
```sql
SELECT toStartOfInterval(TimeUnix, INTERVAL 60 SECOND) AS ts,
       avg(Value) AS val
FROM otel_metrics_gauge
WHERE MetricName = '{metric}'
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
GROUP BY ts
ORDER BY ts
```

**SQL — Counter 类型（如 node_cpu_seconds_total）：**
```sql
SELECT TimeUnix, Value
FROM otel_metrics_sum
WHERE MetricName = '{metric}'
  AND ResourceAttributes['net.host.name'] = '{ip}'
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
ORDER BY TimeUnix
```
Counter 类型在 Go 侧计算 rate（相邻点差值/时间差）。

**Counter 指标列表：**
`node_cpu_seconds_total`, `node_disk_read_bytes_total`, `node_disk_written_bytes_total`,
`node_disk_reads_completed_total`, `node_disk_writes_completed_total`, `node_disk_io_time_seconds_total`,
`node_network_receive_bytes_total`, `node_network_transmit_bytes_total`,
`node_softnet_dropped_total`, `node_softnet_times_squeezed_total`

---

## 6. SLO 查询

### 6.1 ListIngressSLO — Traefik 入口 SLO

**触发路径：** `GET /api/v2/observe/slo/ingress?cluster_id=X&time_range=5m`

**Command 参数：**
```json
{
  "sub_action": "list_ingress",
  "since": "5m"                   // 可选，默认 5 分钟
}
```

**SQL 1 — 请求计数（rate）：**
```sql
SELECT Attributes['service'] AS svc,
       Attributes['code'] AS code,
       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) AS delta
FROM otel_metrics_sum
WHERE MetricName = 'traefik_service_requests_total'
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
GROUP BY svc, code
HAVING count() >= 2
```

**SQL 2 — 延迟 Histogram：**
```sql
SELECT Attributes['service'] AS svc,
       ExplicitBounds,
       BucketCounts
FROM otel_metrics_histogram
WHERE MetricName = 'traefik_service_request_duration_seconds'
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
ORDER BY svc, TimeUnix DESC
```

**产出字段：** ServiceKey, DisplayName, TotalRequests, TotalErrors, RPS, SuccessRate, ErrorRate, StatusCodes, P50Ms, P90Ms, P99Ms

---

### 6.2 ListServiceSLO — Linkerd 服务网格 SLO

**触发路径：** `GET /api/v2/observe/slo/services?cluster_id=X&time_range=5m`

**Command 参数：**
```json
{
  "sub_action": "list_service",
  "since": "5m"
}
```

**SQL 1 — 响应统计：**
```sql
SELECT Attributes['deployment'] AS deploy,
       Attributes['namespace'] AS ns,
       Attributes['status_code'] AS code,
       Attributes['tls'] AS tls,
       sum(Value) AS total
FROM otel_metrics_gauge
WHERE MetricName = 'response_total'
  AND Attributes['direction'] = 'inbound'
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
GROUP BY deploy, ns, code, tls
```

**SQL 2 — 延迟分位数（Histogram bucket）：**
```sql
SELECT Attributes['deployment'] AS deploy,
       Attributes['namespace'] AS ns,
       Attributes['le'] AS le,
       sum(Value) AS total
FROM otel_metrics_gauge
WHERE MetricName = 'response_latency_ms_bucket'
  AND Attributes['direction'] = 'inbound'
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
GROUP BY deploy, ns, le
ORDER BY deploy, ns, toFloat64OrNull(le)
```

**产出字段：** Namespace, Name, RPS, SuccessRate, MTLSRate, StatusCodes, P50Ms, P90Ms, P99Ms

---

### 6.3 ListServiceEdges — 服务间调用拓扑

**触发路径：** `GET /api/v2/observe/slo/edges?cluster_id=X`

**Command 参数：**
```json
{
  "sub_action": "list_edges",
  "since": "5m"
}
```

**SQL：**
```sql
SELECT Attributes['deployment'] AS src,
       Attributes['namespace'] AS src_ns,
       Attributes['dst_deployment'] AS dst,
       Attributes['dst_namespace'] AS dst_ns,
       Attributes['status_code'] AS code,
       sum(Value) AS total
FROM otel_metrics_gauge
WHERE MetricName = 'response_total'
  AND Attributes['direction'] = 'outbound'
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
GROUP BY src, src_ns, dst, dst_ns, code
```

**产出字段：** SrcNamespace, SrcName, DstNamespace, DstName, RPS, SuccessRate

---

### 6.4 GetSLOTimeSeries — SLO 时序数据

**触发路径：** `GET /api/v2/observe/slo/timeseries?cluster_id=X&service=frontend&time_range=1h`

**Command 参数：**
```json
{
  "sub_action": "get_time_series",
  "name": "frontend",             // 必需，service/deployment 名称
  "since": "5m"                   // 可选
}
```

**SQL：**
```sql
SELECT toStartOfInterval(TimeUnix, INTERVAL 300 SECOND) AS ts,
       Attributes['status_code'] AS code,
       sum(Value) AS total
FROM otel_metrics_gauge
WHERE MetricName = 'response_total'
  AND Attributes['direction'] = 'inbound'
  AND (Attributes['deployment'] = '{name}' OR Attributes['service'] = '{name}')
  AND TimeUnix >= now() - INTERVAL {since_seconds} SECOND
GROUP BY ts, code
ORDER BY ts
```

**产出字段：** Name, Points[] { Timestamp, RPS, SuccessRate }

---

### 6.5 GetSLOSummary — SLO 仪表盘摘要

**触发路径：** `GET /api/v2/observe/slo/summary?cluster_id=X`

**Command 参数：**
```json
{
  "sub_action": "get_summary"
}
```

**实现：** 并行调用 `ListIngressSLO(5min)` + `ListServiceSLO(5min)`，在 Go 侧聚合计算 TotalServices, HealthyServices, WarningServices, CriticalServices, AvgSuccessRate, TotalRPS, AvgP99Ms。

---

## 7. Summary 查询（快照上报用，非 Command 触发）

这些查询由 `OTelSummaryRepository` 执行，随快照定期上报，不经过 Command 机制。

### 7.1 GetAPMSummary

```sql
SELECT count(DISTINCT ServiceName)                             AS total_services,
       count(DISTINCT if(err_rate < 0.05, ServiceName, NULL))  AS healthy_services,
       sum(span_cnt) / 300                                     AS total_rps,
       avg(1 - err_rate) * 100                                 AS avg_success_rate,
       avg(p99_ns) / 1e6                                       AS avg_p99_ms
FROM (
    SELECT ServiceName,
           count()                                             AS span_cnt,
           countIf(StatusCode = 'STATUS_CODE_ERROR') / count() AS err_rate,
           quantile(0.99)(Duration)                            AS p99_ns
    FROM otel_traces
    WHERE SpanKind = 'SPAN_KIND_SERVER' AND Timestamp >= now() - INTERVAL 5 MINUTE
    GROUP BY ServiceName
)
```

### 7.2 GetSLOSummary

**SQL 1 — Ingress (Traefik)：**
```sql
SELECT count(DISTINCT svc) AS ingress_services, avg(rate_val) AS avg_rps
FROM (
    SELECT Attributes['service'] AS svc,
           (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
           (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate_val
    FROM otel_metrics_sum
    WHERE MetricName = 'traefik_service_requests_total'
      AND TimeUnix >= now() - INTERVAL 5 MINUTE
    GROUP BY svc HAVING count() >= 2
)
```

**SQL 2 — Mesh (Linkerd)：**
```sql
SELECT count(DISTINCT Attributes['deployment']) AS mesh_services,
       sumIf(Value, Attributes['tls'] = 'true') / if(sum(Value) = 0, 1, sum(Value)) AS avg_mtls
FROM otel_metrics_gauge
WHERE MetricName = 'response_total' AND Attributes['direction'] = 'inbound'
  AND TimeUnix >= now() - INTERVAL 5 MINUTE
```

### 7.3 GetMetricsSummary

**SQL 1 — CPU（同 5.4.1 的 summary 版本）：**
```sql
WITH cpu_rate AS (
    SELECT ResourceAttributes['net.host.name'] AS ip, Attributes['mode'] AS mode,
           (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) /
           (toUnixTimestamp(argMax(TimeUnix, TimeUnix)) - toUnixTimestamp(argMin(TimeUnix, TimeUnix))) AS rate
    FROM otel_metrics_sum
    WHERE MetricName = 'node_cpu_seconds_total' AND TimeUnix >= now() - INTERVAL 5 MINUTE
    GROUP BY ip, Attributes['cpu'], mode HAVING count() >= 2
)
SELECT count(DISTINCT ip), avg(usage) * 100, max(usage) * 100
FROM (
    SELECT ip, 1 - sumIf(rate, mode='idle') / if(sum(rate) = 0, 1, sum(rate)) AS usage
    FROM cpu_rate GROUP BY ip
)
```

**SQL 2 — 内存：**
```sql
SELECT avg(1 - avail / if(total = 0, 1, total)) * 100 AS avg_mem,
       max(1 - avail / if(total = 0, 1, total)) * 100 AS max_mem
FROM (
    SELECT ResourceAttributes['net.host.name'] AS ip,
           argMaxIf(Value, TimeUnix, MetricName='node_memory_MemTotal_bytes') AS total,
           argMaxIf(Value, TimeUnix, MetricName='node_memory_MemAvailable_bytes') AS avail
    FROM otel_metrics_gauge
    WHERE MetricName IN ('node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes')
      AND TimeUnix >= now() - INTERVAL 2 MINUTE
    GROUP BY ip
)
```

---

## 8. 已修复问题

> 以下问题已在 observe API 参数对齐修复中解决。保留记录供参考。

### 8.1 Metrics Series 参数不匹配 — ✅ 已修复

**问题：** Master 不传 `metric` 参数且传 `minutes` 而非 `since`，导致 Agent 始终返回错误。

**修复：**
- Master 转发前端 `metric` 参数
- Master 将 `minutes` 转换为 `since`（如 `minutes=30` → `since="30m"`）
- 前端 `getMetricsNodeSeries` 新增 `metric` 必需参数

### 8.2 SLO time_range 参数未透传到 since — ✅ 已修复

**问题：** Master 传 `time_range`，但 Agent 读 `since`，导致 SLO 查询始终使用默认 5 分钟窗口。

**修复：** Master 将 `time_range` → `since` 重命名后传递给 Agent（4 个 SLO 端点 + SLOTimeSeries 的 `service` → `name`）。

### 8.3 TracesList 参数类型不匹配 — ✅ 已修复

**问题：** Master 将 `min_duration` 作为 string 传入（参数名也不匹配），`limit`/`offset` 也是 string 类型，Agent 的 `getIntParam`/`getFloat64Param` 无法解析。

**修复：**
- Master 将 `min_duration` 重命名为 `min_duration_ms` 并解析为 float64
- Master 将 `limit`/`offset` 解析为 int 后传入
- Agent `getIntParam`/`getFloat64Param` 增加 `case string` 容错分支

---

## 查询统计

| 模块 | 文件 | 查询数 | 触发方式 |
|------|------|--------|---------|
| Traces | `ch/query/trace.go` | 5 | Command（按需） |
| Logs | `ch/query/log.go` | 5（3 并行 + 2 模板） | Command（按需） |
| Metrics | `ch/query/metrics.go` | 20+（10 并行子查询） | Command（按需） |
| SLO | `ch/query/slo.go` | 7 | Command（按需） |
| Summary | `ch/summary.go` | 5 | 定期快照上报 |
| **合计** | | **~42 个 SQL** | |

---

## 9. 实际数据量参考

> 基于 zgmf-x10a 集群实测（6 节点、6 服务、27 Linkerd deployments、6 Traefik services）。
> 采集配置：node-exporter 15s、Linkerd 30s、Traefik 10s。TTL 7 天。

### 9.1 存储概览

| 表 | 行数 | 压缩大小 | 说明 |
|----|------|---------|------|
| `otel_metrics_gauge` | 1.31 亿 | 7.35 GiB | 98%，主要是 `response_latency_ms_bucket`（65%） |
| `otel_metrics_sum` | 1589 万 | 107 MiB | node counter + traefik counter |
| `otel_traces` | 14.8 万 | 15 MiB | ~2 天数据 |
| `otel_logs` | 29.6 万 | 15 MiB | ~2 天数据 |
| `otel_metrics_histogram` | 158 万 | 13 MiB | traefik 延迟 histogram |
| **合计** | | **~7.5 GiB** | 预估 7 天满载 ~6 GiB（优化后） |

### 9.2 各查询返回数据量

#### Traces（默认窗口 5 分钟）

| 查询 | 返回行数 | 说明 |
|------|---------|------|
| ListTraces | 271 条 | 每条 1 个 Trace 聚合（6 服务 × 健康检查） |
| GetTraceDetail | 1 span | 当前服务仅单 span（无跨服务调用链） |
| ListServices | 6 条 | 6 个 geass 微服务 |
| GetTopology — 节点 | 6 条 | 同上 |
| GetTopology — 边 | 0 条 | 无跨服务 span（仅 SPAN_KIND_SERVER） |

#### Logs（默认窗口 15 分钟）

| 查询 | 返回行数 | 说明 |
|------|---------|------|
| QueryLogs 总数 | 1,620 条 | 6 服务 × ~270 条/服务 |
| Facets — services | 6 种 | 6 个服务名 |
| Facets — severities | 1 种 | 仅 INFO（无 ERROR/WARN） |
| Facets — scopes | 6 种 | 6 个 scope name |

#### Metrics

| 查询 | 返回行数 | 说明 |
|------|---------|------|
| 活跃节点数 | 6 个 | 6 个 IP |
| CPU rows/节点（5min） | 640~1280 | 取决于核心数（4 核 vs 8 核） |
| Series 数据点（30min） | 30 个 | 1 点/分钟 |

#### SLO（默认窗口 5 分钟）

| 查询 | 返回行数 | 说明 |
|------|---------|------|
| Ingress — 请求计数 | 36 组 | 6 service × ~6 status code |
| Ingress — histogram | 1,710 条 | 原始 bucket 行 |
| Service SLO — response | 72 组 | 27 deploy × status × tls 组合 |
| Service SLO — latency bucket | 728 组 | 27 deploy × 26 le 值（聚合后） |
| Service Edges | 50 组 | 服务间调用关系 |
| TimeSeries（单服务） | 4 点 | 5min / 300s 间隔 |

#### Summary（快照上报用）

| 查询 | 返回值 | 说明 |
|------|--------|------|
| APM — 服务数 | 6 total, 6 healthy | 全部健康（0 错误） |
| SLO — Ingress 服务数 | 5 | 5 个 Traefik 后端服务 |
| SLO — Mesh 服务数 | 27 | 27 个 Linkerd deployment |
| Metrics — 节点数 | 6 | 6 个节点 |

### 9.3 数据特征备注

| 特征 | 现状 | 影响 |
|------|------|------|
| Traces 仅 `SPAN_KIND_SERVER` | 无 CLIENT/INTERNAL span | 拓扑边查询始终为空 |
| Traces `service.namespace` 全空 | geass 服务未设置 namespace | ListServices 的 ns 列为空 |
| Logs 仅 INFO 级别 | 无 ERROR/WARN 日志 | Facets 只有 1 种 severity |
| `response_latency_ms_bucket` 占 gauge 表 65% | 26 le × 27 deploy × 2 direction | 存储主要消耗来源 |
