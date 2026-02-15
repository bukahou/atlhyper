# AIOps Phase 1 — 依赖图引擎 + 基线引擎

## 概要

构建 AIOps 引擎核心基座：**依赖图引擎 (Correlator)** 从 K8s 快照和 SLO 数据自动构建服务依赖 DAG，**基线引擎 (Baseline Engine)** 对每个实体的每个指标建立动态基线并输出异常分数。两者共同为 Phase 2 的风险评分和状态机提供数据基础。

**前置依赖**: SLO OTel 改造（已完成）— Service/Edge/Ingress 三层 SLO 数据已就绪。

**中心文档**: [`aiops-engine-design.md`](../future/aiops-engine-design.md) §4.1 (M1) + §4.2 (M2)

---

## 1. 文件夹结构

现有文件标 `(现有)`，新增标 `<- NEW`，修改标 `<- 修改`。

```
atlhyper_master_v2/
├── master.go                                (现有)  <- 修改: AIOps 初始化 + OnSnapshotReceived 回调
│
├── aiops/                                            <- NEW (整个目录)
│   ├── interfaces.go                                 <- NEW  AIOpsEngine 对外接口
│   ├── factory.go                                    <- NEW  NewAIOpsEngine()
│   ├── engine.go                                     <- NEW  引擎核心逻辑 (OnSnapshot 编排)
│   ├── types.go                                      <- NEW  共用类型定义
│   │
│   ├── correlator/                                   <- NEW  M1: 依赖图引擎
│   │   ├── builder.go                                <- NEW  从 ClusterSnapshot 构建 DAG
│   │   ├── updater.go                                <- NEW  diff 增量更新
│   │   ├── query.go                                  <- NEW  上下游遍历查询
│   │   └── serializer.go                             <- NEW  gzip/JSON 序列化（持久化/恢复）
│   │
│   └── baseline/                                     <- NEW  M2: 基线引擎
│       ├── detector.go                               <- NEW  EMA + 3σ 异常检测
│       ├── extractor.go                              <- NEW  从 Store/DB 提取指标
│       └── state.go                                  <- NEW  基线状态管理（内存缓存 + SQLite）
│
├── database/
│   ├── interfaces.go                        (现有)  <- 修改: +AIOpsBaselineRepository + AIOpsGraphRepository
│   ├── sqlite/
│   │   ├── migrations.go                    (现有)  <- 修改: +2 张表
│   │   ├── aiops_baseline.go                         <- NEW  基线状态 SQL Dialect
│   │   └── aiops_graph.go                            <- NEW  依赖图快照 SQL Dialect
│   └── repo/
│       ├── aiops_baseline.go                         <- NEW  基线 Repository 实现
│       └── aiops_graph.go                            <- NEW  依赖图 Repository 实现
│
├── service/
│   ├── interfaces.go                        (现有)  <- 修改: Query 接口 +3 方法
│   └── query/
│       └── aiops.go                                  <- NEW  AIOps 查询实现
│
├── gateway/
│   ├── routes.go                            (现有)  <- 修改: +3 路由
│   └── handler/
│       ├── aiops_graph.go                            <- NEW  依赖图 API Handler
│       └── aiops_baseline.go                         <- NEW  基线查询 API Handler
│
└── config/
    └── types.go                             (现有)  <- 修改: +AIOpsConfig
```

### 变更统计

| 操作 | 文件数 | 文件 |
|------|--------|------|
| **新建** | 16 | `aiops/` 下 8 个 + `database/` 下 4 个 + `service/query/aiops.go` + `handler/` 下 2 个 + `repo/` 下 1 个 |
| **修改** | 5 | `master.go`, `database/interfaces.go`, `migrations.go`, `service/interfaces.go`, `gateway/routes.go`, `config/types.go` |

---

## 2. 调用链路

### 2.1 数据写入路径（快照到达 -> 图更新 + 基线更新）

```
Agent 上报 ClusterSnapshot
    ↓
agentsdk/snapshot.go → processor.ProcessSnapshot()
    ↓ OnSnapshotReceived 回调
┌────────────────────────────────────────────────────────────────┐
│  既有回调:                                                      │
│    eventPersist.Sync(clusterID)                                │
│    metricsPersist.Sync(clusterID)                              │
│    sloPersist.Sync(clusterID)                                  │
│                                                                │
│  ★ 新增:                                                       │
│    aiopsEngine.OnSnapshot(ctx, clusterID)                      │
│        │                                                       │
│        ├── 1. correlator.Update(snapshot)                      │
│        │   ├── 提取 K8s 拓扑 (Pod/Service/Node/Ingress)       │
│        │   ├── 提取 SLO 边 (Linkerd outbound edges)            │
│        │   ├── diff 计算 (新增/删除节点和边)                     │
│        │   └── 更新内存图 (DependencyGraph)                     │
│        │                                                       │
│        ├── 2. baseline.Update(clusterID, snapshot)             │
│        │   ├── extractor: 提取各实体指标值                      │
│        │   │   ├── Node: CPU/Memory/Disk (从 NodeMetrics)      │
│        │   │   ├── Pod: RestartCount/Status (从 K8s 快照)      │
│        │   │   ├── Service: ErrorRate/P99 (从 SLO hourly/raw)  │
│        │   │   └── Ingress: ErrorRate/P99 (从 SLO hourly/raw)  │
│        │   ├── detector: EMA + 3σ 异常检测                     │
│        │   │   ├── 更新 EMA/Variance                           │
│        │   │   ├── 计算 deviation                              │
│        │   │   └── 输出 AnomalyResult[]                        │
│        │   └── state: 持久化基线状态                            │
│        │       ├── 内存缓存 (map[entityKey+metricName])        │
│        │       └── 定期 flush 到 SQLite (baseline_states)      │
│        │                                                       │
│        └── 3. 定期持久化依赖图快照 (每 5 分钟)                  │
│            └── serializer.Serialize() -> graphRepo.Save()      │
└────────────────────────────────────────────────────────────────┘
```

### 2.2 数据读取路径（API 查询）

