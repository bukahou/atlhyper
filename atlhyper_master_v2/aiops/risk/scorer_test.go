// atlhyper_master_v2/aiops/risk/scorer_test.go
// 风险评分引擎测试
package risk

import (
	"math"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// ==================== Stage 1: 局部风险测试 ====================

func TestComputeLocalRisks_NoAnomalies(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: false, Score: 0.5},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	if len(risks) != 0 {
		t.Errorf("expected no risks, got %d", len(risks))
	}
}

func TestComputeLocalRisks_SingleAnomaly(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.8},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	expected := 0.20 * 0.8 // pod restart_count weight=0.20, score=0.8
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

func TestComputeLocalRisks_MultipleMetrics(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/service/api", MetricName: "error_rate", IsAnomaly: true, Score: 1.0},
		{EntityKey: "default/service/api", MetricName: "avg_latency", IsAnomaly: true, Score: 0.5},
		{EntityKey: "default/service/api", MetricName: "request_rate", IsAnomaly: false, Score: 0.3},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// error_rate: 0.40 × 1.0 = 0.40
	// avg_latency: 0.30 × 0.5 = 0.15
	// request_rate: not anomaly, skip
	expected := 0.55
	if diff := math.Abs(risks["default/service/api"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/service/api"])
	}
}

func TestComputeLocalRisks_Clamped(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/service/api", MetricName: "error_rate", IsAnomaly: true, Score: 1.0},
		{EntityKey: "default/service/api", MetricName: "avg_latency", IsAnomaly: true, Score: 1.0},
		{EntityKey: "default/service/api", MetricName: "request_rate", IsAnomaly: true, Score: 1.0},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// 0.40 + 0.30 + 0.20 = 0.90 (within [0,1])
	expected := 0.90
	if diff := math.Abs(risks["default/service/api"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/service/api"])
	}
}

func TestComputeLocalRisks_UnknownMetric(t *testing.T) {
	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "unknown_metric", IsAnomaly: true, Score: 1.0},
	}
	config := DefaultRiskConfig()

	risks := ComputeLocalRisks(anomalies, config)
	// 未配置指标默认权重 0.1
	expected := 0.1
	if diff := math.Abs(risks["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, risks["default/pod/api-1"])
	}
}

// ==================== Stage 2: 时序权重测试 ====================

func TestApplyTemporalWeights_NoHistory(t *testing.T) {
	localRisks := map[string]float64{
		"default/pod/api-1": 0.5,
	}
	firstAnomalyTimes := map[string]int64{}

	weighted := ApplyTemporalWeights(localRisks, firstAnomalyTimes, time.Now().Unix(), 300)

	// 无历史记录时 wTime=1.0
	if diff := math.Abs(weighted["default/pod/api-1"] - 0.5); diff > 0.001 {
		t.Errorf("expected 0.5, got %.3f", weighted["default/pod/api-1"])
	}
}

func TestApplyTemporalWeights_RecentAnomaly(t *testing.T) {
	now := time.Now().Unix()
	localRisks := map[string]float64{
		"default/pod/api-1": 0.5,
	}
	firstAnomalyTimes := map[string]int64{
		"default/pod/api-1": now, // 刚刚出现
	}

	weighted := ApplyTemporalWeights(localRisks, firstAnomalyTimes, now, 300)

	// Δt=0, exp(0)=1.0, so weighted = 0.5
	if diff := math.Abs(weighted["default/pod/api-1"] - 0.5); diff > 0.001 {
		t.Errorf("expected 0.5, got %.3f", weighted["default/pod/api-1"])
	}
}

func TestApplyTemporalWeights_OldAnomaly(t *testing.T) {
	now := time.Now().Unix()
	localRisks := map[string]float64{
		"default/pod/api-1": 0.5,
	}
	firstAnomalyTimes := map[string]int64{
		"default/pod/api-1": now - 600, // 10 分钟前
	}

	weighted := ApplyTemporalWeights(localRisks, firstAnomalyTimes, now, 300)

	// Δt=600, τ=300, W = exp(-600/300) = exp(-2) ≈ 0.1353
	expectedW := math.Exp(-2.0)
	expected := 0.5 * expectedW
	if diff := math.Abs(weighted["default/pod/api-1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, weighted["default/pod/api-1"])
	}
}

// ==================== Stage 3: 图传播测试 ====================

func TestPropagate_NoEdges(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("default/node/node1", "node", "", "node1", nil)

	weightedRisks := map[string]float64{
		"default/node/node1": 0.8,
	}

	finalRisks, paths := Propagate(graph, weightedRisks, 0.6)

	// 无依赖: R_final = α × R_weighted = 0.6 × 0.8 = 0.48
	expected := 0.6 * 0.8
	if diff := math.Abs(finalRisks["default/node/node1"] - expected); diff > 0.001 {
		t.Errorf("expected %.3f, got %.3f", expected, finalRisks["default/node/node1"])
	}
	if len(paths) != 0 {
		t.Errorf("expected no paths, got %d", len(paths))
	}
}

func TestPropagate_SimpleChain(t *testing.T) {
	// Pod → Node (pod runs_on node)
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("_cluster/node/node1", "node", "_cluster", "node1", nil)
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.AddEdge("default/pod/api-1", "_cluster/node/node1", "runs_on", 1.0)
	graph.RebuildIndex()

	weightedRisks := map[string]float64{
		"_cluster/node/node1":  0.8,
		"default/pod/api-1": 0.3,
	}

	finalRisks, _ := Propagate(graph, weightedRisks, 0.6)

	// Node (layer=0) 先计算: 无上游依赖
	// R_final(node) = α × R_weighted = 0.6 × 0.8 = 0.48
	nodeExpected := 0.6 * 0.8
	if diff := math.Abs(finalRisks["_cluster/node/node1"] - nodeExpected); diff > 0.001 {
		t.Errorf("node: expected %.3f, got %.3f", nodeExpected, finalRisks["_cluster/node/node1"])
	}

	// Pod (layer=1) 后计算: 有下游依赖 node
	// R_final(pod) = α × R_weighted(pod) + (1-α) × R_final(node)
	//              = 0.6 × 0.3 + 0.4 × 0.48 = 0.18 + 0.192 = 0.372
	podExpected := 0.6*0.3 + 0.4*nodeExpected
	if diff := math.Abs(finalRisks["default/pod/api-1"] - podExpected); diff > 0.001 {
		t.Errorf("pod: expected %.3f, got %.3f", podExpected, finalRisks["default/pod/api-1"])
	}
}

func TestPropagate_PropagationPaths(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("_cluster/node/node1", "node", "_cluster", "node1", nil)
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.AddEdge("default/pod/api-1", "_cluster/node/node1", "runs_on", 1.0)
	graph.RebuildIndex()

	weightedRisks := map[string]float64{
		"_cluster/node/node1":  0.8,
		"default/pod/api-1": 0.3,
	}

	_, paths := Propagate(graph, weightedRisks, 0.6)

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if paths[0].From != "_cluster/node/node1" || paths[0].To != "default/pod/api-1" {
		t.Errorf("unexpected path: %s → %s", paths[0].From, paths[0].To)
	}
}

// ==================== ClusterRisk 聚合测试 ====================

func TestAggregate_BasicClusterRisk(t *testing.T) {
	entityRisks := map[string]*aiops.EntityRisk{
		"default/service/api": {
			EntityKey: "default/service/api",
			RFinal:    0.8,
		},
		"default/pod/api-1": {
			EntityKey: "default/pod/api-1",
			RFinal:    0.3,
		},
	}
	finalRisks := map[string]float64{
		"default/service/api": 0.8,
		"default/pod/api-1":   0.3,
	}

	config := DefaultRiskConfig()
	now := time.Now().Unix()

	clusterRisk := Aggregate("test", entityRisks, finalRisks, nil, config, now)

	// Risk = w1 × max(R_final) × 100 = 0.5 × 0.8 × 100 = 40
	// (no SLO context, so SLO and growth factors are 0)
	expectedRisk := 40.0
	if diff := math.Abs(clusterRisk.Risk - expectedRisk); diff > 0.2 {
		t.Errorf("expected risk %.1f, got %.1f", expectedRisk, clusterRisk.Risk)
	}
	if clusterRisk.Level != "low" {
		t.Errorf("expected level 'low', got '%s'", clusterRisk.Level)
	}
	if clusterRisk.AnomalyCount != 2 {
		t.Errorf("expected 2 anomalies, got %d", clusterRisk.AnomalyCount)
	}
}

func TestAggregate_WithSLOContext(t *testing.T) {
	entityRisks := map[string]*aiops.EntityRisk{
		"default/service/api": {
			EntityKey: "default/service/api",
			RFinal:    0.6,
		},
	}
	finalRisks := map[string]float64{
		"default/service/api": 0.6,
	}

	sloCtx := &SLOContext{
		MaxBurnRate:     2.5,
		ErrorGrowthRate: 0.8,
	}
	config := DefaultRiskConfig()
	now := time.Now().Unix()

	clusterRisk := Aggregate("test", entityRisks, finalRisks, sloCtx, config, now)

	// Risk = 0.5×0.6×100 + 0.3×1.0×100 + 0.2×sigmoid(0.8)×100
	// = 30 + 30 + 0.2×sigmoid×100
	// BurnRate>=2.0 → sloBurnFactor=1.0
	// sigmoid(-2*(0.8-0.5)) = sigmoid(-0.6) ≈ 0.354
	if clusterRisk.Risk < 55 || clusterRisk.Risk > 75 {
		t.Errorf("expected risk in [55, 75], got %.1f", clusterRisk.Risk)
	}
}

// ==================== Scorer 端到端测试 ====================

func TestScorer_Calculate_E2E(t *testing.T) {
	// 构建图: Node → Pod → Service
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("_cluster/node/node1", "node", "_cluster", "node1", nil)
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.AddNode("default/service/api", "service", "default", "api", nil)
	graph.AddEdge("default/pod/api-1", "_cluster/node/node1", "runs_on", 1.0)
	graph.AddEdge("default/service/api", "default/pod/api-1", "selects", 1.0)
	graph.RebuildIndex()

	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "_cluster/node/node1", MetricName: "cpu_usage", IsAnomaly: true, Score: 0.9},
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.5},
		{EntityKey: "default/service/api", MetricName: "error_rate", IsAnomaly: true, Score: 0.7},
	}

	scorer := NewScorer(nil)
	clusterRisk := scorer.Calculate("test", graph, anomalies, nil)

	if clusterRisk == nil {
		t.Fatal("expected non-nil cluster risk")
	}

	// 验证基本属性
	if clusterRisk.ClusterID != "test" {
		t.Errorf("expected cluster ID 'test', got '%s'", clusterRisk.ClusterID)
	}
	if clusterRisk.Risk < 0 || clusterRisk.Risk > 100 {
		t.Errorf("cluster risk out of range: %.1f", clusterRisk.Risk)
	}
	if clusterRisk.TotalEntities != 3 {
		t.Errorf("expected 3 entities, got %d", clusterRisk.TotalEntities)
	}

	// 验证实体风险
	entities := scorer.GetEntityRisks("test", "r_final", 10)
	if len(entities) != 3 {
		t.Fatalf("expected 3 entity risks, got %d", len(entities))
	}

	// 第一个应该是风险最高的
	if entities[0].RFinal < entities[1].RFinal {
		t.Error("entities not sorted by r_final descending")
	}

	// 所有风险值应该在 [0, 1]
	for _, e := range entities {
		if e.RFinal < 0 || e.RFinal > 1 {
			t.Errorf("entity %s r_final out of range: %.3f", e.EntityKey, e.RFinal)
		}
		if e.RLocal < 0 || e.RLocal > 1 {
			t.Errorf("entity %s r_local out of range: %.3f", e.EntityKey, e.RLocal)
		}
	}
}

