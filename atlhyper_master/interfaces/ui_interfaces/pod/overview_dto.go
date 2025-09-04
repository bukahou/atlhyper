// interfaces/ui_api/pod/dto.go
package pod

import "time"

// PodOverviewDTO —— Pod 概览页返回结构
type PodOverviewDTO struct {
    Cards PodCards          `json:"cards"` // 顶部卡片区
    Pods  []PodOverviewItem `json:"pods"`  // 表格区
}

type PodCards struct {
    Running int `json:"running"`
    Pending int `json:"pending"`
    Failed  int `json:"failed"`
    Unknown int `json:"unknown"`
}

type PodOverviewItem struct {
    Namespace  string     `json:"namespace"`
    Deployment string     `json:"deployment,omitempty"`
    Name       string     `json:"name"`
    Ready      string     `json:"ready"`
    Phase      string     `json:"phase"`
    Restarts   int32      `json:"restarts"`

    // 数值字段（保持你的原口径）
    CPU        float64    `json:"cpu"`        // 单位：core
    CPUPercent float64    `json:"cpuPercent"` // 0-100
    Memory     int        `json:"memory"`     // 单位：m（≈Mi）
    MemPercent float64    `json:"memPercent"` // 0-100

    // 展示字段（新增：带单位/百分号，直接可渲染）
    CPUText        string `json:"cpuText,omitempty"`        // 例如 "1m" / "125m" / "0m"
    CPUPercentText string `json:"cpuPercentText,omitempty"` // 例如 "0.100%"
    MemoryText     string `json:"memoryText,omitempty"`     // 例如 "13 m"
    MemPercentText string `json:"memPercentText,omitempty"` // 例如 "2.600%"

    StartTime  time.Time  `json:"startTime"`
    Node       string     `json:"node"`
}
