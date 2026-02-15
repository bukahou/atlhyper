// atlhyper_master_v2/aiops/types.go
// AIOps 引擎共用类型定义
package aiops

import "time"

// ==================== 依赖图类型 ====================

// GraphNode 图节点
type GraphNode struct {
	Key       string            `json:"key"`                 // "default/service/api-server"
	Type      string            `json:"type"`                // "ingress" | "service" | "pod" | "node"
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Metadata  map[string]string `json:"metadata,omitempty"`
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
	Nodes     map[string]*GraphNode `json:"nodes"`
	Edges     []*GraphEdge          `json:"edges"`
	UpdatedAt time.Time             `json:"updatedAt"`

	// 内部索引（不序列化）
	adjacency map[string][]string `json:"-"` // 正向邻接表: from -> [to...]
	reverse   map[string][]string `json:"-"` // 反向邻接表: to -> [from...]
}

// NewDependencyGraph 创建空依赖图
func NewDependencyGraph(clusterID string) *DependencyGraph {
	return &DependencyGraph{
		ClusterID: clusterID,
		Nodes:     make(map[string]*GraphNode),
		Edges:     make([]*GraphEdge, 0),
		UpdatedAt: time.Now(),
		adjacency: make(map[string][]string),
		reverse:   make(map[string][]string),
	}
}

// AddNode 添加节点
func (g *DependencyGraph) AddNode(key, typ, namespace, name string, metadata map[string]string) {
	if _, exists := g.Nodes[key]; exists {
		return
	}
	g.Nodes[key] = &GraphNode{
		Key:       key,
		Type:      typ,
		Namespace: namespace,
		Name:      name,
		Metadata:  metadata,
	}
}

// AddEdge 添加边
func (g *DependencyGraph) AddEdge(from, to, typ string, weight float64) {
	g.Edges = append(g.Edges, &GraphEdge{
		From:   from,
		To:     to,
		Type:   typ,
		Weight: weight,
	})
}

// RebuildIndex 重建邻接表索引
func (g *DependencyGraph) RebuildIndex() {
	g.adjacency = make(map[string][]string)
	g.reverse = make(map[string][]string)
	for _, edge := range g.Edges {
		g.adjacency[edge.From] = append(g.adjacency[edge.From], edge.To)
		g.reverse[edge.To] = append(g.reverse[edge.To], edge.From)
	}
}

// Adjacency 返回正向邻接表（供 risk propagation 使用）
func (g *DependencyGraph) Adjacency() map[string][]string {
	return g.adjacency
}

// Reverse 返回反向邻接表（供 risk propagation 使用）
func (g *DependencyGraph) Reverse() map[string][]string {
	return g.reverse
}

// TraceResult 链路追踪结果
type TraceResult struct {
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`
	Depth int          `json:"depth"`
}

// DiffResult 图变更结果
type DiffResult struct {
	AddedNodes   []string
	RemovedNodes []string
	AddedEdges   []*GraphEdge
	RemovedEdges []*GraphEdge
}

// ==================== 基线类型 ====================

// BaselineState 基线状态（每个实体-指标对）
type BaselineState struct {
	EntityKey  string  `json:"entityKey"`
	MetricName string  `json:"metricName"`
	EMA        float64 `json:"ema"`
	Variance   float64 `json:"variance"`
	Count      int64   `json:"count"`
	UpdatedAt  int64   `json:"updatedAt"`
}

// AnomalyResult 异常检测结果
type AnomalyResult struct {
	EntityKey    string  `json:"entityKey"`
	MetricName   string  `json:"metricName"`
	CurrentValue float64 `json:"currentValue"`
	Baseline     float64 `json:"baseline"`
	Deviation    float64 `json:"deviation"`
	Score        float64 `json:"score"`
	IsAnomaly    bool    `json:"isAnomaly"`
	DetectedAt   int64   `json:"detectedAt"`
}

// EntityBaseline 实体基线汇总（API 响应）
type EntityBaseline struct {
	EntityKey string           `json:"entityKey"`
	States    []*BaselineState `json:"states"`
	Anomalies []*AnomalyResult `json:"anomalies"`
}

// MetricDataPoint 指标数据点
type MetricDataPoint struct {
	EntityKey  string
	MetricName string
	Value      float64
}

// ==================== 常量 ====================

const (
	ColdStartMinCount = 100   // 前 100 个数据点只学习不告警
	DefaultAlpha      = 0.033 // α = 2/(60+1), 窗口 60 个采样点
	AnomalyThreshold  = 3.0   // 3σ 规则
	SigmoidK          = 2.0   // sigmoid 斜率
)
