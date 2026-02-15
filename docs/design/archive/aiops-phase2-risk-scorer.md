# AIOps Phase 2a — 风险评分引擎

## 概要

实现三阶段风险评分流水线：**Stage 1 局部风险**（加权异常分数聚合）→ **Stage 2 时序权重**（因果排序，先出问题的实体权重更高）→ **Stage 3 图传播**（沿依赖图反向拓扑排序传播），最终输出每个实体的 `R_final` 和集群整体的 `ClusterRisk` 分数。

**前置依赖**: Phase 1（依赖图 + 基线引擎）— 需要 `DependencyGraph` 和 `AnomalyResult[]`

**中心文档**: [`aiops-engine-design.md`](../future/aiops-engine-design.md) §4.3 (M3)

**Phase 1 设计**: [`aiops-phase1-graph-baseline.md`](./aiops-phase1-graph-baseline.md)

---

## 1. 文件夹结构

```
atlhyper_master_v2/
├── master.go                                (现有)  不动: Phase 1 已完成 AIOps 初始化
│
├── aiops/
│   ├── interfaces.go                        (Phase 1) <- 修改: +GetClusterRisk / GetEntityRisks / GetEntityRisk
│   ├── engine.go                            (Phase 1) <- 修改: OnSnapshot 中调用 scorer
│   ├── types.go                             (Phase 1) <- 修改: +风险评分相关类型
│   │
│   └── risk/                                          <- NEW (整个目录)
│       ├── scorer.go                                  <- NEW  三阶段流水线主逻辑
│       ├── local.go                                   <- NEW  Stage 1: 局部风险计算
│       ├── temporal.go                                <- NEW  Stage 2: 时序权重
│       ├── propagation.go                             <- NEW  Stage 3: 图传播
│       ├── cluster_risk.go                            <- NEW  ClusterRisk 聚合
│       └── config.go                                  <- NEW  权重配置
│
├── service/
│   ├── interfaces.go                        (现有)  <- 修改: Query 接口 +3 方法
│   └── query/
│       └── aiops.go                         (Phase 1) <- 修改: +风险查询实现
│
└── gateway/
    ├── routes.go                            (现有)  <- 修改: +3 路由
    └── handler/
        └── aiops_risk.go                              <- NEW  风险评分 API Handler
```

### 变更统计

| 操作 | 文件数 | 文件 |
|------|--------|------|
| **新建** | 7 | `risk/` 下 6 个 + `handler/aiops_risk.go` |
| **修改** | 5 | `aiops/interfaces.go`, `aiops/engine.go`, `aiops/types.go`, `service/interfaces.go`, `service/query/aiops.go`, `gateway/routes.go` |

**无新增数据库表** — 风险评分是实时计算的，不持久化。

---

## 2. 调用链路

### 2.1 风险计算路径（OnSnapshot 触发）

```
aiopsEngine.OnSnapshot(ctx, clusterID)
    │
    ├── 1. correlator.Update(snapshot)        ← Phase 1
    ├── 2. baseline.Update(points)            ← Phase 1
    │       → 输出 anomalies []*AnomalyResult
    │
    └── 3. ★ scorer.Calculate(graph, anomalies)  ← Phase 2a NEW
            │
            ├── Stage 1: local.ComputeLocalRisks(anomalies)
            │   ├── 按 entityKey 分组
            │   ├── 对每个实体，按指标权重加权求和
            │   └── → map[entityKey]float64 (R_local)
            │
            ├── Stage 2: temporal.ApplyTemporalWeights(localRisks, firstAnomalyTimes)
            │   ├── 对每个实体计算 W_time = exp(-Δt/τ)
            │   └── → map[entityKey]float64 (R_weighted = R_local × W_time)
            │
            ├── Stage 3: propagation.Propagate(graph, weightedRisks)
            │   ├── 反向拓扑排序: Node → Pod → Service → Ingress
            │   ├── R_final(v) = α × R_weighted(v) + (1-α) × Σ(w_edge × R_final(u))
            │   └── → map[entityKey]float64 (R_final)
            │
            └── ClusterRisk: cluster_risk.Aggregate(finalRisks, sloData)
                ├── max(R_final) × w1
                ├── SLO burn_rate_factor × w2
                ├── error_growth_rate × w3
                └── → ClusterRisk ∈ [0, 100]
```

### 2.2 风险查询路径（API）

```
GET /api/v2/aiops/risk/cluster?cluster={id}
    → handler/aiops_risk.go → service/query/aiops.go
    → aiopsEngine.GetClusterRisk(clusterID)
    → 返回 ClusterRiskResponse

GET /api/v2/aiops/risk/entities?cluster={id}&sort=r_final&limit=20
    → handler/aiops_risk.go → service/query/aiops.go
    → aiopsEngine.GetEntityRisks(clusterID, sort, limit)
    → 返回 []EntityRiskResponse

GET /api/v2/aiops/risk/entity/{key}?cluster={id}
    → handler/aiops_risk.go → service/query/aiops.go
    → aiopsEngine.GetEntityRisk(clusterID, entityKey)
    → 返回 EntityRiskDetailResponse
```

