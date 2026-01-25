// atlhyper_master_v2/notifier/trigger/heartbeat.go
// Agent 心跳检测触发器
package trigger

import (
	"log"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/template"
	"AtlHyper/model_v2"
)

// HeartbeatConfig 心跳检测配置
type HeartbeatConfig struct {
	CheckInterval time.Duration // 检测间隔
	OfflineAfter  time.Duration // 离线阈值
}

// HeartbeatTrigger Agent 心跳检测触发器
type HeartbeatTrigger struct {
	store   datahub.Store
	manager notifier.AlertManager
	config  HeartbeatConfig

	running   bool
	stopCh    chan struct{}
	mu        sync.Mutex
	alerted   map[string]time.Time // clusterID -> 首次告警时间
	alertedMu sync.RWMutex
}

// NewHeartbeatTrigger 创建心跳检测触发器
func NewHeartbeatTrigger(store datahub.Store, manager notifier.AlertManager, cfg HeartbeatConfig) *HeartbeatTrigger {
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 30 * time.Second
	}
	if cfg.OfflineAfter == 0 {
		cfg.OfflineAfter = 60 * time.Second
	}

	return &HeartbeatTrigger{
		store:   store,
		manager: manager,
		config:  cfg,
		stopCh:  make(chan struct{}),
		alerted: make(map[string]time.Time),
	}
}

// Start 启动触发器
func (t *HeartbeatTrigger) Start() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return nil
	}
	t.running = true

	go t.loop()

	log.Printf("[HeartbeatTrigger] 启动: 间隔=%v, 离线阈值=%v", t.config.CheckInterval, t.config.OfflineAfter)
	return nil
}

// Stop 停止触发器
func (t *HeartbeatTrigger) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}
	t.running = false

	close(t.stopCh)
	log.Println("[HeartbeatTrigger] 已停止")
	return nil
}

// loop 检测循环
func (t *HeartbeatTrigger) loop() {
	ticker := time.NewTicker(t.config.CheckInterval)
	defer ticker.Stop()

	t.check()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.check()
		}
	}
}

// check 执行检测
func (t *HeartbeatTrigger) check() {
	agents, err := t.store.ListAgents()
	if err != nil {
		log.Printf("[HeartbeatTrigger] 获取 Agent 列表失败: %v", err)
		return
	}

	now := time.Now()

	for _, agent := range agents {
		offline := t.isOffline(agent, now)
		wasAlerted := t.isAlerted(agent.ClusterID)

		if offline && !wasAlerted {
			t.triggerOffline(agent, now)
		} else if !offline && wasAlerted {
			t.triggerRecovery(agent)
		}
	}
}

// isOffline 判断是否离线
func (t *HeartbeatTrigger) isOffline(agent model_v2.AgentInfo, now time.Time) bool {
	if agent.Status == model_v2.AgentStatusOffline {
		return true
	}
	if agent.LastHeartbeat.IsZero() {
		return true
	}
	return now.Sub(agent.LastHeartbeat) > t.config.OfflineAfter
}

// isAlerted 检查是否已告警
func (t *HeartbeatTrigger) isAlerted(clusterID string) bool {
	t.alertedMu.RLock()
	defer t.alertedMu.RUnlock()
	_, exists := t.alerted[clusterID]
	return exists
}

// triggerOffline 触发离线告警
func (t *HeartbeatTrigger) triggerOffline(agent model_v2.AgentInfo, now time.Time) {
	data := &template.AlertData{
		Title:     "Agent 离线",
		Message:   "Agent 心跳超时，可能已离线或网络异常",
		Severity:  "critical",
		Source:    "agent_heartbeat",
		ClusterID: agent.ClusterID,
		Resource:  "Agent/" + agent.ClusterID,
		Reason:    "HeartbeatTimeout",
		Timestamp: now,
		Fields: map[string]string{
			"last_heartbeat": agent.LastHeartbeat.Format("2006-01-02 15:04:05"),
			"offline_after":  t.config.OfflineAfter.String(),
		},
	}

	if err := t.manager.SendWithTemplate("heartbeat_offline", data); err != nil {
		log.Printf("[HeartbeatTrigger] 发送离线告警失败: cluster=%s, err=%v", agent.ClusterID, err)
		return
	}

	t.alertedMu.Lock()
	t.alerted[agent.ClusterID] = now
	t.alertedMu.Unlock()

	log.Printf("[HeartbeatTrigger] Agent 离线: cluster=%s", agent.ClusterID)
}

// triggerRecovery 触发恢复告警
func (t *HeartbeatTrigger) triggerRecovery(agent model_v2.AgentInfo) {
	t.alertedMu.RLock()
	alertedAt := t.alerted[agent.ClusterID]
	t.alertedMu.RUnlock()

	downtime := time.Since(alertedAt).Round(time.Second).String()

	data := &template.AlertData{
		Title:     "Agent 恢复",
		Message:   "Agent 心跳已恢复正常",
		Severity:  "info",
		Source:    "agent_heartbeat",
		ClusterID: agent.ClusterID,
		Resource:  "Agent/" + agent.ClusterID,
		Reason:    "HeartbeatRecovered",
		Timestamp: time.Now(),
		Fields: map[string]string{
			"downtime": downtime,
		},
	}

	if err := t.manager.SendWithTemplate("heartbeat_recovery", data); err != nil {
		log.Printf("[HeartbeatTrigger] 发送恢复告警失败: cluster=%s, err=%v", agent.ClusterID, err)
		return
	}

	t.alertedMu.Lock()
	delete(t.alerted, agent.ClusterID)
	t.alertedMu.Unlock()

	log.Printf("[HeartbeatTrigger] Agent 恢复: cluster=%s, downtime=%s", agent.ClusterID, downtime)
}