```
┌──── 依赖图查询 ──────────────────────────────────────────────────┐
│                                                                  │
│  GET /api/v2/aiops/graph?cluster={id}                           │
│      ↓                                                          │
│  handler/aiops_graph.go: GraphHandler.GetGraph()                │
│      ↓ 调用 service.Query                                       │
│  service/query/aiops.go: GetAIOpsGraph()                        │
│      ↓ 调用 aiopsEngine                                         │
│  aiops/correlator/query.go: GetGraph(clusterID)                 │
│      → 返回 DependencyGraph (节点 + 边)                         │
│                                                                  │
│  GET /api/v2/aiops/graph/trace?cluster={id}&from={key}&dir=up   │
│      ↓                                                          │
│  handler/aiops_graph.go: GraphHandler.Trace()                   │
│      ↓ 调用 service.Query                                       │
│  service/query/aiops.go: GetAIOpsGraphTrace()                   │
│      ↓ 调用 aiopsEngine                                         │
│  aiops/correlator/query.go: Trace(from, direction)              │
│      → BFS 遍历，返回子图 (受影响的节点 + 边)                   │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘

┌──── 基线查询 ────────────────────────────────────────────────────┐
│                                                                  │
│  GET /api/v2/aiops/baseline?cluster={id}&entity={key}           │
│      ↓                                                          │
│  handler/aiops_baseline.go: BaselineHandler.GetBaseline()       │
│      ↓ 调用 service.Query                                       │
│  service/query/aiops.go: GetAIOpsBaseline()                     │
│      ↓ 调用 aiopsEngine                                         │
│  aiops/baseline/state.go: GetStates(entityKey)                  │
│      → 返回实体的所有指标基线状态 + 异常分数                     │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### 2.3 初始化链路（master.go）

```go
New() {
    // ... 现有初始化 (步骤 1-4.3) ...

    // 4.4 初始化 AIOps 引擎 (在 SLO 组件之后、Service 层之前)
    aiopsEngine := aiops.NewAIOpsEngine(aiops.EngineConfig{
        Store:           store,
        BaselineRepo:    db.AIOpsBaseline,
        GraphRepo:       db.AIOpsGraph,
        SLORepo:         db.SLO,
        SLOServiceRepo:  db.SLOService,
        SLOEdgeRepo:     db.SLOEdge,
        NodeMetricsRepo: db.NodeMetrics,
        Config:          cfg.AIOps,
    })

    // 5. 初始化 Processor — OnSnapshotReceived 回调中追加 AIOps
    proc := processor.New(processor.Config{
        Store: store,
        OnSnapshotReceived: func(clusterID string) {
            eventPersist.Sync(clusterID)
            metricsPersist.Sync(clusterID)
            sloPersist.Sync(clusterID)
            aiopsEngine.OnSnapshot(ctx, clusterID)  // ← 新增
        },
    })

    // 6. 初始化 Query（构造函数注入 AIOps 引擎）
    q := query.NewQueryService(store, bus, db.Event, db.SLO, db.SLOService, db.SLOEdge, aiopsEngine)

    // ... 后续初始化不变 ...
}

Run() {
    // ... 现有启动 ...
    aiopsEngine.Start()  // 启动后台任务（图快照定期持久化）
}

Stop() {
    aiopsEngine.Stop()  // 停止后台任务
    // ... 现有停止 ...
}
```

---

## 3. 数据模型

### 3.1 依赖图数据模型

```go
// aiops/types.go

// GraphNode 图节点
type GraphNode struct {
    Key       string            `json:"key"`       // "default/service/api-server"
    Type      string            `json:"type"`      // "ingress" | "service" | "pod" | "node"
    Namespace string            `json:"namespace"`
    Name      string            `json:"name"`
    Metadata  map[string]string `json:"metadata"`  // 附加信息
}

// GraphEdge 图边
type GraphEdge struct {
    From   string  `json:"from"`   // source node key
    To     string  `json:"to"`     // target node key
    Type   string  `json:"type"`   // "routes_to" | "calls" | "runs_on" | "selects"
    Weight float64 `json:"weight"` // 边权重 (默认 1.0)
}

// DependencyGraph 依赖图
type DependencyGraph struct {
    ClusterID string                `json:"clusterId"`
    Nodes     map[string]*GraphNode `json:"nodes"`   // key -> node
    Edges     []*GraphEdge          `json:"edges"`
    UpdatedAt time.Time             `json:"updatedAt"`

    // 内部索引（不序列化）
    adjacency map[string][]string   // 正向邻接表: from -> [to...]
    reverse   map[string][]string   // 反向邻接表: to -> [from...]
}

// TraceResult 链路追踪结果
type TraceResult struct {
    Nodes []*GraphNode `json:"nodes"` // 链路上的节点
    Edges []*GraphEdge `json:"edges"` // 链路上的边
    Depth int          `json:"depth"` // 遍历深度
}
```

### 3.2 基线数据模型

```go
// aiops/types.go

// BaselineState 基线状态（每个实体-指标对）
type BaselineState struct {
    EntityKey  string  `json:"entityKey"`  // "default/service/api-server"
    MetricName string  `json:"metricName"` // "error_rate" | "p99_latency" | "cpu_usage"
    EMA        float64 `json:"ema"`        // 当前 EMA 值
    Variance   float64 `json:"variance"`   // 当前方差
    Count      int64   `json:"count"`      // 已处理的数据点数
    UpdatedAt  int64   `json:"updatedAt"`  // 最后更新时间戳 (Unix)
}

// AnomalyResult 异常检测结果
type AnomalyResult struct {
    EntityKey    string  `json:"entityKey"`
    MetricName   string  `json:"metricName"`
    CurrentValue float64 `json:"currentValue"`
    Baseline     float64 `json:"baseline"`     // EMA
    Deviation    float64 `json:"deviation"`    // 偏离度（σ 倍数）
    Score        float64 `json:"score"`        // 归一化异常分数 [0, 1]
    IsAnomaly    bool    `json:"isAnomaly"`    // deviation > 3
    DetectedAt   int64   `json:"detectedAt"`   // Unix timestamp
}

