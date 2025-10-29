package alert

import (
	event "AtlHyper/model/event"
)

// ClusterEventGroup —— 表示按集群整合后的事件包
// ------------------------------------------------------------
// ✅ 此结构现在直接符合 AI Service /ai/diagnose 的请求格式：
//    { "clusterID": "...", "events": [...] }
type ClusterEventGroup struct {
	ClusterID string           `json:"clusterID"` // 集群唯一标识
	Events    []event.EventLog `json:"events"`    // 该集群的事件列表
}

// CollectNewEventsGroupedForAI —— 整合增量事件以供 AI 分析
// ------------------------------------------------------------
// 🧠 功能说明：
//   - 从数据库或缓存收集最新事件（由 CollectNewEventLogsForAlert 提供）
//   - 按 ClusterID 聚合事件
//   - 返回结构体切片，可直接作为 /ai/diagnose POST 请求体使用
//
// ✅ 返回值：
//   - []ClusterEventGroup ：每个元素均符合 AI Service 的 JSON 请求格式
//   - 若无新事件，返回 nil
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

	// 3️⃣ 构建返回结构（符合 AI 请求格式）
	out := make([]ClusterEventGroup, 0, len(grouped))
	for clusterID, list := range grouped {
		out = append(out, ClusterEventGroup{
			ClusterID: clusterID,
			Events:    list,
		})
	}

	return out
}
