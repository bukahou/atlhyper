package slo

import "math"

// SummarizeIngress 由 Ingress SLO 列表派生概览 —— 纯函数，不查库。
//
// 分档阈值：成功率 ≥ 99.9 健康、≥ 99.0 警告、其余严重。
// 2026-09-11 起 repository 与 snapshot service 共用此函数，Dashboard 刷新不再
// 为概览单独再查一次 ListIngressSLO（原来 GetSLOSummary 内部就是它）。
func SummarizeIngress(items []IngressSLO) *SLOSummary {
	s := &SLOSummary{TotalServices: len(items)}
	if len(items) == 0 {
		return s
	}

	var totalSuccRate, totalRPS, totalP99 float64
	for i := range items {
		it := &items[i]
		totalSuccRate += it.SuccessRate
		totalRPS += it.RPS
		totalP99 += it.P99Ms
		switch {
		case it.SuccessRate >= 99.9:
			s.HealthyServices++
		case it.SuccessRate >= 99.0:
			s.WarningServices++
		default:
			s.CriticalServices++
		}
	}

	n := float64(len(items))
	s.AvgSuccessRate = roundTo(totalSuccRate/n, 2)
	s.TotalRPS = roundTo(totalRPS, 2)
	s.AvgP99Ms = roundTo(totalP99/n, 2)
	return s
}

func roundTo(v float64, decimals int) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
