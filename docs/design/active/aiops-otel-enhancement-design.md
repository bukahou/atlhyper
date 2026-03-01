# AIOps OTel 信号融合增强

> 状态：活跃
> 创建：2026-02-28
> 前置：AIOps 引擎已完成（EMA+3σ / 三阶段风险评分 / 依赖图传播 / 状态机 / 事件管理）

---

## 1. 背景

### 1.1 现状

AIOps 引擎已具备完整的异常检测 → 风险评分 → 状态机 → 事件管理管道：

```
ExtractMetrics → EMA+3σ 检测 → 三阶段风险评分 → 状态机 → Incident
ExtractDeterministicAnomalies → 确定性直注 ──────┘
```

但 **OTel 信号接入深度不足**：

| 信号 | 现状 | 缺失 |
|------|------|------|
| **K8s 资源** | ✅ 完整（Node 6 指标 + Pod 4 指标 + 容器异常 + Event + Deployment 影响） | — |
| **SLO (Linkerd)** | ✅ `error_rate` + `avg_latency` + `request_rate` | — |
| **SLO (Traefik)** | ✅ `error_rate` + `avg_latency` | — |
| **APM Trace** | ❌ 完全未接入 | 独立于 SLO 的 Trace 级错误率、P99 延迟、RPS 突变 |
| **Logs** | ❌ 完全未接入 | 日志 error 暴增、warn 尖峰 |
| **APM 拓扑** | ❌ 未接入 | 基于 Trace 的 Service→Service 调用拓扑 |

### 1.2 问题

1. **APM 和 SLO 是独立信号源**：SLO 来自 Linkerd/Traefik Sidecar，APM 来自 OTel SDK Trace。两者覆盖范围、精度不同。当前只有 SLO，丢失了 Trace 维度的异常。

2. **日志完全不参与风险评分**：error 日志暴增是强烈的异常信号，但 AIOps 引擎完全看不到。

3. **依赖图只有 K8s 拓扑 + Linkerd Edge**：APM Trace 的 Service 调用拓扑（`APMTopology`）未被利用，风险传播路径不完整。

### 1.3 目标

**不改架构，加宽输入管道** —— 扩展 Extractor 和 Builder，让现有管道自然感知 OTel 信号。

```
现有：K8s + SLO ──→ Pipeline
目标：K8s + SLO + APM + Logs ──→ Pipeline（同一管道）
```

### 1.4 数据源

所有数据已存在于 Master 内存中，无需新增采集：

| 数据 | 来源 | 路径 |
|------|------|------|
| APM 服务指标 | `OTelSnapshot.APMServices` | `[]apm.APMService` |
| APM 拓扑 | `OTelSnapshot.APMTopology` | `*apm.Topology` |
| 日志摘要 | `OTelSnapshot.LogsSummary` | `*log.Summary` |
| 日志条目 | `OTelSnapshot.RecentLogs` | `[]log.Entry` |

---

## 2. 设计

### 2.1 总览

三处扩展点，均在现有管道中：

