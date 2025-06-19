// =======================================================================================
// 📄 alerter/alerter.go
//
// ✨ Description:
//     Implements the EvaluateAlertsFromCleanedEvents function, which evaluates whether
//     an alert should be triggered based on cleaned events.
//     The logic centers around Deployment availability and aggregates alerts for
//     email delivery.
//
// 📦 Responsibilities:
//     - Parse abnormal Pod events and track their parent Deployment state
//     - Determine if alert conditions are met using internal state machines
//     - Build human-readable and grouped AlertGroupData for email notification
//     - Send alert email using throttled mailer logic
//
// 🧩 Dependencies:
//     - diagnosis/types.LogEvent: normalized event structure from diagnosis module
//     - utils.ExtractDeploymentName: extracts Deployment name from Pod name
//     - alerter.UpdatePodEvent: updates internal state and evaluates alert conditions
//     - mailer.SendAlertEmailWithThrottle: email dispatch function with throttling
//
// 📝 Usage Recommendation:
//     - Recommended to be invoked periodically or via diagnosis module callbacks
//     - Future support for multi-channel alerts (e.g. Slack/Webhook) can be added here
//
// ✍️ Author: bukahou (@ZGMF-X10A)
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
