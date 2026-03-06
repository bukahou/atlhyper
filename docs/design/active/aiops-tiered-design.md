# AIOps 分层架构设计

> 状态: active | 创建: 2026-03-06
> 关联文档:
> - [ai-background-analysis-design.md](./ai-background-analysis-design.md) — AI 角色实现（依赖本文档）
> - [ai-role-definition-design.md](./ai-role-definition-design.md) — AI 角色定义

## 1. 背景

### 1.1 问题

AIOps 引擎是 AI 角色（background/analysis）的**信息源**。没有完整的异常检测数据，AI 角色只能基于 K8s 资源状态做表面分析，无法触及业务层（APM 延迟、日志错误率、服务调用链）。

但 AtlHyper 支持两种部署模式：

| 部署模式 | 组件 | OTel 数据 |
|---------|------|----------|
| **基础部署** | Agent + Master | 无（无 OTel Collector / ClickHouse） |
| **全栈部署** | Agent + Master + OTel Collector + ClickHouse | APM / Logs / SLO / Node Metrics |

基础部署的用户没有业务可观测数据，AIOps 只能依赖 K8s 快照。全栈部署的用户有完整的 OTel 信号，AIOps 应充分利用。

### 1.2 目标

设计分层 AIOps 引擎，让两种部署模式都能获得最大化的异常检测能力：

- **Basic 层**：仅依赖 K8s API（ClusterSnapshot + Metrics Server），零 OTel 依赖
- **Enhanced 层**：在 Basic 基础上叠加 OTel 信号（深度 Node 指标 + SLO + APM + Logs + Topology）

**关键约束**：Engine 不应显式切换模式。OTel 数据有就用，没有就跳过——**数据驱动，非配置驱动**。

### 1.3 数据隔离原则

AIOps 模块**不依赖 Observe Service 层**（`service/query/`）。

两者的数据关系：

```
Agent → Processor → DataHub Store ←── AIOps Engine（GetSnapshot → snap.OTel）
                                  ←── Observe Handler（快照直读 + Command/MQ 实时查询）
```

- **AIOps**：直接从 `datahub.Store.GetSnapshot()` 读取 `snap.OTel`（完整 OTelSnapshot）
- **Observe**：双数据路径
  - 15 分钟快照直读（Dashboard 页面，`GetOTelSnapshot()`）
  - Command/MQ 实时查询（自定义时间范围 / Trace Detail / Log Query，Agent ClickHouse 实时查询）
- 两者**共享底层数据存储**（读共享），但**无模块间依赖**
- 符合架构规范：AIOps 可访问 Store（读）和 Database（读写）

### 1.4 AIOps 数据源：snap.OTel 而非 Ring Buffer

**关键设计决策**：AIOps 从 `GetSnapshot()` 返回的 `snap.OTel` 直接读取，**不使用** Ring Buffer 的 `GetOTelTimeline()`。

原因：`datahub/memory/store.go` 中的 `lightweightOTelCopy()` 会剥离 Ring Buffer 中的大体积字段以防止 OOM：

| 字段 | snap.OTel | Ring Buffer |
|------|-----------|-------------|
| 标量摘要（15 字段） | ✅ | ✅ |
| MetricsNodes / APMServices | ✅ | ✅ |
| SLOIngress / SLOServices / SLOEdges | ✅ | ✅ |
| **APMTopology** | ✅ | ❌ 被剥离 |
| **APMOperations** | ✅ | ❌ 被剥离 |
| **RecentTraces** | ✅ | ❌ 被剥离 |
| **RecentLogs** | ✅ | ❌ 被剥离 |
| **LogsSummary** | ✅ | ❌ 被剥离 |
| **SLOWindows** (1d/7d/30d) | ✅ | ❌ 被剥离 |
| **MetricsSummary / SLOSummary** | ✅ | ❌ 被剥离 |
| 预聚合时序（Series） | ✅ | ❌ 被剥离 |

AIOps Enhanced 层需要 APMTopology（拓扑边）、LogsSummary（日志指标）、RecentLogs（per-service 日志统计），这些在 Ring Buffer 中都被剥离了。因此 **必须从 `snap.OTel` 直接读取**。

### 1.5 数据充足性分析

15 分钟快照聚合数据对 AIOps 的充足性评估：

| 数据需求 | snap.OTel 中是否可用 | 说明 |
|---------|-------------------|------|
| APM 每服务指标 (SuccessRate/P99/RPS) | ✅ `APMServices` | 15 分钟窗口聚合 |
| APM 操作级统计 | ✅ `APMOperations` | per-operation 聚合 |
| APM 拓扑（服务调用关系） | ✅ `APMTopology` | 服务间依赖边 |
| SLO 状态码分布 (per-service) | ✅ `SLOIngress[].StatusCodes` / `SLOServices[].StatusCodes` | HTTP 500/400 计数按服务分组 |
| SLO 多窗口 (1d/7d/30d) | ✅ `SLOWindows` | 长期 SLO 趋势 |
| 日志全局 severity 统计 | ✅ `LogsSummary.SeverityCounts` | ERROR/WARN 全局计数 |
| 日志按服务 severity 分布 | ⚠️ 间接可用 | 从 `RecentLogs`（500 条/5min）逐条统计，非预聚合 |
| 节点 Metrics (ClickHouse) | ✅ `MetricsNodes` | per-node 详细指标 |