---

## 3. 数据模型

### 3.1 风险评分类型

```go
// aiops/types.go — 新增

// EntityRisk 实体风险评分
type EntityRisk struct {
    EntityKey    string  `json:"entityKey"`
    EntityType   string  `json:"entityType"`   // "service" | "pod" | "node" | "ingress"
    Namespace    string  `json:"namespace"`
    Name         string  `json:"name"`
    RLocal       float64 `json:"rLocal"`       // Stage 1: 局部风险 [0, 1]
    WTime        float64 `json:"wTime"`        // Stage 2: 时序权重 [0, 1]
    RWeighted    float64 `json:"rWeighted"`    // R_local × W_time
    RFinal       float64 `json:"rFinal"`       // Stage 3: 传播后最终风险 [0, 1]
    RiskLevel    string  `json:"riskLevel"`    // "healthy" | "low" | "medium" | "high" | "critical"
    FirstAnomaly int64   `json:"firstAnomaly"` // 首次异常时间 (Unix, 0 = 无异常)
}

// ClusterRisk 集群整体风险
type ClusterRisk struct {
    ClusterID    string         `json:"clusterId"`
    Risk         float64        `json:"risk"`         // [0, 100]
    Level        string         `json:"level"`        // "healthy" | "low" | "warning" | "critical"
    TopEntities  []*EntityRisk  `json:"topEntities"`  // 风险最高的 Top 5 实体
    TotalEntities int           `json:"totalEntities"` // 图中总实体数
    AnomalyCount  int           `json:"anomalyCount"`  // 当前异常实体数
    UpdatedAt    int64          `json:"updatedAt"`
}

// EntityRiskDetail 实体风险详情（单个实体的完整信息）
type EntityRiskDetail struct {
    EntityRisk                              // 嵌入基础风险
    Metrics      []*AnomalyResult  `json:"metrics"`      // 各指标异常详情
    Propagation  []*PropagationPath `json:"propagation"`  // 传播路径
    CausalChain  []*CausalEntry    `json:"causalChain"`  // 因果链（按时间排序）
}

// PropagationPath 风险传播路径
type PropagationPath struct {
    From       string  `json:"from"`       // 传播源实体
    To         string  `json:"to"`         // 传播目标
    EdgeType   string  `json:"edgeType"`   // 边类型
    Contribution float64 `json:"contribution"` // 该路径对 R_final 的贡献值
}

// CausalEntry 因果链条目
type CausalEntry struct {
    EntityKey  string  `json:"entityKey"`
    MetricName string  `json:"metricName"`
    Deviation  float64 `json:"deviation"`  // σ 倍数
    DetectedAt int64   `json:"detectedAt"` // 首次检测时间
}

// RiskLevel 从 R_final 映射到风险等级
func RiskLevel(rFinal float64) string {
    switch {
    case rFinal >= 0.8:
        return "critical"
    case rFinal >= 0.6:
        return "high"
    case rFinal >= 0.4:
        return "medium"
    case rFinal >= 0.2:
        return "low"
    default:
        return "healthy"
    }
}

// ClusterRiskLevel 从 ClusterRisk 映射到等级
func ClusterRiskLevel(risk float64) string {
    switch {
    case risk >= 80:
        return "critical"
    case risk >= 50:
        return "warning"
    case risk >= 20:
        return "low"
    default:
        return "healthy"
    }
}
```

---

## 4. 详细设计

### 4.1 三阶段流水线主逻辑 (risk/scorer.go)