// EntityBaseline 实体基线汇总（API 响应）
type EntityBaseline struct {
    EntityKey string           `json:"entityKey"`
    States    []*BaselineState `json:"states"`   // 所有指标的基线状态
    Anomalies []*AnomalyResult `json:"anomalies"` // 当前异常（如有）
}

// ColdStartThreshold 冷启动阈值
const (
    ColdStartMinCount = 100  // 前 100 个数据点只学习不告警 (~50 分钟)
    DefaultAlpha      = 0.033 // α = 2/(60+1), 窗口 60 个采样点
    AnomalyThreshold  = 3.0  // 3σ 规则
    SigmoidK          = 2.0  // sigmoid 斜率
)
```

### 3.3 节点 Key 生成规则

```go
// aiops/helpers.go

// EntityKey 生成实体唯一标识
// 格式: "namespace/type/name"
// 示例:
//   "default/pod/api-server-abc-123"
//   "default/service/api-server"
//   "kube-system/node/worker-3"
//   "default/ingress/app.example.com"
func EntityKey(namespace, entityType, name string) string {
    if namespace == "" {
        namespace = "_cluster" // Node 等无 namespace 的资源
    }
    return namespace + "/" + entityType + "/" + name
}
```

---

## 4. 详细设计

### 4.1 依赖图构建 (correlator/builder.go)

从 `ClusterSnapshot` 提取四种边关系：

```go
// BuildFromSnapshot 从快照构建完整依赖图
func BuildFromSnapshot(clusterID string, snap *model_v2.ClusterSnapshot) *DependencyGraph {
    g := NewDependencyGraph(clusterID)

    // 1. Pod → Node (runs_on)
    for _, pod := range snap.Pods {
        podKey := EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
        nodeKey := EntityKey("_cluster", "node", pod.Status.NodeName)
        g.AddNode(podKey, "pod", pod.Summary.Namespace, pod.Summary.Name, nil)
        g.AddNode(nodeKey, "node", "_cluster", pod.Status.NodeName, nil)
        g.AddEdge(podKey, nodeKey, "runs_on", 1.0)
    }

    // 2. Service → Pod (selects)
    for _, svc := range snap.Services {
        svcKey := EntityKey(svc.Summary.Namespace, "service", svc.Summary.Name)
        g.AddNode(svcKey, "service", svc.Summary.Namespace, svc.Summary.Name, nil)
        // 通过 Selector 匹配 Pod
        for _, pod := range snap.Pods {
            if matchSelector(svc.Spec.Selector, pod.Metadata.Labels) &&
               svc.Summary.Namespace == pod.Summary.Namespace {
                podKey := EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
                g.AddEdge(svcKey, podKey, "selects", 1.0)
            }
        }
    }

    // 3. Ingress → Service (routes_to)
    for _, ing := range snap.Ingresses {
        ingKey := EntityKey(ing.Summary.Namespace, "ingress", ing.Summary.Name)
        g.AddNode(ingKey, "ingress", ing.Summary.Namespace, ing.Summary.Name, nil)
        for _, rule := range ing.Spec.Rules {
            for _, path := range rule.Paths {
                svcKey := EntityKey(ing.Summary.Namespace, "service", path.ServiceName)
                g.AddEdge(ingKey, svcKey, "routes_to", 1.0)
            }
        }
    }

    // 4. Service → Service (calls, 从 SLO Edge 数据)
    if snap.SLOData != nil {
        for _, edge := range snap.SLOData.Edges {
            srcKey := EntityKey(edge.SrcNamespace, "service", edge.SrcName)
            dstKey := EntityKey(edge.DstNamespace, "service", edge.DstName)
            g.AddEdge(srcKey, dstKey, "calls", 1.0)
        }
    }

    g.RebuildIndex() // 构建邻接表
    return g
}
```

### 4.2 增量更新 (correlator/updater.go)

```go
// Update 增量更新依赖图
// 避免每次快照全量重建，使用 diff 计算变更
func (g *DependencyGraph) Update(newGraph *DependencyGraph) DiffResult {
    diff := DiffResult{}

    // 1. 检测新增节点
    for key, node := range newGraph.Nodes {
        if _, exists := g.Nodes[key]; !exists {
            g.Nodes[key] = node
            diff.AddedNodes = append(diff.AddedNodes, key)
        }
    }

    // 2. 检测删除节点
    for key := range g.Nodes {
        if _, exists := newGraph.Nodes[key]; !exists {
            delete(g.Nodes, key)
            diff.RemovedNodes = append(diff.RemovedNodes, key)
        }
    }

    // 3. 检测边变更（使用 from+to+type 作为 key）
    oldEdgeSet := edgeSet(g.Edges)
    newEdgeSet := edgeSet(newGraph.Edges)

    for key, edge := range newEdgeSet {
        if _, exists := oldEdgeSet[key]; !exists {
            diff.AddedEdges = append(diff.AddedEdges, edge)
        }
    }
    for key, edge := range oldEdgeSet {
        if _, exists := newEdgeSet[key]; !exists {
            diff.RemovedEdges = append(diff.RemovedEdges, edge)
        }
    }

    // 4. 替换边列表，重建索引
    g.Edges = newGraph.Edges
    g.RebuildIndex()
    g.UpdatedAt = time.Now()

    return diff
}

// DiffResult 图变更结果
type DiffResult struct {
    AddedNodes   []string
    RemovedNodes []string
    AddedEdges   []*GraphEdge
    RemovedEdges []*GraphEdge
}
```

### 4.3 图查询 (correlator/query.go)

```go
// GetGraph 返回指定集群的完整依赖图
func (c *Correlator) GetGraph(clusterID string) *DependencyGraph {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.graphs[clusterID]
}

