# SLO OTel Agent 设计书

## 概要

将 Agent SLO 数据源从 Ingress Controller 直连改为 OTel Collector，采集 Linkerd 服务网格指标 + Ingress Controller 入口指标。

**核心变化**：

| 特性 | 说明 |
|---|---|
| 数据源 | OTel Collector `:8889/metrics` |
| 指标维度 | namespace/deployment（服务）+ serviceKey（入口） |
| 指标来源 | 双源: Linkerd (服务网格) + Ingress Controller (入口) |
| 额外能力 | 服务拓扑发现、mTLS 覆盖率 |
| 计算位置 | Agent 端 per-pod delta → service 聚合 |

**设计约束**：
- 必须经过 OTel Collector（统一数据入口，后续接入更多数据源）
- OTel 无 TSDB，输出裸累积值，Agent 端计算增量
- **先 per-pod delta，再聚合到 service**（避免 Pod 重启导致聚合值错误）

---

## 1. 文件夹结构

现有文件标 `(现有)`，新增/修改标 `← NEW` 或 `← 修改`。

```
atlhyper_agent_v2/
├── agent.go                          (现有)  ← 修改: OTel 模式依赖注入
│
├── config/
│   ├── types.go                      (现有)  ← 修改: SLOConfig 新增 OTel 字段
│   ├── loader.go                     (现有)
│   └── defaults.go                   (现有)  ← 修改: OTel 默认值
│
├── sdk/
│   ├── interfaces.go                 (现有)  ← 修改: 新增 OTelClient 接口
│   ├── types.go                      (现有)  ← 修改: 新增 OTelRawMetrics 等内部类型
│   └── impl/
│       ├── k8s/                      (现有)  不动
│       │   ├── client.go
│       │   ├── core.go
│       │   ├── apps.go
│       │   ├── batch.go
│       │   ├── networking.go
│       │   ├── metrics.go
│       │   └── generic.go
│       │
│       ├── ingress/                  (现有)
│       │   ├── client.go                      ← 删除: 旧 Ingress 直连逻辑
│       │   ├── discover.go                    ← 删除: 旧自动发现
│       │   ├── parser.go                      ← 删除: 旧 Ingress 解析
│       │   └── route_collector.go             (现有) 复用: 路由采集被新 SLORepository 内部调用
│       │
│       ├── otel/                     ← NEW 整个目录
│       │   ├── client.go                      OTelClient 实现: HTTP 采集 + 健康检查
│       │   └── parser.go                      Prometheus 文本解析 → OTelRawMetrics
│       │
│       └── receiver/                 (现有)  不动
│           └── server.go
│
├── repository/
│   ├── interfaces.go                 (现有)  ← 修改: SLORepository 去掉 CollectRoutes()
│   │
│   ├── k8s/                          (现有)  不动
│   │   ├── converter.go
│   │   ├── pod.go
│   │   ├── node.go
│   │   ├── deployment.go
│   │   ├── statefulset.go
│   │   ├── daemonset.go
│   │   ├── replicaset.go
│   │   ├── service.go
│   │   ├── ingress.go
│   │   ├── configmap.go
│   │   ├── secret.go
│   │   ├── namespace.go
│   │   ├── event.go
│   │   ├── job.go
│   │   ├── cronjob.go
│   │   ├── pv.go
│   │   ├── pvc.go
│   │   ├── policy.go
│   │   └── generic.go
│   │
│   ├── slo/                          (现有目录)
│   │   ├── slo.go                    (现有)  ← 重写: SLORepository 主入口，编排 5 个 stage
│   │   ├── types.go                  ← NEW   内部增量类型定义 (delta 中间结构体)
│   │   ├── filter.go                 ← NEW   Stage 1: 过滤 probe/admin/系统ns
│   │   ├── snapshot.go               ← NEW   Stage 2: snapshotManager per-pod delta
│   │   ├── aggregate.go              ← NEW   Stage 3: Pod→Service 聚合 + Edge + Ingress
│   │   └── (不再需要 legacy.go)
│   │
│   └── metrics/                      (现有)  不动
│       └── metrics.go
│
├── service/
│   ├── interfaces.go                 (现有)  不动
│   ├── snapshot/
│   │   └── snapshot.go               (现有)  ← 修改: 删除 CollectRoutes 单独调用
│   └── command/                      (现有)  不动
│       ├── command.go
│       └── summary.go
│
├── gateway/                          (现有)  不动
│   ├── interfaces.go
│   └── master_gateway.go
│
├── scheduler/                        (现有)  不动
│   └── scheduler.go
│
└── model/                            (现有)  不动
    ├── options.go
    ├── command.go
    └── slo.go
```

**共享模型（Agent ↔ Master 合约）**:

```
model_v2/
├── slo.go                            (现有)  ← 重写: SLOSnapshot + ServiceMetrics
│                                              + ServiceEdge + IngressMetrics
│                                              旧类型删除
└── snapshot.go                       (现有)  不动: SLOData *SLOSnapshot 字段已存在
```

### 变更统计

| 操作 | 文件数 | 文件 |
|------|--------|------|
| **新建** | 6 | `sdk/impl/otel/client.go`, `sdk/impl/otel/parser.go`, `repository/slo/types.go`, `repository/slo/filter.go`, `repository/slo/snapshot.go`, `repository/slo/aggregate.go` |
| **重写** | 2 | `model_v2/slo.go`, `repository/slo/slo.go` |
| **删除** | 3 | `sdk/impl/ingress/client.go`, `sdk/impl/ingress/discover.go`, `sdk/impl/ingress/parser.go`（旧 Ingress 直连） |
| **修改** | 5 | `sdk/interfaces.go`, `sdk/types.go`, `repository/interfaces.go`, `config/types.go`, `agent.go` |
| **小改** | 2 | `config/defaults.go`, `service/snapshot/snapshot.go` |
| 不动 | ~35 | 其余所有文件 |