```go
// Scorer 风险评分引擎
type Scorer struct {
    config     *RiskConfig
    mu         sync.RWMutex
    results    map[string]*ClusterRisk      // clusterID -> ClusterRisk
    entityMap  map[string]map[string]*EntityRisk // clusterID -> entityKey -> EntityRisk
    firstAnomaly map[string]int64            // entityKey -> 首次异常时间
}

// Calculate 执行三阶段风险评分
func (s *Scorer) Calculate(
    clusterID string,
    graph *DependencyGraph,
    anomalies []*AnomalyResult,
    sloData *SLOContext, // 从 SLO 仓库获取的 burn rate 等
) *ClusterRisk {
    s.mu.Lock()
    defer s.mu.Unlock()

    now := time.Now().Unix()

    // 更新首次异常时间记录
    s.updateFirstAnomalyTimes(anomalies, now)

    // Stage 1: 局部风险
    localRisks := ComputeLocalRisks(anomalies, s.config)

    // Stage 2: 时序权重
    weightedRisks := ApplyTemporalWeights(localRisks, s.firstAnomaly, now, s.config.TemporalHalfLife)

    // Stage 3: 图传播
    finalRisks, propagationPaths := Propagate(graph, weightedRisks, s.config.SelfWeight)

    // 构建 EntityRisk 列表
    entityRisks := s.buildEntityRisks(graph, localRisks, weightedRisks, finalRisks)
    s.entityMap[clusterID] = entityRisks

    // 聚合 ClusterRisk
    clusterRisk := Aggregate(clusterID, entityRisks, finalRisks, sloData, s.config, now)
    s.results[clusterID] = clusterRisk

    return clusterRisk
}

// updateFirstAnomalyTimes 记录每个实体首次出现异常的时间
func (s *Scorer) updateFirstAnomalyTimes(anomalies []*AnomalyResult, now int64) {
    // 当前异常实体集合
    currentAnomaly := map[string]bool{}
    for _, a := range anomalies {
        if a.IsAnomaly {
            currentAnomaly[a.EntityKey] = true
            if _, exists := s.firstAnomaly[a.EntityKey]; !exists {
                s.firstAnomaly[a.EntityKey] = now
            }
        }
    }
    // 清除已恢复的实体
    for key := range s.firstAnomaly {
        if !currentAnomaly[key] {
            delete(s.firstAnomaly, key)
        }
    }
}
```

### 4.2 Stage 1: 局部风险 (risk/local.go)

```go
// ComputeLocalRisks 计算每个实体的局部风险分数
// R_local(entity) = Σ(w_i × score_i)
func ComputeLocalRisks(anomalies []*AnomalyResult, config *RiskConfig) map[string]float64 {
    // 按 entityKey 分组
    byEntity := map[string][]*AnomalyResult{}
    for _, a := range anomalies {
        byEntity[a.EntityKey] = append(byEntity[a.EntityKey], a)
    }

    localRisks := map[string]float64{}
    for entityKey, results := range byEntity {
        entityType := extractEntityType(entityKey) // 从 key 提取类型
        weights := config.GetWeights(entityType)

        var rLocal float64
        for _, r := range results {
            w, ok := weights[r.MetricName]
            if !ok {
                w = 0.1 // 未配置的指标默认权重
            }
            rLocal += w * r.Score
        }

        // 截断到 [0, 1]
        if rLocal > 1.0 {
            rLocal = 1.0
        }
        localRisks[entityKey] = rLocal
    }

    return localRisks
}
```

### 4.3 权重配置 (risk/config.go)

```go
// RiskConfig 风险评分配置
type RiskConfig struct {
    // Stage 1: 局部风险权重 (按实体类型分组)
    Weights map[string]map[string]float64

    // Stage 2: 时序参数
    TemporalHalfLife float64 // τ (秒)，默认 300 (5 分钟)

    // Stage 3: 传播参数
    SelfWeight float64 // α，默认 0.6 (自身 60%，传播 40%)

    // ClusterRisk 聚合权重
    ClusterWeightMax    float64 // w1，默认 0.5
    ClusterWeightSLO    float64 // w2，默认 0.3
    ClusterWeightGrowth float64 // w3，默认 0.2
}

// DefaultRiskConfig 返回默认配置
func DefaultRiskConfig() *RiskConfig {
    return &RiskConfig{
        Weights: map[string]map[string]float64{
            "service": {
                "error_rate":    0.40,
                "avg_latency":   0.30,
                "request_rate":  0.20,
                "pod_health":    0.10,
            },
            "pod": {
                "restart_count": 0.35,
                "is_running":    0.35,
                "cpu_memory":    0.20,
                "ready":         0.10,
            },
            "node": {
                "memory_usage":  0.30,
                "cpu_usage":     0.25,
                "disk_usage":    0.25,
                "network":       0.10,
                "psi":           0.10,
            },
            "ingress": {
                "error_rate":    0.45,
                "avg_latency":   0.35,
                "request_rate":  0.20,
            },
        },
        TemporalHalfLife:    300,  // 5 分钟
        SelfWeight:          0.6,
        ClusterWeightMax:    0.5,
        ClusterWeightSLO:    0.3,
        ClusterWeightGrowth: 0.2,
    }
}

// GetWeights 获取指定实体类型的指标权重
func (c *RiskConfig) GetWeights(entityType string) map[string]float64 {
    if w, ok := c.Weights[entityType]; ok {
        return w
    }
    return map[string]float64{} // 未知类型返回空
}
```

### 4.4 Stage 2: 时序权重 (risk/temporal.go)