**唯一缺口**：日志按服务的 severity 分布没有预聚合字段（`log.Summary` 只有全局 `SeverityCounts`）。当前通过遍历 `RecentLogs` 逐条统计实现（见变更 5 的 `extractLogMetrics`）。如需更精确的 per-service 日志统计，可后续在 Agent 聚合时补充 `ServiceSeverityCounts` 字段到 `log.Summary`，不阻塞当前设计。

**结论**：`snap.OTel` 的 15 分钟聚合数据**基本满足** AIOps Enhanced 层需求。

---

## 2. 分层定义

### 2.1 Basic 层（K8s 原生）

**前置条件**：Agent + Master 部署即可。Node 指标需要集群安装 Metrics Server（K8s 标配组件）。

#### 2.1.1 现有能力（已实现）

| 能力 | 数据源 | 现状 |
|------|--------|------|
| Pod 状态异常检测 | `snap.Pods`（K8s API） | ✅ 已实现 |
| 容器确定性异常 | `snap.Pods[].Containers`（CrashLoop/OOM/ImagePull） | ✅ 已实现 |
| K8s Event 关联 | `snap.Events`（Critical Event） | ✅ 已实现 |
| Deployment 影响比例 | `snap.Deployments + snap.ReplicaSets + snap.Pods` | ✅ 已实现 |
| 依赖图（K8s 拓扑） | Pod→Node, Service→Pod, Ingress→Service | ✅ 已实现 |
| 风险评分 + 传播 | 3 阶段 Scorer（局部→时序→传播） | ✅ 已实现 |
| 状态机 + 事件管理 | Healthy→Warning→Incident→Recovery→Stable | ✅ 已实现 |

#### 2.1.2 需要新增的能力（本次改造）

| 能力 | 数据源 | 说明 |
|------|--------|------|
| **Node CPU/Memory 异常检测** | `snap.Nodes[].Metrics`（K8s Metrics Server） | 当前只从 `snap.OTel.MetricsNodes` 读取（依赖 ClickHouse），需改为优先读 Metrics Server |
| **Node 压力标志确定性异常** | `snap.Nodes[].Metrics.Pressure`（K8s Node Conditions） | MemoryPressure/DiskPressure/PIDPressure = 确定性异常 |

**关键修正**：当前 `extractNodeMetrics()` 只从 `snap.OTel.MetricsNodes` 读取，这依赖 ClickHouse。但 Agent 已通过 K8s Metrics Server API 采集 Node 资源使用率，数据存在 `snap.Nodes[].Metrics` 中：

```go
// model_v3/cluster/node.go — 已存在的数据结构
type Node struct {
    Metrics *NodeResourceUsage `json:"metrics,omitempty"` // ← 来自 K8s Metrics Server
}

type NodeResourceUsage struct {
    CPU      NodeResourceMetric `json:"cpu"`      // CPU.UtilPct = 使用百分比
    Memory   NodeResourceMetric `json:"memory"`   // Memory.UtilPct = 使用百分比
    Pods     PodCountMetric     `json:"pods"`
    Pressure PressureFlags      `json:"pressure"` // MemoryPressure/DiskPressure/PIDPressure
}
```

**Basic 层的 Node 指标**（来自 Metrics Server，无 ClickHouse 依赖）：

| 指标名 | 数据源 | 说明 |
|--------|--------|------|
| `cpu_usage` | `node.Metrics.CPU.UtilPct` | CPU 使用百分比 |
| `memory_usage` | `node.Metrics.Memory.UtilPct` | 内存使用百分比 |
| `memory_pressure` | `node.Metrics.Pressure.MemoryPressure` | 确定性异常（bool→score） |
| `disk_pressure` | `node.Metrics.Pressure.DiskPressure` | 确定性异常（bool→score） |
| `pid_pressure` | `node.Metrics.Pressure.PIDPressure` | 确定性异常（bool→score） |

**Enhanced 层额外提供的深度 Node 指标**（来自 ClickHouse OTel，`snap.OTel.MetricsNodes`）：

| 指标名 | 数据源 | Basic 层无法提供的原因 |
|--------|--------|----------------------|
| `disk_usage` | OTel MetricsNodes | Metrics Server 不提供磁盘使用率 |
| `psi_cpu` | OTel MetricsNodes | PSI 精确百分比需要 node_exporter |
| `psi_memory` | OTel MetricsNodes | 同上 |
| `psi_io` | OTel MetricsNodes | 同上 |

#### 2.1.3 Basic 层的限制

| 缺失 | 影响 |
|------|------|
| 无 SLO 指标 | Service 实体无 error_rate/latency/rps，只能通过下辖 Pod 异常间接感知 |
| 无 APM 指标 | 无法检测业务级延迟飙升、错误率突变 |
| 无日志分析 | 无法感知 error 日志暴增 |
| 无 Service→Service 调用边 | 依赖图缺少服务间调用关系，风险传播路径不完整 |
| Node 无磁盘/PSI 指标 | 磁盘和 I/O 压力需要 Enhanced 层 |

### 2.2 Enhanced 层（K8s + OTel 全链路）

**前置条件**：全栈部署（Agent + Master + OTel Collector + ClickHouse）。

在 Basic 基础上叠加：

