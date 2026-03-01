package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v3/metrics"
)

// TestNodeMetricsSnapshot_FieldMapping 验证关键字段映射
func TestNodeMetricsSnapshot_FieldMapping(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	src := &metrics.NodeMetrics{
		NodeName:  "node-1",
		Timestamp: ts,
		Kernel:    "6.1.0",
		Uptime:    86400,
		CPU: metrics.NodeCPU{
			UsagePct: 45.5,
			Cores:    4,
			Load1:    1.5,
			Load5:    2.0,
			Load15:   1.8,
		},
		Memory: metrics.NodeMemory{
			TotalBytes:     16 * 1024 * 1024 * 1024,
			AvailableBytes: 7 * 1024 * 1024 * 1024,
			FreeBytes:      5 * 1024 * 1024 * 1024,
			UsagePct:       50.0,
			SwapTotalBytes: 4 * 1024 * 1024 * 1024,
			SwapFreeBytes:  3 * 1024 * 1024 * 1024,
			SwapUsagePct:   25.0,
			CachedBytes:    2 * 1024 * 1024 * 1024,
			BuffersBytes:   512 * 1024 * 1024,
		},
		Disks: []metrics.NodeDisk{
			{
				Device:           "sda1",
				MountPoint:       "/",
				FSType:           "ext4",
				TotalBytes:       500 * 1024 * 1024 * 1024,
				AvailBytes:       280 * 1024 * 1024 * 1024,
				UsagePct:         40.0,
				ReadBytesPerSec:  1024 * 1024,
				WriteBytesPerSec: 512 * 1024,
				ReadIOPS:         100,
				WriteIOPS:        50,
				IOUtilPct:        30.0,
			},
		},
		Networks: []metrics.NodeNetwork{
			{
				Interface:     "eth0",
				Up:            true,
				SpeedBps:      1000,
				RxBytesPerSec: 1024 * 1024,
				TxBytesPerSec: 512 * 1024,
			},
		},
		Temperature: metrics.NodeTemperature{
			CPUTempC: 65.0,
			CPUMaxC:  100.0,
			Sensors: []metrics.TempSensor{
				{Chip: "coretemp", Sensor: "Core 0", CurrentC: 60.0, MaxC: 95.0, CritC: 100.0},
			},
		},
		PSI: metrics.NodePSI{
			CPUSomePct: 1.5,
			MemSomePct: 0.5,
			MemFullPct: 0.1,
			IOSomePct:  2.0,
			IOFullPct:  0.3,
		},
		TCP: metrics.NodeTCP{
			CurrEstab:   100,
			TimeWait:    20,
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

	// CPU: usagePct → usagePercent, cores → coreCount
	if result.CPU.CoreCount != 4 {
		t.Errorf("CPU.CoreCount: got %d, want %d", result.CPU.CoreCount, 4)
	}
	if result.CPU.LoadAvg1 != 1.5 {
		t.Errorf("CPU.LoadAvg1: got %f, want %f", result.CPU.LoadAvg1, 1.5)
	}

	// Memory: totalBytes, usagePct → usagePercent
	if result.Memory.TotalBytes != 16*1024*1024*1024 {
		t.Errorf("Memory.TotalBytes: got %d, want %d", result.Memory.TotalBytes, int64(16*1024*1024*1024))
	}
	// UsedBytes = TotalBytes - AvailableBytes
	expectedUsed := int64(16*1024*1024*1024) - int64(7*1024*1024*1024)
	if result.Memory.UsedBytes != expectedUsed {
		t.Errorf("Memory.UsedBytes: got %d, want %d", result.Memory.UsedBytes, expectedUsed)
	}
	if result.Memory.SwapUsagePercent != 25.0 {
		t.Errorf("Memory.SwapUsagePercent: got %f, want %f", result.Memory.SwapUsagePercent, 25.0)
	}

	// Disk IOPS 聚合: readIOPS + writeIOPS → iops
	if len(result.Disks) != 1 {
		t.Fatalf("Disks length: got %d, want 1", len(result.Disks))
	}
	if result.Disks[0].IOPS != 150 {
		t.Errorf("Disk.IOPS: got %f, want %f (100+50)", result.Disks[0].IOPS, 150.0)
	}
	if result.Disks[0].ReadBytesPS != 1024*1024 {
		t.Errorf("Disk.ReadBytesPS: got %f, want %f", result.Disks[0].ReadBytesPS, float64(1024*1024))
	}

	// Network: rxBytesPerSec → rxBytesPS
	if len(result.Networks) != 1 {
		t.Fatalf("Networks length: got %d, want 1", len(result.Networks))
	}
	if result.Networks[0].RxBytesPS != 1024*1024 {
		t.Errorf("Network.RxBytesPS: got %f, want %f", result.Networks[0].RxBytesPS, float64(1024*1024))
	}
	if result.Networks[0].Status != "up" {
		t.Errorf("Network.Status: got %q, want %q", result.Networks[0].Status, "up")
	}

	// Temperature sensor: chip → name, sensor → label, currentC → temp, maxC → high
	if len(result.Temperature.Sensors) != 1 {
		t.Fatalf("Sensors length: got %d, want 1", len(result.Temperature.Sensors))
	}
	if result.Temperature.Sensors[0].Temp != 60.0 {
		t.Errorf("Sensor.Temp: got %f, want %f (from currentC)", result.Temperature.Sensors[0].Temp, 60.0)
	}
	if result.Temperature.Sensors[0].High != 95.0 {
		t.Errorf("Sensor.High: got %f, want %f (from maxC)", result.Temperature.Sensors[0].High, 95.0)
	}

	// TopProcesses: v3 没有 ProcessMetrics，应为空
	if len(result.TopProcesses) != 0 {
		t.Errorf("TopProcesses length: got %d, want 0", len(result.TopProcesses))
	}

	// PSI
	if result.PSI.CPUSomePercent != 1.5 {
		t.Errorf("PSI.CPUSomePercent: got %f, want %f", result.PSI.CPUSomePercent, 1.5)
	}

	// TCP
	if result.TCP.CurrEstab != 100 {
		t.Errorf("TCP.CurrEstab: got %d, want %d", result.TCP.CurrEstab, int64(100))
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
	src := &metrics.NodeMetrics{
		Disks:    nil,
		Networks: nil,
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

// TestClusterMetricsSummary_FieldMapping 测试集群汇总转换
func TestClusterMetricsSummary_FieldMapping(t *testing.T) {
	src := metrics.Summary{
		TotalNodes:  6,
		OnlineNodes: 5,
		AvgCPUPct:   45.0,
		AvgMemPct:   60.0,
		MaxCPUPct:   85.0,
		MaxMemPct:   90.0,
		MaxCPUTemp:  75.0,
	}

	result := ClusterMetricsSummary(src)

	if result.TotalNodes != 6 {
		t.Errorf("TotalNodes: got %d, want %d", result.TotalNodes, 6)
	}
	if result.AvgCPUUsage != 45.0 {
		t.Errorf("AvgCPUUsage: got %f, want %f", result.AvgCPUUsage, 45.0)
	}
	if result.OnlineNodes != 5 {
		t.Errorf("OnlineNodes: got %d, want %d", result.OnlineNodes, 5)
	}
	if result.MaxCPUTemp != 75.0 {
		t.Errorf("MaxCPUTemp: got %f, want %f", result.MaxCPUTemp, 75.0)
	}
}

// TestNodeMetricsSnapshots_Plural 测试批量转换
func TestNodeMetricsSnapshots_Plural(t *testing.T) {
	src := []*metrics.NodeMetrics{
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
