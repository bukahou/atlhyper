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
	return alerter.EvaluateAlertsFromCleanedEvents(events)
}
