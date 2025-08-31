package pod

import "time"

// PodOverviewDTO —— Pod 概览页返回结构
type PodOverviewDTO struct {
    Cards PodCards          `json:"cards"` // 顶部卡片区
    Pods  []PodOverviewItem `json:"pods"`  // 表格区
}

// ====================== 卡片区 ======================

type PodCards struct {
    Running int `json:"running"`
    Pending int `json:"pending"`
    Failed  int `json:"failed"`
    Unknown int `json:"unknown"`
}

// ====================== 表格区 ======================

type PodOverviewItem struct {
    Namespace  string    `json:"namespace"`
    Deployment string    `json:"deployment,omitempty"` // ControlledBy.Name
    Name       string    `json:"name"`
    Ready      string    `json:"ready"`
    Phase      string    `json:"phase"`
    Restarts   int32     `json:"restarts"`
    CPU        string    `json:"cpu,omitempty"`        // metrics.CPU.Usage
    CPUPercent float64   `json:"cpuPercent,omitempty"` // metrics.CPU.UtilPct
    Memory     string    `json:"memory,omitempty"`
    MemPercent float64   `json:"memPercent,omitempty"`
    StartTime  time.Time `json:"startTime"`
    Node       string    `json:"node"`
}
