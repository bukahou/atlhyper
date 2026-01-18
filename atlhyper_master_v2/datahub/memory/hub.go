// atlhyper_master_v2/datahub/memory/hub.go
// MemoryHub 内存实现
package memory

import (
	"context"
	"log"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// MemoryHub 内存数据中心
type MemoryHub struct {
	// 快照存储
	snapshots   map[string]*model_v2.ClusterSnapshot
	snapshotsMu sync.RWMutex

	// Agent 状态
	agents   map[string]*model_v2.AgentInfo
	agentsMu sync.RWMutex

	// 指令队列
	queues   map[string]*commandQueue
	queuesMu sync.RWMutex

	// 指令状态（全局，用于查询）
	commands   map[string]*model.CommandStatus
	commandsMu sync.RWMutex

	// 指令结果等待者（用于同步等待）
	commandWaiters   map[string]chan *model.CommandResult
	commandWaitersMu sync.Mutex

	// 配置
	eventRetention  time.Duration
	heartbeatExpire time.Duration

	// 控制
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// commandQueue 单个集群的指令队列
type commandQueue struct {
	commands []*model.Command
	waiting  chan struct{} // 通知等待者
	mu       sync.Mutex
}

// New 创建 MemoryHub
func New(eventRetention, heartbeatExpire time.Duration) *MemoryHub {
	return &MemoryHub{
		snapshots:       make(map[string]*model_v2.ClusterSnapshot),
		agents:          make(map[string]*model_v2.AgentInfo),
		queues:          make(map[string]*commandQueue),
		commands:        make(map[string]*model.CommandStatus),
		commandWaiters:  make(map[string]chan *model.CommandResult),
		eventRetention:  eventRetention,
		heartbeatExpire: heartbeatExpire,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动 MemoryHub
func (h *MemoryHub) Start() error {
	// 启动过期数据清理协程
	h.wg.Add(1)
	go h.cleanupLoop()

	log.Println("[MemoryHub] 已启动")
	return nil
}

// Stop 停止 MemoryHub
func (h *MemoryHub) Stop() error {
	close(h.stopCh)
	h.wg.Wait()
	log.Println("[MemoryHub] 已停止")
	return nil
}

// cleanupLoop 定期清理过期数据
func (h *MemoryHub) cleanupLoop() {
	defer h.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.cleanupExpiredEvents()
			h.updateAgentStatus()
		}
	}
}

// cleanupExpiredEvents 清理过期 Events
func (h *MemoryHub) cleanupExpiredEvents() {
	h.snapshotsMu.Lock()
	defer h.snapshotsMu.Unlock()

	cutoff := time.Now().Add(-h.eventRetention)

	for clusterID, snapshot := range h.snapshots {
		var validEvents []model_v2.Event
		for _, e := range snapshot.Events {
			if e.LastTimestamp.After(cutoff) {
				validEvents = append(validEvents, e)
			}
		}
		if len(validEvents) < len(snapshot.Events) {
			snapshot.Events = validEvents
			log.Printf("[MemoryHub] 已清理过期事件: 集群=%s, %d -> %d",
				clusterID, len(snapshot.Events), len(validEvents))
		}
	}
}

// updateAgentStatus 更新 Agent 在线状态
func (h *MemoryHub) updateAgentStatus() {
	h.agentsMu.Lock()
	defer h.agentsMu.Unlock()

	cutoff := time.Now().Add(-h.heartbeatExpire)

	for clusterID, agent := range h.agents {
		if agent.LastHeartbeat.Before(cutoff) && agent.Status == model_v2.AgentStatusOnline {
			agent.Status = model_v2.AgentStatusOffline
			log.Printf("[MemoryHub] Agent 已标记为离线: 集群=%s", clusterID)
		}
	}
}

// ==================== 快照管理 ====================

// SetSnapshot 存储集群快照
func (h *MemoryHub) SetSnapshot(clusterID string, snapshot *model_v2.ClusterSnapshot) error {
	h.snapshotsMu.Lock()
	defer h.snapshotsMu.Unlock()

	h.snapshots[clusterID] = snapshot

	// 同时更新 Agent 状态
	h.agentsMu.Lock()
	if agent, ok := h.agents[clusterID]; ok {
		agent.LastSnapshot = snapshot.FetchedAt
	} else {
		h.agents[clusterID] = &model_v2.AgentInfo{
			ClusterID:     clusterID,
			Status:        model_v2.AgentStatusOnline,
			LastHeartbeat: time.Now(),
			LastSnapshot:  snapshot.FetchedAt,
		}
	}
	h.agentsMu.Unlock()

	return nil
}

// GetSnapshot 获取集群快照
func (h *MemoryHub) GetSnapshot(clusterID string) (*model_v2.ClusterSnapshot, error) {
	h.snapshotsMu.RLock()
	defer h.snapshotsMu.RUnlock()

	snapshot, ok := h.snapshots[clusterID]
	if !ok {
		return nil, nil
	}
	return snapshot, nil
}

// ==================== Agent 状态 ====================

// UpdateHeartbeat 更新 Agent 心跳
func (h *MemoryHub) UpdateHeartbeat(clusterID string) error {
	h.agentsMu.Lock()
	defer h.agentsMu.Unlock()

	if agent, ok := h.agents[clusterID]; ok {
		agent.LastHeartbeat = time.Now()
		agent.Status = model_v2.AgentStatusOnline
	} else {
		h.agents[clusterID] = &model_v2.AgentInfo{
			ClusterID:     clusterID,
			Status:        model_v2.AgentStatusOnline,
			LastHeartbeat: time.Now(),
		}
	}
	return nil
}

// GetAgentStatus 获取 Agent 状态
func (h *MemoryHub) GetAgentStatus(clusterID string) (*model_v2.AgentStatus, error) {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()

	agent, ok := h.agents[clusterID]
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
func (h *MemoryHub) ListAgents() ([]model_v2.AgentInfo, error) {
	h.agentsMu.RLock()
	defer h.agentsMu.RUnlock()

	result := make([]model_v2.AgentInfo, 0, len(h.agents))
	for _, agent := range h.agents {
		result = append(result, *agent)
	}
	return result, nil
}

// ==================== 指令队列 ====================

// getOrCreateQueue 获取或创建队列
func (h *MemoryHub) getOrCreateQueue(clusterID string) *commandQueue {
	h.queuesMu.Lock()
	defer h.queuesMu.Unlock()

	if q, ok := h.queues[clusterID]; ok {
		return q
	}
	q := &commandQueue{
		commands: make([]*model.Command, 0),
		waiting:  make(chan struct{}, 1),
	}
	h.queues[clusterID] = q
	return q
}

// EnqueueCommand 入队指令
func (h *MemoryHub) EnqueueCommand(clusterID string, cmd *model.Command) error {
	q := h.getOrCreateQueue(clusterID)

	q.mu.Lock()
	q.commands = append(q.commands, cmd)
	q.mu.Unlock()

	// 记录指令状态
	h.commandsMu.Lock()
	h.commands[cmd.ID] = &model.CommandStatus{
		CommandID: cmd.ID,
		Status:    model.CommandStatusPending,
		CreatedAt: cmd.CreatedAt,
	}
	h.commandsMu.Unlock()

	// 通知等待者
	select {
	case q.waiting <- struct{}{}:
	default:
	}

	log.Printf("[MemoryHub] 指令已入队: %s -> %s", cmd.ID, clusterID)
	return nil
}

// WaitCommand 等待指令（长轮询）
func (h *MemoryHub) WaitCommand(ctx context.Context, clusterID string, timeout time.Duration) (*model.Command, error) {
	q := h.getOrCreateQueue(clusterID)

	// 先检查是否有待处理的指令
	q.mu.Lock()
	if len(q.commands) > 0 {
		cmd := q.commands[0]
		q.commands = q.commands[1:]
		q.mu.Unlock()

		// 更新状态
		h.updateCommandStatus(cmd.ID, model.CommandStatusRunning)
		return cmd, nil
	}
	q.mu.Unlock()

	// 没有指令，等待
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, nil // 超时，返回 nil
	case <-q.waiting:
		// 有新指令
		q.mu.Lock()
		if len(q.commands) > 0 {
			cmd := q.commands[0]
			q.commands = q.commands[1:]
			q.mu.Unlock()
			h.updateCommandStatus(cmd.ID, model.CommandStatusRunning)
			return cmd, nil
		}
		q.mu.Unlock()
		return nil, nil
	}
}

// updateCommandStatus 更新指令状态
func (h *MemoryHub) updateCommandStatus(cmdID, status string) {
	h.commandsMu.Lock()
	defer h.commandsMu.Unlock()

	if cs, ok := h.commands[cmdID]; ok {
		cs.Status = status
		if status == model.CommandStatusRunning {
			now := time.Now()
			cs.StartedAt = &now
		}
	}
}

// AckCommand 确认指令完成
func (h *MemoryHub) AckCommand(cmdID string, result *model.CommandResult) error {
	h.commandsMu.Lock()

	cs, ok := h.commands[cmdID]
	if !ok {
		h.commandsMu.Unlock()
		return nil
	}

	now := time.Now()
	cs.FinishedAt = &now
	cs.Result = result

	if result.Success {
		cs.Status = model.CommandStatusSuccess
	} else {
		cs.Status = model.CommandStatusFailed
	}

	h.commandsMu.Unlock()

	// 唤醒等待者
	h.commandWaitersMu.Lock()
	if ch, ok := h.commandWaiters[cmdID]; ok {
		select {
		case ch <- result:
		default:
		}
		delete(h.commandWaiters, cmdID)
	}
	h.commandWaitersMu.Unlock()

	log.Printf("[MemoryHub] 指令已完成: %s -> %s", cmdID, cs.Status)
	return nil
}

// GetCommandStatus 获取指令状态
func (h *MemoryHub) GetCommandStatus(cmdID string) (*model.CommandStatus, error) {
	h.commandsMu.RLock()
	defer h.commandsMu.RUnlock()

	cs, ok := h.commands[cmdID]
	if !ok {
		return nil, nil
	}
	return cs, nil
}

// WaitCommandResult 等待指令执行完成（同步等待）
// 阻塞直到 Agent 上报结果或超时
func (h *MemoryHub) WaitCommandResult(cmdID string, timeout time.Duration) (*model.CommandResult, error) {
	// 先检查是否已完成
	h.commandsMu.RLock()
	if cs, ok := h.commands[cmdID]; ok && cs.Result != nil {
		h.commandsMu.RUnlock()
		return cs.Result, nil
	}
	h.commandsMu.RUnlock()

	// 创建等待 channel
	ch := make(chan *model.CommandResult, 1)

	h.commandWaitersMu.Lock()
	h.commandWaiters[cmdID] = ch
	h.commandWaitersMu.Unlock()

	// 确保清理
	defer func() {
		h.commandWaitersMu.Lock()
		delete(h.commandWaiters, cmdID)
		h.commandWaitersMu.Unlock()
	}()

	// 等待结果或超时
	select {
	case result := <-ch:
		return result, nil
	case <-time.After(timeout):
		return nil, nil // 超时返回 nil
	}
}

// ==================== Event 查询 ====================

// GetEvents 获取集群当前所有 Events
func (h *MemoryHub) GetEvents(clusterID string) ([]model_v2.Event, error) {
	h.snapshotsMu.RLock()
	defer h.snapshotsMu.RUnlock()

	snapshot, ok := h.snapshots[clusterID]
	if !ok {
		return nil, nil
	}
	return snapshot.Events, nil
}
