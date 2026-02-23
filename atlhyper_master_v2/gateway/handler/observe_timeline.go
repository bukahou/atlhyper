// atlhyper_master_v2/gateway/handler/observe_timeline.go
// OTel 时间线辅助函数（从内存时间线 / 预聚合时序构建数据）
package handler

import (
	"time"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
)

// buildNodeMetricsSeries 从 OTel 时间线提取单节点时序
// 支持完整 NodeMetrics（100+ 字段），10s 精度
func buildNodeMetricsSeries(entries []cluster.OTelEntry, nodeName, metric string) *metrics.Series {
	series := &metrics.Series{
		Metric: metric,
		Points: make([]metrics.Point, 0, len(entries)),
	}

	for _, e := range entries {
		if e.Snapshot == nil || e.Snapshot.MetricsNodes == nil {
			continue
		}
		for _, node := range e.Snapshot.MetricsNodes {
			if node.NodeName == nodeName {
				value := extractMetricValue(&node, metric)
				if value >= 0 {
					series.Points = append(series.Points, metrics.Point{
						Timestamp: e.Timestamp,
						Value:     value,
					})
				}
				break
			}
		}
	}
	return series
}

// extractMetricValue 从完整 NodeMetrics 提取任意指标值
// 支持所有采集的指标（CPU/Mem/Disk/Net/Temp/PSI/TCP 等）
func extractMetricValue(node *metrics.NodeMetrics, metric string) float64 {
	switch metric {
	// CPU
	case "cpu_usage", "cpu":
		return node.CPU.UsagePct
	case "cpu_user":
		return node.CPU.UserPct
	case "cpu_system":
		return node.CPU.SystemPct
	case "cpu_iowait":
		return node.CPU.IOWaitPct
	case "cpu_load1":
		return node.CPU.Load1
	case "cpu_load5":
		return node.CPU.Load5
	case "cpu_load15":
		return node.CPU.Load15
	// Memory
	case "memory_usage", "memory":
		return node.Memory.UsagePct
	case "swap_usage":
		return node.Memory.SwapUsagePct
	// Temperature
	case "temperature", "cpu_temp":
		return node.Temperature.CPUTempC
	// PSI
	case "psi_cpu":
		return node.PSI.CPUSomePct
	case "psi_memory":
		return node.PSI.MemSomePct
	case "psi_io":
		return node.PSI.IOSomePct
	// TCP
	case "tcp_established":
		return float64(node.TCP.CurrEstab)
	case "sockets_used":
		return float64(node.TCP.SocketsUsed)
	// Disk（主磁盘）
	case "disk_usage", "disk":
		return primaryDiskValue(node, "usagePct")
	case "disk_read_bps":
		return primaryDiskValue(node, "readBps")
	case "disk_write_bps":
		return primaryDiskValue(node, "writeBps")
	case "disk_io_util":
		return primaryDiskValue(node, "ioUtil")
	// Network（主网卡聚合）
	case "net_rx_bps", "net_rx":
		return primaryNetValue(node, "rxBps")
	case "net_tx_bps", "net_tx":
		return primaryNetValue(node, "txBps")
	case "net_rx_pkt":
		return primaryNetValue(node, "rxPkt")
	case "net_tx_pkt":
		return primaryNetValue(node, "txPkt")
	default:
		return node.CPU.UsagePct
	}
}

// primaryDiskValue 从主磁盘提取指标
func primaryDiskValue(node *metrics.NodeMetrics, field string) float64 {
	d := node.GetPrimaryDisk()
	if d == nil {
		return 0
	}
	switch field {
	case "usagePct":
		return d.UsagePct
	case "readBps":
		return d.ReadBytesPerSec
	case "writeBps":
		return d.WriteBytesPerSec
	case "ioUtil":
		return d.IOUtilPct
	default:
		return 0
	}
}

// primaryNetValue 聚合所有活跃非 lo 网卡的指标
func primaryNetValue(node *metrics.NodeMetrics, field string) float64 {
	var total float64
	for _, n := range node.Networks {
		if !n.Up || n.Interface == "lo" {
			continue
		}
		switch field {
		case "rxBps":
			total += n.RxBytesPerSec
		case "txBps":
			total += n.TxBytesPerSec
		case "rxPkt":
			total += n.RxPktPerSec
		case "txPkt":
			total += n.TxPktPerSec
		}
	}
	return total
}

// sloPoint SLO 时序数据点
type sloPoint struct {
	Timestamp interface{} `json:"timestamp"`
	RPS       float64     `json:"rps"`
	SuccRate  float64     `json:"successRate"`
	P50Ms     float64     `json:"p50Ms"`
	P99Ms     float64     `json:"p99Ms"`
	ErrorRate float64     `json:"errorRate"`
}