// Trace 从指定实体出发，BFS 遍历上游或下游链路
func (c *Correlator) Trace(clusterID, fromKey, direction string, maxDepth int) *TraceResult {
    c.mu.RLock()
    defer c.mu.RUnlock()

    graph := c.graphs[clusterID]
    if graph == nil {
        return &TraceResult{}
    }

    if maxDepth <= 0 {
        maxDepth = 10 // 默认最大深度
    }

    visited := map[string]bool{}
    result := &TraceResult{}
    queue := []struct{ key string; depth int }{{fromKey, 0}}

    for len(queue) > 0 {
        current := queue[0]
        queue = queue[1:]

        if visited[current.key] || current.depth > maxDepth {
            continue
        }
        visited[current.key] = true

        if node, ok := graph.Nodes[current.key]; ok {
            result.Nodes = append(result.Nodes, node)
        }

        // 根据方向选择邻接表
        var neighbors []string
        if direction == "downstream" {
            neighbors = graph.adjacency[current.key]
        } else { // upstream
            neighbors = graph.reverse[current.key]
        }

        for _, neighbor := range neighbors {
            if !visited[neighbor] {
                queue = append(queue, struct{ key string; depth int }{neighbor, current.depth + 1})
                // 收集边
                for _, edge := range graph.Edges {
                    if (direction == "downstream" && edge.From == current.key && edge.To == neighbor) ||
                       (direction == "upstream" && edge.To == current.key && edge.From == neighbor) {
                        result.Edges = append(result.Edges, edge)
                    }
                }
            }
        }
        if current.depth > result.Depth {
            result.Depth = current.depth
        }
    }
    return result
}
```

### 4.4 图序列化 (correlator/serializer.go)

```go
// Serialize 将图序列化为 gzip(JSON)
func Serialize(graph *DependencyGraph) ([]byte, error) {
    jsonData, err := json.Marshal(graph)
    if err != nil {
        return nil, fmt.Errorf("marshal graph: %w", err)
    }
    return common.GzipBytes(jsonData)
}

// Deserialize 从 gzip(JSON) 恢复图
func Deserialize(data []byte) (*DependencyGraph, error) {
    jsonData, err := common.GunzipBytes(data)
    if err != nil {
        return nil, fmt.Errorf("gunzip graph: %w", err)
    }
    var graph DependencyGraph
    if err := json.Unmarshal(jsonData, &graph); err != nil {
        return nil, fmt.Errorf("unmarshal graph: %w", err)
    }
    graph.RebuildIndex() // 重建内部索引
    return &graph, nil
}
```

### 4.5 EMA + 3σ 异常检测 (baseline/detector.go)

```go
// Detect 对单个指标执行异常检测
// 返回更新后的状态和异常结果
func Detect(state *BaselineState, value float64, now int64) (*BaselineState, *AnomalyResult) {
    state.Count++

    // 冷启动：只学习，不告警
    if state.Count <= ColdStartMinCount {
        if state.Count == 1 {
            state.EMA = value
            state.Variance = 0
        } else {
            alpha := DefaultAlpha
            state.EMA = alpha*value + (1-alpha)*state.EMA
            diff := value - state.EMA
            state.Variance = alpha*diff*diff + (1-alpha)*state.Variance
        }
        state.UpdatedAt = now
        return state, nil // 冷启动期间不返回异常
    }

    // 正常检测
    alpha := DefaultAlpha
    oldEMA := state.EMA
    state.EMA = alpha*value + (1-alpha)*state.EMA
    diff := value - oldEMA
    state.Variance = alpha*diff*diff + (1-alpha)*state.Variance
    state.UpdatedAt = now

    // 计算偏离度
    sigma := math.Sqrt(state.Variance)
    var deviation float64
    if sigma > 1e-9 {
        deviation = math.Abs(value-state.EMA) / sigma
    }

    // 归一化到 [0, 1]
    score := sigmoid(deviation, AnomalyThreshold, SigmoidK)

    result := &AnomalyResult{
        EntityKey:    state.EntityKey,
        MetricName:   state.MetricName,
        CurrentValue: value,
        Baseline:     state.EMA,
        Deviation:    deviation,
        Score:        score,
        IsAnomaly:    deviation > AnomalyThreshold,
        DetectedAt:   now,
    }

    return state, result
}

