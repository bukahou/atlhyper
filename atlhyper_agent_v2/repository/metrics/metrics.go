package metrics

import (
	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v2"
)

// metricsRepository 节点指标仓库实现
// 从 SDK ReceiverClient 拉取数据，不维护自身存储
type metricsRepository struct {
	receiver sdk.ReceiverClient
}

// NewMetricsRepository 创建节点指标仓库
// receiver 为 SDK 层的数据接收服务器，负责暂存推送数据
func NewMetricsRepository(receiver sdk.ReceiverClient) repository.MetricsRepository {
	return &metricsRepository{receiver: receiver}
}

// GetAll 获取所有节点的指标
// 委托给 ReceiverClient 拉取最新快照
func (r *metricsRepository) GetAll() map[string]*model_v2.NodeMetricsSnapshot {
	return r.receiver.GetAllNodeMetrics()
}
