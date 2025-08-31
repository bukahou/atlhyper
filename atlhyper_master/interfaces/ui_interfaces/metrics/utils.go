package metrics

import (
	"math"
	"strings"

	mod "AtlHyper/model/metrics"
)

func pickDiskUsedPercent(disks []mod.DiskStat) float64 {
	var max float64
	for _, d := range disks {
		if d.Usage*100 > max {
			max = d.Usage * 100
		}
	}
	return round1(max)
}

func findEth0Tx(nics []mod.NetworkStat) float64 {
	for _, n := range nics {
		if strings.EqualFold(n.Interface, "eth0") {
			return round2(n.TxKBps)
		}
	}
	return 0
}
func findEth0Rx(nics []mod.NetworkStat) float64 {
	for _, n := range nics {
		if strings.EqualFold(n.Interface, "eth0") {
			return round2(n.RxKBps)
		}
	}
	return 0
}

func topCPUProcName(ps []mod.TopCPUProcess) string {
	if len(ps) == 0 {
		return ""
	}
	// 按 CPU% 最大
	idx := 0
	for i := 1; i < len(ps); i++ {
		if ps[i].CPUPercent > ps[idx].CPUPercent {
			idx = i
		}
	}
	return ps[idx].Command
}

func safeAvg(sum float64, n int) float64 {
	if n <= 0 {
		return 0
	}
	return round1(sum / float64(n))
}

func round1(x float64) float64 {
	return math.Round(x*10) / 10
}
func round2(x float64) float64 {
	return math.Round(x*100) / 100
}
