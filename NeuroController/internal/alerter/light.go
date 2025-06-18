// =======================================================================================
// 📄 internal/alerter/light.go
//
// 🧊 Description:
//     提供轻量级告警格式化逻辑，用于展示当前清洗池中的所有事件，
//     不依赖 Deployment 异常副本比例或持续时间，仅作为事件总览用。
//     适用于 Slack 或其他需要快速提示的渠道。
//
// 🔎 特点：
//     - 不包含任何触发判断（无 shouldAlert 判断）
//     - 仅负责格式化清洗池中已有的 LogEvent 为 AlertGroupData
//     - 所有资源种类均可纳入格式化（不限于 Pod）
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package alerter

import (
	"NeuroController/internal/monitor"
	"NeuroController/internal/types"
	"fmt"
)

// ✅ 轻量格式化告警信息（不含触发逻辑）
func FormatAllEventsLight(events []types.LogEvent) (bool, string, types.AlertGroupData) {
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

	title := "📋 当前全告警事件"
	data := types.AlertGroupData{
		Title:         title,
		NodeList:      nodeList,
		NamespaceList: nsList,
		AlertCount:    len(alertItems),
		Alerts:        alertItems,
	}

	return true, title, data
}