---

## 2. 数据流总览

```
┌─────────────────────────────────────────────────────────────────┐
│                        数据源                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Linkerd sidecar (per-pod)     Ingress Controller (Traefik/Nginx/...)  │
│       │                              │                          │
│       ▼                              │                          │
│  Linkerd Prometheus (:9090)          │                          │
│       │  /federate                   │                          │
│       ▼                              ▼                          │
│  ┌──────────────────────────────────────┐                      │
│  │      OTel Collector (:8889)          │                      │
│  │      输出: otel_ 前缀的 Prometheus   │                      │
│  └──────────────────┬───────────────────┘                      │
│                     │                                           │
└─────────────────────┼───────────────────────────────────────────┘
                      │ HTTP GET (Prometheus text format)
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Agent                                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  SDK: OTelClient.ScrapeMetrics()                                │
│       │  解析 Prometheus 文本 → OTelRawMetrics                   │
│       ▼                                                          │
│  Repository: SLORepository.Collect()                            │
│       ├── 1. filter     排除 probe/admin/系统ns                  │
│       ├── 2. per-pod delta   snapshotManager 算增量              │
│       ├── 3. aggregate  Pod → Service 聚合 (sum deltas)         │
│       ├── 4. topology   outbound → ServiceEdge                  │
│       └── 5. routes     IngressRoute CRD → RouteInfo            │
│       │                                                          │
│       ▼  SLOSnapshot (service 维度增量)                          │
│  Service: SnapshotService.Collect()                             │
│       │  嵌入 ClusterSnapshot.SLOData                           │
│       ▼                                                          │
│  Scheduler → HTTP POST → Master                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 数据模型 (model_v2/slo.go)

Agent 与 Master 的共享合约。Agent 发送，Master 接收并存储。

```go
package model_v2

// ============================================================
// SLO 快照 — Agent 从 OTel Collector 采集后上报
// ============================================================

// SLOSnapshot SLO 快照数据
// 嵌入 ClusterSnapshot.SLOData，随集群快照统一上报
type SLOSnapshot struct {
    Timestamp int64 `json:"timestamp"` // Unix 时间戳（秒）

    // 服务级黄金指标（Linkerd inbound，增量，已 per-pod delta + service 聚合）
    Services []ServiceMetrics `json:"services,omitempty"`

    // 服务调用拓扑（Linkerd outbound，增量）
    Edges []ServiceEdge `json:"edges,omitempty"`

    // 入口指标（增量，Agent 已将秒转为毫秒，Controller 无关）
    Ingress []IngressMetrics `json:"ingress,omitempty"`

    // 路由映射（K8s IngressRoute CRD / 标准 Ingress，不变）
    Routes []IngressRouteInfo `json:"routes,omitempty"`
}

// ============================================================
// 服务级黄金指标
// ============================================================

// ServiceMetrics 单个服务的黄金指标（增量）
// 维度: namespace + name (deployment/daemonset/statefulset)
// 来源: Linkerd inbound otel_response_total + otel_response_latency_ms_*
type ServiceMetrics struct {
    Namespace string `json:"namespace"`
    Name      string `json:"name"` // workload name (deployment 等)

    // 请求计数增量（按 status_code 分组）
    Requests []RequestDelta `json:"requests"`

    // 延迟直方图增量（毫秒，已跨 Pod 聚合）
    LatencyBuckets map[string]int64 `json:"latency_buckets,omitempty"` // le(ms string) → delta count
    LatencySum     float64          `json:"latency_sum"`               // 总和（毫秒）
    LatencyCount   int64            `json:"latency_count"`

    // mTLS 覆盖率（从 inbound otel_response_total 的 tls 标签聚合）
    TLSRequestDelta   int64 `json:"tls_request_delta"`   // tls="true" 的请求增量
    TotalRequestDelta int64 `json:"total_request_delta"` // 总请求增量（tls=true + tls=false）
    // Master 计算: mtlsPercent = TLSRequestDelta / TotalRequestDelta * 100
}

// RequestDelta 按状态码分组的请求增量
type RequestDelta struct {
    StatusCode     string `json:"status_code"`     // "200", "404", "503"
    Classification string `json:"classification"`  // "success" / "failure" (来自 Linkerd)
    Delta          int64  `json:"delta"`
}

// ============================================================
// 服务拓扑
// ============================================================

// ServiceEdge 服务调用边（增量）
// 来源: Linkerd outbound otel_response_total + otel_response_latency_ms_*
type ServiceEdge struct {
    SrcNamespace string  `json:"src_ns"`
    SrcName      string  `json:"src_name"`
    DstNamespace string  `json:"dst_ns"`
    DstName      string  `json:"dst_name"`
    RequestDelta int64   `json:"request_delta"`  // 增量请求数
    FailureDelta int64   `json:"failure_delta"`  // 失败请求增量（classification=failure）
    LatencySum   float64 `json:"latency_sum"`    // 延迟总和 (ms)
    LatencyCount int64   `json:"latency_count"`  // 延迟请求数
    // Master 计算:
    //   errorRate = FailureDelta / RequestDelta * 100
    //   avgLatency = LatencySum / LatencyCount
}

// ============================================================
// 入口指标（Ingress Controller 无关）
// ============================================================

