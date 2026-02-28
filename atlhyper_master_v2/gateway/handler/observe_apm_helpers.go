// atlhyper_master_v2/gateway/handler/observe_apm_helpers.go
// APM 时序辅助函数（从 observe_timeline.go 拆分）
package handler

import (
	"time"

	"AtlHyper/model_v3/cluster"
)

// filterAPMPointsByMinutes 按时间范围裁剪 APM 时序数据点
func filterAPMPointsByMinutes(points []cluster.APMTimePoint, minutes int) []cluster.APMTimePoint {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	result := make([]cluster.APMTimePoint, 0, len(points))
	for _, p := range points {
		if !p.Timestamp.Before(cutoff) {
			result = append(result, p)
		}
	}
	return result
}
