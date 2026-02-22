// Package agent 定义 Agent 状态和集群信息模型
package agent

import (
	"time"

	"AtlHyper/model_v3/cluster"
)

// AgentStatus Agent 连接状态
type AgentStatus struct {
	ClusterID     string    `json:"clusterId"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	LastSnapshot  time.Time `json:"lastSnapshot"`
}

const (
	StatusOnline  = "online"
	StatusOffline = "offline"
)

func (a *AgentStatus) IsOnline() bool { return a.Status == StatusOnline }

// AgentInfo Agent 信息（列表用）
type AgentInfo struct {
	ClusterID     string    `json:"clusterId"`
	Status        string    `json:"status"`
	LastHeartbeat time.Time `json:"lastHeartbeat"`
	LastSnapshot  time.Time `json:"lastSnapshot"`
}

// ClusterInfo 集群基础信息（列表展示）
type ClusterInfo struct {
	ClusterID string    `json:"clusterId"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"lastSeen"`
	NodeCount int       `json:"nodeCount"`
	PodCount  int       `json:"podCount"`
}

// ClusterDetail 集群详情
type ClusterDetail struct {
	ClusterID string                   `json:"clusterId"`
	Status    *AgentStatus             `json:"status,omitempty"`
	Snapshot  *cluster.ClusterSnapshot `json:"snapshot,omitempty"`
}