// IngressMetrics 入口服务级指标（增量，Controller 无关）
// 维度: service_key（标准化格式: "namespace-service-port"）
//
// 支持的 Ingress Controller:
//   - Traefik: 从 otel_traefik_service_* 指标解析
//   - Nginx:   从 otel_nginx_ingress_controller_* 指标解析
//   - Kong:    从 otel_kong_* 指标解析（预留）
//
// Parser 负责将不同 Controller 的指标归一化到此结构
type IngressMetrics struct {
    ServiceKey string `json:"service_key"` // 标准化: "namespace-service-port"

    // 请求计数增量（按 code + method 分组）
    Requests []IngressRequestDelta `json:"requests"`

    // 延迟直方图增量（毫秒，Agent 已将秒转为毫秒）
    LatencyBuckets map[string]int64 `json:"latency_buckets,omitempty"` // le(ms string) → delta count
    LatencySum     float64          `json:"latency_sum"`               // 毫秒
    LatencyCount   int64            `json:"latency_count"`
}

// IngressRequestDelta 入口请求增量（Controller 无关）
type IngressRequestDelta struct {
    Code   string `json:"code"`   // HTTP 状态码
    Method string `json:"method"` // HTTP 方法
    Delta  int64  `json:"delta"`
}

// ============================================================
// 路由映射（保持现有结构不变）
// ============================================================

// IngressRouteInfo 保持不变
// ...
```

### 模型设计要点

1. **延迟统一为毫秒** — Linkerd 原生 ms，Ingress Controller 秒→ms（Agent 端转换）
2. **所有数值都是增量** — Agent 算好 delta 再上报，Master 不维护 counter snapshot
3. **Service 维度** — 不再有 host 维度，改为 namespace/name
4. **入口层 Controller 无关** — `IngressMetrics` / `IngressRequestDelta` 不绑定特定 Controller，Parser 负责归一化
5. **ServiceKey 标准化** — 统一为 `"namespace-service-port"` 格式，无论底层是 Traefik 还是 Nginx
6. **旧类型删除** — 旧 IngressMetrics、IngressCounterMetric、IngressHistogramMetric 直接删除，不再兼容

---

## 4. SDK 层

### 4.1 OTelClient 接口

**文件**: `sdk/interfaces.go`（在现有 IngressClient 下方新增）

```go
// OTelClient OTel Collector 采集客户端
//
// 从 OTel Collector 的 Prometheus 端点采集原始指标。
// 只做 HTTP 采集和文本解析，不做业务过滤/聚合。
//
// 架构位置:
//
//	SLORepository
//	    ↓ 调用
//	SDK (OTelClient)
//	    ↓ 使用
//	net/http
//	    ↓
//	OTel Collector (:8889/metrics)
type OTelClient interface {
    // ScrapeMetrics 从 OTel Collector 采集原始指标
    // 返回分类后的原始指标（per-pod 级别，累积值）
    ScrapeMetrics(ctx context.Context) (*OTelRawMetrics, error)

    // IsHealthy 检查 Collector 健康状态
    IsHealthy(ctx context.Context) bool
}
```

### 4.2 SDK 内部类型

**文件**: `sdk/types.go`（新增 OTel 相关类型）

```go
// ============================================================
// OTel Collector 原始指标（SDK 内部，不暴露给 Master）
// ============================================================

// OTelRawMetrics OTel 采集的原始指标（per-pod 级别）
type OTelRawMetrics struct {
    // Linkerd 请求计数 (otel_response_total)
    LinkerdResponses []LinkerdResponseMetric

    // Linkerd 延迟 (otel_response_latency_ms_bucket/sum/count)
    LinkerdLatencyBuckets []LinkerdLatencyBucketMetric
    LinkerdLatencySums    []LinkerdLatencySumMetric
    LinkerdLatencyCounts  []LinkerdLatencyCountMetric

    // 入口请求计数（Controller 无关，Parser 归一化后）
    IngressRequests []IngressRequestMetric

    // 入口延迟（Controller 无关，Parser 归一化后，单位: 秒）
    IngressLatencyBuckets []IngressLatencyBucketMetric
    IngressLatencySums    []IngressLatencySumMetric
    IngressLatencyCounts  []IngressLatencyCountMetric
}

// ---- Linkerd 类型 ----

// LinkerdResponseMetric otel_response_total 单条指标
type LinkerdResponseMetric struct {
    Namespace      string  // 源 pod 所在 namespace
    Deployment     string  // 源 deployment
    Pod            string  // 源 pod name
    Direction      string  // "inbound" / "outbound"
    StatusCode     string  // "200", "503"
    Classification string  // "success" / "failure"
    RouteName      string  // "default" / "probe"
    SrvPort        string  // 业务端口 "8200" / admin "4191"
    DstNamespace   string  // outbound: 目标 namespace
    DstDeployment  string  // outbound: 目标 deployment
    TLS            string  // "true" / "false"（inbound 的 mTLS 状态）
    Value          float64 // 累积值
}

// LinkerdLatencyBucketMetric otel_response_latency_ms_bucket 单条
type LinkerdLatencyBucketMetric struct {
    Namespace  string
    Deployment string
    Pod        string
    Direction  string
    Le         string  // bucket 边界 (ms): "1", "5", "100", "+Inf"
    Value      float64
}

// LinkerdLatencySumMetric otel_response_latency_ms_sum 单条
type LinkerdLatencySumMetric struct {
    Namespace  string
    Deployment string
    Pod        string
    Direction  string
    Value      float64 // 毫秒
}

// LinkerdLatencyCountMetric otel_response_latency_ms_count 单条
type LinkerdLatencyCountMetric struct {
    Namespace  string
    Deployment string
    Pod        string
    Direction  string
    Value      float64
}

