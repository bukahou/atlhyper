// atlhyper_master_v2/aiops/risk/temporal.go
// Stage 2: 时序权重
// W_time(entity) = exp(-Δt / τ)
package risk

import "math"

// TemporalFloor 时序权重下限（首次检测到异常时的初始权重）
// 0.7 确保 RLocal=0.315 的确定性异常首次即可跨过 0.2 状态机门槛
const TemporalFloor = 0.7

// ApplyTemporalWeights 应用时序权重
// 效果: 持续异常的实体权重递增（确认是持续问题而非瞬态波动）
// W_time = floor + (1-floor) × (1 - exp(-Δt / τ))
//   - 首次检测 (Δt=0): W_time = floor (0.5)
//   - 持续 5 分钟:       W_time ≈ 0.82
//   - 持续 10 分钟:      W_time ≈ 0.93
func ApplyTemporalWeights(
	localRisks map[string]float64,
	firstAnomalyTimes map[string]int64,
	now int64,
	halfLife float64,
) map[string]float64 {
	weighted := make(map[string]float64, len(localRisks))

	for entityKey, rLocal := range localRisks {
		wTime := 1.0

		if firstTime, ok := firstAnomalyTimes[entityKey]; ok && firstTime > 0 {
			deltaT := float64(now - firstTime)
			if deltaT > 0 && halfLife > 0 {
				// 增长公式：持续越久 → 权重越高
				wTime = TemporalFloor + (1-TemporalFloor)*(1-math.Exp(-deltaT/halfLife))
			} else {
				wTime = TemporalFloor
			}
		}

		weighted[entityKey] = rLocal * wTime
	}

	return weighted
}
