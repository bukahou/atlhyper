package risk

import (
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// 每个实体挂上推高本地风险的信号：只取 IsAnomaly，按 Score 倒序，最多 3 条
func TestScorer_SignalsTop3(t *testing.T) {
	graph := aiops.NewDependencyGraph("test")
	graph.AddNode("default/pod/a", "pod", "default", "a", nil)
	graph.AddNode("default/pod/b", "pod", "default", "b", nil)
	graph.RebuildIndex()

	anomalies := []*aiops.AnomalyResult{
		{EntityKey: "default/pod/a", MetricName: "critical_event", IsAnomaly: true, Score: 0.85, Detail: "Unhealthy: probe"},
		{EntityKey: "default/pod/a", MetricName: "restart_count", IsAnomaly: true, Score: 0.5, CurrentValue: 6, Baseline: 5, Deviation: 3},
		{EntityKey: "default/pod/a", MetricName: "container_anomaly", IsAnomaly: true, Score: 0.6, Detail: "容器 app: NotReady"},
		{EntityKey: "default/pod/a", MetricName: "deployment_impact", IsAnomaly: true, Score: 0.4, Detail: "Deployment default/x 不可用 50%"},
		{EntityKey: "default/pod/a", MetricName: "is_running", IsAnomaly: false, Score: 0.99},
	}

	s := NewScorer(nil)
	s.Calculate("test", graph, anomalies, nil)

	a := s.GetEntityRisk("test", "default/pod/a")
	if a == nil || len(a.Signals) != 3 {
		t.Fatalf("应有 3 条信号，got %+v", a)
	}
	wantOrder := []string{"critical_event", "container_anomaly", "restart_count"}
	for i, m := range wantOrder {
		if a.Signals[i].Metric != m {
			t.Errorf("第 %d 条应为 %s，got %s", i, m, a.Signals[i].Metric)
		}
	}
	if a.Signals[0].Detail != "K8s 事件 Unhealthy: probe" {
		t.Errorf("Detail 应为渲染后的文本，got %q", a.Signals[0].Detail)
	}
	if !strings.Contains(a.Signals[2].Detail, "基线") {
		t.Errorf("统计信号应渲染基线，got %q", a.Signals[2].Detail)
	}

	if b := s.GetEntityRisk("test", "default/pod/b"); b == nil || len(b.Signals) != 0 {
		t.Errorf("无异常实体不应有信号，got %+v", b)
	}
}