// ---- 入口类型（Controller 无关） ----
//
// Parser 将不同 Controller 的原始指标归一化到以下通用结构。
// ServiceKey 统一为 "namespace-service-port" 格式。
//
// 归一化映射:
//   Traefik: service="ns-svc-port@kubernetes"  → ServiceKey="ns-svc-port"（去 @kubernetes 后缀）
//   Nginx:   namespace="ns", ingress="name", service="svc", service_port="port"
//            → ServiceKey="ns-svc-port"

// IngressRequestMetric 入口请求计数指标（归一化后）
type IngressRequestMetric struct {
    ServiceKey string  // 标准化: "namespace-service-port"
    Code       string  // "200"
    Method     string  // "GET"
    Value      float64 // 累积值
}

// IngressLatencyBucketMetric 入口延迟桶指标（归一化后）
type IngressLatencyBucketMetric struct {
    ServiceKey string
    Le         string  // bucket 边界 (秒): "0.1", "0.3", "5", "+Inf"
    Value      float64
}

// IngressLatencySumMetric 入口延迟总和指标（归一化后）
type IngressLatencySumMetric struct {
    ServiceKey string
    Value      float64 // 秒
}

// IngressLatencyCountMetric 入口延迟计数指标（归一化后）
type IngressLatencyCountMetric struct {
    ServiceKey string
    Value      float64
}
```

### 4.3 OTelClient 实现

**文件**: `sdk/impl/otel/client.go`

```go
package otel

// client OTelClient 实现
type client struct {
    metricsURL string       // http://otel-collector.otel.svc:8889/metrics
    healthURL  string       // http://otel-collector.otel.svc:13133
    httpClient *http.Client
}

// NewClient 创建 OTelClient
func NewOTelClient(metricsURL, healthURL string, timeout time.Duration) sdk.OTelClient

// ScrapeMetrics HTTP GET → 解析 Prometheus 文本 → 分类为 OTelRawMetrics
func (c *client) ScrapeMetrics(ctx context.Context) (*sdk.OTelRawMetrics, error)

// IsHealthy 检查 /healthz
func (c *client) IsHealthy(ctx context.Context) bool
```

**文件**: `sdk/impl/otel/parser.go`

```go
package otel

// parsePrometheus 解析 Prometheus 文本格式
// 逐行扫描，按指标名分发到对应结构体
//
// 处理逻辑:
// 1. 跳过 # 注释和空行
// 2. 正则提取: metric_name{labels} value
// 3. 按 metric_name 前缀分类:
//    - otel_response_total          → LinkerdResponseMetric
//    - otel_response_latency_ms_*   → LinkerdLatency*Metric
//    - 入口 Controller 指标（见下方前缀表）→ Ingress*Metric（归一化）
//    其他指标丢弃
// 4. 解析标签为 map，提取所需字段
// 5. 入口指标归一化: 不同 Controller 的标签映射到统一的 IngressRequestMetric 等
func parsePrometheus(r io.Reader) (*sdk.OTelRawMetrics, error)

// parseLabels 解析 key="value",key="value" 格式
func parseLabels(s string) map[string]string

// normalizeServiceKey 将不同 Controller 的 service 标识归一化
// Traefik: "atlantis-atlantis-web-3000@kubernetes" → "atlantis-atlantis-web-3000"
// Nginx:   namespace="atlantis", service="atlantis-web", service_port="3000"
//          → "atlantis-atlantis-web-3000"
func normalizeServiceKey(labels map[string]string, controllerType string) string
```

**Parser 指标名映射表**：

| OTel 输出指标名 | 解析目标 | 来源 |
|---|---|---|
| `otel_response_total` | `LinkerdResponseMetric` | Linkerd |
| `otel_response_latency_ms_bucket` | `LinkerdLatencyBucketMetric` | Linkerd |
| `otel_response_latency_ms_sum` | `LinkerdLatencySumMetric` | Linkerd |
| `otel_response_latency_ms_count` | `LinkerdLatencyCountMetric` | Linkerd |
| `otel_traefik_service_requests_total` | `IngressRequestMetric` (归一化) | Traefik |
| `otel_traefik_service_request_duration_seconds_bucket` | `IngressLatencyBucketMetric` (归一化) | Traefik |
| `otel_traefik_service_request_duration_seconds_sum` | `IngressLatencySumMetric` (归一化) | Traefik |
| `otel_traefik_service_request_duration_seconds_count` | `IngressLatencyCountMetric` (归一化) | Traefik |
| `otel_nginx_ingress_controller_requests` | `IngressRequestMetric` (归一化) | Nginx |
| `otel_nginx_ingress_controller_request_duration_seconds_bucket` | `IngressLatencyBucketMetric` (归一化) | Nginx |
| `otel_nginx_ingress_controller_request_duration_seconds_sum` | `IngressLatencySumMetric` (归一化) | Nginx |
| `otel_nginx_ingress_controller_request_duration_seconds_count` | `IngressLatencyCountMetric` (归一化) | Nginx |

**入口指标前缀检测**：Parser 按前缀自动识别 Controller 类型，无需配置：

```go
switch {
case strings.HasPrefix(name, "otel_traefik_"):
    // → 解析 Traefik 标签 → normalizeServiceKey → IngressRequestMetric
case strings.HasPrefix(name, "otel_nginx_ingress_"):
    // → 解析 Nginx 标签 → normalizeServiceKey → IngressRequestMetric
default:
    // 丢弃未知入口指标
}
```

**标签差异与归一化**：

| 字段 | Traefik 标签 | Nginx 标签 | 归一化结果 |
|---|---|---|---|
| ServiceKey | `service="ns-svc-port@kubernetes"` | `namespace`+`service`+`service_port` | `"ns-svc-port"` |
| Code | `code="200"` | `status="200"` | `"200"` |
| Method | `method="GET"` | `method="GET"` | `"GET"` |
| Le (bucket) | `le="0.1"` | `le="0.1"` | `"0.1"` (秒) |

---

## 5. Repository 层

### 5.1 处理管线

Repository 是 Agent 的核心处理层。按职责拆分为 4 个文件：

```
repository/slo/
├── slo.go         主入口: Collect() 编排各 stage
├── types.go       内部增量类型定义 (linkerdResponseDelta 等)
├── filter.go      Stage 1: 过滤
├── snapshot.go    Stage 2: per-pod delta
└── aggregate.go   Stage 3: 聚合 + 拓扑 + Ingress
```

处理管线：

```
OTelRawMetrics (per-pod 累积值)
      │
      │ Stage 1: Filter                          ← filter.go
      ▼
