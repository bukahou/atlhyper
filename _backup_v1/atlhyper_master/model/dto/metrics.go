// atlhyper_master/dto/ui/metrics.go
// Metrics UI DTOs
package dto

import (
	"time"

	mod "AtlHyper/model/collect"
)

// ====================== Overview ======================

// MetricsOverviewDTO - 概览页
type MetricsOverviewDTO struct {
	Cards MetricsOverviewCards `json:"cards"`
	Rows  []MetricsNodeRow     `json:"rows"`
}

type MetricsOverviewCards struct {
	AvgCPUPercent   float64 `json:"avgCPUPercent"`
	AvgMemPercent   float64 `json:"avgMemPercent"`
	PeakTempC       float64 `json:"peakTempC"`
	PeakTempNode    string  `json:"peakTempNode"`
	PeakDiskPercent float64 `json:"peakDiskPercent"`
	PeakDiskNode    string  `json:"peakDiskNode"`
}

type MetricsNodeRow struct {
	Node        string    `json:"node"`
	CPUPercent  float64   `json:"cpuPercent"`
	MemPercent  float64   `json:"memPercent"`
	CPUTempC    float64   `json:"cpuTempC"`
	DiskUsedPct float64   `json:"diskUsedPercent"`
	Eth0TxKBps  float64   `json:"eth0TxKBps"`
	Eth0RxKBps  float64   `json:"eth0RxKBps"`
	TopCPUProc  string    `json:"topCPUProcess"`
	Timestamp   time.Time `json:"timestamp"`
}

// ====================== Detail ======================

// MetricsDetailDTO - 详情页：单节点时间序列 + 最新快照
type MetricsDetailDTO struct {
	Node      string              `json:"node"`
	Latest    MetricsNodeRow      `json:"latest"`
	Series    MetricsSeries       `json:"series"`
	Processes []mod.TopCPUProcess `json:"processes"`
	TimeRange struct {
		Since time.Time `json:"since"`
		Until time.Time `json:"until"`
	} `json:"timeRange"`
}

type MetricsSeries struct {
	At      []time.Time `json:"at"`
	CPUPct  []float64   `json:"cpuPct"`
	MemPct  []float64   `json:"memPct"`
	TempC   []float64   `json:"tempC"`
	DiskPct []float64   `json:"diskPct"`
	Eth0Tx  []float64   `json:"eth0TxKBps"`
	Eth0Rx  []float64   `json:"eth0RxKBps"`
}