```go
// ApplyTemporalWeights 应用时序权重
// W_time(entity) = exp(-Δt / τ)
// Δt = now - firstAnomalyTime
//
// 效果: 先出问题的实体权重更高（更可能是根因）
func ApplyTemporalWeights(
    localRisks map[string]float64,
    firstAnomalyTimes map[string]int64,
    now int64,
    halfLife float64, // τ 秒
) map[string]float64 {
    weighted := make(map[string]float64, len(localRisks))

    for entityKey, rLocal := range localRisks {
        wTime := 1.0 // 默认无衰减

        if firstTime, ok := firstAnomalyTimes[entityKey]; ok && firstTime > 0 {
            deltaT := float64(now - firstTime)
            if deltaT > 0 {
                wTime = math.Exp(-deltaT / halfLife)
            }
        }

        weighted[entityKey] = rLocal * wTime
    }

    return weighted
}
```

### 4.5 Stage 3: 图传播 (risk/propagation.go)

```go
// Propagate 沿依赖图传播风险
// 按反向拓扑排序: Node(先算) → Pod → Service → Ingress(后算)
//
// R_final(v) = α × R_weighted(v) + (1-α) × Σ(w_edge × R_final(u))
// α = selfWeight
// u ∈ dependencies(v) = v 所依赖的实体（v 的边的目标）
func Propagate(
    graph *DependencyGraph,
    weightedRisks map[string]float64,
    selfWeight float64, // α
) (map[string]float64, []*PropagationPath) {
    finalRisks := make(map[string]float64, len(graph.Nodes))
    var paths []*PropagationPath

    // 1. 拓扑排序（按层级: node=0, pod=1, service=2, ingress=3）
    sorted := topologicalSort(graph)

    // 2. 从底层（Node）到顶层（Ingress）依次计算
    for _, entityKey := range sorted {
        rWeighted := weightedRisks[entityKey] // 可能为 0（无异常的实体）

        // 获取该实体依赖的下游实体（边的目标）
        deps := graph.GetDependencies(entityKey)
        var propagatedRisk float64
        if len(deps) > 0 {
            totalWeight := 0.0
            for _, dep := range deps {
                edgeWeight := dep.Weight
                if edgeWeight == 0 {
                    edgeWeight = 1.0 / float64(len(deps)) // 均匀分配
                }
                propagatedRisk += edgeWeight * finalRisks[dep.Key]
                totalWeight += edgeWeight

                if finalRisks[dep.Key] > 0 {
                    paths = append(paths, &PropagationPath{
                        From:         dep.Key,
                        To:           entityKey,
                        EdgeType:     dep.EdgeType,
                        Contribution: edgeWeight * finalRisks[dep.Key],
                    })
                }
            }
            // 归一化传播风险
            if totalWeight > 0 {
                propagatedRisk /= totalWeight
            }
        }

        // 最终风险 = 自身 × α + 传播 × (1-α)
        finalRisks[entityKey] = selfWeight*rWeighted + (1-selfWeight)*propagatedRisk

        // 截断到 [0, 1]
        if finalRisks[entityKey] > 1.0 {
            finalRisks[entityKey] = 1.0
        }
    }

    return finalRisks, paths
}

// topologicalSort 按层级排序
// 层级定义: node=0, pod=1, service=2, ingress=3
// 先计算底层，再计算顶层
func topologicalSort(graph *DependencyGraph) []string {
    layerOrder := map[string]int{
        "node":    0,
        "pod":     1,
        "service": 2,
        "ingress": 3,
    }

    type entry struct {
        key   string
        layer int
    }
    var entries []entry
    for key, node := range graph.Nodes {
        layer := layerOrder[node.Type]
        entries = append(entries, entry{key, layer})
    }

    sort.Slice(entries, func(i, j int) bool {
        return entries[i].layer < entries[j].layer
    })

    result := make([]string, len(entries))
    for i, e := range entries {
        result[i] = e.key
    }
    return result
}

// Dependency 传播依赖
type Dependency struct {
    Key      string
    EdgeType string
    Weight   float64
}
```

### 4.6 ClusterRisk 聚合 (risk/cluster_risk.go)