// sigmoid 归一化函数
// score = 1 / (1 + exp(-k * (deviation - threshold)))
func sigmoid(deviation, threshold, k float64) float64 {
    return 1.0 / (1.0 + math.Exp(-k*(deviation-threshold)))
}
```

### 4.6 指标提取 (baseline/extractor.go)

```go
// ExtractMetrics 从快照和 SLO 数据中提取所有实体指标
func ExtractMetrics(
    clusterID string,
    snap *model_v2.ClusterSnapshot,
    sloServiceRepo database.SLOServiceRepository,
    sloRepo database.SLORepository,
    nodeMetricsRepo database.NodeMetricsRepository,
) []MetricDataPoint {
    var points []MetricDataPoint
    now := time.Now()
    lookback := now.Add(-5 * time.Minute) // 取最近 5 分钟的 SLO 数据

    // 1. Node 指标（从 ClusterSnapshot.NodeMetrics）
    for nodeName, metrics := range snap.NodeMetrics {
        key := EntityKey("_cluster", "node", nodeName)
        points = append(points,
            MetricDataPoint{EntityKey: key, MetricName: "cpu_usage", Value: metrics.CPU.UsagePercent},
            MetricDataPoint{EntityKey: key, MetricName: "memory_usage", Value: metrics.Memory.UsagePercent},
        )
        if disk := metrics.GetPrimaryDisk(); disk != nil {
            points = append(points,
                MetricDataPoint{EntityKey: key, MetricName: "disk_usage", Value: disk.UsagePercent},
            )
        }
        // PSI
        points = append(points,
            MetricDataPoint{EntityKey: key, MetricName: "psi_cpu", Value: metrics.PSI.CPUSomePercent},
            MetricDataPoint{EntityKey: key, MetricName: "psi_memory", Value: metrics.PSI.MemorySomePercent},
            MetricDataPoint{EntityKey: key, MetricName: "psi_io", Value: metrics.PSI.IOSomePercent},
        )
    }

    // 2. Pod 指标（从 K8s 快照）
    for _, pod := range snap.Pods {
        key := EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
        restarts := float64(pod.Status.TotalRestarts)
        isRunning := 0.0
        if pod.Status.Phase == "Running" {
            isRunning = 1.0
        }
        points = append(points,
            MetricDataPoint{EntityKey: key, MetricName: "restart_count", Value: restarts},
            MetricDataPoint{EntityKey: key, MetricName: "is_running", Value: isRunning},
        )
    }

    // 3. Service 指标（从 SLO Service Raw — 最近 5 分钟的聚合）
    ctx := context.Background()
    for _, svc := range snap.Services {
        key := EntityKey(svc.Summary.Namespace, "service", svc.Summary.Name)
        raws, err := sloServiceRepo.GetServiceRaw(ctx, clusterID, svc.Summary.Namespace, svc.Summary.Name, lookback, now)
        if err != nil || len(raws) == 0 {
            continue
        }
        // 聚合最近的 raw 数据
        var totalReqs, errorReqs int64
        var latencySum float64
        var latencyCount int64
        for _, r := range raws {
            totalReqs += r.TotalRequests
            errorReqs += r.ErrorRequests
            latencySum += r.LatencySum
            latencyCount += r.LatencyCount
        }
        if totalReqs > 0 {
            errorRate := float64(errorReqs) / float64(totalReqs) * 100
            points = append(points,
                MetricDataPoint{EntityKey: key, MetricName: "error_rate", Value: errorRate},
            )
        }
        if latencyCount > 0 {
            avgLatency := latencySum / float64(latencyCount)
            points = append(points,
                MetricDataPoint{EntityKey: key, MetricName: "avg_latency", Value: avgLatency},
            )
        }
        points = append(points,
            MetricDataPoint{EntityKey: key, MetricName: "request_rate", Value: float64(totalReqs) / 300.0}, // 5 分钟平均 RPS
        )
    }

    // 4. Ingress 指标（从 SLO Metrics Raw — 类似 Service）
    // 逻辑与 Service 类似，从 sloRepo.GetRawMetrics 读取

    return points
}

// MetricDataPoint 指标数据点
type MetricDataPoint struct {
    EntityKey  string
    MetricName string
    Value      float64
}
```

### 4.7 基线状态管理 (baseline/state.go)

```go
// StateManager 基线状态管理器
// 维护内存缓存 + 定期 flush 到 SQLite
type StateManager struct {
    mu     sync.RWMutex
    states map[string]*BaselineState // key = entityKey + "|" + metricName
    dirty  map[string]bool           // 需要 flush 的状态
    repo   database.AIOpsBaselineRepository

    // 最新异常结果缓存（供 API 查询）
    anomalies map[string][]*AnomalyResult // key = entityKey
}

// Update 更新指标并检测异常
func (m *StateManager) Update(points []MetricDataPoint) []*AnomalyResult {
    m.mu.Lock()
    defer m.mu.Unlock()

    now := time.Now().Unix()
    var results []*AnomalyResult

    for _, p := range points {
        cacheKey := p.EntityKey + "|" + p.MetricName

        // 获取或创建状态
        state, ok := m.states[cacheKey]
        if !ok {
            state = &BaselineState{
                EntityKey:  p.EntityKey,
                MetricName: p.MetricName,
            }
            m.states[cacheKey] = state
        }

        // 执行异常检测
        _, result := Detect(state, p.Value, now)
        m.dirty[cacheKey] = true

        if result != nil {
            results = append(results, result)
            // 更新异常缓存
            m.anomalies[p.EntityKey] = appendOrReplace(m.anomalies[p.EntityKey], result)
        }
    }

    return results
}

// FlushToDB 将脏状态批量写入数据库
func (m *StateManager) FlushToDB(ctx context.Context) error {
    m.mu.Lock()
    dirtyStates := make([]*BaselineState, 0, len(m.dirty))
    for key := range m.dirty {
        if state, ok := m.states[key]; ok {
            dirtyStates = append(dirtyStates, state)
        }
    }
    m.dirty = make(map[string]bool)
    m.mu.Unlock()

    // 批量写入
    return m.repo.BatchUpsert(ctx, dirtyStates)
}

// LoadFromDB 启动时从数据库恢复状态
func (m *StateManager) LoadFromDB(ctx context.Context) error {
    states, err := m.repo.ListAll(ctx)
    if err != nil {
        return err
    }
    m.mu.Lock()
    defer m.mu.Unlock()
    for _, s := range states {
        m.states[s.EntityKey+"|"+s.MetricName] = s
    }
    return nil
}

// GetStates 返回指定实体的所有基线状态
func (m *StateManager) GetStates(entityKey string) *EntityBaseline {
    m.mu.RLock()
    defer m.mu.RUnlock()

    result := &EntityBaseline{EntityKey: entityKey}
    for key, state := range m.states {
        if strings.HasPrefix(key, entityKey+"|") {
            result.States = append(result.States, state)
        }
    }
    result.Anomalies = m.anomalies[entityKey]
    return result
}
```

### 4.8 AIOps 引擎编排 (aiops/engine.go)

```go
// Engine AIOps 引擎核心
type Engine struct {
    correlator *correlator.Correlator
    baseline   *baseline.StateManager
    store      datahub.Store
    graphRepo  database.AIOpsGraphRepository

    // SLO 仓库（用于基线指标提取）
    sloServiceRepo  database.SLOServiceRepository
    sloRepo         database.SLORepository
    nodeMetricsRepo database.NodeMetricsRepository

    config    EngineConfig
    stopCh    chan struct{}
    lastFlush time.Time
}