```
┌─────────────────────────────────────────────────────────────┐
│                     OnSnapshot 管道                          │
│                                                             │
│  1. BuildFromSnapshot(snap, otel)   ← 扩展：APM 拓扑边      │
│  2. ExtractMetrics(snap, otel)      ← 扩展：APM + Log 指标  │
│  3. ExtractDeterministicAnomalies   ← 扩展：OTel 确定性异常  │
│  4. [不变] StateManager.Update                              │
│  5. [不变] Scorer.Calculate                                 │
│  6. [不变] StateMachine.Evaluate                            │
│                                                             │
│  + DefaultRiskConfig()              ← 扩展：新指标权重       │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Phase 1：扩展 ExtractMetrics（APM + Log 指标）

**文件**：`aiops/baseline/extractor.go`

#### 2.2.1 APM 指标提取

从 `OTelSnapshot.APMServices` 提取，与 SLO 指标**独立共存**（不互斥）：

```go
func extractAPMMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
    var points []aiops.MetricDataPoint
    for _, svc := range otel.APMServices {
        key := aiops.EntityKey(svc.Namespace, "service", svc.Name)
        points = append(points,
            aiops.MetricDataPoint{EntityKey: key, MetricName: "apm_error_rate", Value: (1 - svc.SuccessRate) * 100},
            aiops.MetricDataPoint{EntityKey: key, MetricName: "apm_p99_latency", Value: svc.P99Ms},
            aiops.MetricDataPoint{EntityKey: key, MetricName: "apm_rps", Value: svc.RPS},
        )
    }
    return points
}
```

**指标说明**：

| 指标 | 值域 | 用途 | 与 SLO 的区别 |
|------|------|------|---------------|
| `apm_error_rate` | 0-100 (%) | Trace 级错误率 | SLO `error_rate` 来自 Linkerd，APM 来自 OTel Trace |
| `apm_p99_latency` | ms | Trace P99 延迟尖峰 | SLO `avg_latency` 用 P90，这里用 P99 |
| `apm_rps` | req/s | 流量突变检测 | SLO `request_rate` 来自 Linkerd，这里来自 Trace |

#### 2.2.2 Log 指标提取

从 `OTelSnapshot.LogsSummary` 提取全局日志指标，从 `RecentLogs` 提取 per-service 指标：

```go
func extractLogMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
    var points []aiops.MetricDataPoint

    // 全局日志异常（挂在虚拟实体 _cluster/logs/global 上）
    if s := otel.LogsSummary; s != nil {
        key := aiops.EntityKey("_cluster", "logs", "global")
        errorCount := float64(s.SeverityCounts["ERROR"])
        warnCount := float64(s.SeverityCounts["WARN"])
        points = append(points,
            aiops.MetricDataPoint{EntityKey: key, MetricName: "log_error_count", Value: errorCount},
            aiops.MetricDataPoint{EntityKey: key, MetricName: "log_warn_count", Value: warnCount},
        )
    }

    // Per-service 日志异常（从 RecentLogs 聚合）
    svcErrors := make(map[string]float64)
    svcTotal := make(map[string]float64)
    for _, entry := range otel.RecentLogs {
        svcTotal[entry.ServiceName]++
        if entry.SeverityText == "ERROR" {
            svcErrors[entry.ServiceName]++
        }
    }
    for svcName, total := range svcTotal {
        if total == 0 {
            continue
        }
        key := aiops.EntityKey("_cluster", "service", svcName) // 需要和 SLO 的 key 对齐
        errorRate := (svcErrors[svcName] / total) * 100
        points = append(points,
            aiops.MetricDataPoint{EntityKey: key, MetricName: "log_error_rate", Value: errorRate},
        )
    }

    return points
}
```

> **注意**：`log_error_rate` 的 EntityKey 需要与 SLO/APM 的 service 实体一致。
> 由于 RecentLogs 的 ServiceName 和 APMServices 的 Name 相同，key 匹配没有问题。
> 但需要注意 SLOServices 的 key 格式是 `EntityKey(svc.Namespace, "service", svc.Name)`，
> 而 RecentLogs 没有 namespace。解决方案：遍历 APMServices 建立 name→namespace 映射，
> 或统一用 `_cluster` 命名空间（与 ingress 一致）。
>
> **决策**：使用 APMServices 的 namespace 作为 service 的 namespace。
> 如果 RecentLogs 的 ServiceName 在 APMServices 中找不到，跳过。

#### 2.2.3 在 ExtractMetrics 中整合

```go
func ExtractMetrics(
    clusterID string,
    snap *model_v2.ClusterSnapshot,
    otel *cluster.OTelSnapshot,
) []aiops.MetricDataPoint {
    var points []aiops.MetricDataPoint
    points = append(points, extractNodeMetrics(snap)...)
    points = append(points, extractPodMetrics(snap)...)
    if otel != nil {
        points = append(points, extractServiceMetrics(otel)...)   // 已有：SLO 指标
        points = append(points, extractIngressMetrics(otel)...)   // 已有：Ingress 指标
        points = append(points, extractAPMMetrics(otel)...)       // 新增：APM 指标
        points = append(points, extractLogMetrics(otel)...)       // 新增：Log 指标
    }
    return points
}
```

### 2.3 Phase 2：扩展风险权重配置

**文件**：`aiops/risk/config.go`

```go
"service": {
    // 现有 SLO 指标
    "error_rate":   {Weight: 0.20, Channel: ChannelStatistical},   // 原 0.40 → 0.20
    "avg_latency":  {Weight: 0.15, Channel: ChannelStatistical},   // 原 0.30 → 0.15
    "request_rate": {Weight: 0.10, Channel: ChannelStatistical},   // 原 0.20 → 0.10
    // 新增 APM 指标
    "apm_error_rate":  {Weight: 0.20, Channel: ChannelBoth},       // 新增
    "apm_p99_latency": {Weight: 0.15, Channel: ChannelStatistical}, // 新增
    "apm_rps":         {Weight: 0.05, Channel: ChannelStatistical}, // 新增（流量突变辅助指标）
    // 新增 Log 指标
    "log_error_rate":  {Weight: 0.15, Channel: ChannelBoth},       // 新增
},
```

权重再分配逻辑：
- SLO 和 APM 各占一半信号权重（均为 error_rate + latency）
- Log error rate 作为独立验证信号
- 总权重不变（归一化由 Scorer 自动处理）

新增 `logs` 实体类型（全局日志虚拟实体）：

```go
"logs": {
    "log_error_count": {Weight: 0.60, Channel: ChannelBoth},
    "log_warn_count":  {Weight: 0.40, Channel: ChannelStatistical},
},
```

### 2.4 Phase 3：扩展确定性异常（OTel）

**文件**：`aiops/baseline/extractor.go`

新增函数 `ExtractOTelDeterministicAnomalies`：

```go
func ExtractOTelDeterministicAnomalies(otel *cluster.OTelSnapshot) []*aiops.AnomalyResult {
    if otel == nil {
        return nil
    }
    now := time.Now().Unix()
    var results []*aiops.AnomalyResult

    // 1. APM 服务级确定性异常
    for _, svc := range otel.APMServices {
        key := aiops.EntityKey(svc.Namespace, "service", svc.Name)
        errorRate := 1 - svc.SuccessRate

        // 5xx 爆发：error_rate > 15%
        if errorRate > 0.15 {
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "apm_error_rate",
                CurrentValue: errorRate * 100, Baseline: 0,
                Deviation: errorRate * 10, Score: 0.90,
                IsAnomaly: true, DetectedAt: now,
            })
        }

        // P99 延迟极端：> 5000ms
        if svc.P99Ms > 5000 {
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "apm_p99_latency",
                CurrentValue: svc.P99Ms, Baseline: 500,
                Deviation: svc.P99Ms / 500, Score: 0.75,
                IsAnomaly: true, DetectedAt: now,
            })
        }
    }

    // 2. 日志级确定性异常
    if s := otel.LogsSummary; s != nil {
        errorCount := s.SeverityCounts["ERROR"]
        // 全局 error > 500 条/5 分钟
        if errorCount > 500 {
            key := aiops.EntityKey("_cluster", "logs", "global")
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "log_error_count",
                CurrentValue: float64(errorCount), Baseline: 50,
                Deviation: float64(errorCount) / 50, Score: 0.80,
                IsAnomaly: true, DetectedAt: now,
            })
        }
    }

    return results
}
```

#### 在 engine.go 中调用

```go
// engine.go — OnSnapshot()
// 路径 B: 确定性异常直注
deterministicResults := baseline.ExtractDeterministicAnomalies(snap)
otelDeterministic := baseline.ExtractOTelDeterministicAnomalies(otel)       // 新增
deterministicResults = append(deterministicResults, otelDeterministic...)    // 合并
results = mergeAnomalyResults(results, deterministicResults)
```

### 2.5 Phase 4：扩展依赖图（APM 拓扑边）

**文件**：`aiops/correlator/builder.go`

签名扩展，增加 OTelSnapshot 参数：

```go
func BuildFromSnapshot(
    clusterID string,
    snap *model_v2.ClusterSnapshot,
    otel *cluster.OTelSnapshot,         // 新增参数
) *aiops.DependencyGraph {
    g := aiops.NewDependencyGraph(clusterID)

    // 1-4: 现有逻辑不变
    // ...

    // 5. Service → Service (calls, 从 APM Topology)
    if otel != nil && otel.APMTopology != nil {
        for _, edge := range otel.APMTopology.Edges {
            // TopologyEdge.Source/Target 是节点 ID（= service name）
            // 需要从 TopologyNode 获取 namespace
            srcNS := findTopologyNodeNS(otel.APMTopology.Nodes, edge.Source)
            dstNS := findTopologyNodeNS(otel.APMTopology.Nodes, edge.Target)
            srcKey := aiops.EntityKey(srcNS, "service", edge.Source)
            dstKey := aiops.EntityKey(dstNS, "service", edge.Target)
            g.AddNode(srcKey, "service", srcNS, edge.Source, nil)
            g.AddNode(dstKey, "service", dstNS, edge.Target, nil)
            g.AddEdge(srcKey, dstKey, "calls", 1.0)
        }
    }

    g.RebuildIndex()
    return g
}

