package metrics

import (
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v2"
)

// convertToSnapshot 将 OTelNodeRawMetrics 转换为 NodeMetricsSnapshot
//
// cur: 当前采样数据（必须非 nil）
// prev: 上一次采样数据（首次时为 nil，counter 类指标返回零值）
// elapsed: 两次采样间隔（秒）
func convertToSnapshot(nodeName string, cur, prev *sdk.OTelNodeRawMetrics, elapsed float64) *model_v2.NodeMetricsSnapshot {
	snap := &model_v2.NodeMetricsSnapshot{
		NodeName:  nodeName,
		Timestamp: time.Now(),
		Hostname:  cur.Hostname,
		Kernel:    cur.Kernel,
	}

	// Uptime
	if cur.BootTime > 0 {
		snap.Uptime = int64(time.Now().Unix()) - int64(cur.BootTime)
	}

	// CPU
	convertCPU(snap, cur, prev, elapsed)

	// Memory
	convertMemory(snap, cur)

	// Disk (filesystem + I/O)
	convertDisks(snap, cur, prev, elapsed)

	// Network
	convertNetworks(snap, cur, prev, elapsed)

	// Temperature
	convertTemperature(snap, cur)

	// PSI
	convertPSI(snap, cur, prev, elapsed)

	// TCP/Socket
	snap.TCP = model_v2.TCPMetrics{
		CurrEstab:   cur.TCPCurrEstab,
		TimeWait:    cur.TCPTimeWait,
		Orphan:      cur.TCPOrphan,
		Alloc:       cur.TCPAlloc,
		InUse:       cur.TCPInUse,
		SocketsUsed: cur.SocketsUsed,
	}

	// System
	snap.System = model_v2.SystemMetrics{
		ConntrackEntries: cur.ConntrackEntries,
		ConntrackLimit:   cur.ConntrackLimit,
		FilefdAllocated:  cur.FilefdAllocated,
		FilefdMaximum:    cur.FilefdMaximum,
		EntropyAvailable: cur.EntropyBits,
	}

	// VMStat
	convertVMStat(snap, cur, prev, elapsed)

	// NTP
	snap.NTP = model_v2.NTPMetrics{
		OffsetSeconds: cur.TimexOffsetSeconds,
		Synced:        cur.TimexSyncStatus == 1,
	}

	// Softnet
	snap.Softnet = model_v2.SoftnetMetrics{
		Dropped:  cur.SoftnetDropped,
		Squeezed: cur.SoftnetSqueezed,
	}

	return snap
}

// =============================================================================
// CPU 转换
// =============================================================================

func convertCPU(snap *model_v2.NodeMetricsSnapshot, cur, prev *sdk.OTelNodeRawMetrics, elapsed float64) {
	snap.CPU.Load1 = cur.Load1
	snap.CPU.Load5 = cur.Load5
	snap.CPU.Load15 = cur.Load15
	snap.CPU.Cores = cur.CPUCoreCount
	snap.CPU.Threads = cur.CPUCoreCount // node_exporter 不区分物理核和逻辑线程

	// CPU 频率: 所有核平均值，Hz → MHz
	if len(cur.CPUFreqHertz) > 0 {
		var sum float64
		for _, hz := range cur.CPUFreqHertz {
			sum += hz
		}
		snap.CPU.Frequency = sum / float64(len(cur.CPUFreqHertz)) / 1e6
	}

	// 无 prev 时无法计算使用率
	if prev == nil || elapsed <= 0 {
		return
	}

	// 聚合所有核的 delta
	var totalDelta, idleDelta, userDelta, systemDelta, iowaitDelta float64

	for key, curVal := range cur.CPUSecondsTotal {
		prevVal := prev.CPUSecondsTotal[key]
		delta := counterDelta(curVal, prevVal)
		totalDelta += delta

		// 提取 mode
		parts := strings.SplitN(key, ":", 2)
		if len(parts) == 2 {
			switch parts[1] {
			case "idle":
				idleDelta += delta
			case "user":
				userDelta += delta
			case "system":
				systemDelta += delta
			case "iowait":
				iowaitDelta += delta
			}
		}
	}

	if totalDelta > 0 {
		snap.CPU.UsagePercent = (totalDelta - idleDelta) / totalDelta * 100
		snap.CPU.UserPercent = userDelta / totalDelta * 100
		snap.CPU.SystemPercent = systemDelta / totalDelta * 100
		snap.CPU.IdlePercent = idleDelta / totalDelta * 100
		snap.CPU.IOWaitPercent = iowaitDelta / totalDelta * 100
	}

	// 每核使用率
	if cur.CPUCoreCount > 0 {
		snap.CPU.PerCore = make([]float64, cur.CPUCoreCount)
		for i := 0; i < cur.CPUCoreCount; i++ {
			cpu := strconv.Itoa(i)
			var coreTotal, coreIdle float64
			for _, mode := range []string{"idle", "user", "system", "iowait", "nice", "irq", "softirq", "steal"} {
				key := cpu + ":" + mode
				curVal := cur.CPUSecondsTotal[key]
				prevVal := prev.CPUSecondsTotal[key]
				delta := counterDelta(curVal, prevVal)
				coreTotal += delta
				if mode == "idle" {
					coreIdle = delta
				}
			}
			if coreTotal > 0 {
				snap.CPU.PerCore[i] = (coreTotal - coreIdle) / coreTotal * 100
			}
		}
	}
}