过滤后的 per-pod 数据
      │
      │ Stage 2: Per-pod Delta                   ← snapshot.go
      ▼
per-pod 增量值
      │
      ├── Linkerd inbound ── Stage 3a: Aggregate → ServiceMetrics[]        ┐
      │                                                                     │
      ├── Linkerd outbound ─ Stage 3b: Extract   → ServiceEdge[]           ├─ aggregate.go
      │                                                                     │
      └── Ingress ────────── Stage 3c: Aggregate → IngressMetrics[]        ┘
      │
      │ Stage 4: Collect Routes (IngressClient, 在 slo.go 中调用)
      ▼
SLOSnapshot
```

### 5.2 Stage 1: Filter（过滤）

```go
// 排除规则（按优先级）:
func (r *sloRepository) filter(raw *sdk.OTelRawMetrics) *sdk.OTelRawMetrics {
    // 1. route_name="probe" → K8s 健康检查流量，排除
    // 2. srv_port="4191"    → Linkerd proxy admin 端口，排除
    // 3. namespace ∈ excludeNamespaces → 系统 namespace，排除
    //    默认排除: linkerd, linkerd-viz, kube-system, otel
    // 4. direction: inbound 用于 ServiceMetrics, outbound 用于 Edge
}
```

### 5.3 Stage 2: Per-pod Delta（关键）

**为什么必须 per-pod delta？**

```
场景: 3 个 Pod，其中 Pod-C 重启

        Pod-A    Pod-B    Pod-C    聚合值
t=0     100      200      300      600
t=1     110      210      0(重启)  320

错误方式 (先聚合再 delta):
  delta = 320 - 600 = -280 → 当作重置 → 报 320 (错!)

正确方式 (先 delta 再聚合):
  Pod-A: 110-100 = 10
  Pod-B: 210-200 = 10
  Pod-C: 0 < 300, 重置, delta = 0 (首次采集跳过)
  聚合 = 20 (对!)
```

**SnapshotManager 设计**：

```go
// snapshotManager 维护 per-pod 级别的 prev 值
type snapshotManager struct {
    mu sync.RWMutex

    // Linkerd response per-pod
    // key: "pod|status_code|classification" → prev value
    linkerdResponsePrev map[string]float64

    // Linkerd latency per-pod
    // key: "pod|le" → prev bucket value
    linkerdBucketPrev map[string]float64
    // key: "pod" → prev sum
    linkerdSumPrev    map[string]float64
    // key: "pod" → prev count
    linkerdCountPrev  map[string]float64

    // Ingress per-service-key（Controller 无关，归一化后）
    // key: "service_key|code|method" → prev value
    ingressRequestPrev map[string]float64

    // Ingress latency per-service-key
    // key: "service_key|le" → prev bucket
    ingressBucketPrev map[string]float64
    // key: "service_key" → prev sum
    ingressSumPrev    map[string]float64
    // key: "service_key" → prev count
    ingressCountPrev  map[string]float64

    // Edge (outbound per-pod)
    // key: "pod|dst_ns|dst_name|status_code|classification" → prev total
    edgePrev map[string]float64
    // key: "pod|dst_ns|dst_name" → prev latency sum (ms)
    edgeSumPrev map[string]float64
    // key: "pod|dst_ns|dst_name" → prev latency count
    edgeCountPrev map[string]float64
}