| 能力 | 数据源 | 现状 |
|------|--------|------|
| Node 深度指标（Disk/PSI） | `otel.MetricsNodes`（node_exporter → ClickHouse） | ✅ 已实现 |
| SLO 指标（Mesh） | `otel.SLOServices`（Linkerd Proxy） | ✅ 已实现 |
| SLO 指标（Ingress） | `otel.SLOIngress`（Traefik） | ✅ 已实现 |
| SLO 调用边 | `snap.OTel.SLOEdges`（Linkerd） | ✅ 已实现 |
| **APM 指标** | `otel.APMServices`（OTel Trace） | ❌ 待实现 |
| **APM 确定性异常** | `otel.APMServices`（error>15%, P99>5s） | ❌ 待实现 |
| **APM 拓扑边** | `otel.APMTopology`（Service→Service calls） | ❌ 待实现 |
| **日志指标** | `otel.LogsSummary + otel.RecentLogs` | ❌ 待实现 |
| **日志确定性异常** | `otel.LogsSummary`（error>500/5min） | ❌ 待实现 |

> **确认**：SLO 三项能力（Service 指标 + Ingress 指标 + Service→Service 调用边）已在 AIOps 管道中完整工作。
> 链路验证：Agent ClickHouse 查询 → `snap.OTel` 上报 → DataHub 存储 → `extractServiceMetrics`/`extractIngressMetrics` 提取 → `risk/config.go` 权重 → Scorer 评分。

---

## 3. 设计

### 3.1 核心原则：nil 保护 = 自动降级

Engine 已经在用 `if otel != nil` 做 nil 保护。扩展这个模式到所有新增信号提取。

**不需要任何配置开关或模式切换。** 数据有就提取，没有就跳过。

```
OnSnapshot 管道:

1. BuildFromSnapshot(clusterID, snap, otel)     ← otel=nil 时跳过 SLO Edge + APM Topology
2. ExtractMetrics(clusterID, snap, otel)         ← 始终提取 K8s 指标，otel!=nil 时追加 OTel 指标
3. ExtractDeterministicAnomalies(snap)           ← K8s 确定性异常（始终执行，含 Node Pressure）
   ExtractOTelDeterministicAnomalies(otel)       ← otel=nil 时返回空（新增）
4-6. StateManager → Scorer → StateMachine        ← 不变，处理可用的数据
```

### 3.2 变更 1: extractNodeMetrics 改造（Basic 层核心）

**文件**: `aiops/baseline/extractor.go`

**当前实现**（只从 OTel 读取，依赖 ClickHouse）：

```go
func extractNodeMetrics(snap *cluster.ClusterSnapshot) []aiops.MetricDataPoint {
    if snap.OTel == nil { return points }  // ← 无 OTel 就完全跳过
    for _, nm := range snap.OTel.MetricsNodes {
        // 从 ClickHouse 的 MetricsNodes 读取
    }
}
```

**改造后**（优先 Metrics Server，OTel 补充深度指标）：

```go
func extractNodeMetrics(snap *cluster.ClusterSnapshot) []aiops.MetricDataPoint {
    var points []aiops.MetricDataPoint

    // === Basic 层：从 K8s Metrics Server 读取 CPU/Memory ===
    for i := range snap.Nodes {
        node := &snap.Nodes[i]
        key := aiops.EntityKey("_cluster", "node", node.GetName())

        if node.Metrics != nil {
            points = append(points,
                aiops.MetricDataPoint{EntityKey: key, MetricName: "cpu_usage", Value: node.Metrics.CPU.UtilPct},
                aiops.MetricDataPoint{EntityKey: key, MetricName: "memory_usage", Value: node.Metrics.Memory.UtilPct},
            )
        }
    }

    // === Enhanced 层：从 OTel MetricsNodes 补充深度指标 ===
    if snap.OTel != nil {
        for i := range snap.OTel.MetricsNodes {
            nm := &snap.OTel.MetricsNodes[i]
            key := aiops.EntityKey("_cluster", "node", nm.NodeName)

            // Disk（Metrics Server 不提供）
            if disk := nm.GetPrimaryDisk(); disk != nil {
                points = append(points,
                    aiops.MetricDataPoint{EntityKey: key, MetricName: "disk_usage", Value: disk.UsagePct},
                )
            }

            // PSI 精确值（Metrics Server 不提供，只有 bool 压力标志）
            points = append(points,
                aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_cpu", Value: nm.PSI.CPUSomePct},
                aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_memory", Value: nm.PSI.MemSomePct},
                aiops.MetricDataPoint{EntityKey: key, MetricName: "psi_io", Value: nm.PSI.IOSomePct},
            )

            // 如果 Basic 层没有 Metrics Server 数据，用 OTel 数据兜底
            // （检查该节点是否已有 cpu_usage 指标）
            if !hasMetricsServerData(snap, nm.NodeName) {
                points = append(points,
                    aiops.MetricDataPoint{EntityKey: key, MetricName: "cpu_usage", Value: nm.CPU.UsagePct},
                    aiops.MetricDataPoint{EntityKey: key, MetricName: "memory_usage", Value: nm.Memory.UsagePct},
                )
            }
        }
    }

    return points
}

// hasMetricsServerData 检查指定节点是否有 K8s Metrics Server 数据
func hasMetricsServerData(snap *cluster.ClusterSnapshot, nodeName string) bool {
    for i := range snap.Nodes {
        if snap.Nodes[i].GetName() == nodeName && snap.Nodes[i].Metrics != nil {
            return true
        }
    }
    return false
}
```

**降级行为**：
- Metrics Server 可用 → Basic 层 CPU/Memory 正常工作
- Metrics Server 不可用 → `node.Metrics = nil`，如有 OTel 则兜底
- OTel 不可用 → 无 Disk/PSI 指标

### 3.3 变更 2: Node 压力确定性异常（Basic 层新增）

**文件**: `aiops/baseline/extractor.go`

