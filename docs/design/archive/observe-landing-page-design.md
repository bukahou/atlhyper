# Observe Landing Page — 统一服务健康模型

> 状态：活跃（设计中）
> 创建：2026-02-27
> 前置：[K8s 元数据展示（Phase 0）](k8s-metadata-display-design.md)（已完成）
> 关联：[跨信号穿透式导航](../future/cross-signal-navigation-design.md) Phase 2

---

## 1. 背景

现有 Observe 子页面（APM / Logs / Metrics / SLO）各自独立展示单一信号。
用户排查问题时需要在 4 个页面间手动切换，脑内关联「这个服务的 APM 指标 + SLO 状态 + 最近错误日志 + 所在节点资源」。

**Observe Landing Page 的核心价值**：按服务聚合所有信号，一屏看全，用统一数据模型替代手动关联。

### 关键约束

- **不新建 DB**：所有数据来自 OTelSnapshot（内存）+ ClusterSnapshot（内存）+ ClickHouse（长时间范围聚合查询）
- **大后端小前端**：后端返回聚合好的 `ServiceHealth`，前端直接渲染
- **两层数据策略**：默认 15min 走内存快照（O(1)），1d/7d/30d 走 ClickHouse GROUP BY 聚合（只返回摘要，不拉原始数据）

---

## 2. 数据源分析

### 2.1 实时数据（默认 15min，内存快照）

OTelSnapshot 已包含所有需要的数据：

| 信号 | OTelSnapshot 字段 | 类型 | 关联键 |
|------|-------------------|------|--------|
| APM | `APMServices` | `[]apm.APMService` | `Name` = ServiceName |
| SLO (Mesh) | `SLOServices` | `[]slo.ServiceSLO` | `Name` = ServiceName |
| SLO (Ingress) | `SLOIngress` | `[]slo.IngressSLO` | `ServiceKey` → Ingress Rules 映射 |
| SLO 汇总 | `SLOSummary` | `*slo.SLOSummary` | 全局 |
| Metrics | `MetricsNodes` | `[]metrics.NodeMetrics` | Node name |
| Metrics 汇总 | `MetricsSummary` | `*metrics.Summary` | 全局 |
| Logs | `LogsSummary` | `*log.Summary` | 全局（含 TopServices） |
| Logs | `RecentLogs` | `[]log.Entry` | `ServiceName` |

ClusterSnapshot 补充 K8s 关系：

| 数据 | 用途 |
|------|------|
| Pods | 服务 → Pod 列表（按 `app` label 匹配 ServiceName） |
| Pods.NodeName | Pod → 所在节点 |
| Ingresses | 域名 → 后端 Service 映射（IngressSLO 关联） |

### 2.2 历史数据（1d/7d/30d，ClickHouse 聚合查询）

| 信号 | 查询方式 | 返回量 | 说明 |
|------|----------|--------|------|
| APM | `ListServices(since)` 已有 | ~10 行 | `GROUP BY ServiceName`，复用现有方法 |
| Logs | `LogServiceSummary(since)` **新增** | ~10 行 | `GROUP BY ServiceName, SeverityText` |
| SLO | `SLOWindows` 内存直读 | 内存 | 已有 1d/7d/30d 预聚合，不查 ClickHouse |
| Metrics | 仅当前值 | 内存 | 节点指标无历史需求 |

**关键原则**：ClickHouse 层做 GROUP BY 聚合，只返回 per-service 摘要数字，不拉原始 Trace/Log。
1d 的 otel_traces/otel_logs 可能有百万行，但 GROUP BY ServiceName 后只返回 ~10 行。

### 2.3 新增 Agent 查询 — LogServiceSummary

**路径**: `atlhyper_agent_v2/repository/ch/query/log.go`

```sql
SELECT ServiceName,
       countIf(SeverityText = 'ERROR') AS errorCount,
       countIf(SeverityText = 'WARN')  AS warnCount,
       count()                         AS totalCount
FROM otel_logs
WHERE Timestamp >= now() - INTERVAL {since} SECOND
GROUP BY ServiceName
```

