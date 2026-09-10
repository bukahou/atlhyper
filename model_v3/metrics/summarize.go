package metrics

import "math"

// SummarizeNodes 由节点列表派生集群概览 —— 纯函数，不查库。
//
// 2026-09-11 ClickHouse 负载排查：Agent 的 Dashboard 刷新曾同时调用
// GetMetricsSummary 与 ListAllNodeMetrics，而前者内部就是后者再聚合，
// 于是每个周期 8 节点 × 17 条 SQL 白跑一遍（占单 agent 查询量 48%）。
// 聚合逻辑放到模型层后，repository 与 service 共用同一份，谁拿到列表谁就能算概览。
func SummarizeNodes(nodes []NodeMetrics) *Summary {
	s := &Summary{
		TotalNodes:  len(nodes),
		OnlineNodes: len(nodes),
	}
	if len(nodes) == 0 {
		return s
	}

	var sumCPU, sumMem, maxCPU, maxMem, maxTemp float64
	for i := range nodes {
		nm := &nodes[i]
		sumCPU += nm.CPU.UsagePct
		sumMem += nm.Memory.UsagePct
		if nm.CPU.UsagePct > maxCPU {
			maxCPU = nm.CPU.UsagePct
		}
		if nm.Memory.UsagePct > maxMem {
			maxMem = nm.Memory.UsagePct
		}
		if nm.Temperature.CPUTempC > maxTemp {
			maxTemp = nm.Temperature.CPUTempC
		}
	}

	n := float64(len(nodes))
	s.AvgCPUPct = roundTo(sumCPU/n, 2)
	s.AvgMemPct = roundTo(sumMem/n, 2)
	s.MaxCPUPct = roundTo(maxCPU, 2)
	s.MaxMemPct = roundTo(maxMem, 2)
	s.MaxCPUTemp = roundTo(maxTemp, 1)
	return s
}

// roundTo 四舍五入到指定小数位（NaN/Inf → 0，与 agent 查询层的同名函数一致）
func roundTo(v float64, decimals int) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