// calcDelta 通用增量计算
// delta = current - prev; if delta < 0 → counter 重置, delta = 0 (跳过本周期)
func calcDelta(current, prev float64) float64 {
    delta := current - prev
    if delta < 0 {
        return 0 // Pod 重启，跳过本周期
    }
    return delta
}
```

**注意**: Agent 重启时 prev 为空，首次采集所有 Pod 的 delta 都为 0（因为无 prev），不会产生异常数据。第二次采集开始正常。

### 5.4 Stage 3a: Aggregate to Service

```go
// aggregateServices 将 per-pod delta 聚合为 service 级别
//
// 聚合 key: namespace + deployment
// 聚合方式: 直接 sum 各 pod 的 delta 值
func (r *sloRepository) aggregateServices(
    responseDelta []linkerdResponseDelta,
    bucketDelta   []linkerdBucketDelta,
    sumDelta      []linkerdSumDelta,
    countDelta    []linkerdCountDelta,
) []model_v2.ServiceMetrics {
    // 按 namespace+deployment 分组
    // 每组内:
    //   Requests: 按 status_code+classification 分组 sum delta
    //   LatencyBuckets: 按 le 分组 sum delta
    //   LatencySum: sum
    //   LatencyCount: sum
    //
    // mTLS 聚合（新增）:
    //   遍历 inbound responseDelta:
    //     TotalRequestDelta += delta        （所有请求）
    //     if tls == "true":
    //       TLSRequestDelta += delta        （mTLS 请求）
    //   Master 侧计算: mtlsPercent = TLSRequestDelta / TotalRequestDelta * 100
}
```

### 5.5 Stage 3b: Extract Edges

```go
// extractEdges 从 outbound per-pod delta 提取拓扑（含延迟 + 错误率）
//
// 聚合 key: src_ns+src_name → dst_ns+dst_name
//
// 数据来源:
//   - RequestDelta: outbound otel_response_total delta（sum 所有 pod）
//   - FailureDelta: outbound otel_response_total delta（classification=failure）
//   - LatencySum:   outbound otel_response_latency_ms_sum delta（按 dst 分组）
//   - LatencyCount: outbound otel_response_latency_ms_count delta（按 dst 分组）
//
// 实现:
//   1. 遍历 outboundDelta:
//      - 按 src_ns+src_name+dst_ns+dst_name 分组
//      - RequestDelta += delta
//      - if classification == "failure": FailureDelta += delta
//   2. 遍历 outboundLatencySumDelta / outboundLatencyCountDelta:
//      - 按 src_ns+src_name+dst_ns+dst_name 分组
//      - LatencySum += delta
//      - LatencyCount += delta
func (r *sloRepository) extractEdges(
    outboundDelta       []linkerdResponseDelta,
    outboundSumDelta    []linkerdSumDelta,
    outboundCountDelta  []linkerdCountDelta,
) []model_v2.ServiceEdge
```

### 5.6 Stage 3c: Aggregate Ingress

```go
// aggregateIngress 聚合入口指标（Controller 无关）
//
// 入口指标已经是 service-key 级别（不是 per-pod），
// 但仍需要 delta 计算（counter 是累积的）
// 同时将延迟单位从秒转为毫秒
//
// 注意: 此函数接收的是 Parser 已归一化的通用类型，
// 不关心底层是 Traefik 还是 Nginx
func (r *sloRepository) aggregateIngress(
    requestDelta []ingressRequestDelta,
    bucketDelta  []ingressBucketDelta,
    sumDelta     []ingressSumDelta,
    countDelta   []ingressCountDelta,
) []model_v2.IngressMetrics {
    // 秒→毫秒转换:
    //   bucket le: "0.1" → "100", "0.3" → "300", "5" → "5000"
    //   sum: value * 1000
}
```

### 5.7 SLORepository 完整接口

```go
// SLORepository SLO 数据仓库（重写）
type SLORepository interface {
    // Collect 采集并处理 SLO 数据
    // 完成: scrape → filter → per-pod delta → aggregate → 返回 SLOSnapshot
    Collect(ctx context.Context) (*model_v2.SLOSnapshot, error)
}
```

**注意**: 去掉旧的 `CollectRoutes()` 方法。路由采集合并到 `Collect()` 内部，由 Repository 内部调用 IngressClient.CollectRoutes()。

---

## 6. Config 变更

**文件**: `config/types.go`

```go
// SLOConfig SLO 指标采集配置
type SLOConfig struct {
    Enabled       bool          `yaml:"enabled"`
    ScrapeTimeout time.Duration `yaml:"scrapeTimeout"` // 默认 5s

    // OTel Collector
    OTelMetricsURL string   `yaml:"otelMetricsURL"` // http://otel-collector.otel.svc:8889/metrics
    OTelHealthURL  string   `yaml:"otelHealthURL"`  // http://otel-collector.otel.svc:13133
    ExcludeNamespaces []string `yaml:"excludeNamespaces"` // 默认 [linkerd, linkerd-viz, kube-system, otel]
}
```

**配置示例**:

```yaml
slo:
  enabled: true
  scrapeTimeout: 5s
  otelMetricsURL: "http://otel-collector.otel.svc:8889/metrics"
  otelHealthURL: "http://otel-collector.otel.svc:13133"
  excludeNamespaces:
    - linkerd
    - linkerd-viz
    - kube-system
    - otel
```

---

## 7. Service 层集成

**不需要修改** `service/snapshot/snapshot.go` 的结构。

现有代码已经在并发采集中调用 `sloRepo.Collect()`，返回 `*model_v2.SLOSnapshot`。只要新的 SLORepository 实现同一接口，Service 层无感切换。

唯一变化: `SLORepository` 接口去掉 `CollectRoutes()`（合并到 `Collect()` 内部），Service 层对应删除 CollectRoutes 调用。

---

## 8. 初始化（agent.go 依赖注入）

```go
// 创建 SLO Repository
var sloRepo repository.SLORepository

if cfg.SLO.Enabled {
    otelClient := otel.NewOTelClient(
        cfg.SLO.OTelMetricsURL,
        cfg.SLO.OTelHealthURL,
        cfg.SLO.ScrapeTimeout,
    )
    sloRepo = slo.NewSLORepository(otelClient, ingressClient, cfg.SLO.ExcludeNamespaces)
    //                              ↑ OTel 采指标  ↑ 采路由(IngressRoute CRD)
}
```

---

## 9. OTel 实际指标参考

### 9.1 Linkerd 指标

从 OTel Collector `:8889` 实际输出（都加了 `otel_` 前缀）：

**otel_response_total** (gauge，值为累积):

```
# inbound（被调用方视角，用于 ServiceMetrics）
otel_response_total{
  namespace="elastic", deployment="apm-server",
  pod="apm-server-55c66d695f-rpk7x",
  direction="inbound",
  status_code="202", classification="success",
  route_name="default",    # "probe" = K8s 健康检查
  srv_port="8200",         # "4191" = Linkerd admin
  tls="true",
} 1234

