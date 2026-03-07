# AIOps Enhanced 层 TDD 测试规格

> 状态: active | 创建: 2026-03-07
> 关联文档: [01-aiops-data-tiered-design.md](./01-aiops-data-tiered-design.md)

## 1. 目的

本文档定义 Enhanced 层所有测试用例的**预期输入和输出**，作为 TDD 红灯阶段的规格。
实现代码必须让所有测试从 FAIL 变为 PASS。

---

## 2. 共用 Mock 数据

所有测试共用 `makeOTelSnapshot()` 辅助函数：

```go
func makeOTelSnapshot() *cluster.OTelSnapshot {
    return &cluster.OTelSnapshot{
        // APM 服务指标
        APMServices: []apm.APMService{
            {Name: "api-gateway", Namespace: "default", SuccessRate: 0.82, P99Ms: 6000, RPS: 150},
            // 异常：error_rate = 1-0.82 = 0.18 > 15%, P99 = 6000 > 5000ms
            {Name: "user-svc", Namespace: "default", SuccessRate: 0.99, P99Ms: 200, RPS: 80},
            // 正常
        },
        // APM 拓扑
        APMTopology: &apm.Topology{
            Nodes: []apm.TopologyNode{
                {Id: "api-gateway", Name: "api-gateway", Namespace: "default"},
                {Id: "user-svc", Name: "user-svc", Namespace: "default"},
            },
            Edges: []apm.TopologyEdge{
                {Source: "api-gateway", Target: "user-svc", CallCount: 1000, AvgMs: 50, ErrorRate: 0.02},
            },
        },
        // 日志摘要
        LogsSummary: &log.Summary{
            SeverityCounts: map[string]int64{"ERROR": 600, "WARN": 200, "INFO": 5000},
        },
        // 近期日志（按服务聚合用）
        RecentLogs: []log.Entry{
            {ServiceName: "api-gateway", Severity: "ERROR"},
            {ServiceName: "api-gateway", Severity: "ERROR"},
            {ServiceName: "user-svc", Severity: "WARN"},
        },
        // OTel Node 指标（磁盘 + PSI）
        MetricsNodes: []metrics.NodeMetrics{
            {
                NodeName: "node-1",
                CPU:      metrics.NodeCPU{UsagePct: 65},
                Memory:   metrics.NodeMemory{UsagePct: 70},
                Disks:    []metrics.NodeDisk{{MountPoint: "/", UsagePct: 92}},
                PSI:      metrics.NodePSI{CPUSomePct: 30, MemSomePct: 5, IOSomePct: 45},
            },
        },
        // SLO 边（用于边去重测试）
        SLOEdges: []slo.ServiceEdge{
            {Source: "api-gateway", Target: "user-svc"},   // 与 APM 拓扑重复
            {Source: "api-gateway", Target: "order-svc"},  // 仅 SLO 有
        },
    }
}
```

---

## 3. Extractor Enhanced 测试（`extractor_enhanced_test.go`）

> Enhanced 层实现独立文件 `extractor_enhanced.go`，测试对应 `extractor_enhanced_test.go`。
> Basic 层 `extractor.go` / `extractor_test.go` 零修改。

### 3.1 APM 指标提取

```
测试名: TestExtractAPMMetrics
输入:   makeOTelSnapshot().APMServices
预期输出 MetricPoint 列表:
  - entity="service:default/api-gateway", name="apm_error_rate",  value=0.18
  - entity="service:default/api-gateway", name="apm_p99_latency", value=6000
  - entity="service:default/api-gateway", name="apm_rps",         value=150
  - entity="service:default/user-svc",    name="apm_error_rate",  value=0.01
  - entity="service:default/user-svc",    name="apm_p99_latency", value=200
  - entity="service:default/user-svc",    name="apm_rps",         value=80
断言: 共 6 个点，entity 格式为 "service:{namespace}/{name}"
```

### 3.2 日志指标提取

```
测试名: TestExtractLogMetrics
输入:   makeOTelSnapshot().LogsSummary + RecentLogs
预期输出:
  全局（来自 Summary）:
  - entity="logs:global", name="log_error_count", value=600
  - entity="logs:global", name="log_warn_count",  value=200
  服务级（来自 RecentLogs 聚合）:
  - entity="service:default/api-gateway", name="log_error_count", value=2
  - entity="service:default/user-svc",    name="log_warn_count",  value=1
断言: 全局日志计数来自 Summary，服务级来自 RecentLogs 聚合
```

### 3.3 Enhanced Node 指标