```go
// SLOContext SLO 上下文（从 SLO 仓库获取）
type SLOContext struct {
    MaxBurnRate     float64 // 所有 SLO 中最大的 burn rate
    ErrorGrowthRate float64 // 错误率增长速率
}

// Aggregate 聚合 ClusterRisk
// ClusterRisk = w1 × max(R_final) × 100
//             + w2 × SLO_burn_rate_factor
//             + w3 × error_growth_rate_factor
func Aggregate(
    clusterID string,
    entityRisks map[string]*EntityRisk,
    finalRisks map[string]float64,
    sloCtx *SLOContext,
    config *RiskConfig,
    now int64,
) *ClusterRisk {
    // 1. 找最大 R_final
    var maxRFinal float64
    for _, r := range finalRisks {
        if r > maxRFinal {
            maxRFinal = r
        }
    }

    // 2. SLO burn rate factor
    sloBurnFactor := 0.0
    if sloCtx != nil {
        switch {
        case sloCtx.MaxBurnRate >= 2.0:
            sloBurnFactor = 1.0
        case sloCtx.MaxBurnRate >= 1.0:
            sloBurnFactor = 0.5
        }
    }

    // 3. 错误增长率 factor
    errorGrowthFactor := 0.0
    if sloCtx != nil && sloCtx.ErrorGrowthRate > 0 {
        errorGrowthFactor = sigmoid(sloCtx.ErrorGrowthRate, 0.5, 2.0)
    }

    // 4. 聚合
    risk := config.ClusterWeightMax*maxRFinal*100 +
        config.ClusterWeightSLO*sloBurnFactor*100 +
        config.ClusterWeightGrowth*errorGrowthFactor*100

    if risk > 100 {
        risk = 100
    }

    // 5. Top 5 实体
    topEntities := topN(entityRisks, 5)

    // 6. 异常计数
    anomalyCount := 0
    for _, r := range finalRisks {
        if r > 0.2 { // R_final > 0.2 视为异常
            anomalyCount++
        }
    }

    return &ClusterRisk{
        ClusterID:     clusterID,
        Risk:          math.Round(risk*10) / 10, // 保留一位小数
        Level:         ClusterRiskLevel(risk),
        TopEntities:   topEntities,
        TotalEntities: len(entityRisks),
        AnomalyCount:  anomalyCount,
        UpdatedAt:     now,
    }
}

// topN 返回 R_final 最高的 N 个实体
func topN(risks map[string]*EntityRisk, n int) []*EntityRisk {
    sorted := make([]*EntityRisk, 0, len(risks))
    for _, r := range risks {
        sorted = append(sorted, r)
    }
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].RFinal > sorted[j].RFinal
    })
    if len(sorted) > n {
        sorted = sorted[:n]
    }
    return sorted
}
```

---

## 5. 引擎集成

### 5.1 AIOps 接口变更

```go
// aiops/interfaces.go — 新增方法

type AIOpsEngine interface {
    // Phase 1 (已有)
    OnSnapshot(ctx context.Context, clusterID string)
    GetGraph(clusterID string) *DependencyGraph
    Trace(clusterID, fromKey, direction string, maxDepth int) *TraceResult
    GetBaseline(entityKey string) *EntityBaseline
    Start()
    Stop()

    // Phase 2a (新增)
    GetClusterRisk(clusterID string) *ClusterRisk
    GetEntityRisks(clusterID string, sortBy string, limit int) []*EntityRisk
    GetEntityRisk(clusterID, entityKey string) *EntityRiskDetail
}
```

### 5.2 Engine.OnSnapshot 变更

```go
// aiops/engine.go — OnSnapshot 扩展

func (e *Engine) OnSnapshot(ctx context.Context, clusterID string) {
    snap, err := e.store.GetSnapshot(clusterID)
    if err != nil || snap == nil {
        return
    }

    // Phase 1: 图 + 基线
    newGraph := correlator.BuildFromSnapshot(clusterID, snap)
    e.correlator.Update(clusterID, newGraph)

    points := baseline.ExtractMetrics(clusterID, snap, e.sloServiceRepo, e.sloRepo, e.nodeMetricsRepo)
    anomalies := e.baseline.Update(points)

    // ★ Phase 2a: 风险评分
    graph := e.correlator.GetGraph(clusterID)
    if graph != nil {
        sloCtx := e.buildSLOContext(ctx, clusterID) // 从 SLO 仓库获取 burn rate
        e.scorer.Calculate(clusterID, graph, anomalies, sloCtx)
    }

    // 持久化（Phase 1 逻辑不变）
    if time.Since(e.lastFlush) > e.config.FlushInterval {
        e.baseline.FlushToDB(ctx)
        e.persistGraph(ctx, clusterID)
        e.lastFlush = time.Now()
    }
}

// buildSLOContext 从 SLO 仓库获取上下文
func (e *Engine) buildSLOContext(ctx context.Context, clusterID string) *risk.SLOContext {
    // 查询 SLO 目标和当前状态，计算 burn rate
    // 简化版本：暂时返回 nil，Phase 2b 实现完整版
    return nil
}
```

### 5.3 查询方法实现

