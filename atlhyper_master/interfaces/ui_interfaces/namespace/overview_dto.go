// ui_interfaces/namespace/dto_overview.go
package namespace

import "time"

// NamespaceOverviewDTO —— 概览：顶部卡片 + 表格
type NamespaceOverviewDTO struct {
	Cards OverviewCards     `json:"cards"`
	Rows  []NamespaceRowDTO `json:"rows"`
}

// 顶部卡片（对齐你旧版 UI：总数/Active/Terminating/总 Pod 数）
type OverviewCards struct {
	TotalNamespaces int `json:"totalNamespaces"`
	ActiveCount     int `json:"activeCount"`
	Terminating     int `json:"terminating"`
	TotalPods       int `json:"totalPods"`
}

// 表格行（与你截图列一一对应）
type NamespaceRowDTO struct {
	Name            string    `json:"name"`
	Status          string    `json:"status"` // Active / Terminating
	PodCount        int       `json:"podCount"`
	LabelCount      int       `json:"labelCount"`
	AnnotationCount int       `json:"annotationCount"`
	CreatedAt       time.Time `json:"createdAt"`
}
