package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

// TestNodeMetricsSnapshot_FieldMapping 验证关键字段映射
func TestNodeMetricsSnapshot_FieldMapping(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	src := &model_v2.NodeMetricsSnapshot{
		NodeName:  "node-1",
		Timestamp: ts,
		Hostname:  "host-1",
		OS:        "linux",
		Kernel:    "6.1.0",
		Uptime:    86400,
		CPU: model_v2.CPUMetrics{
			UsagePercent: 45.5,
			Cores:        4,
			Threads:      8,
			PerCore:      []float64{40, 50, 45, 42, 48, 43, 47, 44},
			Load1:        1.5,
			Load5:        2.0,
			Load15:       1.8,
			Model:        "Intel Xeon",
			Frequency:    2400,
		},
		Memory: model_v2.MemoryMetrics{
			Total:        16 * 1024 * 1024 * 1024,
			Used:         8 * 1024 * 1024 * 1024,
			Available:    7 * 1024 * 1024 * 1024,
			UsagePercent: 50.0,
			SwapTotal:    4 * 1024 * 1024 * 1024,
			SwapUsed:     1 * 1024 * 1024 * 1024,
			SwapPercent:  25.0,
			Cached:       2 * 1024 * 1024 * 1024,
			Buffers:      512 * 1024 * 1024,
		},
		Disks: []model_v2.DiskMetrics{
			{
				Device:       "sda1",
				MountPoint:   "/",
				FSType:       "ext4",
				Total:        500 * 1024 * 1024 * 1024,
				Used:         200 * 1024 * 1024 * 1024,
				Available:    280 * 1024 * 1024 * 1024,
				UsagePercent: 40.0,
				ReadRate:     1024 * 1024,
				WriteRate:    512 * 1024,
				ReadIOPS:     100,
				WriteIOPS:    50,
				IOUtil:       30.0,
			},
		},
		Networks: []model_v2.NetworkMetrics{
			{
				Interface:  "eth0",
				IPAddress:  "192.168.1.1",
				MACAddress: "aa:bb:cc:dd:ee:ff",
				Status:     "up",
				Speed:      1000,
				RxRate:     1024 * 1024,
				TxRate:     512 * 1024,
				RxErrors:   5,
				TxErrors:   3,
				RxDropped:  1,
				TxDropped:  0,
			},
		},
		Temperature: model_v2.TemperatureMetrics{
			CPUTemp:    65.0,
			CPUTempMax: 100.0,
			Sensors: []model_v2.SensorReading{
				{Name: "coretemp", Label: "Core 0", Current: 60.0, Max: 95.0, Critical: 100.0},
			},
		},
		Processes: []model_v2.ProcessMetrics{
			{
				PID:        1234,
				Name:       "nginx",
				Cmdline:    "nginx: master process",
				User:       "root",
				Status:     "S",
				CPUPercent: 5.0,
				MemPercent: 2.0,
				MemRSS:     100 * 1024 * 1024,
				Threads:    4,
				StartTime:  1705315800, // 2024-01-15T10:30:00Z
			},
		},
		PSI: model_v2.PSIMetrics{
			CPUSomePercent:    1.5,
			MemorySomePercent: 0.5,
			MemoryFullPercent: 0.1,
			IOSomePercent:     2.0,
			IOFullPercent:     0.3,
		},
		TCP: model_v2.TCPMetrics{
			CurrEstab:   100,
			TimeWait:    20,
			Orphan:      0,
			Alloc:       150,
			InUse:       120,
			SocketsUsed: 200,
		},
	}

	result := NodeMetricsSnapshot(src)

	// 基础字段
	if result.NodeName != "node-1" {
		t.Errorf("NodeName: got %q, want %q", result.NodeName, "node-1")
	}
	if !result.Timestamp.Equal(ts) {
		t.Errorf("Timestamp: got %v, want %v", result.Timestamp, ts)
	}

	// CPU 字段重命名: cores → coreCount, threads → threadCount, per_core → coreUsages
	if result.CPU.CoreCount != 4 {
		t.Errorf("CPU.CoreCount: got %d, want %d", result.CPU.CoreCount, 4)
	}
	if result.CPU.ThreadCount != 8 {
		t.Errorf("CPU.ThreadCount: got %d, want %d", result.CPU.ThreadCount, 8)
	}
	if len(result.CPU.CoreUsages) != 8 {
		t.Errorf("CPU.CoreUsages length: got %d, want %d", len(result.CPU.CoreUsages), 8)
	}
	// load_1 → loadAvg1
	if result.CPU.LoadAvg1 != 1.5 {
		t.Errorf("CPU.LoadAvg1: got %f, want %f", result.CPU.LoadAvg1, 1.5)
	}

	// Memory 字段重命名: total → totalBytes, used → usedBytes
	if result.Memory.TotalBytes != 16*1024*1024*1024 {
		t.Errorf("Memory.TotalBytes: got %d, want %d", result.Memory.TotalBytes, int64(16*1024*1024*1024))
	}
	if result.Memory.SwapUsagePercent != 25.0 {
		t.Errorf("Memory.SwapUsagePercent: got %f, want %f", result.Memory.SwapUsagePercent, 25.0)
	}

	// Disk IOPS 聚合: read_iops + write_iops → iops
	if len(result.Disks) != 1 {
		t.Fatalf("Disks length: got %d, want 1", len(result.Disks))
	}
	if result.Disks[0].IOPS != 150 {
		t.Errorf("Disk.IOPS: got %f, want %f (100+50)", result.Disks[0].IOPS, 150.0)
	}
	if result.Disks[0].ReadBytesPS != 1024*1024 {
		t.Errorf("Disk.ReadBytesPS: got %f, want %f", result.Disks[0].ReadBytesPS, float64(1024*1024))
	}

	// Network: rx_rate → rxBytesPS, packets/s 固定为 0
	if len(result.Networks) != 1 {
		t.Fatalf("Networks length: got %d, want 1", len(result.Networks))
	}
	if result.Networks[0].RxBytesPS != 1024*1024 {
		t.Errorf("Network.RxBytesPS: got %f, want %f", result.Networks[0].RxBytesPS, float64(1024*1024))
	}
	if result.Networks[0].RxPacketsPS != 0 {
		t.Errorf("Network.RxPacketsPS: got %f, want 0", result.Networks[0].RxPacketsPS)
	}

	// Temperature sensor: current → temp, max → high
	if len(result.Temperature.Sensors) != 1 {
		t.Fatalf("Sensors length: got %d, want 1", len(result.Temperature.Sensors))
	}
	if result.Temperature.Sensors[0].Temp != 60.0 {
		t.Errorf("Sensor.Temp: got %f, want %f (from current)", result.Temperature.Sensors[0].Temp, 60.0)
	}
	if result.Temperature.Sensors[0].High != 95.0 {
		t.Errorf("Sensor.High: got %f, want %f (from max)", result.Temperature.Sensors[0].High, 95.0)
	}

	// Process: cmdline → command, status → state, mem_rss → memBytes, start_time → startTime (ISO)
	if len(result.TopProcesses) != 1 {
		t.Fatalf("TopProcesses length: got %d, want 1", len(result.TopProcesses))
	}
	proc := result.TopProcesses[0]
	if proc.Command != "nginx: master process" {
		t.Errorf("Process.Command: got %q, want %q", proc.Command, "nginx: master process")
	}
	if proc.State != "S" {
		t.Errorf("Process.State: got %q, want %q", proc.State, "S")
	}
	if proc.MemBytes != 100*1024*1024 {
		t.Errorf("Process.MemBytes: got %d, want %d", proc.MemBytes, int64(100*1024*1024))
	}
	// StartTime 应为 ISO 8601 字符串
	if proc.StartTime == "" {
		t.Error("Process.StartTime: got empty string, want ISO 8601")
	}
	parsed, err := time.Parse(time.RFC3339, proc.StartTime)
	if err != nil {
		t.Errorf("Process.StartTime: not valid RFC3339: %v", err)
	}
	if parsed.Unix() != 1705315800 {
		t.Errorf("Process.StartTime unix: got %d, want %d", parsed.Unix(), int64(1705315800))
	}

	// PSI
	if result.PSI.CPUSomePercent != 1.5 {
		t.Errorf("PSI.CPUSomePercent: got %f, want %f", result.PSI.CPUSomePercent, 1.5)
	}

	// TCP
	if result.TCP.CurrEstab != 100 {
		t.Errorf("TCP.CurrEstab: got %d, want %d", result.TCP.CurrEstab, int64(100))
	}

	// processes → topProcesses 映射（字段名变了）
	if result.TopProcesses[0].Name != "nginx" {
		t.Errorf("TopProcesses[0].Name: got %q, want %q", result.TopProcesses[0].Name, "nginx")
	}
}

