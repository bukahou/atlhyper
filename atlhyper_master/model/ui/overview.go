// atlhyper_master/dto/ui/overview.go
// Overview UI DTOs (Dashboard)
package ui

import (
	"time"

	"AtlHyper/model/transport"
)

// OverviewDTO - 顶层返回
type OverviewDTO struct {
	ClusterID string                `json:"clusterId"`
	Cards     OverviewCardsDTO      `json:"cards"`
	Trends    OverviewTrendsDTO     `json:"trends"`
	Alerts    OverviewAlertsDTO     `json:"alerts"`
	Nodes     OverviewNodeSection   `json:"nodes"`
}

// ====================== Cards ======================

type OverviewCardsDTO struct {
	ClusterHealth ClusterHealthCard `json:"clusterHealth"`
	NodeReady     NodeReadyCard     `json:"nodeReady"`
	CPUUsage      UsageCard         `json:"cpuUsage"`
	MemUsage      UsageCard         `json:"memUsage"`
	Events24h     int               `json:"events24h"`
}

type ClusterHealthCard struct {
	PodReadyPercent  float64 `json:"podReadyPercent"`
	NodeReadyPercent float64 `json:"nodeReadyPercent"`
	Status           string  `json:"status"`
}

type NodeReadyCard struct {
	Total   int     `json:"total"`
	Ready   int     `json:"ready"`
	Percent float64 `json:"percent"`
}

type UsageCard struct {
	Percent float64 `json:"percent"`
}

// ====================== Trends ======================

type OverviewTrendsDTO struct {
	ResourceUsage []ResourceTrendPoint `json:"resourceUsage"`
	PeakStats     TrendPeakStats       `json:"peakStats"`
}

type ResourceTrendPoint struct {
	At           time.Time `json:"at"`
	CPUPeak      float64   `json:"cpuPeak"`
	CPUPeakNode  string    `json:"cpuPeakNode"`
	MemPeak      float64   `json:"memPeak"`
	MemPeakNode  string    `json:"memPeakNode"`
	TempPeak     float64   `json:"tempPeak"`
	TempPeakNode string    `json:"tempPeakNode"`
}

// TrendPeakStats 当前峰值状态（用于底部状态卡片）
type TrendPeakStats struct {
	PeakCPU      float64 `json:"peakCpu"`      // 当前最高 CPU 使用率 %
	PeakCPUNode  string  `json:"peakCpuNode"`  // 最高 CPU 节点名
	PeakMem      float64 `json:"peakMem"`      // 当前最高内存使用率 %
	PeakMemNode  string  `json:"peakMemNode"`  // 最高内存节点名
	PeakTemp     float64 `json:"peakTemp"`     // 当前最高温度 ℃
	PeakTempNode string  `json:"peakTempNode"` // 最高温度节点名
	NetRxKBps    float64 `json:"netRxKBps"`    // 集群总入流量 KB/s
	NetTxKBps    float64 `json:"netTxKBps"`    // 集群总出流量 KB/s
	HasData      bool    `json:"hasData"`      // 是否有 metrics 插件数据
}

// ====================== Alerts ======================

type OverviewAlertsDTO struct {
	Totals SeverityTotals       `json:"totals"`
	Trend  []AlertHourlyPoint   `json:"trend"`
	Recent []transport.LogEvent `json:"recent"`
}

type SeverityTotals struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

type AlertHourlyPoint struct {
	At       time.Time `json:"at"`
	Critical int       `json:"critical"`
	Warning  int       `json:"warning"`
	Info     int       `json:"info"`
}

// ====================== Nodes ======================

type OverviewNodeSection struct {
	Usage []OverviewNodeUsageRow `json:"usage"`
}

type OverviewNodeUsageRow struct {
	Node     string  `json:"node"`
	CPUUsage float64 `json:"cpuUsage"`
	MemUsage float64 `json:"memUsage"`
}
