// atlhyper_master_v2/processor/processor.go
// Processor 数据处理层
// 负责接收 AgentSDK 传来的数据，进行校验，写入 DataHub
package processor

import (
	"fmt"
	"sync"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var log = logger.Module("Processor")

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
	onSnapshotReceived func(clusterID string)                                          // 快照接收回调（触发 Event/Metrics 持久化）
	onSLODataReceived  func(clusterID string, snapshot *model_v2.ClusterSnapshot) // SLO 数据回调

	// 状态追踪（用于变化检测）
	mu         sync.RWMutex
	lastCounts map[string]snapshotCounts
}

type snapshotCounts struct {
	pods   int
	nodes  int
	events int
}

// Config Processor 配置
type Config struct {
	Store              datahub.Store
	OnSnapshotReceived func(clusterID string)
	OnSLODataReceived  func(clusterID string, snapshot *model_v2.ClusterSnapshot)
}

// New 创建 Processor
func New(cfg Config) Processor {
	return &processorImpl{
		store:              cfg.Store,
		onSnapshotReceived: cfg.OnSnapshotReceived,
		onSLODataReceived:  cfg.OnSLODataReceived,
		lastCounts:         make(map[string]snapshotCounts),
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

	// 3. 检测资源数量变化，决定日志级别
	current := snapshotCounts{
		pods:   len(snapshot.Pods),
		nodes:  len(snapshot.Nodes),
		events: len(snapshot.Events),
	}

	p.mu.RLock()
	last, exists := p.lastCounts[clusterID]
	p.mu.RUnlock()

	if !exists || current != last {
		// 首次或资源数量变化，输出 INFO 日志
		log.Info("快照处理完成",
			"cluster", clusterID,
			"pods", current.pods,
			"nodes", current.nodes,
			"events", current.events,
		)
		p.mu.Lock()
		p.lastCounts[clusterID] = current
		p.mu.Unlock()
	} else {
		// 无变化，输出 DEBUG 日志
		log.Debug("快照处理完成",
			"cluster", clusterID,
			"pods", current.pods,
			"nodes", current.nodes,
			"events", current.events,
		)
	}

	// 4. 触发回调（Event/Metrics 持久化）
	if p.onSnapshotReceived != nil {
		p.onSnapshotReceived(clusterID)
	}

	// 5. 处理 SLO 数据（如果有）
	if snapshot.SLOData != nil && p.onSLODataReceived != nil {
		p.onSLODataReceived(clusterID, snapshot)
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