返回 `[]LogServiceCount`（新类型）：

```go
// model_v3/log/log.go
type ServiceLogCount struct {
    ServiceName string `json:"serviceName"`
    ErrorCount  int64  `json:"errorCount"`
    WarnCount   int64  `json:"warnCount"`
    TotalCount  int64  `json:"totalCount"`
}
```

### 2.4 关联路径

```
ServiceName (APM, Logs)
    ║
    ╠═══ SLOServices[].Name           (直接相等)
    ║
    ╠═══ IngressSLO[].ServiceKey      (Ingress Rules 映射)
    ║     └── ClusterSnapshot.Ingresses → backend Service → Deployment name
    ║
    ╠═══ Pod.Labels["app"]            (直接相等，app label 约定)
    ║     └── Pod.NodeName → MetricsNodes (节点指标)
    ║
    ╚═══ RecentLogs[].ServiceName     (直接相等)
```

---

## 3. 数据模型

### 3.1 后端模型

**路径**: `model_v3/observe/health.go`（新建）

```go
package observe

import model_v3 "AtlHyper/model_v3"

// ServiceHealth 单个服务的跨信号聚合健康视图
type ServiceHealth struct {
    // 标识
    Name      string `json:"name"`      // ServiceName = Deployment name
    Namespace string `json:"namespace"`

    // 综合状态
    Status model_v3.HealthStatus `json:"status"` // healthy/warning/critical

    // APM 维度
    APM *ServiceAPM `json:"apm,omitempty"`

    // SLO 维度（Mesh + Ingress）
    SLO *ServiceSLO `json:"slo,omitempty"`

    // Logs 维度
    Logs *ServiceLogs `json:"logs,omitempty"`

    // Infra 维度
    Infra *ServiceInfra `json:"infra,omitempty"`
}

// ServiceAPM APM 信号摘要
type ServiceAPM struct {
    RPS         float64 `json:"rps"`
    SuccessRate float64 `json:"successRate"` // 0-1
    ErrorRate   float64 `json:"errorRate"`   // 0-1
    P99Ms       float64 `json:"p99Ms"`
    AvgMs       float64 `json:"avgMs"`
    SpanCount   int64   `json:"spanCount"`
    ErrorCount  int64   `json:"errorCount"`
}

// ServiceSLO SLO 信号摘要（Mesh + Ingress 合并）
type ServiceSLO struct {
    // Mesh (Linkerd)
    MeshSuccessRate float64 `json:"meshSuccessRate,omitempty"` // 0-1
    MeshRPS         float64 `json:"meshRps,omitempty"`
    MeshP99Ms       float64 `json:"meshP99Ms,omitempty"`
    MTLSEnabled     bool    `json:"mtlsEnabled"`

    // Ingress (Traefik) — 该服务关联的域名级 SLO
    IngressDomains []IngressBrief `json:"ingressDomains,omitempty"`
}

// IngressBrief 域名级 SLO 摘要
type IngressBrief struct {
    Domain      string  `json:"domain"`      // Ingress Host
    SuccessRate float64 `json:"successRate"` // 0-1
    RPS         float64 `json:"rps"`
    P99Ms       float64 `json:"p99Ms"`
}

// ServiceLogs 日志信号摘要
type ServiceLogs struct {
    ErrorCount int64 `json:"errorCount"`
    WarnCount  int64 `json:"warnCount"`
    TotalCount int64 `json:"totalCount"`
}

// ServiceInfra 基础设施信号摘要
type ServiceInfra struct {
    PodCount int         `json:"podCount"`
    Nodes    []NodeBrief `json:"nodes"` // 该服务 Pod 所在的节点
}

// NodeBrief 节点资源摘要
type NodeBrief struct {
    Name   string  `json:"name"`
    CPUPct float64 `json:"cpuPct"`
    MemPct float64 `json:"memPct"`
}

// HealthOverview Landing Page 顶部汇总卡片
type HealthOverview struct {
    // 服务维度
    TotalServices    int `json:"totalServices"`
    HealthyServices  int `json:"healthyServices"`
    WarningServices  int `json:"warningServices"`
    CriticalServices int `json:"criticalServices"`

    // APM 全局
    TotalRPS       float64 `json:"totalRps"`
    AvgSuccessRate float64 `json:"avgSuccessRate"`

    // SLO 全局
    SLOCompliance float64 `json:"sloCompliance"` // 达标服务比例

    // Infra 全局
    TotalNodes  int     `json:"totalNodes"`
    OnlineNodes int     `json:"onlineNodes"`
    AvgCPUPct   float64 `json:"avgCpuPct"`
    AvgMemPct   float64 `json:"avgMemPct"`

    // Logs 全局
    TotalErrorCount int64 `json:"totalErrorCount"`
}

// LandingPageResponse Landing Page API 响应
type LandingPageResponse struct {
    Overview HealthOverview  `json:"overview"`
    Services []ServiceHealth `json:"services"` // 按风险排序（critical > warning > healthy）
}
```

