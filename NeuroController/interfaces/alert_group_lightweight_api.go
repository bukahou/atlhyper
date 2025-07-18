// =======================================================================================
// 📄 interface/alert_group_lightweight_api.go
//
// 📦 Description:
//     Provides a lightweight alert display interface without strict triggering logic,
//     suitable for visual summaries (e.g., Slack alert views).
//     Wraps FormatAllEventsLight to generate unified event overviews.
//
// 🔌 Responsibilities:
//     - Fetch cleaned event pool (GetCleanedEventLogs)
//     - Build AlertGroupData using FormatAllEventsLight
//
// 🧩 Internal Dependency:
//     - alerter.FormatAllEventsLight
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package interfaces

import (
	"NeuroController/internal/alerter"
	"NeuroController/internal/types"
	"NeuroController/model"
)

// ✅ 提供给外部模块（如 Slack）用于轻量展示事件概览
func GetLightweightAlertGroup(events []model.LogEvent) (bool, string, types.AlertGroupData) {
	shouldDisplay, title, data := alerter.FormatAllEventsLight(events)

	return shouldDisplay, title, data
}
