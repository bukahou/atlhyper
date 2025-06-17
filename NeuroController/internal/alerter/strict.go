// =======================================================================================
// 📄 alerter/alerter.go
//
// ✨ 文件说明：
//     实现清理事件评估函数 EvaluateAlertsFromCleanedEvents，用于从清理后的事件集中判断是否需要触发告警。
//     告警逻辑基于 Deployment 可用性判断，当前默认以邮件方式发送聚合告警信息。
//     本模块核心职责是从清洗池构建具有人类可读性和分组展示的告警载体。
//
// 📦 核心功能：
//     - 解析 Pod 异常事件并追踪其所属 Deployment 状态
//     - 判断是否满足触发告警条件（使用内部状态机）
//     - 构造 AlertGroupData（聚合格式）用于邮件展示
//     - 使用邮件发送器进行发送（含节流控制）
//
// 🧩 模块依赖：
//     - diagnosis/types.LogEvent：来源于诊断模块的标准事件结构
//     - utils.ExtractDeploymentName：解析 Pod 所属的 Deployment 名称
//     - alerter.UpdatePodEvent：更新并判断 Deployment 是否需告警
//     - mailer.SendAlertEmailWithThrottle：封装邮件发送及节流
//
// 📝 使用建议：
//     - 推荐由定时任务或清理器回调调用此模块
//     - 后续若支持多通道（如 Slack/Webhook）可在此基础上扩展输出端
// =======================================================================================

package alerter

import (
	"NeuroController/internal/types"
	"NeuroController/internal/utils"
)

func EvaluateAlertsFromCleanedEvents(events []types.LogEvent) (bool, string, types.AlertGroupData) {
	for _, ev := range events {
		if ev.Kind != "Pod" || ev.Name == "" || ev.Name == "default" {
			continue
		}

		deploymentName := utils.ExtractDeploymentName(ev.Name, ev.Namespace)

		shouldAlert, reasonText := UpdatePodEvent(
			ev.Namespace, ev.Name, deploymentName,
			ev.ReasonCode, ev.Message, ev.Timestamp,
		)

		if shouldAlert {
			subject := reasonText
			nodeSet := make(map[string]struct{})
			nsSet := make(map[string]struct{})
			alertItems := make([]types.AlertItem, 0)

			for _, e := range events {
				nodeSet[e.Node] = struct{}{}
				nsSet[e.Namespace] = struct{}{}

				alertItems = append(alertItems, types.AlertItem{
					Kind:      e.Kind,
					Name:      e.Name,
					Namespace: e.Namespace,
					Node:      e.Node,
					Severity:  e.Severity,
					Reason:    e.ReasonCode,
					Message:   e.Message,
					Time:      e.Timestamp.Format("2006-01-02 15:04:05"),
				})
			}

			nodeList := make([]string, 0, len(nodeSet))
			for k := range nodeSet {
				nodeList = append(nodeList, k)
			}
			nsList := make([]string, 0, len(nsSet))
			for k := range nsSet {
				nsList = append(nsList, k)
			}

			data := types.AlertGroupData{
				Title:         subject,
				NodeList:      nodeList,
				NamespaceList: nsList,
				AlertCount:    len(alertItems),
				Alerts:        alertItems,
			}

			return true, subject, data
		}
	}
	return false, "", types.AlertGroupData{}
}

// ✅ EvaluateAlertsFromCleanedEvents
//
// 核心告警评估函数：输入已清洗的事件池，逐个事件进行 Pod 状态更新，判断是否触发告警。
// 一旦触发告警，将构建邮件内容并通过带节流逻辑的发送函数发送邮件。
//
// 参数：
//   - events: 来自 diagnosis 模块清洗后的事件集合
// func EvaluateAlertsFromCleanedEvents(events []types.LogEvent) {

// 	for _, ev := range events {
// 		// 🎯 只处理 Pod 类型的事件（Deployment 状态依赖于 Pod 状态）
// 		if ev.Kind != "Pod" {
// 			continue
// 		}

// 		// 🛡️ 跳过非法 Pod 名（如空字符串或 default 等特殊名）
// 		if ev.Name == "" || ev.Name == "default" {
// 			utils.Warn(context.TODO(), "⚠️ 跳过非法 Pod 名事件",
// 				zap.String("ev.Name", ev.Name),
// 				zap.String("ev.Namespace", ev.Namespace),
// 				zap.String("ev.Message", ev.Message))
// 			continue
// 		}

// 		// 🔍 提取 Deployment 名称（从 Pod 名中解析）
// 		deploymentName := utils.ExtractDeploymentName(ev.Name, ev.Namespace)

// 		// 🧠 更新 Deployment 内部状态，判断是否触发告警
// 		shouldAlert, reasonText := UpdatePodEvent(
// 			ev.Namespace, ev.Name, deploymentName,
// 			ev.ReasonCode, ev.Message, ev.Timestamp,
// 		)

// 		if shouldAlert {
// 			subject := reasonText

// 			// 📦 构造邮件数据（AlertGroupData）
// 			nodeSet := make(map[string]struct{})
// 			nsSet := make(map[string]struct{})
// 			alertItems := make([]types.AlertItem, 0)

// 			// 🚚 收集当前所有事件用于邮件展示（非只展示触发项）
// 			for _, e := range events {
// 				nodeSet[e.Node] = struct{}{}
// 				nsSet[e.Namespace] = struct{}{}

// 				alertItems = append(alertItems, types.AlertItem{
// 					Kind:      e.Kind,
// 					Name:      e.Name,
// 					Namespace: e.Namespace,
// 					Node:      e.Node,
// 					Severity:  e.Severity,
// 					Reason:    e.ReasonCode,
// 					Message:   e.Message,
// 					Time:      e.Timestamp.Format("2006-01-02 15:04:05"),
// 				})
// 			}

// 			// 📋 将 Set 转换为 List（收件方用于展示）
// 			nodeList := make([]string, 0, len(nodeSet))
// 			for k := range nodeSet {
// 				nodeList = append(nodeList, k)
// 			}
// 			nsList := make([]string, 0, len(nsSet))
// 			for k := range nsSet {
// 				nsList = append(nsList, k)
// 			}

// 			// 📄 构造最终邮件模板数据结构
// 			data := types.AlertGroupData{
// 				Title:         subject,
// 				NodeList:      nodeList,
// 				NamespaceList: nsList,
// 				AlertCount:    len(alertItems),
// 				Alerts:        alertItems,
// 			}

// 			// 📨 日志记录并发送邮件（含节流逻辑）
// 			utils.Info(context.TODO(), "📬 EvaluateAlertsFromCleanedEvents 被调用", zap.Int("事件数", len(events)))
// 			// 📬 收件人列表（由全局配置提供）
// 			recipients := config.GlobalConfig.Mailer.To
// 			err := mailer.SendAlertEmailWithThrottle(recipients, subject, data, time.Now())

// 			if err != nil {
// 				fmt.Printf("❌ 邮件发送失败: %v\n", err)
// 			}

// 			// 📛 ⚠️ 当前版本：只发送一封邮件，因此 break（如需多 Deployment 支持，请去除 break）
// 			break
// 		}
// 	}
// }