### 3.2 IngressSLO 关联逻辑

IngressSLO 是域名级别（Traefik），关联到服务需要两步：

```
IngressSLO.ServiceKey (如 "geass-gateway@kubernetescrd")
  → 解析出 IngressRoute name (如 "geass-gateway")
  → ClusterSnapshot.Ingresses 查找对应 Ingress
  → Ingress.Spec.Rules[].Backend.Service.Name (如 "geass-gateway")
  → Service name == Deployment name == APM ServiceName
```

**简化方案**：IngressSLO 的 `ServiceKey` 格式为 `{name}@kubernetescrd`，`name` 通常就是 K8s Service name，而 Geass 中 Service name == Deployment name。可直接截取 `@` 前部分匹配。

```go
func ingressKeyToServiceName(serviceKey string) string {
    // "geass-gateway@kubernetescrd" → "geass-gateway"
    if idx := strings.Index(serviceKey, "@"); idx > 0 {
        return serviceKey[:idx]
    }
    return serviceKey
}
```

一个服务可能对应 0 个或多个域名（IngressBrief 列表）。

### 3.3 综合状态计算

```go
func computeStatus(apm *ServiceAPM, slo *ServiceSLO) model_v3.HealthStatus {
    // APM 成功率
    if apm != nil && apm.SuccessRate < 0.95 {
        return model_v3.HealthStatusCritical
    }
    // Mesh SLO 成功率
    if slo != nil && slo.MeshSuccessRate > 0 && slo.MeshSuccessRate < 0.95 {
        return model_v3.HealthStatusCritical
    }
    // Ingress SLO 成功率
    if slo != nil {
        for _, ing := range slo.IngressDomains {
            if ing.SuccessRate < 0.95 {
                return model_v3.HealthStatusCritical
            }
        }
    }

    // Warning 阈值
    if apm != nil && apm.SuccessRate < 0.99 {
        return model_v3.HealthStatusWarning
    }
    if slo != nil && slo.MeshSuccessRate > 0 && slo.MeshSuccessRate < 0.99 {
        return model_v3.HealthStatusWarning
    }

    return model_v3.HealthStatusHealthy
}
```

### 3.4 前端模型

**路径**: `atlhyper_web/src/types/model/observe.ts`（新建）