// =============================================================================
// Memory 转换
// =============================================================================

func convertMemory(snap *model_v2.NodeMetricsSnapshot, cur *sdk.OTelNodeRawMetrics) {
	snap.Memory.Total = cur.MemTotal
	snap.Memory.Available = cur.MemAvailable
	snap.Memory.Free = cur.MemFree
	snap.Memory.Cached = cur.MemCached
	snap.Memory.Buffers = cur.MemBuffers
	snap.Memory.SwapTotal = cur.SwapTotal
	snap.Memory.SwapFree = cur.SwapFree

	// Used = Total - Available
	snap.Memory.Used = cur.MemTotal - cur.MemAvailable
	if cur.MemTotal > 0 {
		snap.Memory.UsagePercent = float64(snap.Memory.Used) / float64(cur.MemTotal) * 100
	}

	// Swap
	snap.Memory.SwapUsed = cur.SwapTotal - cur.SwapFree
	if cur.SwapTotal > 0 {
		snap.Memory.SwapPercent = float64(snap.Memory.SwapUsed) / float64(cur.SwapTotal) * 100
	}
}

// =============================================================================
// Disk 转换（合并 filesystem 空间 + I/O rate）
// =============================================================================

func convertDisks(snap *model_v2.NodeMetricsSnapshot, cur, prev *sdk.OTelNodeRawMetrics, elapsed float64) {
	// 先从 filesystem 创建磁盘列表
	for _, fs := range cur.Filesystems {
		used := fs.SizeBytes - fs.AvailBytes
		var usagePercent float64
		if fs.SizeBytes > 0 {
			usagePercent = float64(used) / float64(fs.SizeBytes) * 100
		}
		snap.Disks = append(snap.Disks, model_v2.DiskMetrics{
			Device:       fs.Device,
			MountPoint:   fs.MountPoint,
			FSType:       fs.FSType,
			Total:        fs.SizeBytes,
			Used:         used,
			Available:    fs.AvailBytes,
			UsagePercent: usagePercent,
		})
	}

	// 合并 I/O rate
	if prev == nil || elapsed <= 0 {
		return
	}
	for _, curIO := range cur.DiskIO {
		prevIO := findDiskIO(prev.DiskIO, curIO.Device)
		if prevIO == nil {
			continue
		}
		// 找到匹配的 disk 条目（按设备名模糊匹配）
		disk := findDiskByDevice(snap.Disks, curIO.Device)
		if disk == nil {
			// I/O 设备可能没有对应的 filesystem 条目（如裸磁盘）
			snap.Disks = append(snap.Disks, model_v2.DiskMetrics{Device: curIO.Device})
			disk = &snap.Disks[len(snap.Disks)-1]
		}
		disk.ReadBytes = int64(curIO.ReadBytesTotal)
		disk.WriteBytes = int64(curIO.WrittenBytesTotal)
		disk.ReadRate = counterRate(curIO.ReadBytesTotal, prevIO.ReadBytesTotal, elapsed)
		disk.WriteRate = counterRate(curIO.WrittenBytesTotal, prevIO.WrittenBytesTotal, elapsed)
		disk.ReadIOPS = counterRate(curIO.ReadsCompletedTotal, prevIO.ReadsCompletedTotal, elapsed)
		disk.WriteIOPS = counterRate(curIO.WritesCompletedTotal, prevIO.WritesCompletedTotal, elapsed)
		ioUtil := counterRate(curIO.IOTimeSecondsTotal, prevIO.IOTimeSecondsTotal, elapsed) * 100
		if ioUtil > 100 {
			ioUtil = 100
		}
		disk.IOUtil = ioUtil
	}
}

func findDiskIO(disks []sdk.DiskIORawMetrics, device string) *sdk.DiskIORawMetrics {
	for i := range disks {
		if disks[i].Device == device {
			return &disks[i]
		}
	}
	return nil
}

func findDiskByDevice(disks []model_v2.DiskMetrics, device string) *model_v2.DiskMetrics {
	// 精确匹配或包含匹配 (如 "sda" 匹配 "/dev/sda1")
	for i := range disks {
		if disks[i].Device == device || strings.Contains(disks[i].Device, device) {
			return &disks[i]
		}
	}
	return nil
}

// =============================================================================
// Network 转换
// =============================================================================