在 `ExtractDeterministicAnomalies` 中新增 Node 压力检测：

```go
func ExtractDeterministicAnomalies(snap *cluster.ClusterSnapshot) []*aiops.AnomalyResult {
    now := time.Now().Unix()
    var results []*aiops.AnomalyResult

    // 路径 B1: 容器状态异常（已有）
    results = append(results, extractContainerAnomalies(snap, now)...)
    // 路径 B2: Event 关联异常（已有）
    results = append(results, extractEventAnomalies(snap, now)...)
    // 路径 B3: Deployment 影响比例异常（已有）
    results = append(results, extractDeploymentImpact(snap, now)...)
    // 路径 B4: Node 压力确定性异常（新增，Basic 层）
    results = append(results, extractNodePressure(snap, now)...)

    return results
}

// extractNodePressure 从 Node Conditions 提取确定性压力异常
// 数据来自 K8s API（Metrics Server），不依赖 OTel
func extractNodePressure(snap *cluster.ClusterSnapshot, now int64) []*aiops.AnomalyResult {
    var results []*aiops.AnomalyResult
    for i := range snap.Nodes {
        node := &snap.Nodes[i]
        if node.Metrics == nil {
            continue
        }
        key := aiops.EntityKey("_cluster", "node", node.GetName())

        if node.Metrics.Pressure.MemoryPressure {
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "memory_pressure",
                CurrentValue: 1, Baseline: 0, Deviation: 10,
                Score: 0.85, IsAnomaly: true, DetectedAt: now,
            })
        }
        if node.Metrics.Pressure.DiskPressure {
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "disk_pressure",
                CurrentValue: 1, Baseline: 0, Deviation: 10,
                Score: 0.80, IsAnomaly: true, DetectedAt: now,
            })
        }
        if node.Metrics.Pressure.PIDPressure {
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "pid_pressure",
                CurrentValue: 1, Baseline: 0, Deviation: 10,
                Score: 0.75, IsAnomaly: true, DetectedAt: now,
            })
        }
    }
    return results
}
```

### 3.4 变更 3: Node 风险权重配置更新

**文件**: `aiops/risk/config.go`

```go
"node": {
    // Basic 层（K8s Metrics Server）
    "cpu_usage":    {Weight: 0.25, Channel: ChannelStatistical},
    "memory_usage": {Weight: 0.25, Channel: ChannelStatistical},
    "memory_pressure": {Weight: 0.15, Channel: ChannelDeterministic},  // 新增
    "disk_pressure":   {Weight: 0.10, Channel: ChannelDeterministic},  // 新增
    "pid_pressure":    {Weight: 0.05, Channel: ChannelDeterministic},  // 新增
    // Enhanced 层（OTel/ClickHouse）
    "disk_usage":   {Weight: 0.10, Channel: ChannelStatistical},  // 0.20 → 0.10
    "psi_cpu":      {Weight: 0.05, Channel: ChannelStatistical},  // 0.10 → 0.05
    "psi_memory":   {Weight: 0.03, Channel: ChannelStatistical},  // 0.10 → 0.03
    "psi_io":       {Weight: 0.02, Channel: ChannelStatistical},  // 0.10 → 0.02
},
```

Basic 层权重 = 0.80（CPU + Memory + Pressure），Enhanced 层权重 = 0.20（Disk + PSI）。Scorer 自动归一化。

### 3.5 变更 4: BuildFromSnapshot 签名扩展

**文件**: `aiops/correlator/builder.go`

```go
// 当前:
func BuildFromSnapshot(clusterID string, snap *cluster.ClusterSnapshot) *aiops.DependencyGraph

// 改为:
func BuildFromSnapshot(clusterID string, snap *cluster.ClusterSnapshot, otel *cluster.OTelSnapshot) *aiops.DependencyGraph
```

新增内容：

```go
// 边去重集合（全局，所有步骤共用）
edgeSet := make(map[string]bool)

// 步骤 1-3: 现有 K8s 拓扑逻辑不变（runs_on, selects, routes_to）
// ...

// 步骤 4: SLO Edge（已有，改用 otel 参数而非 snap.OTel）
if otel != nil {
    for _, edge := range otel.SLOEdges {
        srcKey := aiops.EntityKey(edge.SrcNamespace, "service", edge.SrcName)
        dstKey := aiops.EntityKey(edge.DstNamespace, "service", edge.DstName)
        g.AddNode(srcKey, "service", edge.SrcNamespace, edge.SrcName, nil)
        g.AddNode(dstKey, "service", edge.DstNamespace, edge.DstName, nil)
        ek := srcKey + "|" + dstKey + "|calls"
        if !edgeSet[ek] {
            g.AddEdge(srcKey, dstKey, "calls", 1.0)
            edgeSet[ek] = true
        }
    }
}

// 步骤 5: APM Topology 边（新增，Enhanced 层）
if otel != nil && otel.APMTopology != nil {
    for _, edge := range otel.APMTopology.Edges {
        srcNS := findTopologyNodeNS(otel.APMTopology.Nodes, edge.Source)
        dstNS := findTopologyNodeNS(otel.APMTopology.Nodes, edge.Target)
        srcKey := aiops.EntityKey(srcNS, "service", edge.Source)
        dstKey := aiops.EntityKey(dstNS, "service", edge.Target)
        g.AddNode(srcKey, "service", srcNS, edge.Source, nil)
        g.AddNode(dstKey, "service", dstNS, edge.Target, nil)
        ek := srcKey + "|" + dstKey + "|calls"
        if !edgeSet[ek] {
            g.AddEdge(srcKey, dstKey, "calls", 1.0)
            edgeSet[ek] = true
        }
    }
}
```

