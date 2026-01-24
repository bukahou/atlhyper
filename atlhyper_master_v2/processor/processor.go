// atlhyper_master_v2/processor/processor.go
// Processor 数据处理层
// 负责接收 AgentSDK 传来的数据，进行校验，写入 DataHub
package processor

import (
	"fmt"
	"log"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/model_v2"
)

// Processor 数据处理器接口
type Processor interface {
	// ProcessSnapshot 处理集群快照
	// 接收 Agent 上报的快照，校验后写入 DataHub
	ProcessSnapshot(clusterID string, snapshot *model_v2.ClusterSnapshot) error

	// ProcessHeartbeat 处理心跳
	ProcessHeartbeat(clusterID string) error
}

// processorImpl Processor 实现
type processorImpl struct {
	store              datahub.Store
	onSnapshotReceived func(clusterID string) // 快照接收回调（触发 Event 持久化）
}

// Config Processor 配置
type Config struct {
	Store              datahub.Store
	OnSnapshotReceived func(clusterID string)
}

// New 创建 Processor
func New(cfg Config) Processor {
	return &processorImpl{
		store:              cfg.Store,
		onSnapshotReceived: cfg.OnSnapshotReceived,
	}
}

// ProcessSnapshot 处理集群快照
func (p *processorImpl) ProcessSnapshot(clusterID string, snapshot *model_v2.ClusterSnapshot) error {
	// 1. 校验
	if err := p.validateSnapshot(clusterID, snapshot); err != nil {
		return fmt.Errorf("validate snapshot: %w", err)
	}

	// 2. 写入 DataHub
	if err := p.store.SetSnapshot(clusterID, snapshot); err != nil {
		return fmt.Errorf("set snapshot: %w", err)
	}

	log.Printf("[Processor] 快照处理完成: 集群=%s, Pods=%d, Nodes=%d, Events=%d",
		clusterID, len(snapshot.Pods), len(snapshot.Nodes), len(snapshot.Events))

	// 3. 触发回调（Event 持久化）
	if p.onSnapshotReceived != nil {
		p.onSnapshotReceived(clusterID)
	}

	return nil
}

// ProcessHeartbeat 处理心跳
func (p *processorImpl) ProcessHeartbeat(clusterID string) error {
	if clusterID == "" {
		return fmt.Errorf("cluster_id required")
	}
	return p.store.UpdateHeartbeat(clusterID)
}

// validateSnapshot 校验快照
func (p *processorImpl) validateSnapshot(clusterID string, snapshot *model_v2.ClusterSnapshot) error {
	if clusterID == "" {
		return fmt.Errorf("cluster_id required")
	}
	if snapshot == nil {
		return fmt.Errorf("snapshot required")
	}
	if snapshot.ClusterID != "" && snapshot.ClusterID != clusterID {
		return fmt.Errorf("cluster_id mismatch: path=%s, body=%s", clusterID, snapshot.ClusterID)
	}
	return nil
}