func convertNetworks(snap *model_v2.NodeMetricsSnapshot, cur, prev *sdk.OTelNodeRawMetrics, elapsed float64) {
	for _, n := range cur.Networks {
		net := model_v2.NetworkMetrics{
			Interface: n.Device,
			MTU:       n.MTU,
			RxBytes:   int64(n.RxBytesTotal),
			TxBytes:   int64(n.TxBytesTotal),
			RxPackets: int64(n.RxPacketsTotal),
			TxPackets: int64(n.TxPacketsTotal),
			RxErrors:  int64(n.RxErrsTotal),
			TxErrors:  int64(n.TxErrsTotal),
			RxDropped: int64(n.RxDropTotal),
			TxDropped: int64(n.TxDropTotal),
		}
		if n.Up {
			net.Status = "up"
		} else {
			net.Status = "down"
		}
		// Speed: bytes/s → Mbps
		if n.Speed > 0 {
			net.Speed = n.Speed * 8 / 1000000
		}

		// Rate
		if prev != nil && elapsed > 0 {
			if prevNet := findNetRaw(prev.Networks, n.Device); prevNet != nil {
				net.RxRate = counterRate(n.RxBytesTotal, prevNet.RxBytesTotal, elapsed)
				net.TxRate = counterRate(n.TxBytesTotal, prevNet.TxBytesTotal, elapsed)
			}
		}

		snap.Networks = append(snap.Networks, net)
	}
}

func findNetRaw(nets []sdk.NetRawMetrics, device string) *sdk.NetRawMetrics {
	for i := range nets {
		if nets[i].Device == device {
			return &nets[i]
		}
	}
	return nil
}

// =============================================================================
// Temperature 转换
// =============================================================================

func convertTemperature(snap *model_v2.NodeMetricsSnapshot, cur *sdk.OTelNodeRawMetrics) {
	// 转换所有传感器
	for _, t := range cur.HWMonTemps {
		snap.Temperature.Sensors = append(snap.Temperature.Sensors, model_v2.SensorReading{
			Name:     t.Chip,
			Label:    t.Sensor,
			Current:  t.Current,
			Max:      t.Max,
			Critical: t.Critical,
		})
	}

	// CPUTemp 选择策略:
	// 1. x86: platform_coretemp_0 temp1 (Package temperature)
	// 2. arm64: thermal_zone 或 adc 芯片中最高温度
	var cpuTemp, cpuTempMax float64
	found := false

	// 优先找 coretemp (x86)
	for _, t := range cur.HWMonTemps {
		if strings.HasPrefix(t.Chip, "platform_coretemp") && t.Sensor == "temp1" {
			cpuTemp = t.Current
			cpuTempMax = t.Max
			found = true
			break
		}
	}

	// 未找到 coretemp: 取 thermal_zone 或 adc 的最高温度 (arm64)
	if !found {
		for _, t := range cur.HWMonTemps {
			if strings.Contains(t.Chip, "thermal_zone") || strings.Contains(t.Chip, "adc") {
				if t.Current > cpuTemp {
					cpuTemp = t.Current
					cpuTempMax = t.Max
				}
			}
		}
	}

	snap.Temperature.CPUTemp = cpuTemp
	snap.Temperature.CPUTempMax = cpuTempMax
}

// =============================================================================
// PSI 转换
// =============================================================================

func convertPSI(snap *model_v2.NodeMetricsSnapshot, cur, prev *sdk.OTelNodeRawMetrics, elapsed float64) {
	if prev == nil || elapsed <= 0 {
		return
	}
	snap.PSI = model_v2.PSIMetrics{
		CPUSomePercent:    counterRate(cur.PSICPUWaiting, prev.PSICPUWaiting, elapsed) * 100,
		MemorySomePercent: counterRate(cur.PSIMemoryWaiting, prev.PSIMemoryWaiting, elapsed) * 100,
		MemoryFullPercent: counterRate(cur.PSIMemoryStalled, prev.PSIMemoryStalled, elapsed) * 100,
		IOSomePercent:     counterRate(cur.PSIIOWaiting, prev.PSIIOWaiting, elapsed) * 100,
		IOFullPercent:     counterRate(cur.PSIIOStalled, prev.PSIIOStalled, elapsed) * 100,
	}
}

// =============================================================================
// VMStat 转换
// =============================================================================

func convertVMStat(snap *model_v2.NodeMetricsSnapshot, cur, prev *sdk.OTelNodeRawMetrics, elapsed float64) {
	if prev == nil || elapsed <= 0 {
		return
	}
	snap.VMStat = model_v2.VMStatMetrics{
		PgFaultPS:    counterRate(cur.PgFault, prev.PgFault, elapsed),
		PgMajFaultPS: counterRate(cur.PgMajFault, prev.PgMajFault, elapsed),
		PswpInPS:     counterRate(cur.PswpIn, prev.PswpIn, elapsed),
		PswpOutPS:    counterRate(cur.PswpOut, prev.PswpOut, elapsed),
	}
}