**注意**：当前 `builder.go` 步骤 4 从 `snap.OTel.SLOEdges` 读取。改造后统一从 `otel` 参数读取。`otel` 来自 `snap.OTel`（完整数据），不经过 Ring Buffer。

**降级行为**: `otel == nil` → 跳过 SLO Edge + APM Topology，依赖图只有 K8s 拓扑。

### 3.6 变更 5: ExtractMetrics 扩展（APM + Log）

**文件**: `aiops/baseline/extractor.go`

```go
func ExtractMetrics(clusterID string, snap *cluster.ClusterSnapshot, otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
    var points []aiops.MetricDataPoint
    points = append(points, extractNodeMetrics(snap)...)     // Basic + Enhanced Node
    points = append(points, extractPodMetrics(snap)...)      // Basic Pod
    if otel != nil {
        points = append(points, extractServiceMetrics(otel)...)   // Enhanced SLO（已有）
        points = append(points, extractIngressMetrics(otel)...)   // Enhanced SLO（已有）
        points = append(points, extractAPMMetrics(otel)...)       // Enhanced APM（新增）
        points = append(points, extractLogMetrics(otel)...)       // Enhanced Logs（新增）
    }
    return points
}
```

新增两个函数：

```go
// extractAPMMetrics 从 OTelSnapshot.APMServices 提取 APM 指标
// 与 SLO 指标共存（不互斥），同一 service 实体可能同时有两套指标
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

// extractLogMetrics 从 OTelSnapshot 提取日志指标
func extractLogMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
    var points []aiops.MetricDataPoint

    // 全局日志异常
    if s := otel.LogsSummary; s != nil {
        key := aiops.EntityKey("_cluster", "logs", "global")
        points = append(points,
            aiops.MetricDataPoint{EntityKey: key, MetricName: "log_error_count", Value: float64(s.SeverityCounts["ERROR"])},
            aiops.MetricDataPoint{EntityKey: key, MetricName: "log_warn_count", Value: float64(s.SeverityCounts["WARN"])},
        )
    }

    // Per-service 日志 error rate（需 APMServices 提供 namespace 映射）
    nsMap := make(map[string]string)
    for _, svc := range otel.APMServices {
        nsMap[svc.Name] = svc.Namespace
    }
    svcErrors := make(map[string]float64)
    svcTotal := make(map[string]float64)
    for _, entry := range otel.RecentLogs {
        if _, ok := nsMap[entry.ServiceName]; !ok {
            continue
        }
        svcTotal[entry.ServiceName]++
        if entry.SeverityText == "ERROR" {
            svcErrors[entry.ServiceName]++
        }
    }
    for svcName, total := range svcTotal {
        if total == 0 {
            continue
        }
        key := aiops.EntityKey(nsMap[svcName], "service", svcName)
        points = append(points,
            aiops.MetricDataPoint{EntityKey: key, MetricName: "log_error_rate", Value: (svcErrors[svcName] / total) * 100},
        )
    }
    return points
}
```

### 3.7 变更 6: OTel 确定性异常（Enhanced 层）

**文件**: `aiops/baseline/extractor.go`

```go
// ExtractOTelDeterministicAnomalies 从 OTelSnapshot 提取确定性异常
// otel=nil 时返回空——自动降级到 Basic 层
func ExtractOTelDeterministicAnomalies(otel *cluster.OTelSnapshot) []*aiops.AnomalyResult {
    if otel == nil {
        return nil
    }
    now := time.Now().Unix()
    var results []*aiops.AnomalyResult

    // APM 服务级确定性异常
    for _, svc := range otel.APMServices {
        key := aiops.EntityKey(svc.Namespace, "service", svc.Name)
        errorRate := 1 - svc.SuccessRate

        if errorRate > 0.15 { // 5xx 爆发: error_rate > 15%
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "apm_error_rate",
                CurrentValue: errorRate * 100, Baseline: 0,
                Deviation: errorRate * 10, Score: 0.90,
                IsAnomaly: true, DetectedAt: now,
            })
        }
        if svc.P99Ms > 5000 { // P99 延迟极端: > 5000ms
            results = append(results, &aiops.AnomalyResult{
                EntityKey: key, MetricName: "apm_p99_latency",
                CurrentValue: svc.P99Ms, Baseline: 500,
                Deviation: svc.P99Ms / 500, Score: 0.75,
                IsAnomaly: true, DetectedAt: now,
            })
        }
    }

    // 日志级确定性异常: error > 500 条/5min
    if s := otel.LogsSummary; s != nil {
        if errorCount := s.SeverityCounts["ERROR"]; errorCount > 500 {
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

### 3.8 变更 7: Engine OnSnapshot 编排

**文件**: `aiops/core/engine.go`

```go
func (e *engine) OnSnapshot(clusterID string) {
    snap, err := e.store.GetSnapshot(clusterID)
    if err != nil || snap == nil {
        return
    }

    // 直接从最新快照获取 OTelSnapshot（可能为 nil = Basic 层）
    // 注意：不使用 Ring Buffer 的 GetOTelTimeline()，因为 lightweightOTelCopy 会剥离
    // APMTopology/LogsSummary/RecentLogs/SLOWindows 等 AIOps 需要的字段
    otel := snap.OTel

    // 1. 构建依赖图（传入 otel，nil 安全）
    graph := correlator.BuildFromSnapshot(clusterID, snap, otel)   // ← 签名变更
    e.corr.Update(clusterID, graph)
    // ... (持久化逻辑不变) ...

    // 2. 清理过期实体
    activeKeys := extractActiveEntityKeys(snap, otel)
    // ...

    // 3. 提取指标
    points := baseline.ExtractMetrics(clusterID, snap, otel)
    if len(points) == 0 {
        return
    }
    results := e.stateManager.Update(points)

    // 路径 B: 确定性异常
    deterministicResults := baseline.ExtractDeterministicAnomalies(snap)         // K8s（含 Node Pressure）
    otelDeterministic := baseline.ExtractOTelDeterministicAnomalies(otel)       // OTel（APM/Log）
    deterministicResults = append(deterministicResults, otelDeterministic...)
    results = mergeAnomalyResults(results, deterministicResults)

    // 4-6: 不变
    // ...
}
```

### 3.9 变更 8: Service 风险权重再分配

**文件**: `aiops/risk/config.go`

```go
"service": {
    // SLO 指标（已有，降低权重给 APM/Log 腾出空间）
    "error_rate":   {Weight: 0.20, Channel: ChannelStatistical},   // 0.40 → 0.20
    "avg_latency":  {Weight: 0.15, Channel: ChannelStatistical},   // 0.30 → 0.15
    "request_rate": {Weight: 0.10, Channel: ChannelStatistical},   // 0.20 → 0.10
    // APM 指标（新增）
    "apm_error_rate":  {Weight: 0.20, Channel: ChannelBoth},
    "apm_p99_latency": {Weight: 0.15, Channel: ChannelStatistical},
    "apm_rps":         {Weight: 0.05, Channel: ChannelStatistical},
    // Log 指标（新增）
    "log_error_rate":  {Weight: 0.15, Channel: ChannelBoth},
},

