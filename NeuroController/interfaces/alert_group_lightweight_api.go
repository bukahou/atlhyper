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
)

// ✅ 提供给外部模块（如 Slack）用于轻量展示事件概览
func GetLightweightAlertGroup(events []types.LogEvent) (bool, string, types.AlertGroupData) {
	shouldDisplay, title, data := alerter.FormatAllEventsLight(events)

	// if shouldDisplay {
	// 	log.Println("📋 GetLightweightAlertGroup(): 构建轻量级事件概览")
	// 	log.Printf("🧾 标题: %s\n", title)
	// 	log.Printf("📦 AlertGroupData: NodeList=%v, NamespaceList=%v, AlertCount=%d\n", data.NodeList, data.NamespaceList, data.AlertCount)
	// 	for _, item := range data.Alerts {
	// 		log.Printf("🔹 AlertItem: Kind=%s, Name=%s, Namespace=%s, Node=%s, Reason=%s, Message=%s, Time=%s\n",
	// 			item.Kind, item.Name, item.Namespace, item.Node, item.Reason, item.Message, item.Time)
	// 	}
	// } else {
	// 	log.Println("ℹ️ GetLightweightAlertGroup(): 当前无事件概览可展示")
	// }

	return shouldDisplay, title, data
}
