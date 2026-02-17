// atlhyper_master_v2/aiops/baseline/detector.go
// EMA + 3σ 异常检测算法
package baseline

import (
	"math"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Detect 对单个指标执行异常检测
// 返回更新后的状态和异常结果（冷启动期间 result 为 nil）
func Detect(state *aiops.BaselineState, value float64, now int64) (*aiops.BaselineState, *aiops.AnomalyResult) {
	state.Count++
	alpha := aiops.DefaultAlpha

	// 冷启动：只学习，不告警
	if state.Count <= aiops.ColdStartMinCount {
		// 零值计数器快速通道：
		// restart_count / not_ready_containers 等指标正常值恒为 0，
		// 连续 10 个零值建立基线后，首个非零值立即触发正常检测
		// 注意：先检查快速通道条件，再更新 ConsecutiveZero
		if state.ConsecutiveZero >= int64(aiops.ColdStartZeroFastTrack) && value > 0 {
			// 快速通道：基线已确定为 0，跳到正常检测（不在此处更新 EMA，由下方统一处理）
			state.Count = int64(aiops.ColdStartMinCount) + 1
		} else {
			if value == 0 {
				state.ConsecutiveZero++
			} else {
				state.ConsecutiveZero = 0
			}
			// 标准冷启动：更新 EMA 但不产出结果
			if state.Count == 1 {
				state.EMA = value
				state.Variance = 0
			} else {
				state.EMA = alpha*value + (1-alpha)*state.EMA
				diff := value - state.EMA
				state.Variance = alpha*diff*diff + (1-alpha)*state.Variance
			}
			state.UpdatedAt = now
			return state, nil
		}
	}

	// 正常检测
	oldEMA := state.EMA
	state.EMA = alpha*value + (1-alpha)*state.EMA
	diff := value - oldEMA
	state.Variance = alpha*diff*diff + (1-alpha)*state.Variance
	state.UpdatedAt = now

	// 计算偏离度
	sigma := math.Sqrt(state.Variance)
	var deviation float64
	if sigma > 1e-9 {
		deviation = math.Abs(value-state.EMA) / sigma
	}

	// 归一化到 [0, 1]
	score := sigmoid(deviation, aiops.AnomalyThreshold, aiops.SigmoidK)

	result := &aiops.AnomalyResult{
		EntityKey:    state.EntityKey,
		MetricName:   state.MetricName,
		CurrentValue: value,
		Baseline:     state.EMA,
		Deviation:    deviation,
		Score:        score,
		IsAnomaly:    deviation > aiops.AnomalyThreshold,
		DetectedAt:   now,
	}

	return state, result
}

// sigmoid 归一化函数
// score = 1 / (1 + exp(-k * (deviation - threshold)))
func sigmoid(deviation, threshold, k float64) float64 {
	return 1.0 / (1.0 + math.Exp(-k*(deviation-threshold)))
}