```go
// aiops/engine.go — 查询方法

func (e *Engine) GetClusterRisk(clusterID string) *ClusterRisk {
    return e.scorer.GetClusterRisk(clusterID)
}

func (e *Engine) GetEntityRisks(clusterID, sortBy string, limit int) []*EntityRisk {
    return e.scorer.GetEntityRisks(clusterID, sortBy, limit)
}

func (e *Engine) GetEntityRisk(clusterID, entityKey string) *EntityRiskDetail {
    entityRisk := e.scorer.GetEntityRisk(clusterID, entityKey)
    if entityRisk == nil {
        return nil
    }

    // 补充指标详情和因果链
    detail := &EntityRiskDetail{
        EntityRisk:  *entityRisk,
        Metrics:     e.baseline.GetAnomalies(entityKey),
        Propagation: e.scorer.GetPropagationPaths(clusterID, entityKey),
        CausalChain: e.buildCausalChain(clusterID, entityKey),
    }
    return detail
}

// buildCausalChain 构建因果链（按首次异常时间排序的相关实体）
func (e *Engine) buildCausalChain(clusterID, entityKey string) []*CausalEntry {
    // 从依赖图获取上游链路
    trace := e.correlator.Trace(clusterID, entityKey, "upstream", 5)
    if trace == nil {
        return nil
    }

    // 收集所有上游实体的异常信息
    var entries []*CausalEntry
    for _, node := range trace.Nodes {
        anomalies := e.baseline.GetAnomalies(node.Key)
        for _, a := range anomalies {
            if a.IsAnomaly {
                entries = append(entries, &CausalEntry{
                    EntityKey:  a.EntityKey,
                    MetricName: a.MetricName,
                    Deviation:  a.Deviation,
                    DetectedAt: a.DetectedAt,
                })
            }
        }
    }

    // 按时间排序（先出问题的排前面）
    sort.Slice(entries, func(i, j int) bool {
        return entries[i].DetectedAt < entries[j].DetectedAt
    })

    return entries
}
```

---

## 6. API 端点

### 6.1 集群风险概览

```
GET /api/v2/aiops/risk/cluster?cluster={id}

权限: Public (只读)

响应:
{
    "message": "获取成功",
    "data": {
        "clusterId": "cluster-1",
        "risk": 72.0,
        "level": "warning",
        "topEntities": [
            {
                "entityKey": "default/service/api-server",
                "entityType": "service",
                "namespace": "default",
                "name": "api-server",
                "rLocal": 0.70,
                "wTime": 1.0,
                "rWeighted": 0.70,
                "rFinal": 0.85,
                "riskLevel": "critical",
                "firstAnomaly": 1705312800
            },
            ...
        ],
        "totalEntities": 45,
        "anomalyCount": 3,
        "updatedAt": 1705312830
    }
}
```

### 6.2 实体风险列表

```
GET /api/v2/aiops/risk/entities?cluster={id}&sort=r_final&limit=20

权限: Public (只读)

参数:
  - cluster: 集群 ID (必需)
  - sort: 排序字段 (可选, 默认 r_final, 可选 r_local)
  - limit: 返回数量 (可选, 默认 20)

响应:
{
    "message": "获取成功",
    "data": [
        {
            "entityKey": "default/service/api-server",
            "entityType": "service",
            ...
            "rFinal": 0.85,
            "riskLevel": "critical"
        },
        ...
    ],
    "total": 45
}
```

### 6.3 单实体风险详情

```
GET /api/v2/aiops/risk/entity/{key}?cluster={id}

权限: Public (只读)

参数:
  - key: 实体 key (路径参数, URL 编码)
  - cluster: 集群 ID (必需)

响应:
{
    "message": "获取成功",
    "data": {
        "entityKey": "default/service/api-server",
        "entityType": "service",
        "rLocal": 0.70,
        "wTime": 1.0,
        "rWeighted": 0.70,
        "rFinal": 0.85,
        "riskLevel": "critical",
        "firstAnomaly": 1705312800,
        "metrics": [
            {
                "entityKey": "default/service/api-server",
                "metricName": "error_rate",
                "currentValue": 3.2,
                "baseline": 0.5,
                "deviation": 3.5,
                "score": 0.73,
                "isAnomaly": true,
                "detectedAt": 1705312800
            }
        ],
        "propagation": [
            {
                "from": "_cluster/node/worker-3",
                "to": "default/pod/api-server-abc",
                "edgeType": "runs_on",
                "contribution": 0.15
            }
        ],
        "causalChain": [
            {
                "entityKey": "_cluster/node/worker-3",
                "metricName": "memory_usage",
                "deviation": 4.2,
                "detectedAt": 1705312700
            },
            {
                "entityKey": "default/pod/api-server-abc",
                "metricName": "restart_count",
                "deviation": 3.8,
                "detectedAt": 1705312750
            },
            {
                "entityKey": "default/service/api-server",
                "metricName": "error_rate",
                "deviation": 3.5,
                "detectedAt": 1705312800
            }
        ]
    }
}
```

---

## 7. Service 层接口变更

```go
// service/interfaces.go — Query 接口新增

type Query interface {
    // ... 现有方法 + Phase 1 方法 ...

    // ==================== AIOps 风险查询 ====================

    GetAIOpsClusterRisk(ctx context.Context, clusterID string) (*aiops.ClusterRisk, error)
    GetAIOpsEntityRisks(ctx context.Context, clusterID, sortBy string, limit int) ([]*aiops.EntityRisk, int, error)
    GetAIOpsEntityRisk(ctx context.Context, clusterID, entityKey string) (*aiops.EntityRiskDetail, error)
}
```

