package metrics

import "testing"

func TestSummarizeNodes_Empty(t *testing.T) {
	s := SummarizeNodes(nil)
	if s == nil {
		t.Fatal("空列表也必须返回非 nil 概览 —— 前端拿 nil 会 404")
	}
	if s.TotalNodes != 0 || s.OnlineNodes != 0 || s.AvgCPUPct != 0 || s.MaxCPUTemp != 0 {
		t.Fatalf("空列表应全零，得到 %+v", *s)
	}
}

func TestSummarizeNodes_AvgMaxAndRounding(t *testing.T) {
	nodes := []NodeMetrics{
		{CPU: NodeCPU{UsagePct: 10.004}, Memory: NodeMemory{UsagePct: 50}, Temperature: NodeTemperature{CPUTempC: 61.26}},
		{CPU: NodeCPU{UsagePct: 30}, Memory: NodeMemory{UsagePct: 70.009}, Temperature: NodeTemperature{CPUTempC: 68.74}},
		{CPU: NodeCPU{UsagePct: 20}, Memory: NodeMemory{UsagePct: 60}, Temperature: NodeTemperature{CPUTempC: 0}},
	}
	s := SummarizeNodes(nodes)

	tests := []struct {
		name string
		got  float64
		want float64
	}{
		{"AvgCPUPct", s.AvgCPUPct, 20.0}, // (10.004+30+20)/3 = 20.001 → 20.00
		{"AvgMemPct", s.AvgMemPct, 60.0}, // (50+70.009+60)/3 = 60.003 → 60.00
		{"MaxCPUPct", s.MaxCPUPct, 30.0},
		{"MaxMemPct", s.MaxMemPct, 70.01},  // 两位小数
		{"MaxCPUTemp", s.MaxCPUTemp, 68.7}, // 一位小数
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	if s.TotalNodes != 3 || s.OnlineNodes != 3 {
		t.Errorf("TotalNodes/OnlineNodes = %d/%d, want 3/3", s.TotalNodes, s.OnlineNodes)
	}
}
