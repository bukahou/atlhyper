package cluster

import "testing"

// 整体结论、原因代码、两条进度条的结论都出自同一份阈值
func TestClusterHealth(t *testing.T) {
	cases := []struct {
		name                  string
		nodePct, podPct       float64
		status, reason        string
		nodeStatus, podStatus string
	}{
		{"全部健康", 100, 95, HealthHealthy, "", HealthHealthy, HealthHealthy},
		{"Pod 85% 仍健康（Pod 的健康线是 80）", 100, 85, HealthHealthy, "", HealthHealthy, HealthHealthy},
		{"部分 Pod 未就绪", 100, 75, HealthDegraded, HealthReasonSomePodsNotReady, HealthHealthy, HealthDegraded},
		{"部分节点未就绪", 80, 95, HealthDegraded, HealthReasonSomeNodesNotReady, HealthDegraded, HealthHealthy},
		{"节点就绪率过低优先", 50, 50, HealthUnhealthy, HealthReasonNodeReadyLow, HealthUnhealthy, HealthUnhealthy},
		{"Pod 就绪率过低", 95, 40, HealthUnhealthy, HealthReasonPodReadyLow, HealthHealthy, HealthUnhealthy},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CalculateHealthStatus(c.nodePct, c.podPct); got != c.status {
				t.Errorf("status = %s, want %s", got, c.status)
			}
			if got := CalculateHealthReason(c.nodePct, c.podPct); got != c.reason {
				t.Errorf("reason = %q, want %q", got, c.reason)
			}
			if got := NodeReadyStatus(c.nodePct); got != c.nodeStatus {
				t.Errorf("nodeReadyStatus = %s, want %s", got, c.nodeStatus)
			}
			if got := PodReadyStatus(c.podPct); got != c.podStatus {
				t.Errorf("podReadyStatus = %s, want %s", got, c.podStatus)
			}
		})
	}
}
