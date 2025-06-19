// =======================================================================================
// 📄 interface/alert_group_builder_api.go
//
// 📦 Description:
//     Interface bridge for the Alerter module, exposing formatted alert construction logic.
//     Wraps EvaluateAlertsFromCleanedEvents and provides a unified entry point for
//     dispatchers and external modules.
//
// 🔌 Responsibilities:
//     - Evaluate whether an alert should be triggered from a set of events
//     - Construct AlertGroupData for use in email or alert display
//
// 🧩 Internal Dependency:
//     - alerter.EvaluateAlertsFromCleanedEvents
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package interfaces

import (
	"NeuroController/internal/alerter"
	"NeuroController/internal/types"
)

func ComposeAlertGroupIfNecessary(events []types.LogEvent) (bool, string, types.AlertGroupData) {
	shouldAlert, subject, data := alerter.EvaluateAlertsFromCleanedEvents(events)

	// if shouldAlert {
	// 	log.Println("📬 ComposeAlertGroupIfNecessary(): 触发邮件告警")
	// 	log.Printf("🧾 邮件标题: %s\n", subject)
	// 	log.Printf("📦 AlertGroupData: NodeList=%v, NamespaceList=%v, AlertCount=%d\n", data.NodeList, data.NamespaceList, data.AlertCount)
	// 	for _, item := range data.Alerts {
	// 		log.Printf("🔹 AlertItem: Kind=%s, Name=%s, Namespace=%s, Node=%s, Reason=%s, Message=%s, Time=%s\n",
	// 			item.Kind, item.Name, item.Namespace, item.Node, item.Reason, item.Message, item.Time)
	// 	}
	// } else {
	// 	log.Println("ℹ️ ComposeAlertGroupIfNecessary(): 暂不触发告警")
	// }

	return shouldAlert, subject, data
}