// OnSnapshot 快照到达时的处理入口
func (e *Engine) OnSnapshot(ctx context.Context, clusterID string) {
    snap, err := e.store.GetSnapshot(clusterID)
    if err != nil || snap == nil {
        return
    }

    // 1. 更新依赖图
    newGraph := correlator.BuildFromSnapshot(clusterID, snap)
    e.correlator.Update(clusterID, newGraph)

    // 2. 提取指标 + 基线检测
    points := baseline.ExtractMetrics(
        clusterID, snap,
        e.sloServiceRepo, e.sloRepo, e.nodeMetricsRepo,
    )
    anomalies := e.baseline.Update(points)

    // 记录异常日志
    for _, a := range anomalies {
        if a.IsAnomaly {
            log.Warn("AIOps 异常检测",
                "entity", a.EntityKey,
                "metric", a.MetricName,
                "value", a.CurrentValue,
                "baseline", a.Baseline,
                "deviation", fmt.Sprintf("%.1fσ", a.Deviation),
            )
        }
    }

    // 3. 定期持久化（基线状态 + 图快照）
    if time.Since(e.lastFlush) > e.config.FlushInterval {
        e.baseline.FlushToDB(ctx)
        e.persistGraph(ctx, clusterID)
        e.lastFlush = time.Now()
    }
}

// Start 启动后台任务
func (e *Engine) Start() {
    // 从数据库恢复基线状态
    if err := e.baseline.LoadFromDB(context.Background()); err != nil {
        log.Warn("恢复基线状态失败", "err", err)
    }

    // 从数据库恢复依赖图快照
    e.restoreGraphs(context.Background())
}

// Stop 停止并持久化状态
func (e *Engine) Stop() {
    ctx := context.Background()
    e.baseline.FlushToDB(ctx)
    // 持久化所有集群的图
    for clusterID := range e.correlator.ListClusters() {
        e.persistGraph(ctx, clusterID)
    }
}
```

---

## 5. 数据库表结构

### 5.1 基线状态表

```sql
-- 持久化 EMA 状态，用于重启恢复
CREATE TABLE IF NOT EXISTS aiops_baseline_states (
    entity_key  TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    ema         REAL NOT NULL,
    variance    REAL NOT NULL,
    count       INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    PRIMARY KEY (entity_key, metric_name)
);
```

### 5.2 依赖图快照表

```sql
-- 定期持久化图快照，用于重启恢复（每集群一条，覆盖式更新）
CREATE TABLE IF NOT EXISTS aiops_dependency_graph_snapshots (
    cluster_id TEXT NOT NULL,
    snapshot   BLOB NOT NULL,             -- gzip(JSON) 序列化的图
    created_at TEXT DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (cluster_id)
);
```

---

## 6. API 端点

### 6.1 获取依赖图

```
GET /api/v2/aiops/graph?cluster={id}

权限: Public (只读)

响应:
{
    "message": "获取成功",
    "data": {
        "clusterId": "cluster-1",
        "nodes": {
            "default/service/api-server": {
                "key": "default/service/api-server",
                "type": "service",
                "namespace": "default",
                "name": "api-server",
                "metadata": {}
            },
            ...
        },
        "edges": [
            {
                "from": "default/ingress/app.example.com",
                "to": "default/service/api-server",
                "type": "routes_to",
                "weight": 1.0
            },
            ...
        ],
        "updatedAt": "2026-01-15T10:00:00Z"
    }
}
```

### 6.2 链路追踪

```
GET /api/v2/aiops/graph/trace?cluster={id}&from={entity_key}&direction=upstream|downstream&max_depth=5

权限: Public (只读)

参数:
  - cluster: 集群 ID (必需)
  - from: 起始实体 key (必需, URL 编码)
  - direction: 遍历方向 (可选, 默认 upstream)
  - max_depth: 最大深度 (可选, 默认 10)

响应:
{
    "message": "获取成功",
    "data": {
        "nodes": [...],
        "edges": [...],
        "depth": 3
    }
}
```

### 6.3 基线查询

```
GET /api/v2/aiops/baseline?cluster={id}&entity={entity_key}

权限: Public (只读)

参数:
  - cluster: 集群 ID (必需)
  - entity: 实体 key (必需, URL 编码)

响应:
{
    "message": "获取成功",
    "data": {
        "entityKey": "default/service/api-server",
        "states": [
            {
                "entityKey": "default/service/api-server",
                "metricName": "error_rate",
                "ema": 0.5,
                "variance": 0.04,
                "count": 1200,
                "updatedAt": 1705312800
            },
            ...
        ],
        "anomalies": [
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
        ]
    }
}
```

---

## 7. Service 层接口变更

```go
// service/interfaces.go — Query 接口新增 AIOps 查询方法
type Query interface {
    // ... 现有方法 ...

    // ==================== AIOps 查询 ====================

    GetAIOpsGraph(ctx context.Context, clusterID string) (*aiops.DependencyGraph, error)
    GetAIOpsGraphTrace(ctx context.Context, clusterID, fromKey, direction string, maxDepth int) (*aiops.TraceResult, error)
    GetAIOpsBaseline(ctx context.Context, clusterID, entityKey string) (*aiops.EntityBaseline, error)
}
```

```go
// service/query/aiops.go — 查询实现
// aiopsEngine 通过 NewQueryService() 构造函数注入，保证非 nil

func (q *QueryService) GetAIOpsGraph(ctx context.Context, clusterID string) (*aiops.DependencyGraph, error) {
    return q.aiopsEngine.GetGraph(clusterID), nil
}

func (q *QueryService) GetAIOpsGraphTrace(ctx context.Context, clusterID, fromKey, direction string, maxDepth int) (*aiops.TraceResult, error) {
    return q.aiopsEngine.Trace(clusterID, fromKey, direction, maxDepth), nil
}

func (q *QueryService) GetAIOpsBaseline(ctx context.Context, clusterID, entityKey string) (*aiops.EntityBaseline, error) {
    return q.aiopsEngine.GetBaseline(entityKey), nil
}
```

---

## 8. Gateway Handler + 路由注册

### 8.1 Handler 定义

```go
// handler/aiops_graph.go
type AIOpsGraphHandler struct {
    query service.Query
}

func NewAIOpsGraphHandler(query service.Query) *AIOpsGraphHandler {
    return &AIOpsGraphHandler{query: query}
}

