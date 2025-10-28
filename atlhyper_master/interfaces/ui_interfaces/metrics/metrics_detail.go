// atlhyper_master/interfaces/ui_interfaces/metrics/metrics_detail.go
package metrics

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/metrics"
)

// BuildNodeMetricsDetail —— 单节点 15 分钟 / 1 分钟采样 的时序 + 最新快照
func BuildNodeMetricsDetail(ctx context.Context, clusterID, node string) (*NodeMetricsDetailDTO, error) {
	const minutes = 15

	// 1) 设定时间窗（UTC，对齐到整分钟）
	until := time.Now().UTC().Truncate(time.Minute)
	since := until.Add(-time.Duration(minutes) * time.Minute)

	// 2) 拉取时间窗内全集群指标，并筛选该节点
	series, err := datasource.GetClusterMetricsRange(ctx, clusterID, since, until)
	if err != nil {
		return nil, err
	}
	points := make([]mod.NodeMetricsSnapshot, 0, len(series))
	for _, s := range series {
		if s.NodeName == node {
			points = append(points, s)
		}
	}

	// 3) 无数据兜底
	if len(points) == 0 {
		dto := &NodeMetricsDetailDTO{
			Node:   node,
			Latest: NodeMetricsRow{},
			Series: NodeSeries{
				At:     []time.Time{},
				CPUPct: []float64{},
				MemPct: []float64{},
				TempC:  []float64{},
				DiskPct: []float64{},
				Eth0Tx: []float64{},
				Eth0Rx: []float64{},
			},
		}
		dto.TimeRange.Since = since
		dto.TimeRange.Until = until
		return dto, nil
	}

	// 4) 按分钟聚合：同一分钟取“最后一个点”
	minBuckets := make(map[time.Time]mod.NodeMetricsSnapshot, minutes+1)
	for _, p := range points {
		b := p.Timestamp.UTC().Truncate(time.Minute)
		if exist, ok := minBuckets[b]; !ok || p.Timestamp.After(exist.Timestamp) {
			minBuckets[b] = p
		}
	}

	// 5) 生成等距时序（1 分钟步进），缺口沿用上一分钟值
	ser := NodeSeries{
		At:     make([]time.Time, 0, minutes+1),
		CPUPct: make([]float64, 0, minutes+1),
		MemPct: make([]float64, 0, minutes+1),
		TempC:  make([]float64, 0, minutes+1),
		DiskPct: make([]float64, 0, minutes+1),
		Eth0Tx: make([]float64, 0, minutes+1),
		Eth0Rx: make([]float64, 0, minutes+1),
	}

	var last *mod.NodeMetricsSnapshot
	for t := since; !t.After(until); t = t.Add(time.Minute) {
		ser.At = append(ser.At, t)

		if cur, ok := minBuckets[t]; ok {
			// 有当前分钟采样 → 使用它并更新 last
			last = &cur
		}

		if last == nil {
			// 初始若仍无 last，就补 0（也可选择跳过该点）
			ser.CPUPct = append(ser.CPUPct, 0)
			ser.MemPct = append(ser.MemPct, 0)
			ser.TempC = append(ser.TempC, 0)
			ser.DiskPct = append(ser.DiskPct, 0)
			ser.Eth0Tx = append(ser.Eth0Tx, 0)
			ser.Eth0Rx = append(ser.Eth0Rx, 0)
			continue
		}

		// 用 last（当前或最近一分钟的点）填充
		cpuPct := round1(last.CPU.Usage * 100.0)
		memPct := round1(last.Memory.Usage * 100.0)
		tempC := round1(last.Temperature.CPUDegrees)
		diskPct := pickDiskUsedPercent(last.Disk) // 已是百分比并 round1
		tx := findEth0Tx(last.Network)            // 已 round2
		rx := findEth0Rx(last.Network)            // 已 round2

		ser.CPUPct = append(ser.CPUPct, cpuPct)
		ser.MemPct = append(ser.MemPct, memPct)
		ser.TempC = append(ser.TempC, tempC)
		ser.DiskPct = append(ser.DiskPct, diskPct)
		ser.Eth0Tx = append(ser.Eth0Tx, tx)
		ser.Eth0Rx = append(ser.Eth0Rx, rx)
	}

	// 6) Latest：取整个 points 中时间戳最晚的一条
	latest := points[0]
	for i := 1; i < len(points); i++ {
		if points[i].Timestamp.After(latest.Timestamp) {
			latest = points[i]
		}
	}

	// 7) 组装 DTO
	dto := &NodeMetricsDetailDTO{
		Node: node,
		Latest: NodeMetricsRow{
			Node:        node,
			CPUPercent:  round1(latest.CPU.Usage * 100.0),
			MemPercent:  round1(latest.Memory.Usage * 100.0),
			CPUTempC:    round1(latest.Temperature.CPUDegrees),
			DiskUsedPct: pickDiskUsedPercent(latest.Disk),
			Eth0TxKBps:  findEth0Tx(latest.Network),
			Eth0RxKBps:  findEth0Rx(latest.Network),
			TopCPUProc:  topCPUProcName(latest.TopCPUProcesses),
			Timestamp:   latest.Timestamp,
		},
		Series: ser,
	}
	dto.Processes = latest.TopCPUProcesses // ✅ 加这一行
	dto.TimeRange.Since = since
	dto.TimeRange.Until = until



	return dto, nil
}