# outbound（调用方视角，用于 ServiceEdge）
otel_response_total{
  namespace="atlhyper", deployment="atlhyper-agent",
  pod="atlhyper-agent-xxx",
  direction="outbound",
  dst_namespace="kube-system", dst_deployment="traefik",
  status_code="200",
} 5678
```

**otel_response_latency_ms_bucket** (gauge):

```
otel_response_latency_ms_bucket{
  namespace="elastic", deployment="apm-server",
  pod="apm-server-xxx",
  direction="inbound",
  le="100",     # 单位: 毫秒
} 456
```

Bucket 边界 (毫秒): `1, 2, 3, 4, 5, 10, 20, 30, 40, 50, 100, 200, 300, 400, 500, 1000, 2000, 3000, 4000, 5000, 10000, 20000, 30000, +Inf`

### 9.2 入口指标（多 Controller）

#### Traefik 示例

**otel_traefik_service_requests_total** (counter):

```
otel_traefik_service_requests_total{
  service="atlantis-atlantis-web-3000@kubernetes",
  code="200", method="GET", protocol="http",
} 789
```

**otel_traefik_service_request_duration_seconds_bucket** (histogram):

```
otel_traefik_service_request_duration_seconds_bucket{
  service="atlantis-atlantis-web-3000@kubernetes",
  code="200", method="GET",
  le="0.1",    # 单位: 秒 → Agent 转为 "100" (毫秒)
} 123
```

Bucket 边界 (秒): `0.1, 0.3, 1.2, 5, +Inf`

ServiceKey 归一化: `"atlantis-atlantis-web-3000@kubernetes"` → `"atlantis-atlantis-web-3000"`

#### Nginx Ingress 示例

**otel_nginx_ingress_controller_requests** (counter):

```
otel_nginx_ingress_controller_requests{
  namespace="atlantis", ingress="atlantis-web",
  service="atlantis-web", service_port="3000",
  status="200", method="GET",
} 789
```

**otel_nginx_ingress_controller_request_duration_seconds_bucket** (histogram):

```
otel_nginx_ingress_controller_request_duration_seconds_bucket{
  namespace="atlantis", ingress="atlantis-web",
  service="atlantis-web", service_port="3000",
  status="200", method="GET",
  le="0.1",
} 123
```

Bucket 边界 (秒): `0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, +Inf`

ServiceKey 归一化: namespace=`"atlantis"` + service=`"atlantis-web"` + port=`"3000"` → `"atlantis-atlantis-web-3000"`

#### 归一化结果对比

两种 Controller 归一化后产生相同的 `IngressRequestMetric`:

```go
IngressRequestMetric{
    ServiceKey: "atlantis-atlantis-web-3000",
    Code:       "200",
    Method:     "GET",
    Value:      789,
}
```

### 9.3 集群已发现服务

| Namespace | Services |
|---|---|
| atlhyper | atlhyper-agent, atlhyper-controller, atlhyper-web |
| atlantis | atlantis-web |
| elastic | apm-server, elasticsearch, filebeat, kibana |
| geass | geass-auth, geass-favorites, geass-gateway, geass-history, geass-media, geass-user, geass-web |
| nginx | nginx-static |

---

## 10. 过滤规则

| 规则 | 字段 | 值 | 说明 |
|------|------|----|----|
| 排除健康检查 | `route_name` | `"probe"` | K8s liveness/readiness 探针 |
| 排除 Linkerd admin | `srv_port` | `"4191"` | Linkerd proxy 管理端口 |
| 排除系统 namespace | `namespace` | 可配置列表 | 默认: linkerd, linkerd-viz, kube-system, otel |
| 只用 inbound 算 SLO | `direction` | `"inbound"` | outbound 仅用于拓扑发现 |

---

## 11. 文件清单

### 新建

| 文件 | 说明 |
|------|------|
| `sdk/impl/otel/client.go` | OTelClient 实现：HTTP 采集 + 健康检查 |
| `sdk/impl/otel/parser.go` | Prometheus 文本解析 → OTelRawMetrics 分类 |
| `repository/slo/types.go` | 内部增量类型定义（linkerdResponseDelta, ingressRequestDelta 等） |
| `repository/slo/filter.go` | Stage 1: 过滤 probe、admin 端口、系统 namespace |
| `repository/slo/snapshot.go` | Stage 2: snapshotManager，per-pod delta 计算 |
| `repository/slo/aggregate.go` | Stage 3: Pod→Service 聚合 + Edge 提取 + Ingress 聚合(秒→ms) |

### 重写

| 文件 | 说明 |
|------|------|
| `model_v2/slo.go` | 新数据模型（SLOSnapshot + ServiceMetrics + ServiceEdge + IngressMetrics） |
| `repository/slo/slo.go` | SLORepository 主入口：编排 filter → delta → aggregate → routes |

### 修改

| 文件 | 变更 |
|------|------|
| `sdk/interfaces.go` | 新增 OTelClient 接口 |
| `sdk/types.go` | 新增 OTel 内部类型（OTelRawMetrics + 子类型） |
| `repository/interfaces.go` | SLORepository 去掉 CollectRoutes()，只保留 Collect() |
| `config/types.go` | SLOConfig 新增 OTelMetricsURL、OTelHealthURL、ExcludeNamespaces |
| `agent.go` (初始化) | 依赖注入 OTelClient + SLORepository |
| `service/snapshot/snapshot.go` | 删除 CollectRoutes 单独调用（已合入 Collect） |

### 保留不动

| 文件 | 说明 |
|------|------|
| `sdk/impl/ingress/route_collector.go` | 路由采集不变，被新 SLORepository 内部调用 |
| `service/snapshot/snapshot.go` (主体) | Service 层结构不变 |
| `scheduler/` | 调度层不变 |

---

## 12. 实现阶段

```
Phase 1: 数据模型
  - 重写 model_v2/slo.go
  - 验证 JSON 序列化

