// atlhyper_master_v2/aiops/risk/temporal.go
// Stage 2: 时序权重
// W_time(entity) = exp(-Δt / τ)
package risk

import "math"

// ApplyTemporalWeights 应用时序权重
// 效果: 先出问题的实体权重更高（更可能是根因）
func ApplyTemporalWeights(
	localRisks map[string]float64,
	firstAnomalyTimes map[string]int64,
	now int64,
	halfLife float64,
) map[string]float64 {
	weighted := make(map[string]float64, len(localRisks))

	for entityKey, rLocal := range localRisks {
		wTime := 1.0 // 默认无衰减

		if firstTime, ok := firstAnomalyTimes[entityKey]; ok && firstTime > 0 {
			deltaT := float64(now - firstTime)
			if deltaT > 0 && halfLife > 0 {
				wTime = math.Exp(-deltaT / halfLife)
			}
		}

		weighted[entityKey] = rLocal * wTime
	}

	return weighted
}