// TestNodeMetricsSnapshot_NilInput 测试 nil 输入
func TestNodeMetricsSnapshot_NilInput(t *testing.T) {
	result := NodeMetricsSnapshot(nil)
	if result.NodeName != "" {
		t.Errorf("nil input should return zero value, got NodeName=%q", result.NodeName)
	}
	if result.Disks == nil {
		t.Error("nil input should have non-nil empty Disks slice")
	}
	if result.Networks == nil {
		t.Error("nil input should have non-nil empty Networks slice")
	}
	if result.TopProcesses == nil {
		t.Error("nil input should have non-nil empty TopProcesses slice")
	}
}

// TestNodeMetricsSnapshot_EmptySlices 测试空切片
func TestNodeMetricsSnapshot_EmptySlices(t *testing.T) {
	src := &model_v2.NodeMetricsSnapshot{
		Disks:     nil,
		Networks:  nil,
		Processes: nil,
	}
	result := NodeMetricsSnapshot(src)

	if result.Disks == nil || len(result.Disks) != 0 {
		t.Errorf("Disks: expected empty non-nil slice, got %v", result.Disks)
	}
	if result.Networks == nil || len(result.Networks) != 0 {
		t.Errorf("Networks: expected empty non-nil slice, got %v", result.Networks)
	}
	if result.TopProcesses == nil || len(result.TopProcesses) != 0 {
		t.Errorf("TopProcesses: expected empty non-nil slice, got %v", result.TopProcesses)
	}
	if result.Temperature.Sensors == nil || len(result.Temperature.Sensors) != 0 {
		t.Errorf("Temperature.Sensors: expected empty non-nil slice, got %v", result.Temperature.Sensors)
	}
	if result.CPU.CoreUsages == nil || len(result.CPU.CoreUsages) != 0 {
		t.Errorf("CPU.CoreUsages: expected empty non-nil slice, got %v", result.CPU.CoreUsages)
	}
}

