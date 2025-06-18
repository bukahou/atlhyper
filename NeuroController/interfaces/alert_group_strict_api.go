// =======================================================================================
// 📄 interface/alert_group_builder_api.go
//
// 📦 Description:
//     Alerter 模块的接口桥接层，暴露格式化告警构建函数。
//     封装 EvaluateAlertsFromCleanedEvents，提供统一调用点给调度器或 external 模块。
//
// 🔌 Responsibilities:
//     - 从事件集合中评估是否触发告警
//     - 构造用于邮件/告警展示的 AlertGroupData 数据结构
//
// 🧩 内部依赖：
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
