// atlhyper_master_v2/aiops/baseline/detector_test.go
package baseline

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

func TestDetect_ColdStart(t *testing.T) {
	state := &aiops.BaselineState{}
	now := int64(1000)

	// 前 100 个数据点应返回 nil result（只学习不告警）
	for i := 0; i < aiops.ColdStartMinCount; i++ {
		state, result := Detect(state, 50.0, now+int64(i))
		if result != nil {
			t.Fatalf("冷启动期间应返回 nil result, got non-nil at count=%d", state.Count)
		}
	}

	if state.Count != int64(aiops.ColdStartMinCount) {
		t.Fatalf("冷启动后 count 应为 %d, got %d", aiops.ColdStartMinCount, state.Count)
	}
	if state.EMA == 0 {
		t.Fatal("冷启动后 EMA 不应为 0")
	}
}

func TestDetect_NormalValue(t *testing.T) {
	state := &aiops.BaselineState{
		EntityKey:  "default/pod/test",
		MetricName: "cpu_usage",
		EMA:        50.0,
		Variance:   25.0, // σ = 5
		Count:      int64(aiops.ColdStartMinCount),
	}

	// 正常值（在 3σ 内）: 50 ± 15
	_, result := Detect(state, 52.0, 2000)
	if result == nil {
		t.Fatal("冷启动后应返回 result")
	}
	if result.IsAnomaly {
		t.Fatalf("52.0 接近 EMA=50, 不应为异常, deviation=%.2f", result.Deviation)
	}
	if result.Score >= 0.5 {
		t.Fatalf("正常值 score 应 < 0.5, got %.4f", result.Score)
	}
}

func TestDetect_AnomalyValue(t *testing.T) {
	state := &aiops.BaselineState{
		EntityKey:  "default/pod/test",
		MetricName: "cpu_usage",
		EMA:        50.0,
		Variance:   4.0, // σ = 2
		Count:      int64(aiops.ColdStartMinCount),
	}

	// 异常值（偏离 > 3σ）: |90 - 50| / 2 = 20 > 3
	_, result := Detect(state, 90.0, 3000)
	if result == nil {
		t.Fatal("应返回 result")
	}
	if !result.IsAnomaly {
		t.Fatalf("90.0 远离 EMA=50 (σ=2), 应为异常, deviation=%.2f", result.Deviation)
	}
	if result.Score <= 0.5 {
		t.Fatalf("异常值 score 应 > 0.5, got %.4f", result.Score)
	}
}

func TestDetect_FirstSample(t *testing.T) {
	state := &aiops.BaselineState{}
	state, _ = Detect(state, 100.0, 1000)

	if state.EMA != 100.0 {
		t.Fatalf("第一个样本 EMA 应为 100.0, got %.2f", state.EMA)
	}
	if state.Variance != 0 {
		t.Fatalf("第一个样本 Variance 应为 0, got %.2f", state.Variance)
	}
	if state.Count != 1 {
		t.Fatalf("Count 应为 1, got %d", state.Count)
	}
}

func TestDetect_ColdStartZeroFastTrack(t *testing.T) {
	state := &aiops.BaselineState{
		EntityKey:  "default/pod/test",
		MetricName: "restart_count",
	}
	now := int64(1000)

	// 连续喂入 10 个零值（低于 ColdStartMinCount=100，仍在冷启动期间）
	for i := 0; i < aiops.ColdStartZeroFastTrack; i++ {
		state, result := Detect(state, 0.0, now+int64(i))
		if result != nil {
			t.Fatalf("零值冷启动期间应返回 nil result, count=%d", state.Count)
		}
	}

	if state.ConsecutiveZero != int64(aiops.ColdStartZeroFastTrack) {
		t.Fatalf("ConsecutiveZero 应为 %d, got %d", aiops.ColdStartZeroFastTrack, state.ConsecutiveZero)
	}

	// 第 11 个样本：非零值 → 应立即触发检测（跳过剩余冷启动）
	state, result := Detect(state, 5.0, now+11)
	if result == nil {
		t.Fatal("零值快速通道后的首个非零值应触发检测，返回 result")
	}
	if !result.IsAnomaly {
		t.Fatalf("基线为 0 的指标突然变为 5.0 应为异常, deviation=%.2f", result.Deviation)
	}
	if state.Count <= int64(aiops.ColdStartZeroFastTrack) {
		t.Fatalf("Count 应跳过冷启动, got %d", state.Count)
	}
}

func TestDetect_ColdStartZeroFastTrack_NotTriggeredByZero(t *testing.T) {
	state := &aiops.BaselineState{
		EntityKey:  "default/pod/test",
		MetricName: "not_ready_containers",
	}
	now := int64(1000)

	// 连续喂入 10 个零值
	for i := 0; i < aiops.ColdStartZeroFastTrack; i++ {
		state, _ = Detect(state, 0.0, now+int64(i))
	}

	// 第 11 个样本仍然是零值 → 不应触发快速通道（仍在冷启动）
	state, result := Detect(state, 0.0, now+11)
	if result != nil {
		t.Fatal("连续零值后又一个零值不应触发快速通道")
	}
}

func TestDetect_ColdStartZeroFastTrack_BrokenByNonZero(t *testing.T) {
	state := &aiops.BaselineState{
		EntityKey:  "default/pod/test",
		MetricName: "restart_count",
	}
	now := int64(1000)

	// 喂入 5 个零值
	for i := 0; i < 5; i++ {
		state, _ = Detect(state, 0.0, now+int64(i))
	}

	// 插入 1 个非零值 → 打断连续零值计数
	state, _ = Detect(state, 1.0, now+5)
	if state.ConsecutiveZero != 0 {
		t.Fatalf("非零值应重置 ConsecutiveZero, got %d", state.ConsecutiveZero)
	}

	// 再喂入 5 个零值 → 总共连续只有 5 个，不够 10 个
	for i := 0; i < 5; i++ {
		state, _ = Detect(state, 0.0, now+6+int64(i))
	}

	// 非零值 → 不应触发快速通道（因为连续零值不够 10）
	state, result := Detect(state, 3.0, now+11)
	if result != nil {
		t.Fatal("连续零值不足 10 个不应触发快速通道")
	}
}

func TestSigmoid(t *testing.T) {
	tests := []struct {
		name      string
		deviation float64
		wantHigh  bool // score > 0.5
	}{
		{"below_threshold", 1.0, false},
		{"at_threshold", 3.0, false}, // sigmoid(0) = 0.5, 精确阈值约为0.5
		{"above_threshold", 5.0, true},
		{"far_above", 10.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := sigmoid(tt.deviation, aiops.AnomalyThreshold, aiops.SigmoidK)
			if tt.wantHigh && score <= 0.5 {
				t.Errorf("deviation=%.1f: score=%.4f, want > 0.5", tt.deviation, score)
			}
			if !tt.wantHigh && score > 0.5 {
				t.Errorf("deviation=%.1f: score=%.4f, want <= 0.5", tt.deviation, score)
			}
		})
	}
}
