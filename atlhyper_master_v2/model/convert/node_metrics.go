// atlhyper_master_v2/model/convert/node_metrics.go
// metrics → model 节点指标转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/metrics"
)

// NodeMetricsSnapshot 转换单个节点快照
func NodeMetricsSnapshot(src *metrics.NodeMetrics) model.NodeMetricsSnapshot {
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
		Kernel:       src.Kernel,
		CPU:          convertCPU(src.CPU),
		Memory:       convertMemory(src.Memory),
		Disks:        convertDisks(src.Disks),
		Networks:     convertNetworks(src.Networks),
		Temperature:  convertTemperature(src.Temperature),
		TopProcesses: []model.ProcessMetrics{},
		PSI:          convertPSI(src.PSI),
		TCP:          convertTCP(src.TCP),
		System:       convertSystem(src.System),
		VMStat:       convertVMStat(src.VMStat),
		Softnet:      convertSoftnet(src.Softnet),
	}
}

// NodeMetricsSnapshots 批量转换节点快照
func NodeMetricsSnapshots(src []*metrics.NodeMetrics) []model.NodeMetricsSnapshot {
	if src == nil {
		return []model.NodeMetricsSnapshot{}
	}
	result := make([]model.NodeMetricsSnapshot, len(src))
	for i, s := range src {
		result[i] = NodeMetricsSnapshot(s)
	}
	return result
}

// ClusterMetricsSummary 转换集群汇总
func ClusterMetricsSummary(src metrics.Summary) model.ClusterMetricsSummary {
	return model.ClusterMetricsSummary{
		TotalNodes:  src.TotalNodes,
		OnlineNodes: src.OnlineNodes,
		AvgCPUUsage: src.AvgCPUPct,
		MaxCPUUsage: src.MaxCPUPct,
		MaxCPUTemp:  src.MaxCPUTemp,
	}
}

// ==================== 内部转换函数 ====================

func convertCPU(src metrics.NodeCPU) model.CPUMetrics {
	return model.CPUMetrics{
		UsagePercent: src.UsagePct,
		CoreCount:    src.Cores,
		CoreUsages:   []float64{},
		LoadAvg1:     src.Load1,
		LoadAvg5:     src.Load5,
		LoadAvg15:    src.Load15,
	}
}

func convertMemory(src metrics.NodeMemory) model.MemoryMetrics {
	return model.MemoryMetrics{
		TotalBytes:       src.TotalBytes,
		UsedBytes:        src.TotalBytes - src.AvailableBytes,
		AvailableBytes:   src.AvailableBytes,
		UsagePercent:     src.UsagePct,
		SwapTotalBytes:   src.SwapTotalBytes,
		SwapUsedBytes:    src.SwapTotalBytes - src.SwapFreeBytes,
		SwapUsagePercent: src.SwapUsagePct,
		Cached:           src.CachedBytes,
		Buffers:          src.BuffersBytes,
	}
}

func convertDisks(src []metrics.NodeDisk) []model.DiskMetrics {
	if src == nil {
		return []model.DiskMetrics{}
	}
	result := make([]model.DiskMetrics, len(src))
	for i, d := range src {
		result[i] = model.DiskMetrics{
			Device:         d.Device,
			MountPoint:     d.MountPoint,
			FSType:         d.FSType,
			TotalBytes:     d.TotalBytes,
			UsedBytes:      d.TotalBytes - d.AvailBytes,
			AvailableBytes: d.AvailBytes,
			UsagePercent:   d.UsagePct,
			ReadBytesPS:    d.ReadBytesPerSec,
			WriteBytesPS:   d.WriteBytesPerSec,
			IOPS:           d.ReadIOPS + d.WriteIOPS,
			IOUtil:         d.IOUtilPct,
		}
	}
	return result
}

func convertNetworks(src []metrics.NodeNetwork) []model.NetworkMetrics {
	if src == nil {
		return []model.NetworkMetrics{}
	}
	result := make([]model.NetworkMetrics, len(src))
	for i, n := range src {
		status := "down"
		if n.Up {
			status = "up"
		}
		result[i] = model.NetworkMetrics{
			Interface: n.Interface,
			Status:    status,
			Speed:     n.SpeedBps,
			RxBytesPS: n.RxBytesPerSec,
			TxBytesPS: n.TxBytesPerSec,
		}
	}
	return result
}

func convertTemperature(src metrics.NodeTemperature) model.TemperatureMetrics {
	sensors := make([]model.SensorReading, len(src.Sensors))
	for i, s := range src.Sensors {
		sensors[i] = model.SensorReading{
			Name:     s.Chip,
			Label:    s.Sensor,
			Temp:     s.CurrentC,
			High:     s.MaxC,
			Critical: s.CritC,
		}
	}
	return model.TemperatureMetrics{
		CPUTemp:    src.CPUTempC,
		CPUTempMax: src.CPUMaxC,
		Sensors:    sensors,
	}
}

func convertPSI(src metrics.NodePSI) model.PSIMetrics {
	return model.PSIMetrics{
		CPUSomePercent:    src.CPUSomePct,
		MemorySomePercent: src.MemSomePct,
		MemoryFullPercent: src.MemFullPct,
		IOSomePercent:     src.IOSomePct,
		IOFullPercent:     src.IOFullPct,
	}
}

func convertTCP(src metrics.NodeTCP) model.TCPMetrics {
	return model.TCPMetrics{
		CurrEstab:   src.CurrEstab,
		TimeWait:    src.TimeWait,
		Alloc:       src.Alloc,
		InUse:       src.InUse,
		SocketsUsed: src.SocketsUsed,
	}
}

func convertSystem(src metrics.NodeSystem) model.SystemMetrics {
	return model.SystemMetrics{
		ConntrackEntries: src.ConntrackEntries,
		ConntrackLimit:   src.ConntrackLimit,
		FilefdAllocated:  src.FilefdAllocated,
		FilefdMaximum:    src.FilefdMax,
		EntropyAvailable: src.EntropyBits,
	}
}

func convertVMStat(src metrics.NodeVMStat) model.VMStatMetrics {
	return model.VMStatMetrics{
		PgFaultPS:    src.PgFaultPerSec,
		PgMajFaultPS: src.PgMajFaultPerSec,
		PswpInPS:     src.PswpInPerSec,
		PswpOutPS:    src.PswpOutPerSec,
	}
}

func convertSoftnet(src metrics.NodeSoftnet) model.SoftnetMetrics {
	return model.SoftnetMetrics{
		Dropped:  int64(src.DroppedPerSec),
		Squeezed: int64(src.SqueezedPerSec),
	}
}
