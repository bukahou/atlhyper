package overview

import (
	event "AtlHyper/model/event"
	"time"
)

// ====================== 顶层返回 DTO ======================

type OverviewDTO struct {
	ClusterID string         `json:"clusterId"`
	Cards     OverviewCards  `json:"cards"`
	Trends    OverviewTrends `json:"trends"` // 仅资源趋势
	Alerts    OverviewAlerts `json:"alerts"` // Alerts 专属：Totals + Trend + Recent
	Nodes     NodeSection    `json:"nodes"`
}

// ====================== 卡片区 ======================

type OverviewCards struct {
	ClusterHealth ClusterHealthCard `json:"clusterHealth"`
	NodeReady     NodeReadyCard     `json:"nodeReady"`
	CPUUsage      UsageCard         `json:"cpuUsage"`
	MemUsage      UsageCard         `json:"memUsage"`
	Events24h     int               `json:"events24h"`
}

type ClusterHealthCard struct {
	PodReadyPercent  float64 `json:"podReadyPercent"`
	NodeReadyPercent float64 `json:"nodeReadyPercent"`
	Status           string  `json:"status"` // Healthy/Degraded/NoData
}

type NodeReadyCard struct {
	Total   int     `json:"total"`
	Ready   int     `json:"ready"`
	Percent float64 `json:"percent"`
}

type UsageCard struct {
	Percent float64 `json:"percent"`
}

// ====================== 趋势区（仅资源） ======================

type OverviewTrends struct {
	ResourceUsage []ResourceTrendPoint `json:"resourceUsage"`
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

// ====================== Alerts 区 ======================

type OverviewAlerts struct {
	Totals SeverityTotals     `json:"totals"` // 24h 总计
	Trend  []AlertHourlyPoint `json:"trend"`  // 24h 小时分桶
	Recent []event.LogEvent   `json:"recent"`
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

// ====================== Node Usage 区 ======================

type NodeSection struct {
	Usage []NodeUsageRow `json:"usage"`
}

type NodeUsageRow struct {
	Node     string  `json:"node"`
	CPUUsage float64 `json:"cpuUsage"`
	MemUsage float64 `json:"memUsage"`
}