```go
// service/query/aiops.go — 新增实现

func (q *QueryService) GetAIOpsClusterRisk(ctx context.Context, clusterID string) (*aiops.ClusterRisk, error) {
    if q.aiopsEngine == nil {
        return nil, fmt.Errorf("aiops engine not initialized")
    }
    return q.aiopsEngine.GetClusterRisk(clusterID), nil
}

func (q *QueryService) GetAIOpsEntityRisks(ctx context.Context, clusterID, sortBy string, limit int) ([]*aiops.EntityRisk, int, error) {
    if q.aiopsEngine == nil {
        return nil, 0, fmt.Errorf("aiops engine not initialized")
    }
    risks := q.aiopsEngine.GetEntityRisks(clusterID, sortBy, limit)
    total := q.aiopsEngine.GetEntityCount(clusterID)
    return risks, total, nil
}

func (q *QueryService) GetAIOpsEntityRisk(ctx context.Context, clusterID, entityKey string) (*aiops.EntityRiskDetail, error) {
    if q.aiopsEngine == nil {
        return nil, fmt.Errorf("aiops engine not initialized")
    }
    return q.aiopsEngine.GetEntityRisk(clusterID, entityKey), nil
}
```

---

## 8. Gateway Handler + 路由注册

### 8.1 Handler

```go
// handler/aiops_risk.go
type AIOpsRiskHandler struct {
    query service.Query
}

func NewAIOpsRiskHandler(query service.Query) *AIOpsRiskHandler {
    return &AIOpsRiskHandler{query: query}
}

func (h *AIOpsRiskHandler) ClusterRisk(w http.ResponseWriter, r *http.Request) {
    clusterID := r.URL.Query().Get("cluster")
    if clusterID == "" {
        writeError(w, http.StatusBadRequest, "missing cluster parameter")
        return
    }
    result, err := h.query.GetAIOpsClusterRisk(r.Context(), clusterID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, result)
}

func (h *AIOpsRiskHandler) EntityRisks(w http.ResponseWriter, r *http.Request) {
    clusterID := r.URL.Query().Get("cluster")
    sortBy := r.URL.Query().Get("sort")
    if sortBy == "" {
        sortBy = "r_final"
    }
    limit := 20
    if l := r.URL.Query().Get("limit"); l != "" {
        limit, _ = strconv.Atoi(l)
    }
    risks, total, err := h.query.GetAIOpsEntityRisks(r.Context(), clusterID, sortBy, limit)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSONWithTotal(w, http.StatusOK, risks, total)
}

func (h *AIOpsRiskHandler) EntityRisk(w http.ResponseWriter, r *http.Request) {
    clusterID := r.URL.Query().Get("cluster")
    entityKey := extractPathParam(r, "/api/v2/aiops/risk/entity/")
    if clusterID == "" || entityKey == "" {
        writeError(w, http.StatusBadRequest, "missing parameters")
        return
    }
    result, err := h.query.GetAIOpsEntityRisk(r.Context(), clusterID, entityKey)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, result)
}
```

### 8.2 路由注册

```go
// gateway/routes.go — registerRoutes() 中新增

aiopsRiskHandler := handler.NewAIOpsRiskHandler(r.service)

register("/api/v2/aiops/risk/cluster", aiopsRiskHandler.ClusterRisk)
register("/api/v2/aiops/risk/entities", aiopsRiskHandler.EntityRisks)
register("/api/v2/aiops/risk/entity/", aiopsRiskHandler.EntityRisk)
```

---

## 9. 实现阶段（TDD）

```
P1: 数据模型 + 配置
  ├── aiops/types.go 新增风险评分类型
  ├── risk/config.go 权重配置 + 默认值
  └── 单元测试: 配置加载、RiskLevel 映射

P2: Stage 1 — 局部风险
  ├── risk/local.go — ComputeLocalRisks
  └── 单元测试:
      ├── 单指标实体（权重正确应用）
      ├── 多指标实体（加权求和，截断到 [0,1]）
      ├── 不同实体类型（不同权重表）
      └── 空输入（返回空 map）

P3: Stage 2 — 时序权重
  ├── risk/temporal.go — ApplyTemporalWeights
  └── 单元测试:
      ├── 刚出现异常（Δt≈0, W_time≈1.0）
      ├── 5 分钟前异常（W_time≈0.37）
      ├── 20 分钟前异常（W_time≈0.02）
      └── 无异常记录（W_time=1.0，不衰减）

P4: Stage 3 — 图传播
  ├── risk/propagation.go — Propagate + topologicalSort
  └── 单元测试:
      ├── 线性链路: Node → Pod → Service（风险从下层传播到上层）
      ├── 扇入: 多个 Pod 汇聚到 Service
      ├── 无风险图（所有 R_final = 0）
      └── 只有自身风险（无传播源，R_final = α × R_weighted）

P5: ClusterRisk 聚合
  ├── risk/cluster_risk.go — Aggregate
  └── 单元测试:
      ├── 全健康（Risk ≈ 0, Level = "healthy"）
      ├── 单实体高风险（Risk 主要来自 max(R_final)）
      ├── SLO burn rate 影响
      └── Top 5 排序正确

P6: 引擎集成 + API
  ├── aiops/engine.go — OnSnapshot 调用 scorer
  ├── service/query/aiops.go — 查询实现
  ├── handler/aiops_risk.go — API Handler
  ├── gateway/routes.go — 路由注册
  └── 端到端测试: 快照 → 风险计算 → API 查询
```