// TestMetricsHistoryGrouped 测试历史数据按指标分组
func TestMetricsHistoryGrouped(t *testing.T) {
	ts1 := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2025, 1, 15, 10, 5, 0, 0, time.UTC)
	src := []model_v2.MetricsDataPoint{
		{Timestamp: ts1, CPUUsage: 45.5, MemoryUsage: 60.0, DiskUsage: 40.0, CPUTemp: 65.0},
		{Timestamp: ts2, CPUUsage: 50.0, MemoryUsage: 70.0, DiskUsage: 45.0, CPUTemp: 68.0},
	}

	result := MetricsHistoryGrouped(src)

	// 必须包含 4 个指标键
	for _, key := range []string{"cpu", "memory", "disk", "temp"} {
		if _, ok := result[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
		if len(result[key]) != 2 {
			t.Fatalf("%s: got %d points, want 2", key, len(result[key]))
		}
	}

	// 检查 timestamp 为 ISO 8601 格式
	expectedTS1 := ts1.UTC().Format(time.RFC3339)
	if result["cpu"][0].Timestamp != expectedTS1 {
		t.Errorf("cpu[0].Timestamp: got %q, want %q", result["cpu"][0].Timestamp, expectedTS1)
	}

	// 检查值
	if result["cpu"][0].Value != 45.5 {
		t.Errorf("cpu[0].Value: got %f, want 45.5", result["cpu"][0].Value)
	}
	if result["memory"][1].Value != 70.0 {
		t.Errorf("memory[1].Value: got %f, want 70.0", result["memory"][1].Value)
	}
	if result["disk"][0].Value != 40.0 {
		t.Errorf("disk[0].Value: got %f, want 40.0", result["disk"][0].Value)
	}
	if result["temp"][1].Value != 68.0 {
		t.Errorf("temp[1].Value: got %f, want 68.0", result["temp"][1].Value)
	}
}