func (h *AIOpsGraphHandler) GetGraph(w http.ResponseWriter, r *http.Request) {
    clusterID := r.URL.Query().Get("cluster")
    if clusterID == "" {
        writeError(w, http.StatusBadRequest, "missing cluster parameter")
        return
    }
    graph, err := h.query.GetAIOpsGraph(r.Context(), clusterID)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, graph)
}

func (h *AIOpsGraphHandler) Trace(w http.ResponseWriter, r *http.Request) {
    clusterID := r.URL.Query().Get("cluster")
    fromKey := r.URL.Query().Get("from")
    direction := r.URL.Query().Get("direction")
    if direction == "" {
        direction = "upstream"
    }
    maxDepth := 10
    if d := r.URL.Query().Get("max_depth"); d != "" {
        maxDepth, _ = strconv.Atoi(d)
    }
    result, err := h.query.GetAIOpsGraphTrace(r.Context(), clusterID, fromKey, direction, maxDepth)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    writeJSON(w, http.StatusOK, result)
}
```

```go
// handler/aiops_baseline.go
type AIOpsBaselineHandler struct {
    query service.Query
}

func NewAIOpsBaselineHandler(query service.Query) *AIOpsBaselineHandler {
    return &AIOpsBaselineHandler{query: query}
}

func (h *AIOpsBaselineHandler) GetBaseline(w http.ResponseWriter, r *http.Request) {
    clusterID := r.URL.Query().Get("cluster")
    entityKey := r.URL.Query().Get("entity")
    if clusterID == "" || entityKey == "" {
        writeError(w, http.StatusBadRequest, "missing cluster or entity parameter")
        return
    }
    result, err := h.query.GetAIOpsBaseline(r.Context(), clusterID, entityKey)
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

// ---------- AIOps 查询（只读） ----------
aiopsGraphHandler := handler.NewAIOpsGraphHandler(r.service)
aiopsBaselineHandler := handler.NewAIOpsBaselineHandler(r.service)

register("/api/v2/aiops/graph", aiopsGraphHandler.GetGraph)
register("/api/v2/aiops/graph/trace", aiopsGraphHandler.Trace)
register("/api/v2/aiops/baseline", aiopsBaselineHandler.GetBaseline)
```

---

## 9. 配置变更

```go
// config/types.go — 新增

// AIOpsConfig AIOps 引擎配置
type AIOpsConfig struct {
    FlushInterval time.Duration // 基线/图状态持久化间隔（默认 5m）
    BaselineAlpha float64       // EMA 平滑系数 α（默认 0.033, 窗口 60）
    AnomalyThreshold float64   // 异常阈值（σ 倍数，默认 3.0）
}

// AppConfig 中新增字段
type AppConfig struct {
    // ... 现有字段 ...
    AIOps AIOpsConfig
}
```

---

## 10. Database 接口变更

```go
// database/interfaces.go — 新增

// DB 结构体新增字段
type DB struct {
    // ... 现有字段 ...
    AIOpsBaseline AIOpsBaselineRepository
    AIOpsGraph    AIOpsGraphRepository
}

// AIOpsBaselineRepository 基线状态数据访问接口
type AIOpsBaselineRepository interface {
    BatchUpsert(ctx context.Context, states []*AIOpsBaselineState) error
    ListAll(ctx context.Context) ([]*AIOpsBaselineState, error)
    ListByEntity(ctx context.Context, entityKey string) ([]*AIOpsBaselineState, error)
    DeleteByEntity(ctx context.Context, entityKey string) error
}

// AIOpsGraphRepository 依赖图快照数据访问接口
type AIOpsGraphRepository interface {
    Save(ctx context.Context, clusterID string, snapshot []byte) error
    Load(ctx context.Context, clusterID string) ([]byte, error)
    ListClusterIDs(ctx context.Context) ([]string, error)
}

// AIOpsBaselineState 基线状态数据库模型
type AIOpsBaselineState struct {
    EntityKey  string
    MetricName string
    EMA        float64
    Variance   float64
    Count      int64
    UpdatedAt  int64
}

// Dialect 接口新增
type Dialect interface {
    // ... 现有方法 ...
    AIOpsBaseline() AIOpsBaselineDialect
    AIOpsGraph()    AIOpsGraphDialect
}

// AIOpsBaselineDialect 基线 SQL 方言
type AIOpsBaselineDialect interface {
    BatchUpsert(states []*AIOpsBaselineState) (query string, args []any)
    SelectAll() (query string, args []any)
    SelectByEntity(entityKey string) (query string, args []any)
    DeleteByEntity(entityKey string) (query string, args []any)
    ScanRow(rows *sql.Rows) (*AIOpsBaselineState, error)
}

// AIOpsGraphDialect 依赖图 SQL 方言
type AIOpsGraphDialect interface {
    Upsert(clusterID string, snapshot []byte) (query string, args []any)
    SelectByCluster(clusterID string) (query string, args []any)
    SelectAllClusterIDs() (query string, args []any)
    ScanSnapshot(rows *sql.Rows) (clusterID string, data []byte, err error)
}
```

---

## 11. 实现阶段（TDD）

```
P1: 数据模型 + 数据库
  ├── 定义 aiops/types.go (GraphNode, GraphEdge, DependencyGraph, BaselineState, AnomalyResult)
  ├── database/interfaces.go 新增 Repository + Dialect
  ├── migrations.go 新增 2 张表
  ├── SQLite Dialect 实现 (aiops_baseline.go, aiops_graph.go)
  ├── Repository 实现 (repo/aiops_baseline.go, repo/aiops_graph.go)
  └── 单元测试: Repository CRUD 验证

P2: 依赖图引擎
  ├── correlator/builder.go — 从 ClusterSnapshot 构建 DAG
  ├── correlator/updater.go — diff 增量更新
  ├── correlator/query.go — BFS 遍历
  ├── correlator/serializer.go — 序列化/反序列化
  └── 单元测试: 图构建、更新、遍历、序列化

P3: 基线引擎
  ├── baseline/detector.go — EMA + 3σ 异常检测
  ├── baseline/extractor.go — 指标提取
  ├── baseline/state.go — 状态管理（内存 + 持久化）
  └── 单元测试: 异常检测（冷启动、正常、异常场景）

P4: 引擎编排 + 集成
  ├── aiops/interfaces.go — AIOpsEngine 接口
  ├── aiops/factory.go — 工厂函数
  ├── aiops/engine.go — OnSnapshot 编排
  ├── master.go — 初始化 + 回调注入
  └── 集成测试: 快照到达 → 图更新 + 基线更新

P5: API 层
  ├── service/interfaces.go +3 方法
  ├── service/query/aiops.go — 查询实现
  ├── handler/aiops_graph.go + handler/aiops_baseline.go
  ├── gateway/routes.go +3 路由
  ├── config/types.go +AIOpsConfig
  └── API 测试: 端到端验证
```

---

## 12. 文件变更清单

### 新建

| 文件 | 说明 |
|------|------|
| `aiops/interfaces.go` | AIOpsEngine 对外接口 |
| `aiops/factory.go` | NewAIOpsEngine() 工厂 |
| `aiops/engine.go` | 引擎核心（OnSnapshot 编排） |
| `aiops/types.go` | 共用类型（GraphNode, BaselineState, AnomalyResult 等） |
| `aiops/helpers.go` | 工具函数（EntityKey 生成等） |
| `aiops/correlator/builder.go` | 从 ClusterSnapshot 构建 DAG |
| `aiops/correlator/updater.go` | diff 增量更新 |
| `aiops/correlator/query.go` | BFS 上下游遍历 |
| `aiops/correlator/serializer.go` | gzip/JSON 序列化 |
| `aiops/baseline/detector.go` | EMA + 3σ 异常检测 |
| `aiops/baseline/extractor.go` | 从 Store/DB 提取指标 |
| `aiops/baseline/state.go` | 状态管理（内存缓存 + SQLite flush） |
| `database/sqlite/aiops_baseline.go` | 基线 SQL Dialect |
| `database/sqlite/aiops_graph.go` | 依赖图 SQL Dialect |
| `database/repo/aiops_baseline.go` | 基线 Repository 实现 |
| `database/repo/aiops_graph.go` | 依赖图 Repository 实现 |
| `service/query/aiops.go` | AIOps 查询实现 |
| `gateway/handler/aiops_graph.go` | 依赖图 API Handler |
| `gateway/handler/aiops_baseline.go` | 基线查询 API Handler |

### 修改

| 文件 | 变更 |
|------|------|
| `master.go` | AIOps 引擎初始化 + OnSnapshotReceived 追加 + Start/Stop 生命周期 |
| `database/interfaces.go` | +AIOpsBaselineRepository + AIOpsGraphRepository + 模型 + Dialect |
| `database/sqlite/migrations.go` | +2 张表 (aiops_baseline_states, aiops_dependency_graph_snapshots) |
| `service/interfaces.go` | Query 接口 +3 方法 (GetAIOpsGraph, GetAIOpsGraphTrace, GetAIOpsBaseline) |
| `gateway/routes.go` | +3 路由 (/api/v2/aiops/graph, trace, baseline) |
| `config/types.go` | +AIOpsConfig 结构体 + AppConfig 新增字段 |

---

## 13. 测试计划

### 单元测试

| 模块 | 测试文件 | 测试内容 |
|------|---------|---------|
| 图构建 | `correlator/builder_test.go` | 从 mock ClusterSnapshot 构建图，验证节点/边数量和关系 |
| 图更新 | `correlator/updater_test.go` | 新增/删除节点和边的 diff 计算 |
| 图查询 | `correlator/query_test.go` | BFS 遍历（upstream/downstream），环路处理 |
| 序列化 | `correlator/serializer_test.go` | 序列化 → 反序列化 → 一致性验证 |
| 异常检测 | `baseline/detector_test.go` | 冷启动（前 100 点无告警）、正常值（不触发）、3σ 异常（触发） |
| 状态管理 | `baseline/state_test.go` | 内存更新、FlushToDB、LoadFromDB 一致性 |
| Repository | `database/repo/aiops_*_test.go` | CRUD 操作 + 批量 upsert |

### 集成测试

| 场景 | 验证点 |
|------|--------|
| 快照到达 → 图自动更新 | 新 Pod 出现在图中、被删 Pod 从图中移除 |
| 快照到达 → 基线更新 | EMA 值随数据变化、异常值触发告警 |
| 重启恢复 | Stop → Start 后，图和基线状态从 DB 恢复 |
| API 端到端 | GET /aiops/graph 返回完整图、/trace 返回子图 |

---

## 14. 验证命令

```bash
# 构建验证
go build ./atlhyper_master_v2/...

# 单元测试
go test ./atlhyper_master_v2/aiops/... -v
go test ./atlhyper_master_v2/database/repo/ -run AIOps -v

# 集成测试（需要 SQLite）
go test ./atlhyper_master_v2/aiops/ -run Integration -v
```

---

## 15. 阶段实施后评审规范

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
| `aiops-phase2-risk-scorer.md` | `AIOpsEngine` 接口实际签名、`DependencyGraph` 和 `AnomalyResult` 的实际字段、`correlator` 和 `baseline` 的实际 API |
| `aiops-phase2-statemachine-incident.md` | `engine.go` 实际结构、`types.go` 实际类型定义、数据库 Repository 接口模式 |
| `aiops-phase3-frontend.md` | Phase 1 API 端点的实际响应格式 |
| `aiops-phase4-ai-enhancement.md` | `aiops/` 模块实际目录结构、`AIOpsEngine` 实际接口 |

### 评审检查清单

- [ ] 设计文档中引用的接口签名与实际代码一致
- [ ] 设计文档中的文件路径与实际目录结构一致
- [ ] 设计文档中的数据模型与实际 struct 定义一致
- [ ] 设计文档中的初始化链路与 `master.go` 实际代码一致
- [ ] 如有偏差，更新设计文档后再开始下一阶段实施