```typescript
import type { HealthStatus } from "./apm";

export interface ServiceAPM {
  rps: number;
  successRate: number;
  errorRate: number;
  p99Ms: number;
  avgMs: number;
  spanCount: number;
  errorCount: number;
}

export interface IngressBrief {
  domain: string;
  successRate: number;
  rps: number;
  p99Ms: number;
}

export interface ServiceSLO {
  meshSuccessRate?: number;
  meshRps?: number;
  meshP99Ms?: number;
  mtlsEnabled: boolean;
  ingressDomains?: IngressBrief[];
}

export interface ServiceLogs {
  errorCount: number;
  warnCount: number;
  totalCount: number;
}

export interface NodeBrief {
  name: string;
  cpuPct: number;
  memPct: number;
}

export interface ServiceInfra {
  podCount: number;
  nodes: NodeBrief[];
}

export interface ServiceHealth {
  name: string;
  namespace: string;
  status: HealthStatus;
  apm?: ServiceAPM;
  slo?: ServiceSLO;
  logs?: ServiceLogs;
  infra?: ServiceInfra;
}

export interface HealthOverview {
  totalServices: number;
  healthyServices: number;
  warningServices: number;
  criticalServices: number;
  totalRps: number;
  avgSuccessRate: number;
  sloCompliance: number;
  totalNodes: number;
  onlineNodes: number;
  avgCpuPct: number;
  avgMemPct: number;
  totalErrorCount: number;
}

export interface LandingPageResponse {
  overview: HealthOverview;
  services: ServiceHealth[];
}
```

---

## 4. API 设计

### 4.1 端点

```
GET /api/v2/observe/health?cluster_id={id}&time_range={range}
```

| 参数 | 必须 | 默认值 | 说明 |
|------|:----:|--------|------|
| `cluster_id` | 是 | — | 集群 ID |
| `time_range` | 否 | `15m` | 时间范围：`15m` / `1d` / `7d` / `30d` |

### 4.2 两层数据策略

```
time_range=15m (默认)
  → 纯内存快照，O(1)，<10ms
  → APM: OTelSnapshot.APMServices
  → SLO: OTelSnapshot.SLOServices + SLOIngress
  → Logs: OTelSnapshot.RecentLogs (遍历计数)
  → Infra: ClusterSnapshot.Pods + OTelSnapshot.MetricsNodes

time_range=1d/7d/30d
  → APM + Logs: Command/MQ → Agent → ClickHouse GROUP BY (30s 超时)
  → SLO: OTelSnapshot.SLOWindows[timeRange] (内存直读，已预聚合)
  → Infra: 同上（仅当前值）
```

### 4.3 ClickHouse 聚合查询（仅 1d/7d/30d 触发）

**APM**：复用现有 `ListServices(since)`

```sql
-- 已有，无需新增。返回 ~10 行。
SELECT ServiceName, ..., count() / {sec} AS rps,
       1 - countIf(StatusCode = 'STATUS_CODE_ERROR') / count() AS successRate,
       quantile(0.99)(Duration) / 1e6 AS p99Ms
FROM otel_traces
WHERE SpanKind = 'SPAN_KIND_SERVER'
  AND Timestamp >= now() - INTERVAL {sec} SECOND
GROUP BY ServiceName, ns
```

**Logs**：新增 `LogServiceSummary(since)`

```sql
-- 新增。返回 ~10 行。
SELECT ServiceName,
       countIf(SeverityText = 'ERROR') AS errorCount,
       countIf(SeverityText = 'WARN')  AS warnCount,
       count()                         AS totalCount
FROM otel_logs
WHERE Timestamp >= now() - INTERVAL {sec} SECOND
GROUP BY ServiceName
```

**SLO**：不查 ClickHouse，`SLOWindows["1d"]` / `["7d"]` / `["30d"]` 已在内存中。

### 4.4 聚合逻辑（伪代码）

