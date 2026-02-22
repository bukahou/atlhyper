# AtlHyper 全量数据关联参考

> 7 种数据源的关联键和数据流向
> 配合 `clickhouse-otel-data-reference.md` 使用

---

## 目录

1. [七种数据源总览](#1-七种数据源总览)
2. [数据流向：从集群到应用](#2-数据流向从集群到应用)
3. [通用关联键](#3-通用关联键)
4. [逐层关联详解](#4-逐层关联详解)
5. [跨数据源关联矩阵](#5-跨数据源关联矩阵)
6. [关联查询示例](#6-关联查询示例)

---

## 1. 七种数据源总览

| # | 数据源 | 存储 | 粒度 | 核心标识 |
|---|--------|------|------|---------|
| 1 | **集群快照** | Agent → Master DataHub | 资源级 | namespace + name |
| 2 | **K8s Event** | Agent → Master DataHub | 事件级 | involvedObject (kind+ns+name) |
| 3 | **入口指标 (Traefik)** | ClickHouse otel_metrics_sum/histogram | 服务级 | Attributes.`service` |
| 4 | **服务网格 (Linkerd)** | ClickHouse otel_metrics_gauge | Pod/Deployment 级 | Attributes.`namespace` + `deployment` |
| 5 | **基础设施指标 (Node)** | ClickHouse otel_metrics_gauge/sum | 节点级 | ResourceAttributes.`net.host.name` |
| 6 | **应用日志 (Logs)** | ClickHouse otel_logs | 请求级 | ServiceName + TraceId + SpanId |
| 7 | **链路追踪 (Traces)** | ClickHouse otel_traces | Span 级 | TraceId + SpanId + ParentSpanId |

---

## 2. 数据流向：从集群到应用

用户请求从外部进入集群，经过 Ingress → Service → Pod（应用），每一层都有对应的数据源覆盖。

```
                          数据源覆盖
                          ─────────
[用户请求]
    │
    ▼
┌─────────┐
│  Node   │ ◄──── ⑤ 基础设施指标 (node_*, container_*)
│ (节点)   │       关联键: node name / node IP
└────┬────┘
     │
     ▼
┌──────────┐
│ Ingress  │ ◄──── ③ 入口指标 (traefik_entrypoint_*, traefik_service_*)
│ (入口)    │       关联键: Attributes.service → "ns-svc-port@kubernetes"
└────┬─────┘
     │  IngressRule.backend.service.name
     ▼
┌──────────┐
│ Service  │ ◄──── ④ 服务网格 (response_total, request_total, latency)
│ (服务)    │       关联键: Attributes.namespace + deployment
└────┬─────┘
     │  Service.selector → Pod.labels
     ▼
┌──────────┐
│   Pod    │ ◄──── ② K8s Event (involvedObject)
│ (应用)    │ ◄──── ④ 服务网格 (Attributes.pod)
│          │ ◄──── ⑤ 容器指标 (Attributes.pod + namespace)
└────┬─────┘
     │  应用内部
     ▼
┌──────────────┐
│ Trace + Log  │ ◄──── ⑦ 链路追踪 (TraceId → SpanId → ParentSpanId)
│ (请求级)      │ ◄──── ⑥ 应用日志 (TraceId + SpanId 关联)
└──────────────┘
     │
     ▼
① 集群快照: 以上所有 K8s 资源的 spec/status/conditions 状态
```

**自上而下的关联链**：

```
Node.name
  └─→ Pod.nodeName
        └─→ Pod.ownerName → Deployment.name
              └─→ Deployment.selector → Service.selector → Service.name
                    └─→ Ingress.rules[].backend.service.name → Ingress.name
                    └─→ Linkerd: Attributes.deployment + namespace
                    └─→ Traefik: Attributes.service (含 namespace-svc-port)
              └─→ OTel: ResourceAttributes.service.name (= Deployment 名)
                    └─→ Trace: ServiceName + host.name (= Pod 名)
                    └─→ Log: ServiceName + host.name (= Pod 名)
                          └─→ Log.TraceId + SpanId ←→ Trace.TraceId + SpanId
```

---

## 3. 通用关联键

所有数据源最终通过以下 5 个维度关联：

| 关联键 | 含义 | 覆盖数据源 | 示例值 |
|--------|------|-----------|--------|
| **namespace** | K8s 命名空间 | 全部 7 种 | `geass` |
| **workload name** | Deployment/StatefulSet/DaemonSet 名 | 全部 7 种 | `geass-gateway` |
| **pod name** | Pod 完整名称 | 1,2,4,5,6,7 | `geass-gateway-ff5988887-fmztl` |
| **node name** | 节点名称 | 1,2,5 | `raspi-nfs` |
| **TraceId** | 链路唯一 ID | 6,7 | `25db41ab52036ba19daf3588e22ac3ad` |

### 各数据源中的字段名映射

| 关联键 | ① 集群快照 | ② Event | ③ Traefik | ④ Linkerd | ⑤ Node 指标 | ⑥ Logs | ⑦ Traces |
|--------|-----------|---------|-----------|-----------|------------|--------|---------|
| namespace | `Pod.Summary.Namespace` | `InvolvedObject.Namespace` | `service` 前缀解析 | `Attributes.namespace` | `Attributes.namespace` | 从 `service.name` 推断 | 从 `service.name` 推断 |
| workload | `Deployment.Summary.Name` | `InvolvedObject.Name` | `service` 解析 | `Attributes.deployment` | — | `ResourceAttributes.service.name` | `ServiceName` |
| pod | `Pod.Summary.Name` | `InvolvedObject.Name` | — | `Attributes.pod` | `Attributes.pod` | `ResourceAttributes.host.name` | `ResourceAttributes.host.name` |
| node | `Node.Summary.Name` | `InvolvedObject.Name` | — | — | `Attributes.kubernetes_io_hostname` | — | — |
| traceId | — | — | — | — | — | `TraceId` | `TraceId` |

---

## 4. 逐层关联详解

### 4.1 Node → Pod（节点到工作负载）

```
Node.Summary.Name  =  Pod.Summary.NodeName
```

| 来源 | 字段 | 示例 |
|------|------|------|
| Node (快照) | `Summary.Name` | `raspi-nfs` |
| Pod (快照) | `Summary.NodeName` | `raspi-nfs` |
| Node 指标 | `ResourceAttributes['net.host.name']` | `192.168.0.46` (IP) |
| Node 指标 | `Attributes['kubernetes_io_hostname']` | `raspi-nfs` (名称) |
| 容器指标 | `Attributes['kubernetes_io_hostname']` | `raspi-nfs` |

**注意**: Node 指标通过 IP 标识节点，需要 `Node.Addresses.InternalIP` 做 IP ↔ 名称转换。

```
Node.Addresses.InternalIP  =  ResourceAttributes['net.host.name']
Node.Summary.Name          =  Attributes['kubernetes_io_hostname']
```

### 4.2 Pod → Deployment（工作负载归属）

```
Pod.Summary.OwnerKind  =  "Deployment" / "StatefulSet" / "DaemonSet"
Pod.Summary.OwnerName  =  Deployment.Summary.Name
```

Pod 名称中也隐含 Deployment 信息：
```
Pod:        geass-auth-66f49f5c9d-7wfqk
                │          │       │
                │          │       └── 随机后缀
                │          └── ReplicaSet hash
                └── Deployment 名: geass-auth
```

Linkerd 和 OTel 中的映射：
- Linkerd `Attributes.deployment` = `geass-auth` (直接提供 Deployment 名)
- OTel `ResourceAttributes.service.name` = `geass-auth` (= Deployment 名)
- OTel `ResourceAttributes.host.name` = `geass-auth-66f49f5c9d-7wfqk` (= Pod 名)

### 4.3 Deployment → Service（服务发现）

```
Service.Selector  ⊆  Pod.Labels    (标签选择器匹配)
```

| K8s Service | Selector | 匹配的 Pod Labels |
|-------------|----------|------------------|
| `geass-gateway` | `{app: geass-gateway}` | Pod Labels 含 `app: geass-gateway` |
| `geass-auth` | `{app: geass-auth}` | Pod Labels 含 `app: geass-auth` |

Service 的后端端点直接引用 Pod：
```
Service.Backends.Endpoints[].TargetRef = { Kind: "Pod", Name: "geass-auth-66f49f5c9d-7wfqk" }
Service.Backends.Endpoints[].Address   = "10.42.0.142"  (= Pod IP)
```

### 4.4 Service → Ingress（入口路由）

```
Ingress.Spec.Rules[].Paths[].Backend.Service.Name  =  Service.Summary.Name
```

示例：
```
Ingress "geass-ingress":
  Rules:
    - Host: "geass.example.com"
      Paths:
        - Path: "/api/*"
          Backend:
            Service.Name: "geass-gateway"    ← 关联到 K8s Service
            Service.Port: 8080
```

### 4.5 Ingress → Traefik 指标（入口流量）

Traefik 指标中的 `Attributes.service` 字段编码了 K8s 路由信息：

```
Traefik Attributes.service 格式:
    {namespace}-{service-name}-{port}@kubernetes

示例:
    geass-geass-gateway-8080@kubernetes
    │     │              │
    │     │              └── Service Port
    │     └── K8s Service Name: geass-gateway
    └── Namespace: geass
```

**解析规则**：
```
Attributes.service = "geass-geass-gateway-8080@kubernetes"
→ 去掉 @kubernetes 后缀
→ 拆分: namespace=geass, service=geass-gateway, port=8080
→ 关联: K8s Service (namespace=geass, name=geass-gateway)
```

### 4.6 Service → Linkerd 指标（服务网格）

Linkerd 直接在 Attributes 中提供 K8s 维度：

```
Attributes.namespace   = "geass"           → K8s Namespace
Attributes.deployment  = "geass-auth"      → K8s Deployment
Attributes.pod         = "geass-auth-66f49f5c9d-7wfqk" → K8s Pod
Attributes.direction   = "inbound"/"outbound"
Attributes.target_addr = "10.42.0.142:8085" → 下游 Pod IP:Port
```

**服务间调用关联**（outbound 指标）：
```
源服务:   Attributes.deployment = "geass-gateway"  (调用方)
目标地址: Attributes.target_addr = "10.42.0.142:8085"
    └─→ 通过 Pod IP 反查: Pod.Status.PodIP = "10.42.0.142"
    └─→ 得到目标 Pod → 目标 Deployment
```

### 4.7 Pod → Trace / Log（应用级关联）

OTel Java Agent 自动注入 K8s 上下文到 ResourceAttributes：

```
ResourceAttributes:
    service.name        = "geass-gateway"     → = Deployment 名
    host.name           = "geass-gateway-ff5988887-fmztl"  → = Pod 名
    service.instance.id = "b1b58849-..."      → Pod 实例 UUID
    cluster.name        = "zgmf-x10a"         → 集群 ID
```

**Trace 内的服务间调用关联**：
```
Trace (geass-gateway, SERVER):
    SpanAttributes.http.route = "/api/history/list"
    │
    └─ 子 Span (geass-gateway, CLIENT):
        SpanAttributes.url.full = "http://geass-history:8083/history/v2/list"
        SpanAttributes.server.address = "geass-history"  ← 下游 Service DNS 名
        │
        └─ 子 Span (geass-history, SERVER):
            ServiceName = "geass-history"               ← 下游 Deployment 名
            ParentSpanId = 上层 CLIENT Span 的 SpanId   ← 调用链
```

### 4.8 Trace ↔ Log（请求级精确关联）

```
otel_traces.TraceId  =  otel_logs.TraceId
otel_traces.SpanId   =  otel_logs.SpanId
```

同一个 TraceId 下，日志按 SpanId 归属到具体的 Span（操作）。

示例（TraceId: `25db41ab52036ba19daf3588e22ac3ad`）：
```
Span: POST /api/history/list (SpanId: aaa)
  ├── Log: "➡️ [POST] /api/history/list from 10.42.2.123"     (SpanId: aaa)
  ├── Log: "[Auth] userId=1, role=3, uri=/api/history/list"     (SpanId: aaa)
  │
  ├── 子 Span: POST geass-auth/token/verify (SpanId: bbb)
  │   ├── Log: "➡️ [POST] /token/verify from 10.42.0.142"     (SpanId: bbb)
  │   └── Log: "⬅️ [POST] /token/verify - 2ms (status=200)"   (SpanId: bbb)
  │
  └── Log: "⬅️ [POST] /api/history/list - 77ms (status=200)"   (SpanId: aaa)
```

### 4.9 Event → 任意 K8s 资源

```
Event.InvolvedObject.Kind      = "Pod" / "Node" / "Deployment" / ...
Event.InvolvedObject.Namespace = 资源 Namespace
Event.InvolvedObject.Name      = 资源 Name
```

Event 可以关联到 ClusterSnapshot 中的任何 K8s 资源，是唯一的「横切」数据源。

---

## 5. 跨数据源关联矩阵

以下标注任意两种数据源之间的**直接关联键**（非空 = 可直接关联，— = 需间接关联）。

| | ① 快照 | ② Event | ③ Traefik | ④ Linkerd | ⑤ Node指标 | ⑥ Log | ⑦ Trace |
|---|---|---|---|---|---|---|---|
| **① 快照** | — | InvolvedObject (kind+ns+name) | Traefik.service 解析 → K8s Service | namespace + deployment | node name/IP | service.name + host.name | service.name + host.name |
| **② Event** | InvolvedObject | — | — | — | InvolvedObject (Node) | — | — |
| **③ Traefik** | service 解析 | — | — | 同 Service 级别 | — | — | — |
| **④ Linkerd** | namespace + deployment + pod | — | 同 Service 级别 | — | node (通过 Pod) | service.name | service.name |
| **⑤ Node指标** | node name/IP | InvolvedObject (Node) | — | — | — | — | — |
| **⑥ Log** | service.name + host.name | — | — | service.name | — | — | TraceId + SpanId |
| **⑦ Trace** | service.name + host.name | — | — | service.name | — | TraceId + SpanId | — |

### 关联强度

| 强度 | 含义 | 典型场景 |
|------|------|---------|
| **精确** | 唯一键直接匹配 | Trace ↔ Log (TraceId+SpanId) |
| **直接** | K8s 原生字段匹配 | Pod ↔ Deployment (ownerName), Service ↔ Pod (selector) |
| **解析** | 需要字符串解析/拆分 | Traefik.service → K8s Service, Pod name → Deployment name |
| **间接** | 需要经过中间数据源 | Traefik → Linkerd (通过 K8s Service), Event → Trace (通过 Pod) |

---

## 6. 关联查询示例

### 6.1 从 Pod 出发，关联所有数据

已知: `namespace=geass`, `pod=geass-auth-66f49f5c9d-7wfqk`

```sql
-- 1. 该 Pod 的链路 (Traces)
SELECT TraceId, SpanName, Duration/1e6 AS ms, StatusCode
FROM otel_traces
WHERE ResourceAttributes['host.name'] = 'geass-auth-66f49f5c9d-7wfqk'
ORDER BY Timestamp DESC

-- 2. 该 Pod 的日志 (Logs)
SELECT Timestamp, SeverityText, Body, TraceId
FROM otel_logs
WHERE ResourceAttributes['host.name'] = 'geass-auth-66f49f5c9d-7wfqk'
ORDER BY Timestamp DESC

-- 3. 该 Pod 所属 Deployment 的 Linkerd 指标
SELECT MetricName, Attributes['status_code'] AS code, Value
FROM otel_metrics_gauge
WHERE ResourceAttributes['service.name'] = 'linkerd-prometheus'
  AND Attributes['namespace'] = 'geass'
  AND Attributes['deployment'] = 'geass-auth'
  AND MetricName = 'response_total'

-- 4. 该 Pod 的容器资源指标
SELECT MetricName, Value, TimeUnix
FROM otel_metrics_gauge
WHERE Attributes['namespace'] = 'geass'
  AND Attributes['pod'] = 'geass-auth-66f49f5c9d-7wfqk'
  AND MetricName IN ('container_cpu_usage_seconds_total', 'container_memory_working_set_bytes')
ORDER BY TimeUnix DESC
```

K8s 数据（通过 Agent → Master API）：
```
快照:  ClusterSnapshot.Pods[name="geass-auth-66f49f5c9d-7wfqk"]
事件:  ClusterSnapshot.Events[involvedObject.name="geass-auth-66f49f5c9d-7wfqk"]
```

### 6.2 从 Trace 出发，关联全链路

已知: `TraceId=25db41ab52036ba19daf3588e22ac3ad`

```sql
-- 1. 链路中所有 Span
SELECT ServiceName, SpanName, SpanKind, Duration/1e6 AS ms,
       SpanAttributes['http.route'] AS route,
       SpanAttributes['db.sql.table'] AS db_table
FROM otel_traces
WHERE TraceId = '25db41ab52036ba19daf3588e22ac3ad'
ORDER BY Timestamp

-- 2. 链路关联的所有日志
SELECT Timestamp, ServiceName, SeverityText, Body, SpanId
FROM otel_logs
WHERE TraceId = '25db41ab52036ba19daf3588e22ac3ad'
ORDER BY Timestamp

-- 3. 涉及的服务 → 关联 Linkerd 指标
-- 先从 Trace 提取涉及的 ServiceName 列表:
--   geass-gateway, geass-auth, geass-history, geass-media
-- 再查 Linkerd:
SELECT Attributes['deployment'] AS svc,
       Attributes['status_code'] AS code,
       Value AS total
FROM otel_metrics_gauge
WHERE Attributes['namespace'] = 'geass'
  AND Attributes['deployment'] IN ('geass-gateway', 'geass-auth', 'geass-history', 'geass-media')
  AND MetricName = 'response_total'
  AND Attributes['direction'] = 'inbound'
```

K8s 数据：
```
快照: Trace 涉及的 Pod → ClusterSnapshot.Pods[host.name]
      Trace 涉及的 Service → ClusterSnapshot.Services
事件: 相关 Pod 的 Events
```

### 6.3 从 Ingress 入口出发，向下穿透

已知: Traefik entrypoint `websecure`, 目标服务 `geass-geass-gateway-8080@kubernetes`

```sql
-- 1. 入口流量统计
SELECT Attributes['code'] AS status, Attributes['method'] AS method, max(Value) AS total
FROM otel_metrics_sum
WHERE MetricName = 'traefik_entrypoint_requests_total'
  AND Attributes['entrypoint'] = 'websecure'
GROUP BY status, method

-- 2. 入口延迟分布
SELECT Count, Sum, BucketCounts, ExplicitBounds
FROM otel_metrics_histogram
WHERE MetricName = 'traefik_service_request_duration_seconds'
  AND Attributes['service'] = 'geass-geass-gateway-8080@kubernetes'
ORDER BY TimeUnix DESC LIMIT 1

-- 3. 解析 Traefik service → K8s Service (geass/geass-gateway:8080)
-- → 查 Linkerd 同 Service 的网格指标
SELECT Attributes['status_code'] AS code, Value
FROM otel_metrics_gauge
WHERE Attributes['namespace'] = 'geass'
  AND Attributes['deployment'] = 'geass-gateway'
  AND MetricName = 'response_total'
  AND Attributes['direction'] = 'inbound'

-- 4. 该服务的 Trace
SELECT TraceId, SpanName, Duration/1e6 AS ms
FROM otel_traces
WHERE ServiceName = 'geass-gateway'
  AND SpanKind = 'SPAN_KIND_SERVER'
ORDER BY Timestamp DESC LIMIT 20
```

### 6.4 从 Node 出发，关联基础设施

已知: `node=raspi-nfs`, `IP=192.168.0.46`

```sql
-- 1. 节点硬件指标
SELECT MetricName, Value, TimeUnix
FROM otel_metrics_gauge
WHERE ResourceAttributes['net.host.name'] = '192.168.0.46'
  AND MetricName IN ('node_load1', 'node_memory_MemAvailable_bytes')
ORDER BY TimeUnix DESC LIMIT 10

-- 2. 该节点上的容器指标
SELECT Attributes['namespace'] AS ns, Attributes['pod'] AS pod,
       MetricName, Value
FROM otel_metrics_gauge
WHERE Attributes['kubernetes_io_hostname'] = 'raspi-nfs'
  AND MetricName IN ('container_cpu_usage_seconds_total', 'container_memory_working_set_bytes')
ORDER BY TimeUnix DESC
```

K8s 数据：
```
快照: ClusterSnapshot.Nodes[name="raspi-nfs"]  → 节点状态/容量
      ClusterSnapshot.Pods[nodeName="raspi-nfs"]  → 该节点上所有 Pod
事件: ClusterSnapshot.Events[involvedObject.kind="Node", name="raspi-nfs"]
```