func findTopologyNodeNS(nodes []apm.TopologyNode, id string) string {
    for _, n := range nodes {
        if n.Id == id {
            return n.Namespace
        }
    }
    return "_cluster"
}
```

**注意**：APM Topology 的 `calls` 边与 SLO Edge 的 `calls` 边可能重复。
由于 `DependencyGraph.AddEdge` 是追加式的，重复边会导致传播权重偏大。
解决方案：在 `BuildFromSnapshot` 中用 `edgeSet` 去重。

```go
// 在函数开头初始化
edgeSet := make(map[string]bool)

// 在所有 AddEdge 调用时检查
edgeKey := from + "|" + to + "|" + typ
if !edgeSet[edgeKey] {
    g.AddEdge(from, to, typ, weight)
    edgeSet[edgeKey] = true
}
```

#### engine.go 调用签名更新

```go
// 原：graph := correlator.BuildFromSnapshot(clusterID, snap)
// 改：graph := correlator.BuildFromSnapshot(clusterID, snap, otel)
```

---

## 3. 数据流（变更后）

```
Agent 上报 → Master DataHub (MemoryStore)
  ├── ClusterSnapshot (K8s 资源)
  └── OTelSnapshot (APM + SLO + Logs + Metrics)
           ↓
     OnSnapshot() 触发
           ↓
  ┌────────────────────────────────────────────────────────┐
  │ 1. BuildFromSnapshot(snap, otel)                       │
  │    • Pod → Node (runs_on)                [K8s]         │
  │    • Service → Pod (selects)             [K8s]         │
  │    • Ingress → Service (routes_to)       [K8s]         │
  │    • Service → Service (calls)           [SLO Edge]    │
  │    • Service → Service (calls)           [APM Topo] ★  │
  │                                                        │
  │ 2. ExtractMetrics(snap, otel)                          │
  │    • Node: cpu/mem/disk/psi × 6          [K8s]         │
  │    • Pod: restart/running/ready × 4      [K8s]         │
  │    • Service: error_rate/latency/rps × 3 [SLO]         │
  │    • Ingress: error_rate/latency × 2     [SLO]         │
  │    • Service: apm_error/p99/rps × 3      [APM] ★       │
  │    • Service: log_error_rate × 1         [Logs] ★      │
  │    • Global: log_error/warn_count × 2    [Logs] ★      │
  │                                                        │
  │ 3. ExtractDeterministicAnomalies(snap)   [K8s]         │
  │    + ExtractOTelDeterministicAnomalies(otel)    ★       │
  │    • APM error_rate > 15% → score 0.90                 │
  │    • APM P99 > 5s → score 0.75                         │
  │    • Log error > 500/5min → score 0.80                 │
  │                                                        │
  │ 4-6. [不变] StateManager → Scorer → StateMachine       │
  └────────────────────────────────────────────────────────┘
