// =======================================================================================
// 📄 alerter/alerter.go
//
// ✨ 功能说明：
//     - 提供告警评估主入口 EvaluateAlertsFromCleanedEvents，
//       从 diagnosis 模块获取清洗后的事件池，判断是否触发告警并构造邮件内容发送。
//     - 主要处理 Pod 异常事件，配合 pod_tracker.go 中的 UpdatePodEvent 使用。
//     - 内部使用 throttle.go 中的节流机制避免频繁告警。
//
// 🧩 模块依赖：
//     - diagnosis.LogEvent：用于输入清理后的结构化事件。
//     - utils.ExtractDeploymentName：提取 Deployment 名称（根据 Pod 命名推导）。
//     - mailer.AlertGroupData：构造邮件数据。
//     - SendAlertEmailWithThrottle：节流发送邮件。
//
// 📦 提供函数：
//     - EvaluateAlertsFromCleanedEvents([]diagnosis.LogEvent)
//
// 📝 使用说明：
//     - 应在清理完成后调用，例如由控制器或定时任务驱动：
//           diagnosis.RebuildCleanedEventPool()
//           alerter.EvaluateAlertsFromCleanedEvents(diagnosis.GetCleanedEventPool())
//
// 📁 所属模块：alerter （告警判断与发送模块）
// =======================================================================================

package alerter

import (
	"NeuroController/config"
	"NeuroController/internal/mailer"
	"NeuroController/internal/types"
	"NeuroController/internal/utils"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

func EvaluateAlertsFromCleanedEvents(events []types.LogEvent) {

	recipients := config.GlobalConfig.Mailer.To

	for _, ev := range events {
		if ev.Kind != "Pod" {
			continue
		}

		// 🛡️ 新增：过滤非法 Pod 名
		if ev.Name == "" || ev.Name == "default" {
			utils.Warn(context.TODO(), "⚠️ 跳过非法 Pod 名事件",
				zap.String("ev.Name", ev.Name),
				zap.String("ev.Namespace", ev.Namespace),
				zap.String("ev.Message", ev.Message))
			continue
		}

		deploymentName := utils.ExtractDeploymentName(ev.Name, ev.Namespace)

		shouldAlert, reasonText := UpdatePodEvent(
			ev.Namespace, ev.Name, deploymentName,
			ev.ReasonCode, ev.Message, ev.Timestamp,
		)

		if shouldAlert {
			subject := reasonText

			// ✅ 构造 AlertGroupData，提取节点、命名空间等
			nodeSet := make(map[string]struct{})
			nsSet := make(map[string]struct{})
			alertItems := make([]mailer.AlertItem, 0)

			for _, e := range events {
				nodeSet[e.Node] = struct{}{}
				nsSet[e.Namespace] = struct{}{}
				alertItems = append(alertItems, mailer.AlertItem{
					Kind:      e.Kind,
					Name:      e.Name,
					Namespace: e.Namespace,
					Node:      e.Node, // ✅ 补充
					Severity:  e.Severity,
					Reason:    e.ReasonCode, // ✅ 此处应与字段名匹配，如果是 ReasonCode，保持一致
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

			data := mailer.AlertGroupData{
				Title:         subject,
				NodeList:      nodeList,
				NamespaceList: nsList,
				AlertCount:    len(alertItems),
				Alerts:        alertItems,
			}

			utils.Info(context.TODO(), "📬 EvaluateAlertsFromCleanedEvents 被调用", zap.Int("事件数", len(events)))
			err := SendAlertEmailWithThrottle(recipients, subject, data, time.Now())

			if err != nil {
				fmt.Printf("❌ 邮件发送失败: %v\n", err)
			}
			break
		}
	}
}