```go
func (h *ObserveHandler) handleHealth(w, r) {
    timeRange := r.URL.Query().Get("time_range") // "15m" | "1d" | "7d" | "30d"
    if timeRange == "" { timeRange = "15m" }

    otel := svc.GetOTelSnapshot(ctx, clusterID)
    k8s  := svc.GetSnapshot(ctx, clusterID)

    var apmServices []apm.APMService
    var logCounts   []log.ServiceLogCount
    var sloServices []slo.ServiceSLO
    var sloIngress  []slo.IngressSLO

    if timeRange == "15m" {
        // ── 快速路径：纯内存 ──
        apmServices = otel.APMServices
        logCounts   = countLogsFromRecent(otel.RecentLogs)
        sloServices = otel.SLOServices
        sloIngress  = otel.SLOIngress
    } else {
        // ── 慢路径：APM + Logs 查 ClickHouse，SLO 读预聚合 ──
        since := parseDuration(timeRange)

        // 并行发送两个 Command
        apmServices = commandQuery("list_services", since)  // 复用已有
        logCounts   = commandQuery("log_service_summary", since)  // 新增

        // SLO 从预聚合窗口读取
        if window, ok := otel.SLOWindows[timeRange]; ok {
            sloServices = window.MeshServices
            // IngressSLO 的历史数据使用 window.Current（Traefik 指标）
            sloIngress = mapWindowToIngress(window.Current)
        }
    }

    // ── 以下聚合逻辑两条路径共用 ──

    // 1. 以 APMServices 为主列表
    serviceMap := buildFromAPM(apmServices)

    // 2. 关联 SLO Mesh
    for _, s := range sloServices {
        if sh, ok := serviceMap[s.Name]; ok {
            sh.SLO = &ServiceSLO{MeshSuccessRate: s.SuccessRate, ...}
        }
    }

    // 3. 关联 SLO Ingress
    ingressByService := groupIngressByService(sloIngress) // serviceKey → []IngressBrief
    for name, domains := range ingressByService {
        if sh, ok := serviceMap[name]; ok {
            if sh.SLO == nil { sh.SLO = &ServiceSLO{} }
            sh.SLO.IngressDomains = domains
        }
    }

    // 4. 关联 Logs
    for _, lc := range logCounts {
        if sh, ok := serviceMap[lc.ServiceName]; ok {
            sh.Logs = &ServiceLogs{ErrorCount: lc.ErrorCount, ...}
        }
    }

    // 5. 关联 Infra（app label 匹配，仅当前值）
    podsByService := groupPodsByAppLabel(k8s.Pods)
    nodeMetricsMap := indexNodeMetrics(otel.MetricsNodes)
    for name, pods := range podsByService {
        if sh, ok := serviceMap[name]; ok {
            sh.Infra = buildInfra(pods, nodeMetricsMap)
        }
    }

    // 6. 计算状态 + 排序 + 汇总
    services := computeAndSort(serviceMap)
    overview := buildOverview(services, otel)

    writeJSON(w, 200, LandingPageResponse{Overview: overview, Services: services})
}
```

### 4.5 路由注册

```go
// gateway/handler/observe.go — registerObserveRoutes()
mux.HandleFunc("/api/v2/observe/health", h.handleHealth)
```

### 4.6 前端 API

```typescript
// api/observe.ts
type TimeRange = "15m" | "1d" | "7d" | "30d";

export async function getObserveHealth(
  clusterId: string,
  timeRange: TimeRange = "15m"
): Promise<LandingPageResponse> {
  const res = await request.get(`/api/v2/observe/health`, {
    params: { cluster_id: clusterId, time_range: timeRange },
  });
  return res.data.data;
}
```

---

## 5. 数据流

```
time_range=15m（默认，纯内存）:

  OTelSnapshot ──────────────┐
  ├── APMServices            │
  ├── SLOServices + Ingress  ├──→ 聚合 → LandingPageResponse
  ├── RecentLogs             │
  ├── MetricsNodes           │
  ClusterSnapshot ───────────┘
  ├── Pods (app label)
  └── Ingresses


time_range=1d/7d/30d（ClickHouse 聚合 + 内存）:

  ┌─ Command/MQ ─→ Agent ─→ ClickHouse ─┐
  │  ListServices(since)      → ~10 rows │
  │  LogServiceSummary(since) → ~10 rows │
  └──────────────────────────────────────┘
            │
            ├── APM summary (ClickHouse GROUP BY)
            ├── Logs summary (ClickHouse GROUP BY)
  ┌─────────┤
  │  OTelSnapshot
  │  ├── SLOWindows[timeRange]  (内存预聚合)
  │  ├── MetricsNodes           (当前值)
  │  ClusterSnapshot
  │  ├── Pods (app label)
  └──┴──→ 聚合 → LandingPageResponse
```

