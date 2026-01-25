// atlhyper_master/client/alert/build_group.go
package alert

import (
	"AtlHyper/model/integration"
	"sort"
	"time"
)

const hardcodedTitle = "集群告警信息"

func BuildAlertGroupFromEvents() integration.LightweightAlertStub {
	rows := CollectNewEventLogsForAlert()

	// 无数据：返回占位，Display=false
	if len(rows) == 0 {
		return integration.LightweightAlertStub{
			Title:   hardcodedTitle,
			Data:    integration.AlertGroupData{},
			Display: false,
		}
	}

	clusterSet := map[string]struct{}{}
	nsSet := map[string]struct{}{}
	nodeSet := map[string]struct{}{}
	items := make([]integration.AlertItem, 0, len(rows))

	for _, e := range rows {
		// 去重收集汇总字段
		if e.ClusterID != "" {
			clusterSet[e.ClusterID] = struct{}{}
		}
		if e.Namespace != "" {
			nsSet[e.Namespace] = struct{}{}
		}
		if e.Node != "" {
			nodeSet[e.Node] = struct{}{}
		}

		// 充填明细
		items = append(items, integration.AlertItem{
			ClusterID: e.ClusterID,
			Kind:      e.Kind,
			Name:      e.Name,
			Namespace: e.Namespace,
			Node:      e.Node,
			Severity:  e.Severity,
			Reason:    e.Reason,
			Message:   e.Message,
			Time:      safeTime(e.EventTime),
		})
	}

	data := integration.AlertGroupData{
		Title:         hardcodedTitle,
		ClusterID:     toSortedList(clusterSet),
		NodeList:      toSortedList(nodeSet),      // 未来可在这里拼 CPU/Mem 注释
		NamespaceList: toSortedList(nsSet),
		AlertCount:    len(items),
		Alerts:        items,
	}

	return integration.LightweightAlertStub{
		Title:   hardcodedTitle,
		Data:    data,
		Display: true,
	}
}

// ---------- helpers ----------
func toSortedList(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func safeTime(s string) string {
	if s == "" {
		return time.Now().Format(time.RFC3339)
	}
	return s
}
