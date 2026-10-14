// atlhyper_master_v2/aiops/signal.go
// 异常 → 人可读信号文本（时间线 / 风险接口 / AI 上下文共用一份写法）
package aiops

import "fmt"

// signalCategory 确定性异常的类别名；未登记的指标回退为指标名
var signalCategory = map[string]string{
	"critical_event":       "K8s 事件",
	"container_anomaly":    "容器异常",
	"deployment_impact":    "Deployment 影响",
	"memory_pressure":      "节点压力",
	"disk_pressure":        "节点压力",
	"pid_pressure":         "节点压力",
	"apm_high_error_rate":  "APM 高错误率",
	"apm_high_p99_latency": "APM 高延迟",
	"log_error_spike":      "日志错误尖峰",
}

// DescribeAnomaly 把一条异常渲染为一行文本
func DescribeAnomaly(a *AnomalyResult) string {
	if a.Detail != "" {
		if cat, ok := signalCategory[a.MetricName]; ok {
			if a.MetricName == "critical_event" {
				return cat + " " + a.Detail // 「K8s 事件 Unhealthy: …」读起来比加冒号自然
			}
			return cat + ": " + a.Detail
		}
		return a.MetricName + ": " + a.Detail
	}
	if a.Deviation >= MaxDeviation {
		return fmt.Sprintf("%s 当前 %.2f，基线 %.2f（该指标此前从未波动）", a.MetricName, a.CurrentValue, a.Baseline)
	}
	return fmt.Sprintf("%s 当前 %.2f，基线 %.2f（偏离 %.1fσ）", a.MetricName, a.CurrentValue, a.Baseline, a.Deviation)
}