// buildSLOTimeSeries 从 OTel 时间线构建 SLO 时序
// 返回指定服务的 SLO 指标随时间变化（请求量、成功率、延迟）
func buildSLOTimeSeries(entries []cluster.OTelEntry, serviceName string) map[string]interface{} {
	points := make([]sloPoint, 0, len(entries))
	for _, e := range entries {
		if e.Snapshot == nil {
			continue
		}
		// 在 SLO Ingress 或 SLO Services 中查找
		found := false
		if e.Snapshot.SLOIngress != nil {
			for _, svc := range e.Snapshot.SLOIngress {
				if svc.ServiceKey == serviceName || svc.DisplayName == serviceName {
					points = append(points, sloPoint{
						Timestamp: e.Timestamp,
						RPS:       svc.RPS,
						SuccRate:  svc.SuccessRate,
						P50Ms:     svc.P50Ms,
						P99Ms:     svc.P99Ms,
						ErrorRate: svc.ErrorRate,
					})
					found = true
					break
				}
			}
		}
		if !found && e.Snapshot.SLOServices != nil {
			for _, svc := range e.Snapshot.SLOServices {
				if svc.Name == serviceName {
					points = append(points, sloPoint{
						Timestamp: e.Timestamp,
						RPS:       svc.RPS,
						SuccRate:  svc.SuccessRate,
						P50Ms:     svc.P50Ms,
						P99Ms:     svc.P99Ms,
					})
					break
				}
			}
		}
	}

	return map[string]interface{}{
		"service": serviceName,
		"points":  points,
	}
}

// ================================================================
// 预聚合时序辅助函数
// ================================================================

// filterNodePointsByMinutes 按时间范围裁剪节点时序数据点
func filterNodePointsByMinutes(points []cluster.NodeMetricsPoint, minutes int) []cluster.NodeMetricsPoint {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	result := make([]cluster.NodeMetricsPoint, 0, len(points))
	for _, p := range points {
		if !p.Timestamp.Before(cutoff) {
			result = append(result, p)
		}
	}
	return result
}

// extractNodeMetricPoints 从预聚合节点时序中提取指定指标（25 字段全覆盖）
func extractNodeMetricPoints(points []cluster.NodeMetricsPoint, metric string) []metrics.Point {
	result := make([]metrics.Point, 0, len(points))
	for _, p := range points {
		var value float64
		switch metric {
		// CPU
		case "cpu_usage", "cpu":
			value = p.CPUPct
		case "cpu_user":
			value = p.UserPct
		case "cpu_system":
			value = p.SystemPct
		case "cpu_iowait":
			value = p.IOWaitPct
		case "cpu_load1":
			value = p.Load1
		case "cpu_load5":
			value = p.Load5
		case "cpu_load15":
			value = p.Load15
		// Memory
		case "memory_usage", "memory":
			value = p.MemPct
		case "swap_usage":
			value = p.SwapUsagePct
		// Disk
		case "disk_usage", "disk":
			value = p.DiskPct
		case "disk_read_bps":
			value = p.DiskReadBps
		case "disk_write_bps":
			value = p.DiskWriteBps
		case "disk_io_util":
			value = p.DiskIOUtilPct
		// Network
		case "net_rx_bps", "net_rx":
			value = p.NetRxBps
		case "net_tx_bps", "net_tx":
			value = p.NetTxBps
		case "net_rx_pkt":
			value = p.NetRxPktSec
		case "net_tx_pkt":
			value = p.NetTxPktSec
		// Temperature
		case "temperature", "cpu_temp":
			value = p.CPUTempC
		// PSI
		case "psi_cpu":
			value = p.CPUSomePct
		case "psi_memory":
			value = p.MemSomePct
		case "psi_io":
			value = p.IOSomePct
		// TCP
		case "tcp_established":
			value = float64(p.TCPEstab)
		case "sockets_used":
			value = float64(p.SocketsUsed)
		default:
			value = p.CPUPct
		}
		result = append(result, metrics.Point{
			Timestamp: p.Timestamp,
			Value:     value,
		})
	}
	return result
}

// filterSLOPointsByMinutes 按时间范围裁剪 SLO 时序数据点
func filterSLOPointsByMinutes(points []cluster.SLOTimePoint, minutes int) []cluster.SLOTimePoint {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	result := make([]cluster.SLOTimePoint, 0, len(points))
	for _, p := range points {
		if !p.Timestamp.Before(cutoff) {
			result = append(result, p)
		}
	}
	return result
}

// filterAPMPointsByMinutes 按时间范围裁剪 APM 时序数据点
func filterAPMPointsByMinutes(points []cluster.APMTimePoint, minutes int) []cluster.APMTimePoint {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	result := make([]cluster.APMTimePoint, 0, len(points))
	for _, p := range points {
		if !p.Timestamp.Before(cutoff) {
			result = append(result, p)
		}
	}
	return result
}
