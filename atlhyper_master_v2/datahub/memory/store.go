// atlhyper_master_v2/datahub/memory/store.go
// MemoryStore 内存存储实现
package memory

import (
	"sync"
	"time"

	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
	"AtlHyper/model_v3/cluster"
)

var log = logger.Module("MemoryStore")

// 默认 OTel 时间线容量：15min / 10s = 90 条
const defaultOTelRingCapacity = 90

// MemoryStore 内存数据存储
type MemoryStore struct {
	// 快照存储
	snapshots   map[string]*model_v2.ClusterSnapshot
	snapshotsMu sync.RWMutex

	// OTel 时间线（Ring Buffer per cluster）
	otelTimeline   map[string]*OTelRing
	otelTimelineMu sync.RWMutex

	// Agent 状态
	agents   map[string]*model_v2.AgentInfo
	agentsMu sync.RWMutex

	// 配置
	eventRetention    time.Duration
	heartbeatExpire   time.Duration
	snapshotRetention time.Duration

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New 创建 MemoryStore
func New(eventRetention, heartbeatExpire, snapshotRetention time.Duration) *MemoryStore {
	if snapshotRetention <= 0 {
		snapshotRetention = 15 * time.Minute
	}
	return &MemoryStore{
		snapshots:         make(map[string]*model_v2.ClusterSnapshot),
		otelTimeline:      make(map[string]*OTelRing),
		agents:            make(map[string]*model_v2.AgentInfo),
		eventRetention:    eventRetention,
		heartbeatExpire:   heartbeatExpire,
		snapshotRetention: snapshotRetention,
		stopCh:            make(chan struct{}),
	}
}

// Start 启动 MemoryStore
func (s *MemoryStore) Start() error {
	s.wg.Add(1)
	go s.cleanupLoop()

	log.Info("已启动", "snapshotRetention", s.snapshotRetention)
	return nil
}

// Stop 停止 MemoryStore
func (s *MemoryStore) Stop() error {
	close(s.stopCh)
	s.wg.Wait()
	log.Info("已停止")
	return nil
}

// cleanupLoop 定期清理过期数据
func (s *MemoryStore) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.updateAgentStatus()
			s.cleanupOfflineClusterData()
		}
	}
}

// updateAgentStatus 更新 Agent 在线状态
func (s *MemoryStore) updateAgentStatus() {
	s.agentsMu.Lock()
	defer s.agentsMu.Unlock()

	cutoff := time.Now().Add(-s.heartbeatExpire)

	for clusterID, agent := range s.agents {
		if agent.LastHeartbeat.Before(cutoff) && agent.Status == model_v2.AgentStatusOnline {
			agent.Status = model_v2.AgentStatusOffline
			log.Warn("Agent 已标记为离线", "cluster", clusterID)
		}
	}
}

// cleanupOfflineClusterData 清理离线集群的 OTel 时间线数据
func (s *MemoryStore) cleanupOfflineClusterData() {
	s.agentsMu.RLock()
	offlineClusters := make([]string, 0)
	offlineCutoff := time.Now().Add(-s.snapshotRetention * 2) // 离线超过 2 倍保留时间才清理
	for clusterID, agent := range s.agents {
		if agent.Status == model_v2.AgentStatusOffline && agent.LastHeartbeat.Before(offlineCutoff) {
			offlineClusters = append(offlineClusters, clusterID)
		}
	}
	s.agentsMu.RUnlock()

	if len(offlineClusters) == 0 {
		return
	}

	s.otelTimelineMu.Lock()
	for _, clusterID := range offlineClusters {
		if _, ok := s.otelTimeline[clusterID]; ok {
			delete(s.otelTimeline, clusterID)
			log.Info("已清理离线集群 OTel 时间线", "cluster", clusterID)
		}
	}
	s.otelTimelineMu.Unlock()
}

// ==================== 快照管理 ====================

