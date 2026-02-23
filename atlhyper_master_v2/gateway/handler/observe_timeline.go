// atlhyper_master_v2/gateway/handler/observe_timeline.go
// OTel 时间线辅助函数（从内存时间线 / 预聚合时序构建数据）
package handler

import (
	"time"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
)

// buildNodeMetricsSeries 从 OTel 时间线提取单节点时序
func buildNodeMetricsSeries(entries []cluster.OTelEntry, nodeName, metric string) *metrics.Series {
	series := &metrics.Series{
		Metric: metric,
		Points: make([]metrics.Point, 0, len(entries)),
	}

	for _, e := range entries {
		if e.Snapshot == nil || e.Snapshot.MetricsNodes == nil {
			continue
		}
		for _, node := range e.Snapshot.MetricsNodes {
			if node.NodeName == nodeName {
				value := extractMetricValue(&node, metric)
				if value >= 0 {
					series.Points = append(series.Points, metrics.Point{
						Timestamp: e.Timestamp,
						Value:     value,
					})
				}
				break
			}
		}
	}
	return series
}

// extractMetricValue 从 NodeMetrics 中提取指定指标值
func extractMetricValue(node *metrics.NodeMetrics, metric string) float64 {
	switch metric {
	case "cpu_usage", "cpu":
		return node.CPU.UsagePct
	case "memory_usage", "memory":
		return node.Memory.UsagePct
	case "cpu_load1":
		return node.CPU.Load1
	case "cpu_load5":
		return node.CPU.Load5
	case "cpu_load15":
		return node.CPU.Load15
	default:
		// 默认返回 CPU 使用率
		return node.CPU.UsagePct
	}
}

// sloPoint SLO 时序数据点
type sloPoint struct {
	Timestamp interface{} `json:"timestamp"`
	RPS       float64     `json:"rps"`
	SuccRate  float64     `json:"successRate"`
	P99Ms     float64     `json:"p99Ms"`
}

// buildSLOTimeSeries 从 OTel 时间线构建 SLO 时序
// 返回指定服务的 SLO 指标随时间变化（请求量、成功率、延迟）
func buildSLOTimeSeries(entries []cluster.OTelEntry, serviceName string) map[string]interface{} {
	points := make([]sloPoint, 0, len(entries))
	for _, e := range entries {
		if e.Snapshot == nil {
			continue
		}
		// 在 SLO Ingress 或 SLO Services 中查找
		found := false
		if e.Snapshot.SLOIngress != nil {
			for _, svc := range e.Snapshot.SLOIngress {
				if svc.ServiceKey == serviceName || svc.DisplayName == serviceName {
					points = append(points, sloPoint{
						Timestamp: e.Timestamp,
						RPS:       svc.RPS,
						SuccRate:  svc.SuccessRate,
						P99Ms:     svc.P99Ms,
					})
					found = true
					break
				}
			}
		}
		if !found && e.Snapshot.SLOServices != nil {
			for _, svc := range e.Snapshot.SLOServices {
				if svc.Name == serviceName {
					points = append(points, sloPoint{
						Timestamp: e.Timestamp,
						RPS:       svc.RPS,
						SuccRate:  svc.SuccessRate,
						P99Ms:     svc.P99Ms,
					})
					break
				}
			}
		}
	}

	return map[string]interface{}{
		"service": serviceName,
		"points":  points,
	}
}

// ================================================================
// 预聚合时序辅助函数
// ================================================================

// filterNodePointsByMinutes 按时间范围裁剪节点时序数据点
func filterNodePointsByMinutes(points []cluster.NodeMetricsPoint, minutes int) []cluster.NodeMetricsPoint {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	result := make([]cluster.NodeMetricsPoint, 0, len(points))
	for _, p := range points {
		if !p.Timestamp.Before(cutoff) {
			result = append(result, p)
		}
	}
	return result
}

// extractNodeMetricPoints 从预聚合节点时序中提取指定指标
func extractNodeMetricPoints(points []cluster.NodeMetricsPoint, metric string) []metrics.Point {
	result := make([]metrics.Point, 0, len(points))
	for _, p := range points {
		var value float64
		switch metric {
		case "cpu_usage", "cpu":
			value = p.CPUPct
		case "memory_usage", "memory":
			value = p.MemPct
		case "disk_usage", "disk":
			value = p.DiskPct
		case "net_rx":
			value = p.NetRxBps
		case "net_tx":
			value = p.NetTxBps
		case "cpu_load1":
			value = p.Load1
		default:
			value = p.CPUPct
		}
		result = append(result, metrics.Point{
			Timestamp: p.Timestamp,
			Value:     value,
		})
	}
	return result
}

// filterSLOPointsByMinutes 按时间范围裁剪 SLO 时序数据点
func filterSLOPointsByMinutes(points []cluster.SLOTimePoint, minutes int) []cluster.SLOTimePoint {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	result := make([]cluster.SLOTimePoint, 0, len(points))
	for _, p := range points {
		if !p.Timestamp.Before(cutoff) {
			result = append(result, p)
		}
	}
	return result
}