// TestClusterMetricsSummary_FieldMapping 测试集群汇总转换
func TestClusterMetricsSummary_FieldMapping(t *testing.T) {
	src := model_v2.ClusterMetricsSummary{
		TotalNodes:     6,
		OnlineNodes:    5,
		OfflineNodes:   1,
		AvgCPUUsage:    45.0,
		AvgMemoryUsage: 60.0,
		AvgDiskUsage:   35.0,
		MaxCPUUsage:    85.0,
		MaxMemoryUsage: 90.0,
		MaxDiskUsage:   70.0,
		AvgCPUTemp:     55.0,
		MaxCPUTemp:     75.0,
		TotalMemory:    96 * 1024 * 1024 * 1024,
		UsedMemory:     60 * 1024 * 1024 * 1024,
		TotalDisk:      3 * 1024 * 1024 * 1024 * 1024,
		UsedDisk:       1 * 1024 * 1024 * 1024 * 1024,
		TotalNetworkRx: 10 * 1024 * 1024,
		TotalNetworkTx: 5 * 1024 * 1024,
	}

	result := ClusterMetricsSummary(src)

	if result.TotalNodes != 6 {
		t.Errorf("TotalNodes: got %d, want %d", result.TotalNodes, 6)
	}
	if result.AvgCPUUsage != 45.0 {
		t.Errorf("AvgCPUUsage: got %f, want %f", result.AvgCPUUsage, 45.0)
	}
	if result.TotalMemory != 96*1024*1024*1024 {
		t.Errorf("TotalMemory: got %d, want %d", result.TotalMemory, int64(96*1024*1024*1024))
	}
}

// TestNodeMetricsSnapshots_Plural 测试批量转换
func TestNodeMetricsSnapshots_Plural(t *testing.T) {
	src := []*model_v2.NodeMetricsSnapshot{
		{NodeName: "node-1"},
		{NodeName: "node-2"},
	}
	result := NodeMetricsSnapshots(src)
	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
	if result[0].NodeName != "node-1" {
		t.Errorf("[0].NodeName: got %q, want %q", result[0].NodeName, "node-1")
	}
	if result[1].NodeName != "node-2" {
		t.Errorf("[1].NodeName: got %q, want %q", result[1].NodeName, "node-2")
	}
}

// TestNodeMetricsSnapshots_Nil 测试 nil 列表
func TestNodeMetricsSnapshots_Nil(t *testing.T) {
	result := NodeMetricsSnapshots(nil)
	if result == nil {
		t.Error("nil input should return non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("nil input should return empty slice, got len=%d", len(result))
	}
}

// TestMetricsHistoryGrouped_Empty 测试空输入
func TestMetricsHistoryGrouped_Empty(t *testing.T) {
	result := MetricsHistoryGrouped(nil)
	for _, key := range []string{"cpu", "memory", "disk", "temp"} {
		if result[key] == nil {
			t.Errorf("%s: should be non-nil empty slice", key)
		}
		if len(result[key]) != 0 {
			t.Errorf("%s: got %d points, want 0", key, len(result[key]))
		}
	}
}