```
测试名: TestExtractEnhancedNodeMetrics
输入:   makeOTelSnapshot().MetricsNodes
预期输出（追加到已有 Node 指标之后）:
  - entity="node:node-1", name="disk_usage",  value=92
  - entity="node:node-1", name="psi_cpu",     value=30
  - entity="node:node-1", name="psi_memory",  value=5
  - entity="node:node-1", name="psi_io",      value=45
断言: 磁盘取 GetPrimaryDisk().UsagePct，PSI 三个指标全部提取
```

### 3.4 OTel 确定性异常

```
测试名: TestExtractOTelDeterministicAnomalies
输入:   makeOTelSnapshot()
预期输出 DeterministicAnomaly 列表:
  - entity="service:default/api-gateway", type="apm_high_error_rate",
    message 包含 "18.0%", severity="warning"
  - entity="service:default/api-gateway", type="apm_high_p99_latency",
    message 包含 "6000ms", severity="warning"
  - entity="logs:global", type="log_error_spike",
    message 包含 "600", severity="warning"   (600 > 500 阈值)
断言: user-svc 无异常（正常），共 3 条异常
```

### 3.5 nil 降级

```
测试名: TestExtractAPMMetrics_NilOTel
输入:   otel = nil
预期:   返回空切片，不 panic

测试名: TestExtractLogMetrics_NilSummary
输入:   otel.LogsSummary = nil, otel.RecentLogs = nil
预期:   返回空切片

测试名: TestExtractOTelDeterministicAnomalies_NilOTel
输入:   otel = nil
预期:   返回空切片，不 panic
```

---

## 4. Builder 测试（`builder_test.go`）

### 4.1 APM 拓扑边

```
测试名: TestBuildFromSnapshot_APMTopologyEdges
输入:   snap 含 OTel（只有 APMTopology，无 SLOEdges）
预期:
  - 图中存在边 "service:default/api-gateway" -> "service:default/user-svc"
  - 边类型 = "calls"
断言: APM 拓扑边成功加入因果图
```

### 4.2 SLO + APM 边去重

```
测试名: TestBuildFromSnapshot_EdgeDedup
输入:
  - SLOEdges:          [{api-gateway -> user-svc}, {api-gateway -> order-svc}]
  - APMTopology.Edges: [{api-gateway -> user-svc}]  (与 SLO 重复)
预期:
  - "service:default/api-gateway" -> "service:default/user-svc" 只出现 1 条边
  - "service:default/api-gateway" -> "service:default/order-svc" 存在 1 条边
  - 总 "calls" 类型边数 = 2（不是 3）
断言: 重复边被 edgeSet 去重
```

### 4.3 已有测试适配

```
测试名: TestBuildFromSnapshot_Basic (修改已有测试)
变更:   BuildFromSnapshot(clusterID, snap, snap.OTel)
预期:   所有已有断言不变（Pod->Node, Service->Pod 边不受影响）

测试名: TestBuildFromSnapshot_NilOTel (修改已有测试)
变更:   BuildFromSnapshot(clusterID, snap, nil)
预期:   不 panic，只有 K8s 边
```

---

## 5. Risk Config 测试（`scorer_test.go`）

### 5.1 Service 权重重分配

```
测试名: TestServiceWeights_Enhanced
验证:   所有 service 类型权重之和 = 1.0
预期权重:
  - error_rate:       0.25 (从 0.40 降低)
  - avg_latency:      0.15 (从 0.30 降低)
  - request_rate:     0.10 (从 0.20 降低)
  - apm_error_rate:   0.20 (新增)
  - apm_p99_latency:  0.15 (新增)
  - log_error_count:  0.10 (新增)
  - 其余:             0.05
```

### 5.2 Node 权重重分配

```
测试名: TestNodeWeights_Enhanced
验证:   所有 node 类型权重之和 = 1.0
预期权重:
  - cpu_usage:    0.25 (从 0.30 降低)
  - memory_usage: 0.25 (从 0.30 降低)
  - disk_usage:   0.15 (新增)
  - psi_cpu:      0.10 (新增)
  - psi_memory:   0.10 (新增)
  - psi_io:       0.10 (新增)
  - 其余:         0.05
```

### 5.3 logs 实体类型

```
测试名: TestLogsEntityWeights
验证:   entityWeights["logs"] 存在且包含 log_error_count 权重
```

---

## 6. Engine 集成测试（`engine_test.go`）

### 6.1 完整流程

```
测试名: TestEngine_EnhancedTier_FullFlow
输入:   snap.OTel = makeOTelSnapshot()
预期:
  - 风险评分中包含 "service:default/api-gateway"（APM 异常推高评分）
  - 确定性异常包含 apm_high_error_rate
  - 因果图包含 APM 拓扑边
  - 活跃实体包含 "logs:global"
```

