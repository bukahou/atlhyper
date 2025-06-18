// =======================================================================================
// 📄 interface/alert_group_lightweight_api.go
//
// 📦 Description:
//     提供轻量级告警展示接口，无需严格触发判断，适用于 Slack 可视化场景。
//     封装 FormatAllEventsLight，统一对外展示事件概览格式。
//
// 🔌 Responsibilities:
//     - 获取清洗池数据（GetCleanedEventLogs）
//     - 使用轻量格式 FormatAllEventsLight 构造 AlertGroupData
//
// 🧩 内部依赖：
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
