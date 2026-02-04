// atlhyper_master_v2/datahub/memory/store.go
// MemoryStore 内存存储实现
package memory

import (
	"sync"
	"time"

	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var log = logger.Module("MemoryStore")

// MemoryStore 内存数据存储
type MemoryStore struct {
	// 快照存储
	snapshots   map[string]*model_v2.ClusterSnapshot
	snapshotsMu sync.RWMutex

	// Agent 状态
	agents   map[string]*model_v2.AgentInfo
	agentsMu sync.RWMutex

	// 配置
	eventRetention  time.Duration
	heartbeatExpire time.Duration

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New 创建 MemoryStore
func New(eventRetention, heartbeatExpire time.Duration) *MemoryStore {
	return &MemoryStore{
		snapshots:       make(map[string]*model_v2.ClusterSnapshot),
		agents:          make(map[string]*model_v2.AgentInfo),
		eventRetention:  eventRetention,
		heartbeatExpire: heartbeatExpire,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动 MemoryStore
func (s *MemoryStore) Start() error {
	// 启动过期数据清理协程
	s.wg.Add(1)
	go s.cleanupLoop()

	log.Info("已启动")
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
			s.cleanupExpiredEvents()
			s.updateAgentStatus()
		}
	}
}

// cleanupExpiredEvents 清理过期 Events
func (s *MemoryStore) cleanupExpiredEvents() {
	s.snapshotsMu.Lock()
	defer s.snapshotsMu.Unlock()

	cutoff := time.Now().Add(-s.eventRetention)

	for clusterID, snapshot := range s.snapshots {
		beforeCount := len(snapshot.Events)
		var validEvents []model_v2.Event
		for _, e := range snapshot.Events {
			if e.LastTimestamp.After(cutoff) {
				validEvents = append(validEvents, e)
			}
		}
		if len(validEvents) < beforeCount {
			snapshot.Events = validEvents
			// 有事件被清理时输出 INFO
			log.Info("已清理过期事件",
				"cluster", clusterID,
				"before", beforeCount,
				"after", len(validEvents),
			)
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

// ==================== 快照管理 ====================

// SetSnapshot 存储集群快照
func (s *MemoryStore) SetSnapshot(clusterID string, snapshot *model_v2.ClusterSnapshot) error {
	s.snapshotsMu.Lock()
	defer s.snapshotsMu.Unlock()

	s.snapshots[clusterID] = snapshot

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
