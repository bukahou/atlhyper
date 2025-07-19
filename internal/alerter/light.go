// =======================================================================================
// 📄 internal/alerter/light.go
//
// 🧊 Description:
//     Provides lightweight alert formatting logic to present all current events in the
//     cleaned event pool. This does not rely on Deployment-level thresholds or duration.
//     Primarily intended for quick overviews in channels like Slack.
//
// 🔎 Features:
//     - No alert triggering logic (no shouldAlert evaluation)
//     - Only responsible for formatting existing LogEvent data into AlertGroupData
//     - Supports all resource kinds (not limited to Pods)
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package alerter

import (
	"NeuroController/internal/monitor"
	"NeuroController/internal/types"
	"NeuroController/model"
	"fmt"
)

// ✅ 轻量格式化告警信息（不含触发逻辑）
func FormatAllEventsLight(events []model.LogEvent) (bool, string, types.AlertGroupData) {
	if len(events) == 0 {
		return false, "", types.AlertGroupData{}
	}

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

	// 🎯 获取节点资源使用情况
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

	title := "📋 現在発生中の全アラート一覧"
	data := types.AlertGroupData{
		Title:         title,
		NodeList:      nodeList,
		NamespaceList: nsList,
		AlertCount:    len(alertItems),
		Alerts:        alertItems,
	}

	return true, title, data
}
