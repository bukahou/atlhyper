package metrics

import "time"

// 概览页：卡片 + 表格
type NodeMetricsOverviewDTO struct {
	Cards OverviewCards     `json:"cards"`
	Rows  []NodeMetricsRow  `json:"rows"`
}

type OverviewCards struct {
	AvgCPUPercent   float64 `json:"avgCPUPercent"`   // 全集群平均 CPU%
	AvgMemPercent   float64 `json:"avgMemPercent"`   // 全集群平均内存%
	PeakTempC       float64 `json:"peakTempC"`       // 峰值温度(℃)
	PeakTempNode    string  `json:"peakTempNode"`    // 峰值温度节点
	PeakDiskPercent float64 `json:"peakDiskPercent"` // 峰值磁盘使用%
	PeakDiskNode    string  `json:"peakDiskNode"`    // 峰值磁盘节点
}

// 表格行（与你截图一致）
type NodeMetricsRow struct {
	Node         string    `json:"node"`
	CPUPercent   float64   `json:"cpuPercent"`
	MemPercent   float64   `json:"memPercent"`
	CPUTempC     float64   `json:"cpuTempC"`
	DiskUsedPct  float64   `json:"diskUsedPercent"` // 主挂载点或聚合后最大值
	Eth0TxKBps   float64   `json:"eth0TxKBps"`
	Eth0RxKBps   float64   `json:"eth0RxKBps"`
	TopCPUProc   string    `json:"topCPUProcess"`   // 例如 "k3s-server"
	Timestamp    time.Time `json:"timestamp"`
}