### 6.2 降级流程

```
测试名: TestEngine_EnhancedTier_Degraded
输入:   snap.OTel = nil
预期:
  - 只有 Basic 层数据（K8s 指标、容器异常）
  - 无 APM/Log 相关风险
  - 不 panic
```

---

## 7. 关键设计决策

| 问题 | 决策 |
|------|------|
| 字段名 | 设计文档写 `entry.SeverityText`，实际模型是 `entry.Severity`，以实际为准 |
| APM error_rate 计算 | `1 - SuccessRate`（SuccessRate=0.82 -> error_rate=0.18） |
| 日志异常阈值 | ERROR > 500/5min（全局 Summary 的 ERROR count） |
| 边去重策略 | `map[string]bool`，key = `"{source}->{target}"` |
| OTel 数据来源 | `snap.OTel` 直接读取（不走 Ring Buffer） |
| entity 命名格式 | service: `"service:{ns}/{name}"`，node: `"node:{name}"`，logs: `"logs:global"` |

---

## 8. 文件结构

```
atlhyper_master_v2/aiops/
├── baseline/
│   ├── extractor.go                  # [不动] Basic 层 + ExtractMetrics 编排入口
│   ├── extractor_test.go             # [不动] Basic 层测试
│   ├── extractor_enhanced.go         # [新增] Enhanced 层函数
│   │   ├── extractAPMMetrics(otel)
│   │   ├── extractLogMetrics(otel)
│   │   ├── extractEnhancedNodeMetrics(otel)
│   │   └── ExtractOTelDeterministicAnomalies(otel)
│   └── extractor_enhanced_test.go    # [新增] Enhanced 层测试
│       ├── makeOTelSnapshot()
│       ├── TestExtractAPMMetrics
│       ├── TestExtractAPMMetrics_NilOTel
│       ├── TestExtractLogMetrics
│       ├── TestExtractLogMetrics_NilSummary
│       ├── TestExtractEnhancedNodeMetrics
│       ├── TestExtractOTelDeterministicAnomalies
│       └── TestExtractOTelDeterministicAnomalies_NilOTel
│
├── correlator/
│   ├── builder.go                    # [修改] 签名变更 + APM 边 + 边去重
│   └── builder_test.go               # [修改] 新增 2 测试 + 已有测试适配
│
├── risk/
│   ├── config.go                     # [修改] 权重重分配 + logs 实体
│   └── scorer_test.go                # [修改] 新增 3 测试
│
└── core/
    └── engine.go                     # [修改] OTel 数据源 + 调用 Enhanced 函数
```

编排入口 `ExtractMetrics()` 在 `extractor.go` 中调用 Enhanced 函数：
```go
// extractor.go（仅修改编排入口，Basic 函数体零改动）
func ExtractMetrics(..., otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
    // Basic（不动）
    points = append(points, extractNodeMetrics(snap)...)
    points = append(points, extractPodMetrics(snap)...)
    // Enhanced（新增调用，函数实现在 extractor_enhanced.go）
    if otel != nil {
        points = append(points, extractServiceMetrics(otel)...)
        points = append(points, extractIngressMetrics(otel)...)
        points = append(points, extractAPMMetrics(otel)...)
        points = append(points, extractLogMetrics(otel)...)
        points = append(points, extractEnhancedNodeMetrics(otel)...)
    }
    return points
}
```

---

## 9. 验证命令

```bash
# 红灯阶段 — Enhanced 测试全部 FAIL
go test ./atlhyper_master_v2/aiops/baseline/ -v -run "TestExtractAPMMetrics|TestExtractLogMetrics|TestExtractEnhancedNodeMetrics|TestExtractOTelDeterministicAnomalies"
go test ./atlhyper_master_v2/aiops/correlator/ -v -run "TestBuildFromSnapshot_APMTopology|TestBuildFromSnapshot_EdgeDedup"
go test ./atlhyper_master_v2/aiops/risk/ -v -run "TestServiceWeights_Enhanced|TestNodeWeights_Enhanced|TestLogsEntityWeights"

# 红灯阶段 — Basic 测试仍然 PASS（零修改保证）
go test ./atlhyper_master_v2/aiops/baseline/ -v -run "TestExtractNodeMetrics|TestExtractPodMetrics|TestExtractContainerAnomalies|TestExtractEventAnomalies"

# 绿灯阶段 — 全部 PASS
go test ./atlhyper_master_v2/aiops/... -v
```