// 新增: logs 虚拟实体
"logs": {
    "log_error_count": {Weight: 0.60, Channel: ChannelBoth},
    "log_warn_count":  {Weight: 0.40, Channel: ChannelStatistical},
},
```

**权重自动归一化**：Scorer 已有归一化逻辑。当 Basic 层无 OTel service 指标时，该实体类型无指标点，Scorer 不处理。

### 3.10 变更 9: extractActiveEntityKeys 扩展

```go
func extractActiveEntityKeys(snap *cluster.ClusterSnapshot, otel *cluster.OTelSnapshot) map[string]bool {
    keys := make(map[string]bool, ...)

    // K8s 实体（始终可用）
    // ... 现有 Pod/Node/Service/Ingress 逻辑 ...

    if otel != nil {
        // SLO 实体（已有）
        for _, svc := range otel.SLOServices { ... }
        for _, ing := range otel.SLOIngress { ... }

        // APM 实体（新增）
        for _, svc := range otel.APMServices {
            keys[aiops.EntityKey(svc.Namespace, "service", svc.Name)] = true
        }

        // Logs 虚拟实体（新增）
        if otel.LogsSummary != nil {
            keys[aiops.EntityKey("_cluster", "logs", "global")] = true
        }
    }
    return keys
}
```

---

## 4. 降级行为矩阵

| 部署场景 | Node 指标 | Service 指标 | 依赖图边 | 确定性异常 |
|---------|----------|-------------|---------|----------|
| **纯 K8s（无 Metrics Server）** | 无 | 无 | K8s 拓扑 | Container + Event + Deployment |
| **K8s + Metrics Server** | CPU + Memory + Pressure | 无 | K8s 拓扑 | + Node Pressure |
| **+ ClickHouse（OTel）** | + Disk + PSI | + SLO | + SLO Edge | 同上 |
| **全栈（+ APM + Logs）** | 同上 | + APM + Log | + APM Topology | + APM error + Log error |

> 每一行都是上一行的超集。Engine 不需要感知当前处于哪一层。

---

## 5. 数据流（变更后完整版）

```
Agent 上报 → Master DataHub (MemoryStore)
  ├── ClusterSnapshot
  │   ├── Nodes[].Metrics       ← K8s Metrics Server（Basic 层）
  │   ├── Pods/Events/...       ← K8s API（Basic 层）
  │   └── OTel *OTelSnapshot    ← Agent ClickHouse 查询（Enhanced 层，可选）
  ├── OTelTimeline (Ring Buffer) ← 轻量副本（Observe Dashboard 用）
  │     注意: Ring Buffer 会剥离 APMTopology/LogsSummary/RecentLogs/SLOWindows
  │
  └── AIOps 读取路径: GetSnapshot() → snap.OTel（完整数据，不经过 Ring Buffer）
           ↓
     OnSnapshot() 触发
           ↓
  ┌────────────────────────────────────────────────────────────┐
  │ 1. BuildFromSnapshot(snap, otel)                           │
  │    [Basic]    Pod → Node (runs_on)                         │
  │    [Basic]    Service → Pod (selects)                      │
  │    [Basic]    Ingress → Service (routes_to)                │
  │    [Enhanced] Service → Service (calls)  [SLO Edge]        │
  │    [Enhanced] Service → Service (calls)  [APM Topology] *  │
  │                                                            │
  │ 2. ExtractMetrics(snap, otel)                              │
  │    [Basic]    Node: cpu_usage/memory_usage × 2 [MetricsSrv]│
  │    [Basic]    Pod: restart/running/ready/max_restarts × 4  │
  │    [Enhanced] Node: disk_usage/psi × 4 [OTel]              │
  │    [Enhanced] Service: error_rate/latency/rps × 3 [SLO]    │
  │    [Enhanced] Ingress: error_rate/latency × 2 [SLO]        │
  │    [Enhanced] Service: apm_error/p99/rps × 3 [APM] *       │
  │    [Enhanced] Service: log_error_rate × 1 [Logs] *         │
  │    [Enhanced] Global: log_error/warn_count × 2 [Logs] *    │
  │                                                            │
  │ 3. Deterministic Anomalies                                 │
  │    [Basic]    ContainerAnomalies (CrashLoop/OOM/ImagePull) │
  │    [Basic]    EventAnomalies (Critical K8s Events)          │
  │    [Basic]    DeploymentImpact (>=50% unavailable)          │
  │    [Basic]    NodePressure (Memory/Disk/PID Pressure) *    │
  │    [Enhanced] APM error_rate > 15% → score 0.90 *          │
  │    [Enhanced] APM P99 > 5s → score 0.75 *                  │
  │    [Enhanced] Log error > 500/5min → score 0.80 *          │
  │                                                            │
  │ 4-6. [不变] StateManager → Scorer → StateMachine           │
  └────────────────────────────────────────────────────────────┘

