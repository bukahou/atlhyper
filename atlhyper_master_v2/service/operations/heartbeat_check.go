// atlhyper_master_v2/service/operations/heartbeat_check.go
// Agent 心跳检测服务
// 定期检查 Agent 状态，对离线的 Agent 发送告警
package operations

import (
	"context"
	"log"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/manager"
	"AtlHyper/model_v2"
)

// HeartbeatCheckConfig 心跳检测配置
type HeartbeatCheckConfig struct {
	CheckInterval time.Duration // 检测间隔（默认 30 秒）
	OfflineAfter  time.Duration // 多久未心跳视为离线（默认 60 秒）
}

// HeartbeatCheckService Agent 心跳检测服务
type HeartbeatCheckService struct {
	store        datahub.Store
	alertManager *manager.AlertManager
	config       HeartbeatCheckConfig

	running bool
	stopCh  chan struct{}
	mu      sync.Mutex
}

// NewHeartbeatCheckService 创建心跳检测服务
func NewHeartbeatCheckService(store datahub.Store, alertMgr *manager.AlertManager, cfg HeartbeatCheckConfig) *HeartbeatCheckService {
	// 设置默认值
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = 30 * time.Second
	}
	if cfg.OfflineAfter == 0 {
		cfg.OfflineAfter = 60 * time.Second
	}

	return &HeartbeatCheckService{
		store:        store,
		alertManager: alertMgr,
		config:       cfg,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动心跳检测服务
func (s *HeartbeatCheckService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}
	s.running = true

	go s.checkLoop()

	log.Printf("[HeartbeatCheck] 启动心跳检测服务: 间隔=%v, 离线阈值=%v",
		s.config.CheckInterval, s.config.OfflineAfter)
	return nil
}

// Stop 停止心跳检测服务
func (s *HeartbeatCheckService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}
	s.running = false

	close(s.stopCh)
	log.Println("[HeartbeatCheck] 停止心跳检测服务")
	return nil
}

// checkLoop 检测循环
func (s *HeartbeatCheckService) checkLoop() {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	// 启动后立即检查一次
	s.check()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.check()
		}
	}
}

// check 执行一次检测
func (s *HeartbeatCheckService) check() {
	// 获取所有 Agent
	agents, err := s.store.ListAgents()
	if err != nil {
		log.Printf("[HeartbeatCheck] 获取 Agent 列表失败: %v", err)
		return
	}

	now := time.Now()

	for _, agent := range agents {
		// 检查心跳是否超时
		if s.isOffline(agent, now) {
			s.sendOfflineAlert(agent)
		}
	}
}

// isOffline 判断 Agent 是否离线
func (s *HeartbeatCheckService) isOffline(agent model_v2.AgentInfo, now time.Time) bool {
	// 如果状态已经是 offline，检查心跳时间
	if agent.Status == model_v2.AgentStatusOffline {
		return true
	}

	// 如果心跳时间为零，说明从未心跳过
	if agent.LastHeartbeat.IsZero() {
		return true
	}

	// 检查心跳是否超时
	return now.Sub(agent.LastHeartbeat) > s.config.OfflineAfter
}

// sendOfflineAlert 发送离线告警
func (s *HeartbeatCheckService) sendOfflineAlert(agent model_v2.AgentInfo) {
	ctx := context.Background()

	alert := &notifier.Alert{
		Title:     "Agent 离线",
		Message:   "Agent 心跳超时，可能已离线或网络异常",
		Severity:  notifier.SeverityCritical,
		Source:    notifier.SourceAgentHeartbeat,
		ClusterID: agent.ClusterID,
		Resource:  "Agent/" + agent.ClusterID,
		Reason:    "HeartbeatTimeout",
		Fields: map[string]string{
			"cluster_id":     agent.ClusterID,
			"last_heartbeat": agent.LastHeartbeat.Format("2006-01-02 15:04:05"),
			"offline_after":  s.config.OfflineAfter.String(),
		},
		Timestamp: time.Now(),
	}

	if err := s.alertManager.Send(ctx, alert); err != nil {
		log.Printf("[HeartbeatCheck] 发送告警失败: cluster=%s, err=%v", agent.ClusterID, err)
	}
}
