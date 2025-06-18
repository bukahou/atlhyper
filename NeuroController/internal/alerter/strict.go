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
	"NeuroController/internal/monitor"
	"NeuroController/internal/types"
	"NeuroController/internal/utils"
	"fmt"
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

			// nodeList := make([]string, 0, len(nodeSet))
			// for k := range nodeSet {
			// 	nodeList = append(nodeList, k)
			// }
			// nsList := make([]string, 0, len(nsSet))
			// for k := range nsSet {
			// 	nsList = append(nsList, k)
			// }

			// 获取节点指标
			nodeMetrics := monitor.GetNodeResourceUsage()

			nodeList := make([]string, 0, len(nodeSet))
			for nodeName := range nodeSet {
				if usage, ok := nodeMetrics[nodeName]; ok {
					nodeList = append(nodeList,
						fmt.Sprintf("%s (CPU: %s, Mem: %s)", nodeName, usage.CPUUsage, usage.MemoryUsage),
					)
				} else {
					nodeList = append(nodeList, nodeName)
				}
			}

			nsList := make([]string, 0, len(nsSet))
			for ns := range nsSet {
				nsList = append(nsList, ns)
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
