package aiops

import (
	"strings"
	"testing"
)

// 确定性异常：有 Detail 时用「类别: Detail」，人一眼能看出触发源
func TestDescribeAnomaly_DeterministicUsesDetail(t *testing.T) {
	got := DescribeAnomaly(&AnomalyResult{MetricName: "critical_event", Detail: "Unhealthy: Readiness probe failed"})
	if got != "K8s 事件 Unhealthy: Readiness probe failed" {
		t.Errorf("got %q", got)
	}
	got = DescribeAnomaly(&AnomalyResult{MetricName: "some_new_metric", Detail: "x"})
	if got != "some_new_metric: x" {
		t.Errorf("未登记的指标应回退为指标名，got %q", got)
	}
}

// 统计异常：无 Detail，由当前值 / 基线 / 偏离 σ 渲染
func TestDescribeAnomaly_StatisticalRendersValues(t *testing.T) {
	got := DescribeAnomaly(&AnomalyResult{MetricName: "cpu_usage", CurrentValue: 92.14, Baseline: 41.3, Deviation: 4.21})
	for _, want := range []string{"cpu_usage", "当前 92.14", "基线 41.30", "偏离 4.2σ"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q 应含 %q", got, want)
		}
	}
	// 偏离封顶（先验方差为 0）不写成天文数字
	got = DescribeAnomaly(&AnomalyResult{MetricName: "restart_count", CurrentValue: 6, Baseline: 5, Deviation: MaxDeviation})
	if !strings.Contains(got, "从未波动") {
		t.Errorf("封顶偏离应写成「从未波动」，got %q", got)
	}
}