```

★ = 本次新增

---

## 4. 两层查询策略

### 4.1 Hot 层（实时，自动触发）

| 属性 | 说明 |
|------|------|
| 数据源 | Master 内存：ClusterSnapshot + OTelSnapshot Ring Buffer (15min/10s=90条) |
| 触发 | 每次 Agent 上报快照时 `OnSnapshot()` |
| 计算 | 全量异常检测 + 风险评分 + 状态机评估 |
| 延迟 | < 100ms |
| 用途 | 实时异常发现 → Incident 创建 |

### 4.2 Cold 层（回溯，按需触发）

现有的 AI 增强模块（`aiops/ai/`）已通过 LLM 做事件分析。
Cold 层不需要新建 —— 当用户在前端点击「深度分析」时，
AI enhancer 的 `context_builder.go` 可以通过 Command 机制查询 ClickHouse 历史数据，
作为 LLM prompt 的上下文传入。

**不在本次范围内实现。** 现有 AI 增强模块已具备基础能力，后续可扩展 context builder。

---

## 5. Observe Landing Page 定位调整

### 5.1 现状

刚完成的三层 drill-down 实现（K8s 资源 + 可观测信号关联式视图）。

### 5.2 问题

与已有的 APM/Log/Metrics/SLO 独立页面功能重叠。
GPT 分析正确：全量展示在正常时无意义，异常时才有价值。

### 5.3 定位

**暂时保留现有实现**，后续考虑改造为 AIOps 入口：

- Level 1：服务列表 → 按 AIOps 风险分降序排列（替代简单的 critical/warning/healthy 排序）
- Level 2：服务详情 → 无异常面板显示「正常」，有异常面板展开因果链
- Level 3：直接跳转到 AIOps incidents 页面的 EntityRiskDetail

**不在本次范围内改造。** 先完成后端 AIOps OTel 融合。

---

## 6. 文件变更清单

| # | 文件 | 操作 | 说明 |
|---|------|------|------|
| 1 | `aiops/baseline/extractor.go` | 修改 | 新增 `extractAPMMetrics` + `extractLogMetrics` + `ExtractOTelDeterministicAnomalies` |
| 2 | `aiops/risk/config.go` | 修改 | service 权重再分配 + 新增 logs 实体类型 |
| 3 | `aiops/correlator/builder.go` | 修改 | 签名增加 OTelSnapshot 参数 + APM 拓扑边 + 边去重 |
| 4 | `aiops/core/engine.go` | 修改 | `OnSnapshot` 调用新的 OTel 确定性异常 + 传递 otel 给 builder |
| **合计** | | **4 文件修改** | 无新建文件 |

---

## 7. 指标总览（变更后）

### 7.1 实体类型 → 指标映射

| 实体类型 | 指标名 | 来源 | 权重 | 通道 |
|----------|--------|------|------|------|
| **node** | `cpu_usage` | K8s NodeMetrics | 0.25 | Statistical |
| | `memory_usage` | K8s NodeMetrics | 0.25 | Statistical |
| | `disk_usage` | K8s NodeMetrics | 0.20 | Statistical |
| | `psi_cpu` | K8s NodeMetrics | 0.10 | Statistical |
| | `psi_memory` | K8s NodeMetrics | 0.10 | Statistical |
| | `psi_io` | K8s NodeMetrics | 0.10 | Statistical |
| **pod** | `restart_count` | K8s Pod | 0.20 | Both |
| | `is_running` | K8s Pod | 0.10 | Both |
| | `not_ready_containers` | K8s Pod | 0.20 | Both |
| | `max_container_restarts` | K8s Pod | 0.10 | Both |
| | `container_anomaly` | K8s Container | 0.25 | Deterministic |
| | `critical_event` | K8s Event | 0.15 | Deterministic |
| | `deployment_impact` | K8s Deployment | 0.25 | Deterministic |
| **service** | `error_rate` | SLO (Linkerd) | 0.20 | Statistical |
| | `avg_latency` | SLO (Linkerd) | 0.15 | Statistical |
| | `request_rate` | SLO (Linkerd) | 0.10 | Statistical |
| | `apm_error_rate` ★ | APM (OTel Trace) | 0.20 | Both |
| | `apm_p99_latency` ★ | APM (OTel Trace) | 0.15 | Statistical |
| | `apm_rps` ★ | APM (OTel Trace) | 0.05 | Statistical |
| | `log_error_rate` ★ | Logs (OTel Logs) | 0.15 | Both |
| **ingress** | `error_rate` | SLO (Traefik) | 0.50 | Statistical |
| | `avg_latency` | SLO (Traefik) | 0.50 | Statistical |
| **logs** ★ | `log_error_count` | Logs (OTel Logs) | 0.60 | Both |
| | `log_warn_count` | Logs (OTel Logs) | 0.40 | Statistical |

★ = 本次新增

### 7.2 确定性异常触发条件

| 来源 | 条件 | 指标名 | Score |
|------|------|--------|-------|
| K8s Container | CrashLoopBackOff | `container_anomaly` | 0.90 |
| K8s Container | OOMKilled | `container_anomaly` | 0.95 |
| K8s Container | ImagePullBackOff | `container_anomaly` | 0.70 |
| K8s Container | NotReady | `container_anomaly` | 0.60 |
| K8s Event | Critical Event 5min 内 | `critical_event` | 0.85 |
| K8s Deployment | 不可用比例 ≥ 75% | `deployment_impact` | 0.95 |
| K8s Deployment | 不可用比例 ≥ 50% | `deployment_impact` | 0.80 |
| APM ★ | error_rate > 15% | `apm_error_rate` | 0.90 |
| APM ★ | P99 > 5000ms | `apm_p99_latency` | 0.75 |
| Logs ★ | error > 500 条/5min | `log_error_count` | 0.80 |

### 7.3 依赖图边类型

| 边类型 | 方向 | 来源 |
|--------|------|------|
| `runs_on` | Pod → Node | K8s Pod.NodeName |
| `selects` | Service → Pod | K8s Service.Selector |
| `routes_to` | Ingress → Service | K8s Ingress.Rules |
| `calls` | Service → Service | SLO Edge (Linkerd) |
| `calls` ★ | Service → Service | APM Topology (OTel Trace) |

---

## 8. 验证

```bash
# 编译验证
go build ./atlhyper_master_v2/...

