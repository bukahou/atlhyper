// atlhyper_master/dto/ui/event.go
// Event UI DTOs
package ui

import "AtlHyper/model/transport"

// EventOverviewDTO - 总体返回
type EventOverviewDTO struct {
	Cards EventCards            `json:"cards"`
	Rows  []transport.EventLog  `json:"rows"`
}

// EventCards - 卡片统计
type EventCards struct {
	TotalAlerts     int `json:"totalAlerts"`
	TotalEvents     int `json:"totalEvents"`
	Warning         int `json:"warning"`
	Info            int `json:"info"`
	Error           int `json:"error"`
	CategoriesCount int `json:"categoriesCount"`
	KindsCount      int `json:"kindsCount"`
}
