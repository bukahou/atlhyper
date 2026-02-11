// Package otel OTel Collector 采集客户端实现
//
// node_parser.go - node_exporter 指标解析
//
// 逐行扫描 OTel Collector 输出的 Prometheus 文本，
// 解析 otel_node_* 指标并按 instance 分组为 OTelNodeRawMetrics。
//
// 解析策略:
//  1. 只处理 "otel_node_" 前缀的行
//  2. 去除 "otel_" 前缀后匹配指标名
//  3. 提取 label (instance, cpu, mode, device, mountpoint, fstype, chip, sensor 等)
//  4. 按 instance label 分组填充 OTelNodeRawMetrics
//  5. NodeName 从 node_uname_info{nodename=...} 提取
//  6. 在解析阶段应用过滤规则 (文件系统/网络/磁盘I/O)
package otel

import (
	"strconv"
	"strings"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// parseNodeMetrics 解析 OTel Prometheus 文本中的 node_exporter 指标
//
// 返回 map[nodeName]*OTelNodeRawMetrics
// 如果 nodeName 尚未从 uname_info 中获取，使用 instance 作为临时 key
func parseNodeMetrics(text string) map[string]*sdk.OTelNodeRawMetrics {
	// instance → *OTelNodeRawMetrics
	byInstance := make(map[string]*sdk.OTelNodeRawMetrics)
	// 追踪每个 instance 的 CPU 核心集合 (用于去重计数)
	cpuCores := make(map[string]map[string]bool)

	for _, line := range strings.Split(text, "\n") {
		// 快速前缀过滤
		if !strings.HasPrefix(line, "otel_node_") {
			continue
		}

		// 提取 metric_name, labels, value
		matches := metricLineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		name := matches[1]
		labelsRaw := matches[2]
		valueStr := matches[3]

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		var labels map[string]string
		if labelsRaw != "" {
			labels = parseLabels(labelsRaw[1 : len(labelsRaw)-1])
		} else {
			labels = make(map[string]string)
		}

		instance := labels["instance"]
		if instance == "" {
			continue
		}

		// 获取或创建节点
		node := byInstance[instance]
		if node == nil {
			node = &sdk.OTelNodeRawMetrics{
				Instance:        instance,
				CPUSecondsTotal: make(map[string]float64),
				CPUFreqHertz:    make(map[string]float64),
			}
			byInstance[instance] = node
			cpuCores[instance] = make(map[string]bool)
		}

		// 去除 otel_ 前缀后匹配
		metricName := name[len("otel_"):]

		switch metricName {
		// ---- 系统信息 ----
		case "node_uname_info":
			node.NodeName = labels["nodename"]
			node.Hostname = labels["nodename"]
			node.Machine = labels["machine"]
			node.Kernel = labels["release"]

		case "node_boot_time_seconds":
			node.BootTime = value

		// ---- CPU ----
		case "node_cpu_seconds_total":
			cpu := labels["cpu"]
			mode := labels["mode"]
			key := cpu + ":" + mode
			node.CPUSecondsTotal[key] = value
			cpuCores[instance][cpu] = true
			node.CPUCoreCount = len(cpuCores[instance])

		case "node_cpu_scaling_frequency_hertz":
			cpu := labels["cpu"]
			node.CPUFreqHertz[cpu] = value

		case "node_cpu_scaling_frequency_max_hertz":
			if value > node.CPUFreqMaxHertz {
				node.CPUFreqMaxHertz = value
			}

		// ---- Load ----
		case "node_load1":
			node.Load1 = value
		case "node_load5":
			node.Load5 = value
		case "node_load15":
			node.Load15 = value

		// ---- Memory ----
		case "node_memory_MemTotal_bytes":
			node.MemTotal = int64(value)
		case "node_memory_MemAvailable_bytes":
			node.MemAvailable = int64(value)
		case "node_memory_MemFree_bytes":
			node.MemFree = int64(value)
		case "node_memory_Cached_bytes":
			node.MemCached = int64(value)
		case "node_memory_Buffers_bytes":
			node.MemBuffers = int64(value)
		case "node_memory_SwapTotal_bytes":
			node.SwapTotal = int64(value)
		case "node_memory_SwapFree_bytes":
			node.SwapFree = int64(value)

		// ---- Filesystem ----
		case "node_filesystem_size_bytes":
			device := labels["device"]
			fstype := labels["fstype"]
			mountpoint := labels["mountpoint"]
			if !shouldKeepFilesystemNode(device) {
				continue
			}
			fs := getOrCreateFS(node, device, mountpoint)
			fs.FSType = fstype
			fs.SizeBytes = int64(value)

		case "node_filesystem_avail_bytes":
			device := labels["device"]
			mountpoint := labels["mountpoint"]
			if !shouldKeepFilesystemNode(device) {
				continue
			}
			fs := getOrCreateFS(node, device, mountpoint)
			fs.AvailBytes = int64(value)

		// ---- Disk I/O ----
		case "node_disk_read_bytes_total":
			device := labels["device"]
			if !shouldKeepDiskIONode(device) {
				continue
			}
			disk := getOrCreateDiskIO(node, device)
			disk.ReadBytesTotal = value

		case "node_disk_written_bytes_total":
			device := labels["device"]
			if !shouldKeepDiskIONode(device) {
				continue
			}
			disk := getOrCreateDiskIO(node, device)
			disk.WrittenBytesTotal = value

		case "node_disk_reads_completed_total":
			device := labels["device"]
			if !shouldKeepDiskIONode(device) {
				continue
			}
			disk := getOrCreateDiskIO(node, device)
			disk.ReadsCompletedTotal = value

		case "node_disk_writes_completed_total":
			device := labels["device"]
			if !shouldKeepDiskIONode(device) {
				continue
			}
			disk := getOrCreateDiskIO(node, device)
			disk.WritesCompletedTotal = value

		case "node_disk_io_time_seconds_total":
			device := labels["device"]
			if !shouldKeepDiskIONode(device) {
				continue
			}
			disk := getOrCreateDiskIO(node, device)
			disk.IOTimeSecondsTotal = value

		// ---- Network ----
		case "node_network_up":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.Up = value == 1

		case "node_network_speed_bytes":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.Speed = int64(value)

		case "node_network_mtu_bytes":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.MTU = int(value)

		case "node_network_receive_bytes_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.RxBytesTotal = value

		case "node_network_transmit_bytes_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.TxBytesTotal = value

		case "node_network_receive_packets_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.RxPacketsTotal = value

		case "node_network_transmit_packets_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.TxPacketsTotal = value

		case "node_network_receive_errs_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.RxErrsTotal = value

		case "node_network_transmit_errs_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.TxErrsTotal = value

		case "node_network_receive_drop_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.RxDropTotal = value

		case "node_network_transmit_drop_total":
			device := labels["device"]
			if !shouldKeepNetworkNode(device) {
				continue
			}
			net := getOrCreateNet(node, device)
			net.TxDropTotal = value

		// ---- Temperature ----
		case "node_hwmon_temp_celsius":
			chip := labels["chip"]
			sensor := labels["sensor"]
			temp := getOrCreateTemp(node, chip, sensor)
			temp.Current = value

		case "node_hwmon_temp_max_celsius":
			chip := labels["chip"]
			sensor := labels["sensor"]
			temp := getOrCreateTemp(node, chip, sensor)
			temp.Max = value

		case "node_hwmon_temp_crit_celsius":
			chip := labels["chip"]
			sensor := labels["sensor"]
			temp := getOrCreateTemp(node, chip, sensor)
			temp.Critical = value

		// ---- PSI ----
		case "node_pressure_cpu_waiting_seconds_total":
			node.PSICPUWaiting = value
		case "node_pressure_memory_waiting_seconds_total":
			node.PSIMemoryWaiting = value
		case "node_pressure_memory_stalled_seconds_total":
			node.PSIMemoryStalled = value
		case "node_pressure_io_waiting_seconds_total":
			node.PSIIOWaiting = value
		case "node_pressure_io_stalled_seconds_total":
			node.PSIIOStalled = value

		// ---- TCP/Socket ----
		case "node_netstat_Tcp_CurrEstab":
			node.TCPCurrEstab = int64(value)
		case "node_sockstat_TCP_tw":
			node.TCPTimeWait = int64(value)
		case "node_sockstat_TCP_orphan":
			node.TCPOrphan = int64(value)
		case "node_sockstat_TCP_alloc":
			node.TCPAlloc = int64(value)
		case "node_sockstat_TCP_inuse":
			node.TCPInUse = int64(value)
		case "node_sockstat_sockets_used":
			node.SocketsUsed = int64(value)

		// ---- System ----
		case "node_nf_conntrack_entries":
			node.ConntrackEntries = int64(value)
		case "node_nf_conntrack_entries_limit":
			node.ConntrackLimit = int64(value)
		case "node_filefd_allocated":
			node.FilefdAllocated = int64(value)
		case "node_filefd_maximum":
			node.FilefdMaximum = int64(value)
		case "node_entropy_available_bits":
			node.EntropyBits = int64(value)

		// ---- VMStat ----
		case "node_vmstat_pgfault":
			node.PgFault = value
		case "node_vmstat_pgmajfault":
			node.PgMajFault = value
		case "node_vmstat_pswpin":
			node.PswpIn = value
		case "node_vmstat_pswpout":
			node.PswpOut = value

		// ---- NTP ----
		case "node_timex_offset_seconds":
			node.TimexOffsetSeconds = value
		case "node_timex_sync_status":
			node.TimexSyncStatus = value

		// ---- Softnet (per-cpu counter, 求和) ----
		case "node_softnet_dropped_total":
			node.SoftnetDropped += int64(value)
		case "node_softnet_times_squeezed_total":
			node.SoftnetSqueezed += int64(value)
		}
	}

	// 将 byInstance 转为 byNodeName
	result := make(map[string]*sdk.OTelNodeRawMetrics, len(byInstance))
	for _, node := range byInstance {
		key := node.NodeName
		if key == "" {
			key = node.Instance // 降级用 instance
		}
		result[key] = node
	}
	return result
}

// =============================================================================
// 过滤规则（解析阶段应用）
// =============================================================================

// shouldKeepFilesystemNode 判断文件系统是否应保留
func shouldKeepFilesystemNode(device string) bool {
	return strings.HasPrefix(device, "/dev/")
}

// shouldKeepNetworkNode 判断网络接口是否应保留
func shouldKeepNetworkNode(device string) bool {
	switch {
	case device == "lo":
		return false
	case strings.HasPrefix(device, "veth"):
		return false
	case strings.HasPrefix(device, "flannel"):
		return false
	case strings.HasPrefix(device, "cni"):
		return false
	case strings.HasPrefix(device, "cali"):
		return false
	}
	return true
}

// shouldKeepDiskIONode 判断磁盘 I/O 设备是否应保留
func shouldKeepDiskIONode(device string) bool {
	return !strings.HasPrefix(device, "dm-")
}

// =============================================================================
// 辅助函数：获取或创建子结构
// =============================================================================

func getOrCreateFS(node *sdk.OTelNodeRawMetrics, device, mountpoint string) *sdk.FSRawMetrics {
	for i := range node.Filesystems {
		if node.Filesystems[i].Device == device && node.Filesystems[i].MountPoint == mountpoint {
			return &node.Filesystems[i]
		}
	}
	node.Filesystems = append(node.Filesystems, sdk.FSRawMetrics{
		Device:     device,
		MountPoint: mountpoint,
	})
	return &node.Filesystems[len(node.Filesystems)-1]
}

func getOrCreateDiskIO(node *sdk.OTelNodeRawMetrics, device string) *sdk.DiskIORawMetrics {
	for i := range node.DiskIO {
		if node.DiskIO[i].Device == device {
			return &node.DiskIO[i]
		}
	}
	node.DiskIO = append(node.DiskIO, sdk.DiskIORawMetrics{
		Device: device,
	})
	return &node.DiskIO[len(node.DiskIO)-1]
}

func getOrCreateNet(node *sdk.OTelNodeRawMetrics, device string) *sdk.NetRawMetrics {
	for i := range node.Networks {
		if node.Networks[i].Device == device {
			return &node.Networks[i]
		}
	}
	node.Networks = append(node.Networks, sdk.NetRawMetrics{
		Device: device,
	})
	return &node.Networks[len(node.Networks)-1]
}

func getOrCreateTemp(node *sdk.OTelNodeRawMetrics, chip, sensor string) *sdk.HWMonRawTemp {
	for i := range node.HWMonTemps {
		if node.HWMonTemps[i].Chip == chip && node.HWMonTemps[i].Sensor == sensor {
			return &node.HWMonTemps[i]
		}
	}
	node.HWMonTemps = append(node.HWMonTemps, sdk.HWMonRawTemp{
		Chip:   chip,
		Sensor: sensor,
	})
	return &node.HWMonTemps[len(node.HWMonTemps)-1]
}