* = 本次新增
```

---

## 6. 指标总览

### 6.1 实体类型 → 指标映射

| 实体类型 | 指标名 | 来源 | 层级 | 权重 | 通道 |
|----------|--------|------|------|------|------|
| **node** | `cpu_usage` | K8s Metrics Server | Basic | 0.25 | Statistical |
| | `memory_usage` | K8s Metrics Server | Basic | 0.25 | Statistical |
| | `memory_pressure` * | K8s Node Conditions | Basic | 0.15 | Deterministic |
| | `disk_pressure` * | K8s Node Conditions | Basic | 0.10 | Deterministic |
| | `pid_pressure` * | K8s Node Conditions | Basic | 0.05 | Deterministic |
| | `disk_usage` | OTel NodeMetrics | Enhanced | 0.10 | Statistical |
| | `psi_cpu` | OTel NodeMetrics | Enhanced | 0.05 | Statistical |
| | `psi_memory` | OTel NodeMetrics | Enhanced | 0.03 | Statistical |
| | `psi_io` | OTel NodeMetrics | Enhanced | 0.02 | Statistical |
| **pod** | `restart_count` | K8s Pod | Basic | 0.20 | Both |
| | `is_running` | K8s Pod | Basic | 0.10 | Both |
| | `not_ready_containers` | K8s Pod | Basic | 0.20 | Both |
| | `max_container_restarts` | K8s Pod | Basic | 0.10 | Both |
| | `container_anomaly` | K8s Container | Basic | 0.25 | Deterministic |
| | `critical_event` | K8s Event | Basic | 0.15 | Deterministic |
| | `deployment_impact` | K8s Deployment | Basic | 0.25 | Deterministic |
| **service** | `error_rate` | SLO (Linkerd) | Enhanced | 0.20 | Statistical |
| | `avg_latency` | SLO (Linkerd) | Enhanced | 0.15 | Statistical |
| | `request_rate` | SLO (Linkerd) | Enhanced | 0.10 | Statistical |
| | `apm_error_rate` * | APM (OTel Trace) | Enhanced | 0.20 | Both |
| | `apm_p99_latency` * | APM (OTel Trace) | Enhanced | 0.15 | Statistical |
| | `apm_rps` * | APM (OTel Trace) | Enhanced | 0.05 | Statistical |
| | `log_error_rate` * | Logs (OTel Logs) | Enhanced | 0.15 | Both |
| **ingress** | `error_rate` | SLO (Traefik) | Enhanced | 0.50 | Statistical |
| | `avg_latency` | SLO (Traefik) | Enhanced | 0.50 | Statistical |
| **logs** * | `log_error_count` | Logs (OTel Logs) | Enhanced | 0.60 | Both |
| | `log_warn_count` | Logs (OTel Logs) | Enhanced | 0.40 | Statistical |

\* = 本次新增

### 6.2 确定性异常触发条件

| 来源 | 条件 | 指标名 | Score | 层级 |
|------|------|--------|-------|------|
| K8s Container | CrashLoopBackOff | `container_anomaly` | 0.90 | Basic |
| K8s Container | OOMKilled | `container_anomaly` | 0.95 | Basic |
| K8s Container | ImagePullBackOff | `container_anomaly` | 0.70 | Basic |
| K8s Container | NotReady | `container_anomaly` | 0.60 | Basic |
| K8s Event | Critical Event 5min 内 | `critical_event` | 0.85 | Basic |
| K8s Deployment | 不可用比例 >= 75% | `deployment_impact` | 0.95 | Basic |
| K8s Deployment | 不可用比例 >= 50% | `deployment_impact` | 0.80 | Basic |
| K8s Node * | MemoryPressure=true | `memory_pressure` | 0.85 | Basic |
| K8s Node * | DiskPressure=true | `disk_pressure` | 0.80 | Basic |
| K8s Node * | PIDPressure=true | `pid_pressure` | 0.75 | Basic |
| APM * | error_rate > 15% | `apm_error_rate` | 0.90 | Enhanced |
| APM * | P99 > 5000ms | `apm_p99_latency` | 0.75 | Enhanced |
| Logs * | error > 500 条/5min | `log_error_count` | 0.80 | Enhanced |

### 6.3 依赖图边类型

| 边类型 | 方向 | 来源 | 层级 |
|--------|------|------|------|
| `runs_on` | Pod → Node | K8s Pod.NodeName | Basic |
| `selects` | Service → Pod | K8s Service.Selector | Basic |
| `routes_to` | Ingress → Service | K8s Ingress.Rules | Basic |
| `calls` | Service → Service | SLO Edge (Linkerd) | Enhanced |
| `calls` * | Service → Service | APM Topology (OTel Trace) | Enhanced |

---

## 7. AI 角色的信息源

分层 AIOps 完成后，AI 角色可以根据层级获取不同深度的信息：

### 7.1 background 角色（自动摘要）

| 层级 | 可用信息 | 摘要质量 |
|------|---------|---------|
| Basic | K8s 资源状态、容器异常、Deployment 影响、Node CPU/Memory/Pressure | 基础：「Node X MemoryPressure，Pod Y OOMKilled，Deployment Z 75% 不可用」 |
| Enhanced | + SLO 违规、APM 延迟飙升、日志错误暴增、服务调用链 | 完整：「Service A P99 延迟 8s，上游 Service B error rate 25%，日志显示 DB 连接超时 500+/5min」 |

### 7.2 analysis 角色（深度分析）

| 层级 | 可用信息 | 分析深度 |
|------|---------|---------|
| Basic | 因果树（K8s 拓扑）、事件时间线、Node 资源趋势 | 基础设施层分析 |
| Enhanced | + 完整因果树（含服务调用链）、APM 指标、日志上下文 | 全链路根因分析 |

---

## 8. 文件变更清单

| # | 文件 | 操作 | 说明 |
|---|------|------|------|
| 1 | `aiops/baseline/extractor.go` | 修改 | `extractNodeMetrics` 改造（优先 Metrics Server）+ `extractNodePressure` + `extractAPMMetrics` + `extractLogMetrics` + `ExtractOTelDeterministicAnomalies` |
| 2 | `aiops/correlator/builder.go` | 修改 | 签名增加 `otel` 参数 + APM 拓扑边 + SLO Edge 改用 otel 参数 + 边去重 |
| 3 | `aiops/risk/config.go` | 修改 | node 权重再分配（+Pressure）+ service 权重再分配（+APM/Log）+ 新增 logs 实体类型 |
| 4 | `aiops/core/engine.go` | 修改 | 传递 otel 给 builder + 调用 OTel 确定性异常 + 扩展 activeKeys |
| **合计** | | **4 文件修改** | 无新增文件 |

---

## 9. 验证

```bash
# 编译验证
go build ./atlhyper_master_v2/...

