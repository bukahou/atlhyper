// atlhyper_master_v2/notifier/source/interfaces.go
// 告警来源：把事件变化、Agent 心跳、K8s Warning 事件变成统一的告警，提交给通知管道
package source

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/common/logger"
	agentmodel "AtlHyper/model_v3/agent"
)

var log = logger.Module("NotifySource")

// AlertSubmitter 通知管道的入口。两个方法都只入队、立即返回；队列满时返回 false
type AlertSubmitter interface {
	Submit(a alert.Alert) bool
	// SubmitDeferred 入队一个构造函数：管道读到当前策略后调用它，由它决定是否产出告警
	SubmitDeferred(build func(policy *database.NotifyPolicy) (alert.Alert, bool)) bool
}

// FiringAlertReader 读库里持续中的告警。
// 心跳与 K8s 事件两个来源自己不记状态，每轮从这里看「哪些告警在持续」，Master 重启后不会丢
type FiringAlertReader interface {
	ListFiring(ctx context.Context) ([]*database.NotifyAlert, error)
}

// PolicyReader 读当前的通知策略
type PolicyReader interface {
	Get(ctx context.Context) (*database.NotifyPolicy, error)
}

// AgentLister 列出各集群 Agent 的在线状态
type AgentLister interface {
	ListAgents() ([]agentmodel.AgentInfo, error)
}

// ClusterEventReader 读持久化的 K8s Warning 事件
type ClusterEventReader interface {
	GetLatestEventID(ctx context.Context) (int64, error)
	GetEventsSince(ctx context.Context, sinceID int64) ([]*database.ClusterEvent, error)
	ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*database.ClusterEvent, error)
}
