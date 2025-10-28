// atlhyper_master/client/alert/alert_group_ai.go
package alert

import (
	event "AtlHyper/model/event"
)

//
// ClusterEventGroup —— 表示按集群整合后的事件包
// ------------------------------------------------------------
// 仅用于 AI 分析模块的输入准备阶段。
// 每个结构体对应一个集群的增量事件集合。
//
type ClusterEventGroup struct {
	ClusterID string           `json:"clusterID"`
	Events    []event.EventLog `json:"events"`
	Count     int              `json:"count"`
}

//
// CollectNewEventsGroupedForAI —— 整合增量事件以供 AI 分析
// ------------------------------------------------------------
// 🧠 功能说明：
//   - 调用 CollectNewEventLogsForAlert() 获取最新的增量事件。
//   - 按 ClusterID 分组整合（每个集群一组）。
//   - 不做过滤、不做网络请求。
//   - 结果供上层 handler 或调度逻辑调用，用于发送至 AI Service。
//
// ✅ 返回值：
//   - []ClusterEventGroup ：每个集群一组事件。
//   - 若无新事件，返回 nil。
//
func CollectNewEventsGroupedForAI() []ClusterEventGroup {
	// 1️⃣ 收集所有增量事件
	events := CollectNewEventLogsForAlert()
	if len(events) == 0 {
		return nil
	}

	// 2️⃣ 按 ClusterID 聚合
	grouped := make(map[string][]event.EventLog)
	for _, e := range events {
		clusterID := e.ClusterID
		if clusterID == "" {
			clusterID = "unknown"
		}
		grouped[clusterID] = append(grouped[clusterID], e)
	}

	// 3️⃣ 构建返回结构
	out := make([]ClusterEventGroup, 0, len(grouped))
	for clusterID, list := range grouped {
		out = append(out, ClusterEventGroup{
			ClusterID: clusterID,
			Events:    list,
			Count:     len(list),
		})
	}

	return out
}
