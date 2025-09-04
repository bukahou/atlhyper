// interfaces/ui_interfaces/event/dto.go
package event

import model "AtlHyper/model/event"

// 卡片统计
type EventCards struct {
	TotalAlerts     int `json:"totalAlerts"`     // 数据总数（len(rows)）
	TotalEvents     int `json:"totalEvents"`     // Category == "Event" 的数量
	Warning         int `json:"warning"`         // Severity=warning 数
	Info            int `json:"info"`            // Severity=info/normal 数
	Error           int `json:"error"`           // Severity=error 数（没有则为 0）
	CategoriesCount int `json:"categoriesCount"` // 类别去重数
	KindsCount      int `json:"kindsCount"`      // 资源种类去重数
}

// 总体返回
type EventOverviewDTO struct {
	Cards EventCards       `json:"cards"`
	Rows  []model.EventLog `json:"rows"`
}
