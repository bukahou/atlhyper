# AIOps 算法范围扩展 — 接入 APM/Logs 信号

> 状态：未来规划
> 创建：2026-02-27
> 前置：AIOps 核心引擎（已完成）

## 1. 问题陈述

AtlHyper AIOps 引擎当前只消费 K8s 资源状态 + 节点指标 + SLO 聚合数据。
OTelSnapshot 中大量可观测性数据（APM Traces、Logs、时序数据）完全未被算法利用。

这导致 AIOps 的异常检测存在**信号盲区**：

| 场景 | 当前检测能力 | 缺失原因 |
|------|-------------|----------|
| 服务延迟突增但 Pod 正常 | 能检测（通过 SLOServices.avg_latency） | ✅ 部分覆盖 |
| 某个 API 端点错误率飙升 | **无法检测** | AIOps 不消费 APM Operations 数据 |
| ERROR 日志突增 | **无法检测** | AIOps 不消费日志数据 |
| Trace 中某 Span 超时 | **无法检测** | AIOps 不消费 Trace 详情 |
| 服务间调用错误率上升 | **无法检测** | APMTopology 的边权重（ErrorRate/AvgMs）未接入 |
| 节点 CPU 趋势上升（非瞬时） | **检测延迟** | 只用单点快照，未用 NodeMetricsSeries 时序 |

## 2. 当前 AIOps 数据消费分析

### 已消费的数据

```
engine.go OnSnapshot()
├── ClusterSnapshot
│   ├── Nodes            → 图节点（node 类型）
│   ├── Pods             → 图节点 + restart_count/container_anomaly/deployment_impact
│   ├── Services         → 图节点 + selects 边
│   ├── Ingresses        → 图节点 + routes_to 边
│   ├── Events           → K8s Critical Event → 确定性异常
│   ├── Deployments      → 不可用副本比例 → 确定性异常
│   ├── ReplicaSets      → 反向关联 Deployment
│   ├── NodeMetrics      → cpu_usage/memory_usage/disk_usage/psi_*
│   └── SLOData.Edges    → calls 边（服务间调用拓扑）
│
└── OTelSnapshot（部分消费）
    ├── SLOServices      → error_rate/avg_latency/request_rate（统计通道）
    └── SLOIngress       → error_rate/avg_latency + burn_rate（用于 ClusterRisk）
```

### 未消费的数据（OTelSnapshot 盲区）

| 数据 | 内容 | 潜在算法价值 |
|------|------|-------------|
| `APMServices[]` | 服务级 APM 聚合（SuccessRate、P50/P99、RPS） | 可增强/替代 SLOServices，带 Namespace |
| `APMTopology` | OTel 实测调用图（每条边: CallCount、AvgMs、ErrorRate） | 加权 `calls` 边，检测链路级异常 |
| `APMOperations[]` | 操作/端点级统计（ServiceName × SpanName × P99/ErrorRate/RPS） | **端点级异常检测**（最细粒度） |
| `RecentTraces[]` | 最近 Trace 摘要（HasError、ErrorType、ErrorMessage） | 高错误率 Trace 作为确定性异常信号 |
| `RecentLogs[]` | 最近 500 条日志（Severity、Body、TraceId） | ERROR 日志突增检测 |
| `LogsSummary` | 5min 窗口日志统计 | 错误日志比例 → service 指标 |
| `NodeMetricsSeries[]` | 节点指标时序（1min×60点） | 趋势检测（替代单点快照） |
| `SLOTimeSeries[]` | SLO 时序（1min×60点） | 服务级时序异常检测 |
| `APMTimeSeries[]` | APM 时序（1min×60点） | 端点级时序异常检测 |

## 3. 扩展方案

### 3.1 新增指标提取（extractor.go 改动）

在现有 `extractMetrics()` 流程中新增以下提取逻辑：

#### A. APM 操作级异常（优先级：高）

```go
// 从 OTelSnapshot.APMOperations 提取端点级指标
for _, op := range otel.APMOperations {
    entityKey := fmt.Sprintf("service:%s/%s:%s", op.Namespace, op.ServiceName, op.SpanName)

    // 端点错误率
    e.submit(entityKey, "operation_error_rate", 1.0 - op.SuccessRate)
    // 端点 P99 延迟
    e.submit(entityKey, "operation_p99_latency", op.P99Ms)
    // 端点 RPS
    e.submit(entityKey, "operation_rps", op.RPS)
}
```

**挑战**：操作级实体太多（每个服务可能有几十个端点），需要：
- 只对 RPS > 阈值的端点做基线检测（过滤冷端点）
- 或只做确定性检测（ErrorRate > 50% 且 RPS > 1 → 直注异常）

#### B. 日志异常信号（优先级：高）

```go
// 从 OTelSnapshot.LogsSummary 提取错误日志比例
if otel.LogsSummary != nil {
    for _, svc := range otel.LogsSummary.Services {
        entityKey := fmt.Sprintf("service:%s/%s", svc.Namespace, svc.ServiceName)
        errorRatio := float64(svc.ErrorCount) / float64(svc.TotalCount)
        e.submit(entityKey, "log_error_ratio", errorRatio)
    }
}
```

