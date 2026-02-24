// atlhyper_master_v2/model/convert/node_metrics.go
// model_v2 → model 节点指标转换函数
package convert

import (
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// NodeMetricsSnapshot 转换单个节点快照
func NodeMetricsSnapshot(src *model_v2.NodeMetricsSnapshot) model.NodeMetricsSnapshot {
	if src == nil {
		return model.NodeMetricsSnapshot{
			Disks:        []model.DiskMetrics{},
			Networks:     []model.NetworkMetrics{},
			TopProcesses: []model.ProcessMetrics{},
			CPU:          model.CPUMetrics{CoreUsages: []float64{}},
			Temperature:  model.TemperatureMetrics{Sensors: []model.SensorReading{}},
		}
	}
	return model.NodeMetricsSnapshot{
		NodeName:     src.NodeName,
		Timestamp:    src.Timestamp,
		Uptime:       src.Uptime,
		OS:           src.OS,
		Kernel:       src.Kernel,
		CPU:          convertCPU(src.CPU),
		Memory:       convertMemory(src.Memory),
		Disks:        convertDisks(src.Disks),
		Networks:     convertNetworks(src.Networks),
		Temperature:  convertTemperature(src.Temperature),
		TopProcesses: convertProcesses(src.Processes),
		PSI:          convertPSI(src.PSI),
		TCP:          convertTCP(src.TCP),
		System:       convertSystem(src.System),
		VMStat:       convertVMStat(src.VMStat),
		NTP:          convertNTP(src.NTP),
		Softnet:      convertSoftnet(src.Softnet),
	}
}

// NodeMetricsSnapshots 批量转换节点快照
func NodeMetricsSnapshots(src []*model_v2.NodeMetricsSnapshot) []model.NodeMetricsSnapshot {
	if src == nil {
		return []model.NodeMetricsSnapshot{}
	}
	result := make([]model.NodeMetricsSnapshot, len(src))
	for i, s := range src {
		result[i] = NodeMetricsSnapshot(s)
	}
	return result
}

// MetricsHistoryGrouped 将扁平历史数据按指标分组
// 输出: { "cpu": [...], "memory": [...], "disk": [...], "temp": [...] }
func MetricsHistoryGrouped(src []model_v2.MetricsDataPoint) map[string][]model.TimeSeriesPoint {
	result := map[string][]model.TimeSeriesPoint{
		"cpu":    {},
		"memory": {},
		"disk":   {},
		"temp":   {},
	}
	for _, pt := range src {
		ts := pt.Timestamp.UTC().Format(time.RFC3339)
		result["cpu"] = append(result["cpu"], model.TimeSeriesPoint{Timestamp: ts, Value: pt.CPUUsage})
		result["memory"] = append(result["memory"], model.TimeSeriesPoint{Timestamp: ts, Value: pt.MemoryUsage})
		result["disk"] = append(result["disk"], model.TimeSeriesPoint{Timestamp: ts, Value: pt.DiskUsage})
		result["temp"] = append(result["temp"], model.TimeSeriesPoint{Timestamp: ts, Value: pt.CPUTemp})
	}
	return result
}

// ClusterMetricsSummary 转换集群汇总
func ClusterMetricsSummary(src model_v2.ClusterMetricsSummary) model.ClusterMetricsSummary {
	return model.ClusterMetricsSummary{
		TotalNodes:     src.TotalNodes,
		OnlineNodes:    src.OnlineNodes,
		OfflineNodes:   src.OfflineNodes,
		AvgCPUUsage:    src.AvgCPUUsage,
		AvgMemoryUsage: src.AvgMemoryUsage,
		AvgDiskUsage:   src.AvgDiskUsage,
		MaxCPUUsage:    src.MaxCPUUsage,
		MaxMemoryUsage: src.MaxMemoryUsage,
		MaxDiskUsage:   src.MaxDiskUsage,
		AvgCPUTemp:     src.AvgCPUTemp,
		MaxCPUTemp:     src.MaxCPUTemp,
		TotalMemory:    src.TotalMemory,
		UsedMemory:     src.UsedMemory,
		TotalDisk:      src.TotalDisk,
		UsedDisk:       src.UsedDisk,
		TotalNetworkRx: src.TotalNetworkRx,
		TotalNetworkTx: src.TotalNetworkTx,
	}
}

// ==================== 内部转换函数 ====================

func convertCPU(src model_v2.CPUMetrics) model.CPUMetrics {
	coreUsages := src.PerCore
	if coreUsages == nil {
		coreUsages = []float64{}
	}
	return model.CPUMetrics{
		UsagePercent: src.UsagePercent,
		CoreCount:    src.Cores,
		ThreadCount:  src.Threads,
		CoreUsages:   coreUsages,
		LoadAvg1:     src.Load1,
		LoadAvg5:     src.Load5,
		LoadAvg15:    src.Load15,
		Model:        src.Model,
		Frequency:    src.Frequency,
	}
}

func convertMemory(src model_v2.MemoryMetrics) model.MemoryMetrics {
	return model.MemoryMetrics{
		TotalBytes:       src.Total,
		UsedBytes:        src.Used,
		AvailableBytes:   src.Available,
		UsagePercent:     src.UsagePercent,
		SwapTotalBytes:   src.SwapTotal,
		SwapUsedBytes:    src.SwapUsed,
		SwapUsagePercent: src.SwapPercent,
		Cached:           src.Cached,
		Buffers:          src.Buffers,
	}
}

func convertDisks(src []model_v2.DiskMetrics) []model.DiskMetrics {
	if src == nil {
		return []model.DiskMetrics{}
	}
	result := make([]model.DiskMetrics, len(src))
	for i, d := range src {
		result[i] = model.DiskMetrics{
			Device:         d.Device,
			MountPoint:     d.MountPoint,
			FSType:         d.FSType,
			TotalBytes:     d.Total,
			UsedBytes:      d.Used,
			AvailableBytes: d.Available,
			UsagePercent:   d.UsagePercent,
			ReadBytesPS:    d.ReadRate,
			WriteBytesPS:   d.WriteRate,
			IOPS:           d.ReadIOPS + d.WriteIOPS,
			IOUtil:         d.IOUtil,
		}
	}
	return result
}

func convertNetworks(src []model_v2.NetworkMetrics) []model.NetworkMetrics {
	if src == nil {
		return []model.NetworkMetrics{}
	}
	result := make([]model.NetworkMetrics, len(src))
	for i, n := range src {
		result[i] = model.NetworkMetrics{
			Interface:   n.Interface,
			IPAddress:   n.IPAddress,
			MACAddress:  n.MACAddress,
			Status:      n.Status,
			Speed:       n.Speed,
			RxBytesPS:   n.RxRate,
			TxBytesPS:   n.TxRate,
			RxPacketsPS: 0,
			TxPacketsPS: 0,
			RxErrors:    n.RxErrors,
			TxErrors:    n.TxErrors,
			RxDropped:   n.RxDropped,
			TxDropped:   n.TxDropped,
		}
	}
	return result
}

func convertTemperature(src model_v2.TemperatureMetrics) model.TemperatureMetrics {
	sensors := make([]model.SensorReading, len(src.Sensors))
	for i, s := range src.Sensors {
		sensors[i] = model.SensorReading{
			Name:     s.Name,
			Label:    s.Label,
			Temp:     s.Current,
			High:     s.Max,
			Critical: s.Critical,
		}
	}
	return model.TemperatureMetrics{
		CPUTemp:    src.CPUTemp,
		CPUTempMax: src.CPUTempMax,
		Sensors:    sensors,
	}
}

func convertProcesses(src []model_v2.ProcessMetrics) []model.ProcessMetrics {
	if src == nil {
		return []model.ProcessMetrics{}
	}
	result := make([]model.ProcessMetrics, len(src))
	for i, p := range src {
		result[i] = model.ProcessMetrics{
			PID:        p.PID,
			Name:       p.Name,
			User:       p.User,
			State:      p.Status,
			CPUPercent: p.CPUPercent,
			MemPercent: p.MemPercent,
			MemBytes:   p.MemRSS,
			Threads:    p.Threads,
			StartTime:  time.Unix(p.StartTime, 0).UTC().Format(time.RFC3339),
			Command:    p.Cmdline,
		}
	}
	return result
}

func convertPSI(src model_v2.PSIMetrics) model.PSIMetrics {
	return model.PSIMetrics{
		CPUSomePercent:    src.CPUSomePercent,
		MemorySomePercent: src.MemorySomePercent,
		MemoryFullPercent: src.MemoryFullPercent,
		IOSomePercent:     src.IOSomePercent,
		IOFullPercent:     src.IOFullPercent,
	}
}

func convertTCP(src model_v2.TCPMetrics) model.TCPMetrics {
	return model.TCPMetrics{
		CurrEstab:   src.CurrEstab,
		TimeWait:    src.TimeWait,
		Orphan:      src.Orphan,
		Alloc:       src.Alloc,
		InUse:       src.InUse,
		SocketsUsed: src.SocketsUsed,
	}
}

func convertSystem(src model_v2.SystemMetrics) model.SystemMetrics {
	return model.SystemMetrics{
		ConntrackEntries: src.ConntrackEntries,
		ConntrackLimit:   src.ConntrackLimit,
		FilefdAllocated:  src.FilefdAllocated,
		FilefdMaximum:    src.FilefdMaximum,
		EntropyAvailable: src.EntropyAvailable,
	}
}

func convertVMStat(src model_v2.VMStatMetrics) model.VMStatMetrics {
	return model.VMStatMetrics{
		PgFaultPS:    src.PgFaultPS,
		PgMajFaultPS: src.PgMajFaultPS,
		PswpInPS:     src.PswpInPS,
		PswpOutPS:    src.PswpOutPS,
	}
}

func convertNTP(src model_v2.NTPMetrics) model.NTPMetrics {
	return model.NTPMetrics{
		OffsetSeconds: src.OffsetSeconds,
		Synced:        src.Synced,
	}
}

func convertSoftnet(src model_v2.SoftnetMetrics) model.SoftnetMetrics {
	return model.SoftnetMetrics{
		Dropped:  src.Dropped,
		Squeezed: src.Squeezed,
	}
}