# 单元测试
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractAPMMetrics
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractLogMetrics
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractOTelDeterministicAnomalies
go test ./atlhyper_master_v2/aiops/correlator/ -v -run TestBuildFromSnapshot_APMTopology
go test ./atlhyper_master_v2/aiops/risk/ -v -run TestServiceRisk_WithAPM
```

端到端验证：
1. 部署带 OTel 采集的服务到 K3s 集群
2. 等待 AIOps 引擎冷启动完成（~100 个数据点，约 17 分钟）
3. 检查 AIOps 风险仪表盘：service 实体应显示 APM + Log 指标
4. 人为注入错误（如 5xx 响应），验证确定性异常触发
5. 检查依赖图：应包含 APM Topology 的 service→service 边

---

## 9. 已确认决策

| 问题 | 决策 |
|------|------|
| APM 和 SLO 指标是否互斥 | **共存**：同一 service 实体可能同时有 SLO `error_rate` 和 APM `apm_error_rate`，各自独立检测 |
| Log 指标的 EntityKey | **与 APM 对齐**：使用 APMServices 的 namespace + name 构建 key |
| APM 和 SLO 的拓扑边重复 | **去重**：用 `edgeSet` 确保同一 from→to→type 只添加一次 |
| 新指标的权重分配 | **再分配**：SLO 原始权重降半，腾出空间给 APM + Log |
| logs 虚拟实体 | **新增实体类型** `logs`：挂载全局日志指标，参与风险传播 |
| Cold 层实现 | **不在本次范围**：现有 AI enhancer 已具备 LLM 分析能力 |
| Landing Page 改造 | **不在本次范围**：先完成后端，前端后续迭代 |