---

## 10. 文件变更清单

### 新建

| 文件 | 说明 |
|------|------|
| `aiops/risk/scorer.go` | 三阶段流水线主逻辑 |
| `aiops/risk/local.go` | Stage 1: 局部风险计算 |
| `aiops/risk/temporal.go` | Stage 2: 时序权重 |
| `aiops/risk/propagation.go` | Stage 3: 图传播 |
| `aiops/risk/cluster_risk.go` | ClusterRisk 聚合 |
| `aiops/risk/config.go` | 权重配置 |
| `gateway/handler/aiops_risk.go` | 风险评分 API Handler |

### 修改

| 文件 | 变更 |
|------|------|
| `aiops/interfaces.go` | +GetClusterRisk / GetEntityRisks / GetEntityRisk |
| `aiops/engine.go` | OnSnapshot 中调用 scorer.Calculate |
| `aiops/types.go` | +EntityRisk, ClusterRisk, EntityRiskDetail 等类型 |
| `service/interfaces.go` | Query 接口 +3 方法 |
| `service/query/aiops.go` | +3 风险查询实现 |
| `gateway/routes.go` | +3 路由 |

---

## 11. 测试计划

### 单元测试

| 模块 | 测试文件 | 测试内容 |
|------|---------|---------|
| 局部风险 | `risk/local_test.go` | 权重应用、多指标聚合、不同实体类型 |
| 时序权重 | `risk/temporal_test.go` | 衰减曲线验证、边界值 |
| 图传播 | `risk/propagation_test.go` | 线性/扇入/扇出拓扑、拓扑排序 |
| 聚合 | `risk/cluster_risk_test.go` | ClusterRisk 范围 [0,100]、SLO 因子、Top N |
| 流水线 | `risk/scorer_test.go` | 完整三阶段流水线端到端 |

### 集成测试

| 场景 | 验证点 |
|------|--------|
| Node 内存飙升 → Pod OOM → Service 错误率上升 | R_final 传播正确，Node 排在根因首位 |
| 多个 Pod 同时异常 | Service 的 R_final 反映聚合影响 |
| 异常恢复 | firstAnomaly 清除，R_local 降低 |
| ClusterRisk 响应格式 | API 返回完整的 ClusterRisk 结构 |

---

## 12. 验证命令

```bash
# 构建验证
go build ./atlhyper_master_v2/...

# 单元测试
go test ./atlhyper_master_v2/aiops/risk/... -v

# API 测试
curl "http://localhost:8080/api/v2/aiops/risk/cluster?cluster=test-cluster"
curl "http://localhost:8080/api/v2/aiops/risk/entities?cluster=test-cluster&limit=5"
```

---

## 13. 阶段实施后评审规范

> **本阶段实施完成后，必须对后续所有阶段的设计文档进行重新评审。**

### 原因

每个阶段的实施可能导致代码结构、接口签名、数据模型与设计文档中的预期产生偏差。提前编写的设计文档基于「假设的代码状态」，而实际实施后的代码才是唯一真实状态。不经过评审就直接实施下一阶段，可能导致：

- 接口签名不匹配（设计文档引用的方法名/参数与实际实现不一致）
- 文件路径变更（实施中因重构调整了目录结构）
- 数据模型演变（字段增删或类型变更）
- 新增的约束或依赖未在后续设计中体现

### 本阶段实施后需评审的文档

| 文档 | 重点评审内容 |
|------|-------------|
| `aiops-phase2-statemachine-incident.md` | `EntityRisk` / `ClusterRisk` 实际字段、`scorer` 实际 API（`GetEntityRiskMap` / `GetClusterRisk`）、`engine.go` 中 scorer 调用方式 |
| `aiops-phase3-frontend.md` | Phase 2a API 端点实际响应格式（ClusterRisk / EntityRisk JSON 结构） |
| `aiops-phase4-ai-enhancement.md` | 风险评分数据获取方式、`AIOpsEngine` 接口中风险查询方法的实际签名 |

### 评审检查清单

- [ ] 设计文档中引用的接口签名与实际代码一致
- [ ] 设计文档中的文件路径与实际目录结构一致
- [ ] 设计文档中的数据模型与实际 struct 定义一致
- [ ] 设计文档中的初始化链路与 `master.go` 实际代码一致
- [ ] 如有偏差，更新设计文档后再开始下一阶段实施