---

## 6. 关联键映射（总结）

```
ServiceName (APM, Logs)
    ║
    ╠═══ SLOServices[].Name                 (直接相等)
    ║
    ╠═══ IngressSLO[].ServiceKey            ("geass-gateway@kubernetescrd" → "geass-gateway")
    ║     └── 截取 "@" 前部分 == ServiceName
    ║
    ╠═══ Pod.Labels["app"]                  (直接相等)
    ║     └── Pod.NodeName → MetricsNodes
    ║
    ╚═══ RecentLogs[].ServiceName           (直接相等)
        / LogServiceSummary[].ServiceName   (直接相等)
```

---

## 7. 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model_v3/observe/health.go` | **新建** | ServiceHealth / HealthOverview / LandingPageResponse |
| `model_v3/log/log.go` | 修改 | 新增 `ServiceLogCount` 类型 |
| `atlhyper_agent_v2/repository/interfaces.go` | 修改 | 新增 `LogServiceSummary(since)` 方法签名 |
| `atlhyper_agent_v2/repository/ch/query/log.go` | 修改 | 实现 `LogServiceSummary` ClickHouse 查询 |
| `atlhyper_agent_v2/service/command/ch_query.go` | 修改 | 新增 `log_service_summary` 命令处理 |
| `atlhyper_master_v2/gateway/handler/observe.go` | 修改 | 新增 `handleHealth` + 路由 |
| `atlhyper_web/src/types/model/observe.ts` | **新建** | 前端类型 |
| `atlhyper_web/src/api/observe.ts` | 修改 | 新增 `getObserveHealth` |
| `atlhyper_web/src/app/observe/page.tsx` | **新建** | Landing Page 页面 |
| `atlhyper_web/src/types/i18n.ts` | 修改 | 新增 ObserveTranslations |
| `atlhyper_web/src/i18n/locales/zh.ts` | 修改 | 中文翻译 |
| `atlhyper_web/src/i18n/locales/ja.ts` | 修改 | 日文翻译 |
| **合计** | **3 新建 + 9 修改** | |

---

## 8. 验证

```bash
# 后端
go build ./model_v3/... && go build ./atlhyper_agent_v2/... && go build ./atlhyper_master_v2/...

# 前端
cd atlhyper_web && npm run build
```

端到端：
1. 访问 `/observe` → 默认 15m → 快照直读 → 应显示 HealthOverview + ServiceHealth 列表
2. 切换到 1d → ClickHouse 聚合查询 → 数据应更新（总量更大，成功率可能不同）
3. 每个服务卡片显示 APM + SLO（Mesh + Ingress 域名）+ Logs + Infra
4. 服务按风险排序：critical > warning > healthy

---

## 9. 已确认决策

| 问题 | 决策 |
|------|------|
| Pod → Service 匹配方式 | **app label**：`Pod.Labels["app"] == ServiceName`，一级查找 |
| IngressSLO 关联 | **本阶段实现**：`ServiceKey` 截取 `@` 前部分匹配 ServiceName |
| 长时间范围数据策略 | **ClickHouse GROUP BY 聚合**：只返回 per-service 摘要（~10 行），不拉原始数据 |
| 以谁为主列表 | **APMServices**：SLO/Logs/Infra 作为可选关联（`omitempty`） |

---

## 10. 开放问题

1. **SLOServices 覆盖范围可能与 APMServices 不同**
   - Linkerd 网格覆盖的服务 vs OTel 覆盖的服务可能不完全一致
   - 方案：以 APMServices 为主列表，SLO 只在有匹配时展示

2. **IngressSLO ServiceKey 格式假设**
   - 假设格式为 `{name}@kubernetescrd`，截取 `@` 前等于 Service name
   - 需实际验证 Traefik IngressRoute 的 ServiceKey 格式