// SetSnapshot 存储集群快照
func (s *MemoryStore) SetSnapshot(clusterID string, snapshot *model_v2.ClusterSnapshot) error {
	s.snapshotsMu.Lock()
	s.snapshots[clusterID] = snapshot
	s.snapshotsMu.Unlock()

	// 追加 OTel 到时间线
	if snapshot.OTel != nil {
		s.appendOTel(clusterID, snapshot.OTel, snapshot.FetchedAt)
	}

	// 同时更新 Agent 状态
	s.agentsMu.Lock()
	if agent, ok := s.agents[clusterID]; ok {
		agent.LastSnapshot = snapshot.FetchedAt
	} else {
		s.agents[clusterID] = &model_v2.AgentInfo{
			ClusterID:     clusterID,
			Status:        model_v2.AgentStatusOnline,
			LastHeartbeat: time.Now(),
			LastSnapshot:  snapshot.FetchedAt,
		}
	}
	s.agentsMu.Unlock()

	return nil
}

// appendOTel 追加 OTel 快照到时间线
func (s *MemoryStore) appendOTel(clusterID string, otel *cluster.OTelSnapshot, ts time.Time) {
	s.otelTimelineMu.Lock()
	defer s.otelTimelineMu.Unlock()

	ring, ok := s.otelTimeline[clusterID]
	if !ok {
		ring = NewOTelRing(defaultOTelRingCapacity)
		s.otelTimeline[clusterID] = ring
	}
	ring.Add(otel, ts)
}

// GetSnapshot 获取集群快照
func (s *MemoryStore) GetSnapshot(clusterID string) (*model_v2.ClusterSnapshot, error) {
	s.snapshotsMu.RLock()
	defer s.snapshotsMu.RUnlock()

	snapshot, ok := s.snapshots[clusterID]
	if !ok {
		return nil, nil
	}
	return snapshot, nil
}

// ==================== Agent 状态 ====================

// UpdateHeartbeat 更新 Agent 心跳
func (s *MemoryStore) UpdateHeartbeat(clusterID string) error {
	s.agentsMu.Lock()
	defer s.agentsMu.Unlock()

	if agent, ok := s.agents[clusterID]; ok {
		agent.LastHeartbeat = time.Now()
		agent.Status = model_v2.AgentStatusOnline
	} else {
		s.agents[clusterID] = &model_v2.AgentInfo{
			ClusterID:     clusterID,
			Status:        model_v2.AgentStatusOnline,
			LastHeartbeat: time.Now(),
		}
	}
	return nil
}

// GetAgentStatus 获取 Agent 状态
func (s *MemoryStore) GetAgentStatus(clusterID string) (*model_v2.AgentStatus, error) {
	s.agentsMu.RLock()
	defer s.agentsMu.RUnlock()

	agent, ok := s.agents[clusterID]
	if !ok {
		return nil, nil
	}
	return &model_v2.AgentStatus{
		ClusterID:     agent.ClusterID,
		Status:        agent.Status,
		LastHeartbeat: agent.LastHeartbeat,
		LastSnapshot:  agent.LastSnapshot,
	}, nil
}

// ListAgents 列出所有 Agent
func (s *MemoryStore) ListAgents() ([]model_v2.AgentInfo, error) {
	s.agentsMu.RLock()
	defer s.agentsMu.RUnlock()

	result := make([]model_v2.AgentInfo, 0, len(s.agents))
	for _, agent := range s.agents {
		result = append(result, *agent)
	}
	return result, nil
}

// ==================== Event 查询 ====================

// GetEvents 获取集群当前所有 Events
func (s *MemoryStore) GetEvents(clusterID string) ([]model_v2.Event, error) {
	s.snapshotsMu.RLock()
	defer s.snapshotsMu.RUnlock()

	snapshot, ok := s.snapshots[clusterID]
	if !ok {
		return nil, nil
	}
	return snapshot.Events, nil
}

// ==================== OTel 时间线 ====================

// GetOTelTimeline 获取 OTel 时间线数据
func (s *MemoryStore) GetOTelTimeline(clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	s.otelTimelineMu.RLock()
	defer s.otelTimelineMu.RUnlock()

	ring, ok := s.otelTimeline[clusterID]
	if !ok {
		return nil, nil
	}

	snapshots, timestamps := ring.Since(since)
	if len(snapshots) == 0 {
		return nil, nil
	}

	entries := make([]cluster.OTelEntry, len(snapshots))
	for i := range snapshots {
		entries[i] = cluster.OTelEntry{
			Snapshot:  snapshots[i],
			Timestamp: timestamps[i],
		}
	}
	return entries, nil
}