**注意**：当前 `LogsSummary` 结构可能需要扩展以支持按服务分组统计。

#### C. 调用链路异常（优先级：中）

```go
// 从 OTelSnapshot.APMTopology 提取边权重
for _, edge := range otel.APMTopology {
    // 将 APM Topology 的 ErrorRate 和 AvgMs 注入 calls 边
    // 当某条调用链路错误率突增时触发异常
    edgeKey := fmt.Sprintf("edge:%s→%s", edge.Source, edge.Target)
    e.submit(edgeKey, "call_error_rate", edge.ErrorRate)
    e.submit(edgeKey, "call_avg_latency", edge.AvgMs)
}
```

**挑战**：当前 AIOps 的实体模型是节点（node/pod/service/ingress），
边（calls/selects/routes_to/runs_on）只用于风险传播，不独立作为实体。
需要决定是否将边升级为可独立评分的实体。

#### D. 时序趋势检测（优先级：低）

当前 `baseline/detector.go` 使用 EMA+3σ，数据来源是每次 OnSnapshot 的单点值。
OTelSnapshot 中已有 60 点时序（NodeMetricsSeries/SLOTimeSeries/APMTimeSeries），
可以：
- 一次性喂入 60 个点加速冷启动
- 或用更高级的时序算法（如 ARIMA、Prophet）替代 EMA

### 3.2 风险评分权重调整（config.go 改动）

现有权重表需要扩展：

```go
// 新增 service 级指标权重
"service": {
    "error_rate":          {Weight: 0.30, Channel: Statistical},  // 原 0.40
    "avg_latency":         {Weight: 0.20, Channel: Statistical},  // 原 0.30
    "request_rate":        {Weight: 0.10, Channel: Statistical},  // 原 0.20
    "log_error_ratio":     {Weight: 0.20, Channel: Statistical},  // 新增
    "operation_anomaly":   {Weight: 0.20, Channel: Deterministic}, // 新增
}
```

权重需要重新平衡，确保新信号不会淹没原有信号。

### 3.3 依赖图增强（builder.go 改动）

#### 加权 calls 边

当前 `calls` 边权重固定为 1.0，来源是 SLOData.Edges。
可以用 APMTopology 的边指标（ErrorRate、AvgMs、CallCount）来加权：

```go
// 风险传播时，高错误率的调用链路传播更多风险
weight := 1.0 + edge.ErrorRate * 2.0  // 错误率越高，传播权重越大
```

#### 新增 APM 实测 calls 边

APMTopology 可能包含 SLOData.Edges 中没有的调用关系
（SLO 基于网格代理，APM 基于代码级追踪，覆盖范围不同）。
应将两者合并。

## 4. 实施路线

### Phase 1：日志异常信号接入（低风险，高价值）

1. 在 `extractor.go` 新增日志错误比例提取
2. `config.go` 增加 `log_error_ratio` 权重
3. 验证：人工构造 ERROR 日志突增场景，观察 Risk Score 变化

**前置条件**：OTelSnapshot.LogsSummary 需要按服务分组统计（可能需要 Agent 端改造）

### Phase 2：APM 操作级确定性异常（中风险，高价值）

1. 从 APMOperations 提取端点级异常
2. 只做确定性检测（ErrorRate > 阈值 + RPS > 阈值）
3. 关联到对应 service 实体

### Phase 3：调用链路加权（中风险，中价值）

1. APMTopology 的 ErrorRate/AvgMs 注入 calls 边权重
2. 风险传播算法适配加权边

### Phase 4：时序算法升级（高风险，长期）

1. 利用 OTelSnapshot 中的 60 点时序数据
2. 评估是否需要替换 EMA+3σ 或作为补充

## 5. 风险与注意事项

1. **指标爆炸**：APM Operations 可能产生大量实体（服务数 × 端点数），
   需要设置过滤阈值（如只监控 RPS > 1 的端点）

2. **信号重复**：SLOServices 和 APMServices 可能是同一服务的两个数据源，
   如何避免重复计分？
   - 方案：APMServices 作为 SLOServices 的增强/替代，而非并列

3. **冷启动问题**：新增指标需要 100 个快照周期（EMA 冷启动），
   约 100 × 采集间隔 才能生效。确定性通道可即时生效。

4. **权重再平衡**：新增指标后，原有指标权重需要同比下调，
   否则总体灵敏度会上升，可能导致误报增加

## 6. 文件变更预估

| 文件 | 改动 | Phase |
|------|------|-------|
| `aiops/baseline/extractor.go` | 新增 APM/Log 指标提取函数 | 1-2 |
| `aiops/risk/config.go` | 新增指标权重 + 重平衡 | 1-2 |
| `aiops/correlator/builder.go` | APMTopology 加权 calls 边 | 3 |
| `aiops/risk/propagation.go` | 适配加权边（当前等权） | 3 |
| `aiops/core/engine.go` | OnSnapshot 消费更多 OTel 字段 | 1-3 |
| `model_v3/cluster/snapshot.go` | LogsSummary 按服务分组（如需） | 1 |
| Agent ClickHouse 查询 | 日志统计摘要查询（如需） | 1 |
| **合计** | | ~5-8 |
