// atlhyper_master_v2/gateway/handler/observe_slo_helpers.go
// SLO 时序辅助函数（从 observe_timeline.go 拆分）
package handler

import (
	"time"

	"AtlHyper/model_v3/cluster"
)

// sloPoint SLO 时序数据点
type sloPoint struct {
	Timestamp interface{} `json:"timestamp"`
	RPS       float64     `json:"rps"`
	SuccRate  float64     `json:"successRate"`
	P50Ms     float64     `json:"p50Ms"`
	P99Ms     float64     `json:"p99Ms"`
	ErrorRate float64     `json:"errorRate"`
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
						P50Ms:     svc.P50Ms,
						P99Ms:     svc.P99Ms,
						ErrorRate: svc.ErrorRate,
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
						P50Ms:     svc.P50Ms,
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
