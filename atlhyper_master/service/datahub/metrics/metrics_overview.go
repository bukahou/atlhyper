package metrics

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/model/dto"
	"AtlHyper/atlhyper_master/repository"
	mod "AtlHyper/model/collect"
)

// BuildNodeMetricsOverview —— 取各节点最近一次快照，生成概览
func BuildNodeMetricsOverview(ctx context.Context, clusterID string) (*dto.MetricsOverviewDTO, error) {
	now := time.Now().UTC()
	since := now.Add(-2 * time.Minute)

	// 如果你有 “最新快照” 接口，优先用：
	// snaps, err := repository.GetClusterMetricsLatest(ctx, clusterID)
	// 下面是通用兜底：从近2分钟范围内取最后一条作为“最新”
	series, err := repository.Mem.GetClusterMetricsRange(ctx, clusterID, since, now)
	if err != nil {
		return nil, err
	}

	latestByNode := pickLatestByNode(series)

	rows := make([]dto.MetricsNodeRow, 0, len(latestByNode))
	var cpuSum, memSum float64
	var cpuCnt, memCnt int
	var peakTemp float64
	var peakTempNode string
	var peakDisk float64
	var peakDiskNode string

	for node, snap := range latestByNode {
		row := dto.MetricsNodeRow{
			Node:        node,
			CPUPercent:  snap.CPU.Usage * 100,      // 0~1 -> %
			MemPercent:  snap.Memory.Usage * 100,   // 0~1 -> %
			CPUTempC:    snap.Temperature.CPUDegrees,
			DiskUsedPct: pickDiskUsedPercent(snap.Disk),
			Eth0TxKBps:  findEth0Tx(snap.Network),
			Eth0RxKBps:  findEth0Rx(snap.Network),
			TopCPUProc:  topCPUProcName(snap.TopCPUProcesses),
			Timestamp:   snap.Timestamp,
		}
		rows = append(rows, row)

		// 汇总
		if row.CPUPercent > 0 {
			cpuSum += row.CPUPercent
			cpuCnt++
		}
		if row.MemPercent > 0 {
			memSum += row.MemPercent
			memCnt++
		}
		if row.CPUTempC > peakTemp {
			peakTemp = row.CPUTempC
			peakTempNode = node
		}
		if row.DiskUsedPct > peakDisk {
			peakDisk = row.DiskUsedPct
			peakDiskNode = node
		}
	}

	dto := &dto.MetricsOverviewDTO{
		Cards: dto.MetricsOverviewCards{
			AvgCPUPercent:   safeAvg(cpuSum, cpuCnt),
			AvgMemPercent:   safeAvg(memSum, memCnt),
			PeakTempC:       round1(peakTemp),
			PeakTempNode:    peakTempNode,
			PeakDiskPercent: round1(peakDisk),
			PeakDiskNode:    peakDiskNode,
		},
		Rows: rows,
	}
	return dto, nil
}

// 取最近一条快照
func pickLatestByNode(snaps []mod.NodeMetricsSnapshot) map[string]mod.NodeMetricsSnapshot {
	out := make(map[string]mod.NodeMetricsSnapshot, 16)
	for _, s := range snaps {
		if cur, ok := out[s.NodeName]; !ok || s.Timestamp.After(cur.Timestamp) {
			out[s.NodeName] = s
		}
	}
	return out
}
