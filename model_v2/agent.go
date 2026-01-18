package model_v2

import "time"

// ============================================================
// Agent 状态模型
// ============================================================

// AgentStatus Agent 状态
//
// Master 维护的 Agent 连接状态。
type AgentStatus struct {
	ClusterID     string    `json:"cluster_id"`     // 集群 ID
	Status        string    `json:"status"`         // online, offline
	LastHeartbeat time.Time `json:"last_heartbeat"` // 最后心跳时间
	LastSnapshot  time.Time `json:"last_snapshot"`  // 最后快照时间
}

// Agent 状态常量
const (
	AgentStatusOnline  = "online"
	AgentStatusOffline = "offline"
)

// IsOnline 判断 Agent 是否在线
func (a *AgentStatus) IsOnline() bool {
	return a.Status == AgentStatusOnline
}

// AgentInfo Agent 信息（用于列表）
//
// 与 AgentStatus 类似，用于 ListAgents 接口。
type AgentInfo struct {
	ClusterID     string    `json:"cluster_id"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	LastSnapshot  time.Time `json:"last_snapshot"`
}

// ============================================================
// ClusterInfo 集群信息
// ============================================================

// ClusterInfo 集群基础信息
//
// 用于集群列表展示。
type ClusterInfo struct {
	ClusterID string    `json:"cluster_id"` // 集群 ID
	Status    string    `json:"status"`     // Agent 状态
	LastSeen  time.Time `json:"last_seen"`  // 最后活跃时间
	NodeCount int       `json:"node_count"` // 节点数量
	PodCount  int       `json:"pod_count"`  // Pod 数量
}

// ClusterDetail 集群详情
//
// 包含完整的快照和 Agent 状态。
type ClusterDetail struct {
	ClusterID string           `json:"cluster_id"`
	Status    *AgentStatus     `json:"status,omitempty"`
	Snapshot  *ClusterSnapshot `json:"snapshot,omitempty"`
}
