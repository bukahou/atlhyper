package slo

import "testing"

func TestSummarizeIngress_Empty(t *testing.T) {
	s := SummarizeIngress(nil)
	if s == nil {
		t.Fatal("空列表也必须返回非 nil 概览")
	}
	if s.TotalServices != 0 || s.AvgSuccessRate != 0 || s.TotalRPS != 0 || s.AvgP99Ms != 0 {
		t.Fatalf("空列表应全零，得到 %+v", *s)
	}
}

func TestSummarizeIngress_BucketsAndAverages(t *testing.T) {
	items := []IngressSLO{
		{SuccessRate: 99.95, RPS: 10, P99Ms: 100},        // healthy  (>= 99.9)
		{SuccessRate: 99.9, RPS: 20, P99Ms: 200},         // healthy  (边界含)
		{SuccessRate: 99.5, RPS: 30, P99Ms: 300},         // warning  (>= 99.0)
		{SuccessRate: 98.0, RPS: 40.004, P99Ms: 400.006}, // critical
	}
	s := SummarizeIngress(items)

	if s.TotalServices != 4 {
		t.Fatalf("TotalServices = %d, want 4", s.TotalServices)
	}
	if s.HealthyServices != 2 || s.WarningServices != 1 || s.CriticalServices != 1 {
		t.Errorf("buckets = %d/%d/%d, want 2/1/1", s.HealthyServices, s.WarningServices, s.CriticalServices)
	}
	if s.AvgSuccessRate != 99.34 { // (99.95+99.9+99.5+98)/4 = 99.3375 → 99.34
		t.Errorf("AvgSuccessRate = %v, want 99.34", s.AvgSuccessRate)
	}
	if s.TotalRPS != 100.0 { // 100.004 → 100.00
		t.Errorf("TotalRPS = %v, want 100", s.TotalRPS)
	}
	if s.AvgP99Ms != 250.0 { // 1000.006/4 = 250.0015 → 250.00
		t.Errorf("AvgP99Ms = %v, want 250", s.AvgP99Ms)
	}
}