func TestScorer_GetEntityRisk(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.RebuildIndex()

	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.8},
	}

	scorer := NewScorer(nil)
	scorer.Calculate("test", graph, anomalies, nil)

	risk := scorer.GetEntityRisk("test", "default/pod/api-1")
	if risk == nil {
		t.Fatal("expected non-nil entity risk")
	}
	if risk.EntityType != "pod" {
		t.Errorf("expected type 'pod', got '%s'", risk.EntityType)
	}
	if risk.RLocal <= 0 {
		t.Error("expected positive r_local")
	}
}

func TestScorer_NonExistentCluster(t *testing.T) {
	scorer := NewScorer(nil)

	if r := scorer.GetClusterRisk("unknown"); r != nil {
		t.Error("expected nil for unknown cluster")
	}
	if r := scorer.GetEntityRisks("unknown", "r_final", 10); r != nil {
		t.Error("expected nil for unknown cluster")
	}
}

func TestScorer_UpdateFirstAnomalyTimes(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("default/pod/api-1", "pod", "default", "api-1", nil)
	graph.RebuildIndex()

	scorer := NewScorer(nil)

	// 第一次: 有异常
	anomalies1 := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.5},
	}
	scorer.Calculate("test", graph, anomalies1, nil)

	// 获取首次异常时间
	entityRisk1 := scorer.GetEntityRisk("test", "default/pod/api-1")
	if entityRisk1 == nil || entityRisk1.FirstAnomaly == 0 {
		t.Fatal("expected first anomaly time to be set")
	}
	firstTime := entityRisk1.FirstAnomaly

	// 第二次: 仍然异常，首次时间不变
	anomalies2 := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: true, Score: 0.6},
	}
	scorer.Calculate("test", graph, anomalies2, nil)
	entityRisk2 := scorer.GetEntityRisk("test", "default/pod/api-1")
	if entityRisk2.FirstAnomaly != firstTime {
		t.Error("first anomaly time should not change")
	}

	// 第三次: 恢复正常
	anomalies3 := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/api-1", MetricName: "restart_count", IsAnomaly: false, Score: 0.0},
	}
	scorer.Calculate("test", graph, anomalies3, nil)
	entityRisk3 := scorer.GetEntityRisk("test", "default/pod/api-1")
	if entityRisk3 != nil && entityRisk3.FirstAnomaly != 0 {
		t.Error("first anomaly time should be cleared after recovery")
	}
}

// ==================== RiskLevel 映射测试 ====================

func TestRiskLevel(t *testing.T) {
	tests := []struct {
		rFinal   float64
		expected string
	}{
		{0.0, "healthy"},
		{0.1, "healthy"},
		{0.2, "low"},
		{0.4, "medium"},
		{0.6, "high"},
		{0.8, "critical"},
		{1.0, "critical"},
	}

	for _, tt := range tests {
		got := aiops.RiskLevel(tt.rFinal)
		if got != tt.expected {
			t.Errorf("RiskLevel(%.1f) = %s, want %s", tt.rFinal, got, tt.expected)
		}
	}
}

func TestClusterRiskLevel(t *testing.T) {
	tests := []struct {
		risk     float64
		expected string
	}{
		{0, "healthy"},
		{10, "healthy"},
		{20, "low"},
		{50, "warning"},
		{80, "critical"},
		{100, "critical"},
	}

	for _, tt := range tests {
		got := aiops.ClusterRiskLevel(tt.risk)
		if got != tt.expected {
			t.Errorf("ClusterRiskLevel(%.0f) = %s, want %s", tt.risk, got, tt.expected)
		}
	}
}
