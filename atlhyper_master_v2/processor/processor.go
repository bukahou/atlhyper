// atlhyper_master_v2/processor/processor.go
// Processor 数据处理层
// 负责接收 AgentSDK 传来的数据，进行校验，写入 DataHub
package processor

import (
	"fmt"
	"strings"
	"sync"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/cluster"
)

var log = logger.Module("Processor")

// Processor 数据处理器接口
type Processor interface {
	// ProcessSnapshot 处理集群快照
	// 接收 Agent 上报的快照，校验后写入 DataHub
	ProcessSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error

	// ProcessHeartbeat 处理心跳
	ProcessHeartbeat(clusterID string) error
}

// processorImpl Processor 实现
type processorImpl struct {
	store              datahub.Store
	onSnapshotReceived func(clusterID string) // 快照接收回调（触发 Event/Metrics/SLO 持久化）

	// 状态追踪（用于变化检测）
	mu         sync.RWMutex
	lastCounts map[string]snapshotCounts
	lastStale  map[string]string // 集群 → 上一份快照里没采到的资源类别（变化时才记日志）
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
}

// NewProcessor 创建 Processor
func NewProcessor(cfg Config) Processor {
	return &processorImpl{
		store:              cfg.Store,
		onSnapshotReceived: cfg.OnSnapshotReceived,
		lastCounts:         make(map[string]snapshotCounts),
		lastStale:          make(map[string]string),
	}
}

// ProcessSnapshot 处理集群快照
func (p *processorImpl) ProcessSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
	// 1. 校验
	if err := p.validateSnapshot(clusterID, snapshot); err != nil {
		return fmt.Errorf("validate snapshot: %w", err)
	}

	// 2. 清洗：整份清单副本类注解不进入存储（不依赖上报方已经清理过）
	snapshot.DropManifestCopyAnnotations()

	// 3. 写入 DataHub
	if err := p.store.SetSnapshot(clusterID, snapshot); err != nil {
		return fmt.Errorf("set snapshot: %w", err)
	}

	// 4. 检测资源数量变化，决定日志级别
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

	// 5. Agent 有资源类别没采到（沿用旧数据或已留空）：变化时记一条，不在每份快照里重复
	p.logStaleKindsChange(clusterID, snapshot.StaleKinds)

	// 6. 触发回调（Event/Metrics/SLO 持久化）
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
func (p *processorImpl) validateSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
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

// staleKindsKey 把没采到的类别写成可比较的字符串：deployments（沿用）/ ingresses（留空）
func staleKindsKey(kinds []cluster.StaleKind) string {
	parts := make([]string, 0, len(kinds))
	for _, k := range kinds {
		state := "沿用"
		if k.Unavailable {
			state = "留空"
		}
		parts = append(parts, k.Kind+"（"+state+"）")
	}
	return strings.Join(parts, " / ")
}

func (p *processorImpl) logStaleKindsChange(clusterID string, kinds []cluster.StaleKind) {
	key := staleKindsKey(kinds)
	p.mu.Lock()
	previous, seen := p.lastStale[clusterID]
	p.lastStale[clusterID] = key
	p.mu.Unlock()
	if previous == key || (!seen && key == "") {
		return
	}
	if key == "" {
		log.Info("集群的资源类别已全部采到", "cluster", clusterID, "之前", previous)
		return
	}
	log.Warn("集群有资源类别本轮没采到", "cluster", clusterID, "类别", key)
}