Phase 2: SDK 层
  - sdk/interfaces.go 新增 OTelClient
  - sdk/types.go 新增内部类型
  - sdk/impl/otel/client.go + parser.go
  - 单元测试: mock OTel 输出 → 验证解析结果

Phase 3: Repository 层
  - repository/slo/slo.go 重写主入口 (Collect 编排)
  - repository/slo/filter.go 过滤逻辑
  - repository/slo/snapshot.go snapshotManager per-pod delta
  - repository/slo/aggregate.go Pod→Service 聚合 + Edge + Ingress
  - 删除旧 Ingress 直连代码 (sdk/impl/ingress/client.go, discover.go, parser.go)
  - 单元测试: mock OTelRawMetrics → 验证 filter/delta/aggregate 各阶段

Phase 4: 集成
  - config 新增字段
  - agent.go 依赖注入
  - repository/interfaces.go 简化
  - service 层适配

Phase 5: 端到端验证
  - 对接真实 OTel Collector (已部署)
  - 验证指标采集 + 增量计算正确性
  - 验证 ClusterSnapshot 上报 Master
```

---

## 13. 前端数据映射

前端展示两层数据，本节说明 Agent 上报字段到前端展示字段的完整映射关系，以及哪些计算在 Agent 侧完成、哪些留给 Master。

### 13.1 服务网格层（ServiceNode → ServiceMetrics）

| 前端字段 | 含义 | Agent 上报字段 | OTel 原始指标 | 计算方式 |
|---|---|---|---|---|
| `rps` | 请求/秒 | `ServiceMetrics.LatencyCount` ÷ 采集间隔 | `otel_response_latency_ms_count` (inbound) | **Master 计算**: delta_count / interval_seconds |
| `avgLatency` | 平均延迟 ms | `LatencySum` / `LatencyCount` | `otel_response_latency_ms_sum` / `_count` | **Master 计算**: sum / count |
| `p50Latency` | P50 ms | `LatencyBuckets` | `otel_response_latency_ms_bucket` (inbound) | **Master 计算**: histogram 插值 |
| `p95Latency` | P95 ms | `LatencyBuckets` | 同上 | **Master 计算**: histogram 插值 |
| `p99Latency` | P99 ms | `LatencyBuckets` | 同上 | **Master 计算**: histogram 插值 |
| `errorRate` | 错误率 % | `Requests[].Classification=="failure"` | `otel_response_total` (inbound, classification=failure) | **Master 计算**: failure_delta / total_delta * 100 |
| `mtlsPercent` | mTLS 覆盖率 | `TLSRequestDelta` / `TotalRequestDelta` | `otel_response_total` (inbound, `tls` 标签) | **Agent 聚合**: tls=true 请求 / 总请求 |
| `latencyDistribution` | 24桶直方图 | `LatencyBuckets` | `otel_response_latency_ms_bucket` | **Master**: 存储 bucket delta |
| `requestBreakdown` | 按HTTP方法分布 | **不可用** | Linkerd inbound 无 `method` 标签 | **仅入口层提供** |
| `statusCodeBreakdown` | 状态码分布 | `Requests[]` 按 StatusCode 聚合 | `otel_response_total` (status_code) | **Master 聚合**: 按 status_code 前缀分组 |
| `totalRequests` | 总请求数 | `LatencyCount` 累积 | `otel_response_latency_ms_count` | **Master 累积** |

### 13.2 拓扑层（ServiceEdge）

| 前端字段 | 含义 | Agent 上报字段 | OTel 原始指标 |
|---|---|---|---|
| `source` / `target` | 调用方/被调方 | `SrcNamespace+SrcName` / `DstNamespace+DstName` | `otel_response_total` (outbound, `dst_deployment`+`dst_namespace`) |
| `rps` | 边的 RPS | `RequestDelta` ÷ 间隔 | 同上 |
| `avgLatency` | 边的平均延迟 | `LatencySum` / `LatencyCount` | `otel_response_latency_ms_sum/count` (outbound, 按 dst 分组) |
| `errorRate` | 边的错误率 | `FailureDelta` / `RequestDelta` | `otel_response_total` (outbound, classification=failure) |

### 13.3 入口层（DomainSLO → IngressMetrics + IngressRouteInfo）

| 前端字段 | Agent 上报字段 | OTel 原始指标（Controller 无关） |
|---|---|---|
| `host` | `IngressRouteInfo.Domain` | IngressRoute CRD / 标准 Ingress |
| `current.requestsPerSec` | `IngressMetrics.LatencyCount` ÷ 间隔 | 入口延迟 count 指标 |
| `current.p50/p95/p99Latency` | `IngressMetrics.LatencyBuckets` | 入口延迟 bucket 指标 |
| `current.errorRate` | `IngressMetrics.Requests` (code 5xx) | 入口请求计数指标 |
| `latencyDistribution` | `IngressMetrics.LatencyBuckets` | 同上 (Agent 已转 ms) |
| `requestBreakdown` | `IngressMetrics.Requests` (按 method) | 入口请求计数指标 (Traefik/Nginx 均有 method) |
| `statusCodeBreakdown` | `IngressMetrics.Requests` (按 code) | 同上 |
| `backendServices` | `IngressRouteInfo.ServiceKey` → 匹配 | IngressRoute CRD / 标准 Ingress → ServiceKey |

### 13.4 数据不可用说明

| 场景 | 原因 | 前端处理 |
|---|---|---|
| 服务网格层无 `requestBreakdown` | Linkerd `otel_response_total` 无 `method` 标签 | 服务详情传空数组，不展示该区块 |
| 入口层有 `requestBreakdown` | Traefik 和 Nginx 均提供 `method` 标签 | 正常展示 GET/POST/PUT/DELETE 分布 |
