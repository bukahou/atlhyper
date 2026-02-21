# ClickHouse OTel 数据参考手册

> 数据源: 集群 `zgmf-x10a` / 数据库 `atlhyper`
> 采集时间: 2026-02-20
> OTel Collector: v0.96.0 / OTel Java Agent: v2.25.0
> 用途: AtlHyper 模型建立 + 前端 APM/监控页面开发

---

## 目录

**Part A: ClickHouse OTel 数据**

1. [数据总量](#1-数据总量)
2. [表结构 (Schema)](#2-表结构-schema)
3. [数据源分类](#3-数据源分类)
4. [Traces 数据](#4-traces-数据)
5. [Logs 数据](#5-logs-数据)
6. [Metrics — Gauge 表](#6-metrics--gauge-表)
7. [Metrics — Sum 表](#7-metrics--sum-表)
8. [Metrics — Histogram 表](#8-metrics--histogram-表)
9. [信号关联 (Signal Correlation)](#9-信号关联-signal-correlation)
10. [关键查询模式](#10-关键查询模式)

**Part B: Agent 采集数据（K8s API Server 独占）**

11. [Agent 数据模型](#11-agent-数据模型)
    - [11.1 ClusterSnapshot 总体结构](#111-clustersnapshot-总体结构)
    - [11.2 公共模型](#112-公共模型)
    - [11.3 工作负载](#113-工作负载) — Pod, Deployment, StatefulSet, DaemonSet, ReplicaSet, Job, CronJob
    - [11.4 网络](#114-网络) — Service, Ingress
    - [11.5 配置](#115-配置) — Namespace, ConfigMap, Secret
    - [11.6 策略](#116-策略) — ResourceQuota, LimitRange, NetworkPolicy, ServiceAccount
    - [11.7 存储](#117-存储) — PersistentVolume, PersistentVolumeClaim
    - [11.8 集群](#118-集群) — Node, Event
    - [11.9 ClusterSummary](#119-clustersummary)

---

## 1. 数据总量

| 表名 | 行数 | 说明 |
|------|------|------|
| `otel_traces` | 819 | 分布式链路追踪 Span |
| `otel_logs` | 1,592 | 应用日志（OTel SDK 采集） |
| `otel_metrics_gauge` | 486,980 | Gauge 型指标（瞬时值） |
| `otel_metrics_sum` | 52,394 | Sum 型指标（累计值） |
| `otel_metrics_histogram` | 5,184 | Histogram 型指标（延迟分布） |

---

## 2. 表结构 (Schema)

### 2.1 otel_traces

| 列名 | 类型 | 说明 |
|------|------|------|
| `Timestamp` | DateTime64(9) | Span 开始时间（纳秒精度） |
| `TraceId` | String | 链路 ID（32 字符 hex） |
| `SpanId` | String | Span ID（16 字符 hex） |
| `ParentSpanId` | String | 父 Span ID（空 = 根 Span） |
| `TraceState` | String | W3C TraceState |
| `SpanName` | LowCardinality(String) | 操作名（如 `POST /api/history/list`） |
| `SpanKind` | LowCardinality(String) | `SPAN_KIND_SERVER` / `SPAN_KIND_CLIENT` |
| `ServiceName` | LowCardinality(String) | 服务名（如 `geass-gateway`） |
| `ResourceAttributes` | Map(LowCardinality(String), String) | 资源属性（服务实例信息） |
| `ScopeName` | String | 插桩库名（如 `io.opentelemetry.tomcat-7.0`） |
| `ScopeVersion` | String | 插桩库版本 |
| `SpanAttributes` | Map(LowCardinality(String), String) | Span 属性（HTTP/DB 详情） |
| `Duration` | Int64 | 耗时（纳秒） |
| `StatusCode` | LowCardinality(String) | `STATUS_CODE_UNSET` / `STATUS_CODE_ERROR` |
| `StatusMessage` | String | 错误消息 |
| `Events.Timestamp` | Array(DateTime64(9)) | 事件时间戳数组 |
| `Events.Name` | Array(LowCardinality(String)) | 事件名数组 |
| `Events.Attributes` | Array(Map(...)) | 事件属性数组 |
| `Links.TraceId` | Array(String) | 关联链路 ID 数组 |
| `Links.SpanId` | Array(String) | 关联 Span ID 数组 |
| `Links.TraceState` | Array(String) | 关联 TraceState 数组 |
| `Links.Attributes` | Array(Map(...)) | 关联属性数组 |

### 2.2 otel_logs

| 列名 | 类型 | 说明 |
|------|------|------|
| `Timestamp` | DateTime64(9) | 日志时间 |
| `TraceId` | String | 关联链路 ID（**关联键**） |
| `SpanId` | String | 关联 Span ID（**关联键**） |
| `TraceFlags` | UInt32 | Trace 标志（1 = sampled） |
| `SeverityText` | LowCardinality(String) | `INFO` / `DEBUG` / `WARN` / `ERROR` |
| `SeverityNumber` | Int32 | 严重级别数字（9=INFO, 5=DEBUG） |
| `ServiceName` | LowCardinality(String) | 服务名 |
| `Body` | String | 日志内容 |
| `ResourceSchemaUrl` | String | 资源 Schema URL |
| `ResourceAttributes` | Map(LowCardinality(String), String) | 资源属性 |
| `ScopeSchemaUrl` | String | Scope Schema URL |
| `ScopeName` | String | 日志来源类（如 `com.geass.gateway.util.RemoteCaller`） |
| `ScopeVersion` | String | Scope 版本 |
| `ScopeAttributes` | Map(LowCardinality(String), String) | Scope 属性 |
| `LogAttributes` | Map(LowCardinality(String), String) | 日志属性 |

### 2.3 otel_metrics_gauge

| 列名 | 类型 | 说明 |
|------|------|------|
| `ResourceAttributes` | Map(LowCardinality(String), String) | 数据源资源属性 |
| `ResourceSchemaUrl` | String | 资源 Schema URL |
| `ScopeName` | String | 采集器名（如 `otelcol/prometheusreceiver`） |
| `ScopeVersion` | String | 采集器版本 |
| `ScopeAttributes` | Map(LowCardinality(String), String) | Scope 属性 |
| `ScopeDroppedAttrCount` | UInt32 | 丢弃属性计数 |
| `ScopeSchemaUrl` | String | Scope Schema URL |
| `MetricName` | String | 指标名（如 `node_load1`） |
| `MetricDescription` | String | 指标描述 |
| `MetricUnit` | String | 单位 |
| `Attributes` | Map(LowCardinality(String), String) | 标签键值对 |
| `StartTimeUnix` | DateTime64(9) | 开始时间 |
| `TimeUnix` | DateTime64(9) | 采样时间 |
| `Value` | Float64 | 指标值 |
| `Flags` | UInt32 | 标志位 |
| `Exemplars.*` | Array(...) | Exemplar 数组（FilteredAttributes, TimeUnix, Value, SpanId, TraceId） |

### 2.4 otel_metrics_sum

与 `otel_metrics_gauge` 结构相同，额外增加：

| 列名 | 类型 | 说明 |
|------|------|------|
| `AggTemp` | Int32 | 聚合时间性（2 = Cumulative） |
| `IsMonotonic` | Bool | 是否单调递增 |

### 2.5 otel_metrics_histogram

| 列名 | 类型 | 说明 |
|------|------|------|
| `ResourceAttributes` | Map(...) | 同 gauge |
| `ScopeName` | String | 采集器名 |
| `MetricName` | String | 指标名 |
| `MetricDescription` | String | 指标描述 |
| `Attributes` | Map(...) | 标签 |
| `StartTimeUnix` | DateTime64(9) | 开始时间 |
| `TimeUnix` | DateTime64(9) | 采样时间 |
| `Count` | UInt64 | 样本总数 |
| `Sum` | Float64 | 值的总和 |
| `BucketCounts` | Array(UInt64) | 各桶计数 |
| `ExplicitBounds` | Array(Float64) | 桶边界值 |
| `Flags` | UInt32 | 标志位 |
| `Min` | Float64 | 最小值 |
| `Max` | Float64 | 最大值 |
| `Exemplars.*` | Array(...) | Exemplar 数组 |

---

## 3. 数据源分类

### 3.1 Traces 来源（OTel Java Agent 自动插桩）

| 服务 | Span 数 | 说明 |
|------|---------|------|
| geass-media | 237 | 媒体服务（数据库查询最多） |
| geass-gateway | 235 | API 网关（下游调用最多） |
| geass-auth | 127 | 认证服务（Token 验证） |
| geass-favorites | 118 | 收藏服务 |
| geass-history | 114 | 播放历史服务 |
| geass-user | 109 | 用户服务 |

### 3.2 Logs 来源（OTel Java Agent → OTLP）

| 服务 | 级别 | 数量 | 说明 |
|------|------|------|------|
| geass-gateway | INFO | 384 | LoggingFilter + RemoteCaller + AuthVerifyFilter |
| geass-media | INFO | 288 | LoggingFilter |
| geass-media | DEBUG | 273 | MyBatis SQL 日志 |
| geass-auth | INFO | 256 | LoggingFilter + TokenServiceImpl |
| geass-favorites | INFO | 214 | LoggingFilter + FavoriteServiceImpl |
| geass-history | INFO | 211 | LoggingFilter + HistoryServiceImpl |
| geass-user | INFO | 208 | LoggingFilter + AuthenticationServiceImpl |

### 3.3 Metrics 来源（Prometheus Scrape → OTel Collector）

| 数据源 | service.name | 表 | 指标类型 |
|--------|-------------|-----|---------|
| **Linkerd 服务网格** | `linkerd-prometheus` | gauge | response_total, request_total, response_latency_ms_*, tcp_*, container_* |
| **Node Exporter** | `node-exporter` | gauge + sum | node_cpu_*, node_memory_*, node_filesystem_*, node_disk_*, node_network_*, node_load*, node_hwmon_temp_* |
| **Traefik Ingress** | `traefik` | sum + histogram | traefik_entrypoint_*, traefik_service_* |

---

## 4. Traces 数据

### 4.1 SpanName 分布

| 服务 | SpanName | SpanKind | 数量 | 说明 |
|------|----------|----------|------|------|
| geass-gateway | `POST` | CLIENT | 75 | 下游调用（RemoteCaller） |
| geass-auth | `POST /token/verify` | SERVER | 26 | Token 验证 |
| geass-media | `SELECT geass_v2.ura_anime` | CLIENT | 25 | 数据库查询 |
| geass-media | `SELECT geass_v2.anime` | CLIENT | 23 | 数据库查询 |
| geass-media | `SELECT geass_v2.av` | CLIENT | 19 | 数据库查询 |
| geass-media | `SELECT geass_v2.drama` | CLIENT | 12 | 数据库查询 |
| geass-favorites | `SELECT geass_v2.favorites` | CLIENT | 11 | 数据库查询 |
| geass-gateway | `GET` | CLIENT | 9 | 下游 GET 调用 |
| geass-history | `SELECT geass_v2.history` | CLIENT | 9 | 数据库查询 |
| geass-gateway | `POST /api/ura-anime/sort/release` | SERVER | 8 | API 端点 |
| geass-media | `POST /ura-anime/sort/release` | SERVER | 8 | 业务端点 |
| geass-media | `POST /batch/fetch` | SERVER | 8 | 批量获取 |
| geass-gateway | `POST /public/anime/sort/release` | SERVER | 7 | 公开 API |
| geass-media | `POST /anime/v2/sort/release` | SERVER | 7 | 业务端点 |
| geass-user | `POST /auth/login` | SERVER | 2 | 登录 |
| geass-user | `INSERT geass_v2.user_logins` | CLIENT | 2 | 登录记录写入 |

### 4.2 Trace 样本（完整链路）

#### 样本: `POST /api/history/list` (TraceId: `25db41ab52036ba19daf3588e22ac3ad`)

```
geass-gateway (SERVER)  POST /api/history/list     Duration: 87ms
├── geass-gateway (CLIENT)  POST geass-auth:8085/token/verify   Duration: 6ms
│   └── geass-auth (SERVER)  POST /token/verify                  Duration: 2ms
├── geass-gateway (CLIENT)  POST geass-history:8083/history/v2/list  Duration: 35ms
│   └── geass-history (SERVER)  POST /history/v2/list             Duration: 32ms
│       ├── geass-history (CLIENT)  SELECT geass_v2.history       Duration: 3ms (列表查询)
│       └── geass-history (CLIENT)  SELECT geass_v2.history       Duration: 2ms (COUNT 查询)
└── geass-gateway (CLIENT)  POST geass-media:8081/batch/fetch     Duration: 30ms
    └── geass-media (SERVER)  POST /batch/fetch                   Duration: 26ms
        ├── geass-media (CLIENT)  SELECT geass_v2.anime           Duration: 1.9ms
        ├── geass-media (CLIENT)  SELECT geass_v2.av              Duration: 1.4ms
        ├── geass-media (CLIENT)  SELECT geass_v2.ura_anime       Duration: 1.2ms
        └── geass-media (CLIENT)  SELECT geass_v2.drama           Duration: 1.3ms
```

### 4.3 关键 SpanAttributes

#### HTTP Server Span

```json
{
  "http.request.method": "POST",
  "http.route": "/api/history/list",
  "http.response.status_code": "200",
  "url.path": "/api/history/list",
  "url.scheme": "https",
  "server.address": "geass-gateway",
  "server.port": "8080",
  "client.address": "10.42.2.123",
  "network.peer.address": "10.42.1.125",
  "network.peer.port": "52970",
  "network.protocol.version": "1.1",
  "user_agent.original": "Mozilla/5.0 ... Chrome/145.0.0.0 ...",
  "thread.name": "http-nio-8080-exec-4",
  "thread.id": "31"
}
```

#### HTTP Client Span (Gateway → 下游)

```json
{
  "http.request.method": "POST",
  "http.response.status_code": "200",
  "url.full": "http://geass-history:8083/history/v2/list",
  "server.address": "geass-history",
  "server.port": "8083",
  "network.protocol.version": "1.1",
  "thread.name": "http-nio-8080-exec-4",
  "thread.id": "31"
}
```

#### 数据库 Span (JDBC)

```json
{
  "db.system": "mysql",
  "db.name": "geass_v2",
  "db.operation": "SELECT",
  "db.sql.table": "history",
  "db.statement": "SELECT * FROM history WHERE user_id = ? ORDER BY played_at DESC LIMIT ? OFFSET ?",
  "db.connection_string": "mysql://192.168.0.182:3306",
  "db.user": "bukahou",
  "server.address": "192.168.0.182",
  "server.port": "3306",
  "thread.name": "http-nio-8083-exec-2",
  "thread.id": "29"
}
```

### 4.4 ResourceAttributes（所有 Trace/Log 共用）

```json
{
  "service.name": "geass-gateway",
  "service.version": "0.0.1-SNAPSHOT",
  "service.instance.id": "b1b58849-73a8-4746-b25c-7a51afab663d",
  "deployment.environment": "production",
  "cluster.name": "zgmf-x10a",
  "host.name": "geass-gateway-ff5988887-fmztl",
  "host.arch": "amd64",
  "os.type": "linux",
  "os.description": "Linux 6.8.0-85-generic",
  "process.pid": "7",
  "process.executable.path": "/opt/java/openjdk/bin/java",
  "process.command_args": "[\"...\",\"-javaagent:/otel/opentelemetry-javaagent.jar\",\"-jar\",\"/app/app.jar\"]",
  "process.runtime.name": "OpenJDK Runtime Environment",
  "process.runtime.version": "17.0.18+8",
  "process.runtime.description": "Eclipse Adoptium OpenJDK 64-Bit Server VM 17.0.18+8",
  "telemetry.sdk.name": "opentelemetry",
  "telemetry.sdk.language": "java",
  "telemetry.sdk.version": "1.59.0",
  "telemetry.distro.name": "opentelemetry-java-instrumentation",
  "telemetry.distro.version": "2.25.0"
}
```

### 4.5 ScopeName 映射

| ScopeName | 含义 |
|-----------|------|
| `io.opentelemetry.tomcat-7.0` | HTTP Server（Tomcat 入站请求） |
| `io.opentelemetry.http-url-connection` | HTTP Client（出站请求） |
| `io.opentelemetry.jdbc` | 数据库查询（JDBC 自动插桩） |

---

## 5. Logs 数据

### 5.1 ScopeName 分布（日志来源类）

| ScopeName | 数量 | 类别 |
|-----------|------|------|
| `com.geass.gateway.common.LoggingFilter` | 300 | HTTP 进出日志 |
| `com.geass.media.common.LoggingFilter` | 288 | HTTP 进出日志 |
| `com.geass.auth.common.LoggingFilter` | 254 | HTTP 进出日志 |
| `com.geass.favorites.common.LoggingFilter` | 212 | HTTP 进出日志 |
| `com.geass.history.common.LoggingFilter` | 210 | HTTP 进出日志 |
| `com.geass.user.common.LoggingFilter` | 204 | HTTP 进出日志 |
| `com.geass.gateway.util.RemoteCaller` | 58 | **下游调用日志** |
| `com.geass.gateway.filter.AuthVerifyFilter` | 26 | **认证日志** |
| `com.geass.media.mapper.*.selectByIds` | 24 | MyBatis SQL（DEBUG） |
| `com.geass.media.mapper.*.countAll` | ~20 | MyBatis SQL（DEBUG） |
| `com.geass.user.service.impl.AuthenticationServiceImpl` | 4 | **登录业务日志** |
| `com.geass.auth.service.impl.TokenServiceImpl` | 2 | **Token 业务日志** |
| `com.geass.favorites.service.impl.FavoriteServiceImpl` | 2 | **收藏业务日志** |
| `com.geass.history.service.impl.HistoryServiceImpl` | 1 | **播放业务日志** |

### 5.2 日志类别与 Body 格式

#### LoggingFilter（HTTP 进出）
```
➡️ [POST] /api/history/list from 10.42.2.123
⬅️ [POST] /api/history/list - 77ms (status=200)
```

#### RemoteCaller（下游调用）
```
[Downstream] POST http://geass-history:8083/history/v2/list -> 200 (38ms)
[Downstream] POST http://geass-media:8081/batch/fetch -> 200 (34ms)
```

#### AuthVerifyFilter（认证）
```
[Auth] userId=1, role=3, uri=/api/history/list
```

#### AuthenticationServiceImpl（登录）
```
[Login] Attempt: username=xxx
[Login] Success: userId=1, username=xxx
[Login] Failed: username=xxx, reason=密码错误
```

#### MyBatis SQL（DEBUG 级别）
```
==>  Preparing: SELECT * FROM anime WHERE id IN ( ? , ? , ? )
==> Parameters: 23(Long), 39(Long), 10(Long)
<==      Total: 16
```

### 5.3 日志样本

```json
{
  "Timestamp": "2026-02-20 13:56:02.025000000",
  "TraceId": "25db41ab52036ba19daf3588e22ac3ad",
  "SpanId": "124cd8a7dfed717e",
  "TraceFlags": 1,
  "SeverityText": "INFO",
  "SeverityNumber": 9,
  "ServiceName": "geass-gateway",
  "Body": "[Auth] userId=1, role=3, uri=/api/history/list",
  "ResourceSchemaUrl": "https://opentelemetry.io/schemas/1.24.0",
  "ScopeName": "com.geass.gateway.filter.AuthVerifyFilter",
  "ScopeVersion": "",
  "ScopeAttributes": {},
  "LogAttributes": {}
}
```

---

## 6. Metrics — Gauge 表

存储瞬时值指标，每个时间点一个 Value。

### 6.1 指标清单

#### Linkerd 服务网格指标（来源: `linkerd-prometheus`）

| 指标名 | 数据量 | 说明 |
|--------|--------|------|
| `response_latency_ms_bucket` | 406,900 | 响应延迟分布桶（最大量） |
| `response_latency_ms_count` | 15,650 | 响应延迟计数 |
| `response_latency_ms_sum` | 15,650 | 响应延迟总和 |
| `response_total` | 16,149 | 总响应数 |
| `request_total` | 14,212 | 总请求数 |
| `tcp_open_connections` | 22,655 | TCP 打开连接数 |
| `tcp_read_bytes_total` | 22,655 | TCP 读取字节总数 |
| `tcp_write_bytes_total` | 22,655 | TCP 写入字节总数 |
| `container_cpu_usage_seconds_total` | 16,333 | 容器 CPU 使用 |
| `container_memory_working_set_bytes` | 16,334 | 容器内存使用 |
| `container_network_receive_bytes_total` | 8,351 | 容器网络接收 |
| `container_network_transmit_bytes_total` | 8,351 | 容器网络发送 |

#### Node Exporter 硬件指标（来源: `node-exporter`）

| 指标名 | 数据量 | 说明 |
|--------|--------|------|
| `node_load1` / `node_load5` / `node_load15` | 270 each | 系统负载 |
| `node_memory_MemTotal_bytes` | 270 | 总内存 |
| `node_memory_MemAvailable_bytes` | 270 | 可用内存 |
| `node_memory_MemFree_bytes` | 270 | 空闲内存 |
| `node_memory_Cached_bytes` | 270 | 缓存内存 |
| `node_memory_Buffers_bytes` | 270 | Buffer 内存 |
| `node_memory_SwapTotal_bytes` | 270 | Swap 总量 |
| `node_memory_SwapFree_bytes` | 270 | Swap 空闲 |
| `node_filesystem_size_bytes` | 3,694 | 文件系统大小 |
| `node_filesystem_avail_bytes` | 3,694 | 文件系统可用 |
| `node_hwmon_temp_celsius` | 1,395 | 硬件温度 |
| `node_hwmon_temp_max_celsius` | 990 | 温度阈值 (max) |
| `node_hwmon_temp_crit_celsius` | 810 | 温度阈值 (critical) |
| `node_cpu_scaling_frequency_hertz` | 1,620 | CPU 频率 |
| `node_network_mtu_bytes` | 720 | 网络 MTU |
| `node_network_up` | 720 | 网络接口状态 |
| `node_network_speed_bytes` | 315 | 网络速度 |
| `node_boot_time_seconds` | 270 | 启动时间 |
| `node_entropy_available_bits` | 270 | 熵可用位 |
| `node_filefd_allocated` / `node_filefd_maximum` | 270 each | 文件描述符 |
| `node_nf_conntrack_entries` / `_limit` | 270 each | 连接跟踪 |
| `node_netstat_Tcp_CurrEstab` | 270 | TCP 已建立连接 |
| `node_sockstat_TCP_*` | 270 each | Socket 统计 |
| `node_vmstat_pgfault` / `pgmajfault` | 270 each | 页面故障 |
| `node_vmstat_pswpin` / `pswpout` | 270 each | Swap I/O |
| `node_timex_offset_seconds` | 270 | 时钟偏移 |
| `node_uname_info` | 270 | 内核信息 |

### 6.2 Gauge 样本

#### Node Exporter 样本

```json
{
  "ResourceAttributes": {
    "net.host.name": "192.168.0.46",
    "service.instance.id": "192.168.0.46:9100",
    "net.host.port": "9100",
    "http.scheme": "http",
    "cluster.name": "zgmf-x10a",
    "service.name": "node-exporter"
  },
  "ScopeName": "otelcol/prometheusreceiver",
  "ScopeVersion": "0.96.0",
  "MetricName": "node_load1",
  "MetricDescription": "1m load average.",
  "Attributes": {},
  "TimeUnix": "2026-02-20 14:01:53.224000000",
  "Value": 0.2
}
```

#### Linkerd 响应延迟桶样本

```json
{
  "ResourceAttributes": {
    "service.name": "linkerd-prometheus",
    "cluster.name": "zgmf-x10a"
  },
  "MetricName": "response_latency_ms_bucket",
  "Attributes": {
    "le": "10",
    "deployment": "geass-auth",
    "namespace": "geass",
    "pod": "geass-auth-66f49f5c9d-7wfqk",
    "direction": "inbound",
    "status_code": "200",
    "tls": "no_identity",
    "srv_name": "all-unauthenticated",
    "route_name": "probe",
    "authz_name": "probe"
  },
  "Value": 201
}
```

#### 容器资源样本

```json
{
  "ResourceAttributes": {
    "service.name": "linkerd-prometheus",
    "cluster.name": "zgmf-x10a"
  },
  "MetricName": "container_memory_working_set_bytes",
  "Attributes": {
    "id": "/kubepods.slice",
    "kubernetes_io_hostname": "raspi-nfs",
    "namespace": "",
    "exported_instance": "raspi-nfs",
    "exported_job": "kubernetes-nodes-cadvisor"
  },
  "Value": 568340480
}
```

### 6.3 Linkerd Metrics 关键 Attributes

| Attribute | 说明 | 示例值 |
|-----------|------|--------|
| `deployment` | K8s Deployment 名 | `geass-auth`, `geass-media` |
| `namespace` | K8s Namespace | `geass` |
| `pod` | Pod 名 | `geass-auth-66f49f5c9d-7wfqk` |
| `direction` | 流量方向 | `inbound` / `outbound` |
| `status_code` | HTTP 状态码 | `200`, `404`, `500` |
| `tls` | TLS 状态 | `true`, `no_identity` |
| `srv_name` | Linkerd Server 名 | `all-unauthenticated` |
| `route_name` | 路由名 | `probe`, `default` |
| `authz_name` | 授权策略名 | `probe`, `all-unauthenticated` |
| `target_addr` | 目标地址 | `10.42.0.142:8085` |
| `le` | 延迟桶上限（ms） | `1`, `2`, `4`, `10`, `40`, `40000` |
| `client_id` | mTLS 客户端身份 | `prometheus.linkerd-viz.serviceaccount...` |

### 6.4 Container Metrics 关键 Attributes

| Attribute | 说明 | 示例值 |
|-----------|------|--------|
| `id` | cgroup 路径 | `/kubepods.slice/kubepods-burstable.slice/...` |
| `namespace` | K8s Namespace | `atlhyper`, `geass` |
| `pod` | Pod 名 | `node-exporter-lksqt` |
| `kubernetes_io_hostname` | 节点名 | `raspi-nfs` |
| `exported_instance` | Prometheus 实例 | `raspi-nfs` |
| `exported_job` | Prometheus Job | `kubernetes-nodes-cadvisor` |
| `cpu` | CPU 编号 | `total` |

---

## 7. Metrics — Sum 表

存储累积计数器指标，具有 `AggTemp`（聚合时间性）和 `IsMonotonic`（单调性）。

### 7.1 指标清单

#### Node Exporter 累计指标

| 指标名 | 数据量 | 说明 |
|--------|--------|------|
| `node_cpu_seconds_total` | 12,960 | CPU 时间（按 mode 分） |
| `node_network_receive_bytes_total` | 3,109 | 网络接收字节 |
| `node_network_transmit_bytes_total` | 3,109 | 网络发送字节 |
| `node_network_receive_packets_total` | 3,109 | 网络接收包数 |
| `node_network_transmit_packets_total` | 3,109 | 网络发送包数 |
| `node_network_receive_errs_total` | 3,109 | 网络接收错误 |
| `node_network_transmit_errs_total` | 3,109 | 网络发送错误 |
| `node_network_receive_drop_total` | 3,109 | 网络接收丢包 |
| `node_network_transmit_drop_total` | 3,109 | 网络发送丢包 |
| `node_disk_read_bytes_total` | 495 | 磁盘读字节 |
| `node_disk_written_bytes_total` | 495 | 磁盘写字节 |
| `node_disk_reads_completed_total` | 495 | 磁盘读完成数 |
| `node_disk_writes_completed_total` | 495 | 磁盘写完成数 |
| `node_disk_io_time_seconds_total` | 495 | 磁盘 I/O 时间 |
| `node_pressure_cpu_waiting_seconds_total` | 270 | PSI: CPU 等待 |
| `node_pressure_memory_*_seconds_total` | 270 each | PSI: 内存压力 |
| `node_pressure_io_*_seconds_total` | 270 each | PSI: I/O 压力 |
| `node_softnet_dropped_total` | 1,620 | 软中断丢包 |
| `node_softnet_times_squeezed_total` | 1,620 | 软中断挤压 |

#### Traefik Ingress 累计指标

| 指标名 | 数据量 | 说明 |
|--------|--------|------|
| `traefik_entrypoint_requests_total` | 2,747 | 入口点请求总数 |
| `traefik_entrypoint_requests_bytes_total` | 2,747 | 入口点请求字节 |
| `traefik_entrypoint_responses_bytes_total` | 2,747 | 入口点响应字节 |
| `traefik_entrypoint_requests_tls_total` | 469 | TLS 请求总数 |
| `traefik_service_requests_total` | 3,685 | 后端服务请求总数 |
| `traefik_service_requests_bytes_total` | 3,685 | 后端服务请求字节 |
| `traefik_service_responses_bytes_total` | 3,685 | 后端服务响应字节 |
| `traefik_service_requests_tls_total` | 670 | 后端 TLS 请求 |

### 7.2 Traefik Sum 样本

```json
{
  "ResourceAttributes": {
    "service.name": "traefik",
    "net.host.name": "traefik-metrics.kube-system.svc.cluster.local",
    "cluster.name": "zgmf-x10a"
  },
  "MetricName": "traefik_entrypoint_requests_total",
  "MetricDescription": "How many HTTP requests processed on an entrypoint, partitioned by status code, protocol, and method.",
  "Attributes": {
    "code": "200",
    "entrypoint": "websecure",
    "method": "POST",
    "protocol": "http"
  },
  "StartTimeUnix": "2026-02-20 13:51:25.768000000",
  "TimeUnix": "2026-02-20 14:01:45.768000000",
  "Value": 2395,
  "AggTemp": 2,
  "IsMonotonic": true
}
```

### 7.3 Traefik Attributes

| Attribute | 说明 | 示例值 |
|-----------|------|--------|
| `code` | HTTP 状态码 | `200`, `302`, `404`, `401` |
| `method` | HTTP 方法 | `GET`, `POST`, `PUT` |
| `protocol` | 协议 | `http` |
| `entrypoint` | Traefik 入口点 | `websecure` |
| `service` | 后端服务 | `geass-geass-web-3000@kubernetes` |

---

## 8. Metrics — Histogram 表

存储延迟分布数据，每条记录包含 BucketCounts 和 ExplicitBounds。

### 8.1 指标清单

| 指标名 | 数据量 | 说明 |
|--------|--------|------|
| `traefik_service_request_duration_seconds` | ~2,600 | 后端服务请求延迟分布 |
| `traefik_entrypoint_request_duration_seconds` | ~2,600 | 入口点请求延迟分布 |

### 8.2 Histogram 样本

```json
{
  "ResourceAttributes": {
    "service.name": "traefik",
    "cluster.name": "zgmf-x10a"
  },
  "MetricName": "traefik_service_request_duration_seconds",
  "MetricDescription": "How long it took to process the request on a service, partitioned by status code, protocol, and method.",
  "Attributes": {
    "code": "404",
    "method": "GET",
    "protocol": "http",
    "service": "atlhyper-atlhyper-web-3000@kubernetes"
  },
  "Count": "8762",
  "Sum": 47.408,
  "BucketCounts": ["6768", "1592", "282", "51", "22", "21", "13", "6", "3", "4", "0", "0", "0", "0", "0", "0"],
  "ExplicitBounds": [0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.3, 0.5, 0.75, 1, 2.5, 5, 10]
}
```

**解读**: 8762 个请求中:
- 6768 (77.2%) < 5ms
- 1592 (18.2%) < 10ms
- 282 (3.2%) < 25ms
- 平均延迟 = 47.408 / 8762 = 5.41ms

### 8.3 Histogram Attributes

| Attribute | 说明 | 示例值 |
|-----------|------|--------|
| `code` | HTTP 状态码 | `200`, `404`, `401` |
| `method` | HTTP 方法 | `GET`, `POST`, `HEAD` |
| `protocol` | 协议 | `http` |
| `service` | Traefik 后端服务 | `geass-geass-web-3000@kubernetes`, `atlhyper-atlhyper-web-3000@kubernetes` |

---

## 9. 信号关联 (Signal Correlation)

### 9.1 关联机制

Traces 和 Logs 通过 `TraceId` + `SpanId` 关联。OTel Java Agent 自动将当前 Span 上下文注入到日志中。

```
otel_traces.TraceId  ←→  otel_logs.TraceId
otel_traces.SpanId   ←→  otel_logs.SpanId
```

Metrics 与 Traces/Logs 没有直接的 TraceId 关联，但可以通过以下维度间接关联：
- `ServiceName`（Traces/Logs）↔ `deployment`/`pod`（Linkerd Metrics Attributes）
- `host.name`（ResourceAttributes）↔ `kubernetes_io_hostname`（Container Metrics Attributes）

### 9.2 关联查询示例

已验证的关联数据（TraceId: `25db41ab52036ba19daf3588e22ac3ad`）：

| 时间线 | 信号 | 服务 | 内容 |
|--------|------|------|------|
| 02.015 | Trace | geass-gateway | `POST /api/history/list` (根 Span) |
| 02.017 | Trace | geass-gateway | → `POST geass-auth:8085/token/verify` |
| 02.021 | Trace | geass-auth | `POST /token/verify` |
| 02.021 | **Log** | geass-auth | `➡️ [POST] /token/verify from 10.42.0.142` |
| 02.023 | **Log** | geass-auth | `⬅️ [POST] /token/verify - 2ms (status=200)` |
| 02.025 | **Log** | geass-gateway | `[Auth] userId=1, role=3, uri=/api/history/list` |
| 02.025 | **Log** | geass-gateway | `➡️ [POST] /api/history/list from 10.42.2.123` |
| 02.029 | Trace | geass-gateway | → `POST geass-history:8083/history/v2/list` |
| 02.030 | Trace | geass-history | `POST /history/v2/list` |
| 02.031 | **Log** | geass-history | `➡️ [POST] /history/v2/list from 10.42.3.62` |
| 02.035 | Trace | geass-history | `SELECT geass_v2.history` (列表) |
| 02.057 | Trace | geass-history | `SELECT geass_v2.history` (COUNT) |
| 02.062 | **Log** | geass-history | `⬅️ [POST] /history/v2/list - 31ms (status=200)` |
| 02.066 | **Log** | geass-gateway | `[Downstream] POST .../history/v2/list -> 200 (38ms)` |
| 02.067 | Trace | geass-gateway | → `POST geass-media:8081/batch/fetch` |
| 02.071 | Trace | geass-media | `POST /batch/fetch` |
| 02.071 | **Log** | geass-media | `➡️ [POST] /batch/fetch from 10.42.6.97` |
| 02.074 | **Log** | geass-media | `==> Preparing: SELECT * FROM anime WHERE id IN (...)` |
| 02.076 | Trace | geass-media | `SELECT geass_v2.anime` |
| 02.083 | Trace | geass-media | `SELECT geass_v2.av` |
| 02.087 | Trace | geass-media | `SELECT geass_v2.ura_anime` |
| 02.090 | Trace | geass-media | `SELECT geass_v2.drama` |
| 02.096 | **Log** | geass-media | `⬅️ [POST] /batch/fetch - 25ms (status=200)` |
| 02.100 | **Log** | geass-gateway | `[Downstream] POST .../batch/fetch -> 200 (34ms)` |
| 02.102 | **Log** | geass-gateway | `⬅️ [POST] /api/history/list - 77ms (status=200)` |

---

## 10. 关键查询模式

### 10.1 服务拓扑发现

```sql
-- 从 Trace 中发现服务间调用关系
SELECT
    t1.ServiceName AS caller,
    t2.ServiceName AS callee,
    t2.SpanName AS operation,
    count() AS call_count,
    avg(t2.Duration) / 1e6 AS avg_ms
FROM atlhyper.otel_traces t1
JOIN atlhyper.otel_traces t2 ON t1.SpanId = t2.ParentSpanId AND t1.TraceId = t2.TraceId
WHERE t1.ServiceName != t2.ServiceName
GROUP BY caller, callee, operation
ORDER BY call_count DESC
```

### 10.2 Trace-Log 关联查询

```sql
-- 获取某个 Trace 的所有日志
SELECT Timestamp, ServiceName, SeverityText, Body, SpanId
FROM atlhyper.otel_logs
WHERE TraceId = '25db41ab52036ba19daf3588e22ac3ad'
ORDER BY Timestamp
```

### 10.3 服务错误率

```sql
-- 从 Trace 计算服务错误率
SELECT
    ServiceName,
    countIf(SpanAttributes['http.response.status_code'] NOT IN ('200', '')) AS errors,
    count() AS total,
    round(errors / total * 100, 2) AS error_rate_pct
FROM atlhyper.otel_traces
WHERE SpanKind = 'SPAN_KIND_SERVER'
GROUP BY ServiceName
```

### 10.4 P50/P99 延迟

```sql
-- 从 Trace 计算服务端点延迟
SELECT
    ServiceName,
    SpanName,
    quantile(0.5)(Duration / 1e6) AS p50_ms,
    quantile(0.99)(Duration / 1e6) AS p99_ms,
    count() AS cnt
FROM atlhyper.otel_traces
WHERE SpanKind = 'SPAN_KIND_SERVER'
GROUP BY ServiceName, SpanName
ORDER BY cnt DESC
```

### 10.5 Node 内存使用率

```sql
-- 节点内存使用率（需要关联 Total 和 Available）
SELECT
    t.ResourceAttributes['net.host.name'] AS node,
    t.Value AS total_bytes,
    a.Value AS available_bytes,
    round((1 - a.Value / t.Value) * 100, 2) AS used_pct
FROM atlhyper.otel_metrics_gauge t
JOIN atlhyper.otel_metrics_gauge a
    ON t.ResourceAttributes['net.host.name'] = a.ResourceAttributes['net.host.name']
    AND t.TimeUnix = a.TimeUnix
WHERE t.MetricName = 'node_memory_MemTotal_bytes'
  AND a.MetricName = 'node_memory_MemAvailable_bytes'
ORDER BY t.TimeUnix DESC
LIMIT 10
```

### 10.6 Traefik 请求分布

```sql
-- Traefik 入口点按状态码统计
SELECT
    Attributes['code'] AS status_code,
    Attributes['method'] AS method,
    max(Value) AS total_requests
FROM atlhyper.otel_metrics_sum
WHERE MetricName = 'traefik_entrypoint_requests_total'
GROUP BY status_code, method
ORDER BY total_requests DESC
```

### 10.7 日志搜索

```sql
-- 按关键词搜索日志
SELECT Timestamp, ServiceName, SeverityText, Body, TraceId
FROM atlhyper.otel_logs
WHERE Body LIKE '%[Downstream]%'
  AND SeverityText = 'INFO'
ORDER BY Timestamp DESC
LIMIT 20
```

---

## 11. Agent 数据模型

Agent 每 ~30s 从 K8s API Server（含 metrics.k8s.io API）采集数据，打包为 `model_v2.ClusterSnapshot`，通过 HTTP POST (gzip) 上报 Master DataHub。

这些数据只有通过 K8s API Server 才能获取，ClickHouse 无此数据。源码路径: `model_v2/`

### 11.1 ClusterSnapshot 总体结构

**源文件**: `model_v2/snapshot.go`

```
ClusterSnapshot
├── ClusterID    string          // 集群标识
├── FetchedAt    time.Time       // 采集时间
│
├── [工作负载]
│   ├── Pods          []Pod
│   ├── Deployments   []Deployment
│   ├── StatefulSets  []StatefulSet
│   ├── DaemonSets    []DaemonSet
│   ├── ReplicaSets   []ReplicaSet
│   ├── Jobs          []Job
│   └── CronJobs      []CronJob
│
├── [网络]
│   ├── Services      []Service
│   └── Ingresses     []Ingress
│
├── [配置]
│   ├── Namespaces    []Namespace
│   ├── ConfigMaps    []ConfigMap
│   └── Secrets       []Secret
│
├── [策略]
│   ├── ResourceQuotas   []ResourceQuota
│   ├── LimitRanges      []LimitRange
│   ├── NetworkPolicies  []NetworkPolicy
│   └── ServiceAccounts  []ServiceAccount
│
├── [存储]
│   ├── PersistentVolumes       []PersistentVolume
│   └── PersistentVolumeClaims  []PersistentVolumeClaim
│
├── [集群]
│   ├── Nodes         []Node          // 含 metrics-server 数据（CPU/内存用量）
│   └── Events        []Event
│
└── [摘要]
    └── Summary       ClusterSummary
```

> **注**: `NodeMetrics` 和 `SLOData` 字段数据源为 OTel Collector (Prometheus)，已被 ClickHouse 覆盖（见 Part A §6-§8），不属于 API Server 独占数据。

---

### 11.2 公共模型

**源文件**: `model_v2/common.go`

#### CommonMeta（所有简单资源嵌入）

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| UID | string | `uid` | K8s 资源唯一 ID |
| Name | string | `name` | 资源名称 |
| Namespace | string | `namespace` | 命名空间（集群级资源为空） |
| Kind | string | `kind` | 资源类型 |
| NodeName | string | `node_name` | 所在 Node（Pod 填充） |
| OwnerKind | string | `owner_kind` | 所有者类型 |
| OwnerName | string | `owner_name` | 所有者名称 |
| Labels | map[string]string | `labels` | 标签 |
| CreatedAt | time.Time | `created_at` | 创建时间 |

#### ResourceRequirements（容器资源需求）

```
ResourceRequirements
├── Requests: ResourceList { CPU, Memory }
└── Limits:   ResourceList { CPU, Memory }
```

#### ResourceRef（Event 关联对象引用）

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| Kind | string | `kind` | 资源类型 |
| Namespace | string | `namespace` | 命名空间 |
| Name | string | `name` | 资源名称 |
| UID | string | `uid` | 资源 UID |

---

### 11.3 工作负载

#### Pod（嵌套结构）

**源文件**: `model_v2/pod.go`

```
Pod
├── Summary: PodSummary
│   ├── Name, Namespace, NodeName
│   ├── OwnerKind, OwnerName        // 关联 Deployment/DaemonSet 等
│   ├── CreatedAt, Age
│
├── Spec: PodSpec
│   ├── RestartPolicy, ServiceAccountName
│   ├── NodeSelector, Tolerations, Affinity
│   ├── DNSPolicy, HostNetwork, RuntimeClassName
│   ├── PriorityClassName, TerminationGracePeriodSeconds
│   └── ImagePullSecrets
│
├── Status: PodStatus
│   ├── Phase                       // Running, Pending, Succeeded, Failed, Unknown
│   ├── Ready                       // "2/3" 格式
│   ├── Restarts                    // 总重启次数
│   ├── QoSClass                    // Guaranteed, Burstable, BestEffort
│   ├── PodIP, PodIPs, HostIP
│   ├── Reason, Message             // Pending/Failed 原因
│   ├── Conditions []PodCondition
│   ├── CPUUsage, MemoryUsage       // metrics-server 数据
│
├── Containers []PodContainerDetail
│   ├── Name, Image, ImagePullPolicy
│   ├── Command, Args, WorkingDir
│   ├── Ports, Envs, VolumeMounts
│   ├── Requests, Limits            // 资源需求
│   ├── LivenessProbe, ReadinessProbe, StartupProbe
│   ├── State, StateReason, StateMessage  // running/waiting/terminated
│   ├── Ready, RestartCount
│   └── LastTerminationReason/Message/Time
│
├── InitContainers []PodContainerDetail
├── Volumes []VolumeSpec            // {Name, Type, Source}
├── Labels, Annotations
```

**辅助方法**: `IsRunning()`, `IsPending()`, `IsFailed()`, `IsReady()`, `HasRestarts()`

#### Deployment（嵌套结构）

**源文件**: `model_v2/deployment.go`

```
Deployment
├── Summary: DeploymentSummary
│   ├── Name, Namespace, Strategy
│   ├── Replicas, Updated, Ready, Available, Unavailable
│   ├── Paused, CreatedAt, Age, Selector
│
├── Spec: DeploymentSpec
│   ├── Replicas, Selector
│   ├── Strategy { Type, RollingUpdate { MaxUnavailable, MaxSurge } }
│   ├── MinReadySeconds, RevisionHistoryLimit, ProgressDeadlineSeconds
│
├── Template: PodTemplate
│   ├── Labels, Annotations
│   ├── Containers []ContainerDetail    // 同 PodContainerDetail 但无运行状态
│   ├── Volumes, ServiceAccountName
│   ├── NodeSelector, Tolerations, Affinity
│
├── Status: DeploymentStatus
│   ├── Replicas, UpdatedReplicas, ReadyReplicas
│   ├── AvailableReplicas, UnavailableReplicas
│   └── Conditions []DeploymentCondition
│
├── Rollout: *DeploymentRollout      // { Phase, Message, Badges }
├── ReplicaSets []ReplicaSetBrief    // 关联的 RS 简要信息
├── Labels, Annotations
```

**辅助方法**: `IsHealthy()`, `IsUpdating()`, `IsPaused()`

#### StatefulSet（嵌套结构）

**源文件**: `model_v2/workload.go`

```
StatefulSet
├── Summary: StatefulSetSummary
│   ├── Name, Namespace, Replicas, Ready, Current, Updated, Available
│   ├── CreatedAt, Age, ServiceName, Selector
│
├── Spec: StatefulSetSpec
│   ├── Replicas, ServiceName, PodManagementPolicy
│   ├── UpdateStrategy, RevisionHistoryLimit, MinReadySeconds
│   ├── PersistentVolumeClaimRetentionPolicy
│   ├── Selector, VolumeClaimTemplates
│
├── Template: PodTemplate
├── Status: StatefulSetStatus        // 同 DeploymentStatus 类似
├── Rollout: *WorkloadRollout        // { Phase, Message, Badges }
├── Labels, Annotations
```

**辅助方法**: `IsHealthy()`, `IsUpdating()`

#### DaemonSet（嵌套结构）

**源文件**: `model_v2/workload.go`

```
DaemonSet
├── Summary: DaemonSetSummary
│   ├── Name, Namespace
│   ├── DesiredNumberScheduled, CurrentNumberScheduled
│   ├── NumberReady, NumberAvailable, NumberUnavailable, NumberMisscheduled
│   ├── UpdatedNumberScheduled, CreatedAt, Age, Selector
│
├── Spec: DaemonSetSpec
│   ├── UpdateStrategy, MinReadySeconds, RevisionHistoryLimit, Selector
│
├── Template: PodTemplate
├── Status: DaemonSetStatus
├── Rollout: *WorkloadRollout
├── Labels, Annotations
```

**辅助方法**: `IsHealthy()`, `IsUpdating()`, `HasMisscheduled()`

#### ReplicaSet（简单结构，嵌入 CommonMeta）

**源文件**: `model_v2/deployment.go`

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | UID, Name, Namespace, Kind 等 |
| Replicas | int32 | `replicas` | 期望副本数 |
| ReadyReplicas | int32 | `ready_replicas` | 就绪副本数 |
| AvailableReplicas | int32 | `available_replicas` | 可用副本数 |
| Selector | map[string]string | `selector` | 标签选择器 |

#### Job（嵌入 CommonMeta）

**源文件**: `model_v2/job.go`

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | |
| Active | int32 | `active` | 运行中 Pod 数 |
| Succeeded | int32 | `succeeded` | 成功 Pod 数 |
| Failed | int32 | `failed` | 失败 Pod 数 |
| Complete | bool | `complete` | 是否完成 |
| Completions | *int32 | `completions` | 期望完成数 |
| Parallelism | *int32 | `parallelism` | 并行数 |
| BackoffLimit | *int32 | `backoff_limit` | 重试上限 |
| Template | PodTemplate | `template` | Pod 模板 |
| Conditions | []WorkloadCondition | `conditions` | 状态条件 |
| StartTime | *time.Time | `start_time` | 开始时间 |
| FinishTime | *time.Time | `finish_time` | 完成时间 |

#### CronJob（嵌入 CommonMeta）

**源文件**: `model_v2/job.go`

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | |
| Schedule | string | `schedule` | Cron 表达式 |
| Suspend | bool | `suspend` | 是否暂停 |
| ConcurrencyPolicy | string | `concurrency_policy` | Allow/Forbid/Replace |
| SuccessfulJobsHistoryLimit | *int32 | `successful_jobs_history_limit` | 成功保留数 |
| FailedJobsHistoryLimit | *int32 | `failed_jobs_history_limit` | 失败保留数 |
| Template | PodTemplate | `template` | Pod 模板 |
| ActiveJobs | int32 | `active_jobs` | 当前活跃 Job 数 |
| LastScheduleTime | *time.Time | `last_schedule_time` | 上次调度时间 |
| LastSuccessfulTime | *time.Time | `last_successful_time` | 上次成功时间 |

---

### 11.4 网络

#### Service（嵌套结构）

**源文件**: `model_v2/service.go`

```
Service
├── Summary: ServiceSummary
│   ├── Name, Namespace
│   ├── Type                        // ClusterIP, NodePort, LoadBalancer, ExternalName
│   ├── CreatedAt, Age
│   ├── PortsCount, HasSelector, Badges
│   ├── ClusterIP, ExternalName
│
├── Spec: ServiceSpec
│   ├── Type, SessionAffinity, SessionAffinityTimeoutSeconds
│   ├── ExternalTrafficPolicy, InternalTrafficPolicy
│   ├── IPFamilies, IPFamilyPolicy
│   ├── ClusterIPs, ExternalIPs
│   ├── LoadBalancerClass, LoadBalancerSourceRanges
│   ├── PublishNotReadyAddresses, AllocateLoadBalancerNodePorts
│   ├── HealthCheckNodePort, ExternalName
│
├── Ports []ServicePort             // { Name, Protocol, Port, TargetPort, NodePort, AppProtocol }
├── Selector map[string]string
│
├── Network: ServiceNetwork
│   ├── ClusterIPs, ExternalIPs, LoadBalancerIngress
│   ├── IPFamilies, IPFamilyPolicy
│   ├── ExternalTrafficPolicy, InternalTrafficPolicy
│
├── Backends: *ServiceBackends
│   ├── Summary { Ready, NotReady, Total, Slices, Updated }
│   ├── Ports []EndpointPort
│   └── Endpoints []BackendEndpoint  // { Address, Ready, NodeName, Zone, TargetRef }
│
├── Labels, Annotations
```

**辅助方法**: `IsClusterIP()`, `IsNodePort()`, `IsLoadBalancer()`, `IsHeadless()`, `HasEndpoints()`

#### Ingress（嵌套结构）

**源文件**: `model_v2/service.go`

```
Ingress
├── Summary: IngressSummary
│   ├── Name, Namespace, CreatedAt, Age
│   ├── IngressClass, HostsCount, PathsCount
│   ├── TLSEnabled, Hosts
│
├── Spec: IngressSpec
│   ├── IngressClassName
│   ├── DefaultBackend { Type, Service { Name, PortName, PortNumber }, Resource }
│   ├── Rules []IngressRule
│   │   └── { Host, Paths []{ Path, PathType, Backend } }
│   └── TLS []IngressTLS { Hosts, SecretName }
│
├── Status: IngressStatus { LoadBalancer }
├── Labels, Annotations
```

---

### 11.5 配置

#### Namespace（嵌套结构）

**源文件**: `model_v2/namespace.go`

```
Namespace
├── Summary: NamespaceSummary { Name, CreatedAt, Age }
├── Status:  NamespaceStatus { Phase }    // Active, Terminating
├── Resources: NamespaceResources
│   ├── Pods, PodsRunning, PodsPending, PodsFailed, PodsSucceeded
│   ├── Deployments, StatefulSets, DaemonSets, ReplicaSets, Jobs, CronJobs
│   ├── Services, Ingresses, NetworkPolicies
│   ├── ConfigMaps, Secrets, ServiceAccounts
│   └── PVCs
├── Quotas []ResourceQuota
├── LimitRanges []LimitRange
├── Labels, Annotations
```

#### ConfigMap（嵌入 CommonMeta）

**源文件**: `model_v2/namespace.go`

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | |
| DataKeys | []string | `data_keys` | 数据键列表（**不存储值**，安全考虑） |

#### Secret（嵌入 CommonMeta）

**源文件**: `model_v2/namespace.go`

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | |
| Type | string | `type` | Opaque, kubernetes.io/tls, kubernetes.io/dockerconfigjson 等 |
| DataKeys | []string | `data_keys` | 数据键列表（**不存储值**） |

---

### 11.6 策略

**源文件**: `model_v2/policy.go`

#### ResourceQuota

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| Name, Namespace | string | — | 基础标识 |
| CreatedAt, Age | string | — | 时间 |
| Scopes | []string | `scopes` | 配额范围 |
| Hard | map[string]string | `hard` | 硬限制（如 `cpu: "4"`, `memory: "8Gi"`） |
| Used | map[string]string | `used` | 已使用量 |

#### LimitRange

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| Name, Namespace | string | — | 基础标识 |
| Items | []LimitRangeItem | `items` | 限制项 |

LimitRangeItem: `{ Type, Default, DefaultRequest, Max, Min, MaxLimitRequestRatio }` — 均为 `map[string]string`

#### NetworkPolicy

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| Name, Namespace | string | — | 基础标识 |
| PodSelector | string | `podSelector` | JSON 字符串 |
| PolicyTypes | []string | `policyTypes` | Ingress/Egress |
| IngressRuleCount | int | `ingressRuleCount` | 入站规则数 |
| EgressRuleCount | int | `egressRuleCount` | 出站规则数 |
| IngressRules | []NetworkPolicyRule | `ingressRules` | 入站规则详情 |
| EgressRules | []NetworkPolicyRule | `egressRules` | 出站规则详情 |

NetworkPolicyRule: `{ Peers []{ Type, Selector, CIDR, Except }, Ports []{ Protocol, Port, EndPort } }`

#### ServiceAccount

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| Name, Namespace | string | — | 基础标识 |
| SecretsCount | int | `secretsCount` | 关联 Secret 数 |
| ImagePullSecretsCount | int | `imagePullSecretsCount` | 镜像拉取 Secret 数 |
| AutomountServiceAccountToken | *bool | `automountServiceAccountToken` | 自动挂载 token |
| SecretNames | []string | `secretNames` | Secret 名称列表 |
| ImagePullSecretNames | []string | `imagePullSecretNames` | 镜像拉取 Secret 列表 |

---

### 11.7 存储

**源文件**: `model_v2/storage.go`

#### PersistentVolume（嵌入 CommonMeta）

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | |
| Capacity | string | `capacity` | 容量（如 "10Gi"） |
| Phase | string | `phase` | Available, Bound, Released, Failed |
| StorageClass | string | `storage_class` | 存储类 |
| AccessModes | []string | `access_modes` | ReadWriteOnce, ReadOnlyMany 等 |
| ReclaimPolicy | string | `reclaim_policy` | Retain, Recycle, Delete |
| VolumeSourceType | string | `volume_source_type` | NFS, HostPath, CSI, Local |
| ClaimRefName | string | `claim_ref_name` | 绑定的 PVC 名 |
| ClaimRefNS | string | `claim_ref_namespace` | 绑定的 PVC 命名空间 |

#### PersistentVolumeClaim（嵌入 CommonMeta）

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | |
| Phase | string | `phase` | Pending, Bound, Lost |
| VolumeName | string | `volume_name` | 绑定的 PV 名 |
| StorageClass | string | `storage_class` | 存储类 |
| AccessModes | []string | `access_modes` | 访问模式 |
| RequestedCapacity | string | `requested_capacity` | 请求容量 |
| ActualCapacity | string | `actual_capacity` | 实际容量 |
| VolumeMode | string | `volume_mode` | Filesystem, Block |

---

### 11.8 集群

#### Node（嵌套结构）

**源文件**: `model_v2/node.go`

```
Node
├── Summary: NodeSummary
│   ├── Name, Roles, Ready          // Ready: "True"/"False"/"Unknown"
│   ├── Schedulable, Age, CreationTime
│   ├── Badges, Reason, Message
│
├── Spec: NodeSpec
│   ├── PodCIDRs, ProviderID, Unschedulable
│
├── Capacity: NodeResources          // { CPU, Memory, Pods, EphemeralStorage, ScalarResources }
├── Allocatable: NodeResources       // 同上，可分配量
│
├── Addresses: NodeAddresses
│   ├── Hostname, InternalIP, ExternalIP
│   └── All []NodeAddr
│
├── Info: NodeInfo
│   ├── OSImage, OperatingSystem, Architecture
│   ├── KernelVersion, ContainerRuntimeVersion
│   ├── KubeletVersion, KubeProxyVersion
│
├── Conditions []NodeCondition       // { Type, Status, Reason, Message, LastHeartbeatTime, LastTransitionTime }
├── Taints []NodeTaint               // { Key, Value, Effect, TimeAdded }
├── Labels
│
├── Metrics: *NodeMetrics
│   ├── CPU    { Usage, Allocatable, Capacity, UtilPct }
│   ├── Memory { Usage, Allocatable, Capacity, UtilPct }
│   ├── Pods   { Used, Capacity, UtilPct }
│   └── Pressure { MemoryPressure, DiskPressure, PIDPressure, NetworkUnavailable }
```

**辅助方法**: `IsReady()`, `IsSchedulable()`, `IsMaster()`

#### Event

**源文件**: `model_v2/event.go`

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| CommonMeta | (嵌入) | — | |
| Type | string | `type` | Normal, Warning |
| Reason | string | `reason` | 事件原因码 |
| Message | string | `message` | 详细信息 |
| Source | string | `source` | 来源组件（kubelet, scheduler 等） |
| InvolvedObject | ResourceRef | `involved_object` | 关联资源 |
| Count | int32 | `count` | 发生次数 |
| FirstTimestamp | time.Time | `first_timestamp` | 首次发生 |
| LastTimestamp | time.Time | `last_timestamp` | 最后发生 |

**严重事件判定** (`IsCritical()`)：Warning 类型 + 以下 Reason 之一：

`Failed`, `FailedScheduling`, `FailedMount`, `FailedAttachVolume`, `OOMKilled`, `BackOff`, `CrashLoopBackOff`, `Unhealthy`, `NodeNotReady`

**严重级别** (`GetSeverity()`): `"critical"` / `"warning"` / `"info"`

---

### 11.9 ClusterSummary

**源文件**: `model_v2/snapshot.go`

由 `ClusterSnapshot.GenerateSummary()` 自动生成。

| 字段 | 类型 | JSON | 说明 |
|------|------|------|------|
| TotalNodes | int | `total_nodes` | 总节点数 |
| ReadyNodes | int | `ready_nodes` | 就绪节点数 |
| TotalPods | int | `total_pods` | 总 Pod 数 |
| RunningPods | int | `running_pods` | Running 状态 |
| PendingPods | int | `pending_pods` | Pending 状态 |
| FailedPods | int | `failed_pods` | Failed 状态 |
| TotalDeployments | int | `total_deployments` | 总 Deployment 数 |
| HealthyDeployments | int | `healthy_deployments` | 健康 Deployment 数 |
| TotalStatefulSets | int | `total_statefulsets` | |
| TotalDaemonSets | int | `total_daemonsets` | |
| TotalServices | int | `total_services` | |
| TotalIngresses | int | `total_ingresses` | |
| TotalNamespaces | int | `total_namespaces` | |
| TotalEvents | int | `total_events` | 总事件数 |
| WarningEvents | int | `warning_events` | Warning 事件数 |
