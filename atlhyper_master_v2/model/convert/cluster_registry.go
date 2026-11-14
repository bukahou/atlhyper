// atlhyper_master_v2/model/convert/cluster_registry.go
// database.Cluster + agent.AgentInfo → model.ClusterRegistryItem 转换函数
package convert

import (
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/agent"
)

// ClusterRegistryItem 把一个集群的登记信息、在线状态与认证状态拼成集群管理页的一行。
//
// registered 为 nil 表示未登记（只在上报）；presence 为 nil 表示从未上报。
// 两者不会同时为 nil —— 那样的集群不存在于任何一侧。
// 令牌哈希只用来判断是否已签发，不写入结果。
func ClusterRegistryItem(registered *database.Cluster, presence *agent.AgentInfo, authenticated bool) model.ClusterRegistryItem {
	item := model.ClusterRegistryItem{
		Status:        model.ClusterStatusNever,
		Authenticated: authenticated,
	}
	if registered != nil {
		item.ClusterID = registered.ClusterID
		item.Name = registered.Name
		item.Description = registered.Description
		item.Registered = true
		item.TokenIssued = registered.TokenHash != ""
		item.TokenPrefix = registered.TokenPrefix
		if registered.TokenIssuedAt != nil {
			item.TokenIssuedAt = formatUTC(*registered.TokenIssuedAt)
		}
	}
	if presence != nil {
		item.ClusterID = presence.ClusterID
		item.Status = clusterPresenceStatus(presence.Status)
		item.LastSeen = formatUTC(laterOf(presence.LastHeartbeat, presence.LastSnapshot))
	}
	return item
}

// clusterPresenceStatus 上报过的集群只有在线与离线两种；认不出的取值按离线，不冒充在线
func clusterPresenceStatus(agentStatus string) string {
	if agentStatus == agent.StatusOnline {
		return model.ClusterStatusOnline
	}
	return model.ClusterStatusOffline
}

func laterOf(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// formatUTC 统一按 UTC 输出，零值返回空串。
// 注册表的时间来自数据库与内存两处，时区各不相同，统一后前端不必分辨
func formatUTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeFormat)
}