# Basic 层测试
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractNodeMetrics_MetricsServer
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractNodeMetrics_NilMetrics
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractNodePressure
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractMetrics_NilOTel
go test ./atlhyper_master_v2/aiops/correlator/ -v -run TestBuildFromSnapshot_NilOTel

# Enhanced 层测试
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractNodeMetrics_OTelFallback
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractAPMMetrics
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractLogMetrics
go test ./atlhyper_master_v2/aiops/baseline/ -v -run TestExtractOTelDeterministicAnomalies
go test ./atlhyper_master_v2/aiops/correlator/ -v -run TestBuildFromSnapshot_APMTopology
go test ./atlhyper_master_v2/aiops/correlator/ -v -run TestBuildFromSnapshot_EdgeDedup
go test ./atlhyper_master_v2/aiops/risk/ -v -run TestServiceRisk_WithAPM
```

### 9.1 降级验证

| 测试场景 | 预期行为 |
|---------|---------|
| 无 Metrics Server + 无 OTel | 只有 Pod/Container/Event/Deployment 异常检测 |
| 有 Metrics Server + 无 OTel | + Node CPU/Memory/Pressure 异常检测 |
| 有 Metrics Server + 有 OTel（部分） | + SLO 指标 + 深度 Node 指标 |
| 全部可用 | 所有指标 + 所有边 + 所有确定性异常 |
| 有 OTel 但无 Metrics Server | OTel 数据兜底 CPU/Memory |

---

## 10. 已确认决策

| 问题 | 决策 |
|------|------|
| 两层如何切换 | **不切换**：nil 保护自动降级，数据驱动而非配置驱动 |
| Basic 层 Node 指标来源 | **K8s Metrics Server**（`snap.Nodes[].Metrics`），OTel 兜底 |
| Basic 层 Node 压力 | **确定性异常**：MemoryPressure/DiskPressure/PIDPressure |
| APM 和 SLO 指标是否互斥 | **共存**：同一 service 可同时有 SLO 和 APM 指标 |
| Enhanced 权重在 Basic 下是否影响评分 | **不影响**：无数据 = 无指标点 = Scorer 自动归一化 |
| AIOps 与 Observe 模块关系 | **读共享**：都从 DataHub Store 读取，无模块间依赖。Observe 有双路径（快照直读 + Command/MQ 实时查询） |
| AIOps OTel 数据源 | **snap.OTel 直接读取**：不使用 Ring Buffer（lightweightCopy 会剥离 APMTopology/LogsSummary/RecentLogs/SLOWindows） |
| SLO Edge 数据源 | **改用 otel 参数**：统一从 snap.OTel 获取 |
| 15 分钟聚合数据充足性 | **基本满足**：唯一缺口是日志 per-service severity 无预聚合字段，通过遍历 RecentLogs 补偿 |
| APM 和 SLO 的拓扑边重复 | **去重**：用 edgeSet 确保同一 from→to→type 只添加一次 |
| logs 虚拟实体 | **新增实体类型** `logs`：全局日志指标 |

