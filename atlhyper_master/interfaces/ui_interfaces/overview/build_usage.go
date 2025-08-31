package overview

import (
	"AtlHyper/model/node"
)

// buildNodeUsagesAndTotals
// -----------------------------------------------------------------------------
// 一次循环完成：
// 1. 每个节点的 CPU/内存使用率 (NodeUsageRow)
// 2. 整体 CPU 使用率 (UsageCard)
// 3. 整体内存使用率 (UsageCard)
func buildNodeUsagesAndTotals(nodes []node.Node) ([]NodeUsageRow, UsageCard, UsageCard) {
	out := make([]NodeUsageRow, 0, len(nodes))

	var totalCPUUsed, totalCPUTotal float64
	var totalMemUsed, totalMemTotal float64

	for _, n := range nodes {
		// ---- CPU ----
		cu := parseMilliCPU(n.Metrics.CPU.Usage)
		ct := parseCPU(n.Metrics.CPU.Capacity)
		if cu > 0 && ct > 0 {
			totalCPUUsed += cu
			totalCPUTotal += ct
		}
		cpuPct := 0.0
		if ct > 0 {
			cpuPct = cu / ct * 100
		} else if n.Metrics.CPU.UtilPct > 0 {
			// 容错：如果 capacity 解析失败，就用 utilPct
			cpuPct = n.Metrics.CPU.UtilPct
		}

		// ---- Memory ----
		mu := parseKiToBytes(n.Metrics.Memory.Usage)
		mt := parseKiToBytes(n.Metrics.Memory.Capacity)
		if mu > 0 && mt > 0 {
			totalMemUsed += mu
			totalMemTotal += mt
		}
		memPct := 0.0
		if mt > 0 {
			memPct = mu / mt * 100
		} else if n.Metrics.Memory.UtilPct > 0 {
			memPct = n.Metrics.Memory.UtilPct
		}

		// ---- append row ----
		out = append(out, NodeUsageRow{
			Node:     n.Summary.Name,
			CPUUsage: cpuPct,
			MemUsage: memPct,
		})
	}

	// ---- totals ----
	cpuTotalPct := 0.0
	memTotalPct := 0.0
	if totalCPUTotal > 0 {
		cpuTotalPct = totalCPUUsed / totalCPUTotal * 100
	}
	if totalMemTotal > 0 {
		memTotalPct = totalMemUsed / totalMemTotal * 100
	}

	return out, UsageCard{Percent: cpuTotalPct}, UsageCard{Percent: memTotalPct}
}
