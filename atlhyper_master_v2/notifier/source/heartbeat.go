// atlhyper_master_v2/notifier/source/heartbeat.go
// Agent 心跳 → 告警：Agent 离线时触发一条 critical 告警，恢复上报时结束
package source

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/notifier/alert"
	agentmodel "AtlHyper/model_v3/agent"
)

// sourceStoreTimeout 来源单次读库的超时
const sourceStoreTimeout = 10 * time.Second

// HeartbeatConfig 心跳检测配置
type HeartbeatConfig struct {
	CheckInterval time.Duration // 检测间隔；0 = 30 秒
	OfflineAfter  time.Duration // 心跳多久没更新算离线；0 = 60 秒
	Now           func() time.Time
}

// HeartbeatSource 定期检查各集群 Agent 的心跳。
//
// 自己不记「哪些集群已经告过警」：每轮从库里看心跳告警是否在持续，
// 只在状态翻转时提交（离线且没有告警 → 触发；在线且有告警 → 结束）。Master 重启后判断不受影响
type HeartbeatSource struct {
	agents    AgentLister
	alerts    FiringAlertReader
	submitter AlertSubmitter
	config    HeartbeatConfig

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	done    chan struct{}
}

// NewHeartbeatSource 创建 HeartbeatSource
func NewHeartbeatSource(agents AgentLister, alerts FiringAlertReader, submitter AlertSubmitter, cfg HeartbeatConfig) *HeartbeatSource {
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 30 * time.Second
	}
	if cfg.OfflineAfter <= 0 {
		cfg.OfflineAfter = 60 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &HeartbeatSource{agents: agents, alerts: alerts, submitter: submitter, config: cfg}
}

// Start 启动检测协程。重复调用无效果
func (s *HeartbeatSource) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	s.running = true
	s.stopCh, s.done = make(chan struct{}), make(chan struct{})
	go runPeriodically(s.config.CheckInterval, s.stopCh, s.done, s.check)
	log.Info("心跳告警来源已启动", "interval", s.config.CheckInterval.String(), "offlineAfter", s.config.OfflineAfter.String())
	return nil
}

// Stop 停止检测协程并等它退出。重复调用无效果
func (s *HeartbeatSource) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.running = false
	close(s.stopCh)
	<-s.done
	return nil
}

// check 一轮检测
func (s *HeartbeatSource) check() {
	agents, err := s.agents.ListAgents()
	if err != nil {
		log.Error("获取 Agent 列表失败，本轮跳过", "err", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sourceStoreTimeout)
	defer cancel()
	firing, err := s.alerts.ListFiring(ctx)
	if err != nil {
		// 不知道哪些告警在持续就无法判断「翻转」：宁可晚一轮，也不重复触发或误结束
		log.Error("读取持续中的告警失败，本轮跳过", "err", err)
		return
	}
	alerted := make(map[string]bool)
	for _, row := range firing {
		if row.Source == string(alert.SourceHeartbeat) {
			alerted[row.ClusterID] = true
		}
	}

	now := s.config.Now()
	for _, agent := range agents {
		offline := s.isOffline(agent, now)
		switch {
		case offline && !alerted[agent.ClusterID]:
			s.submitter.Submit(heartbeatOfflineAlert(agent, now))
		case !offline && alerted[agent.ClusterID]:
			s.submitter.Submit(heartbeatRecoveredAlert(agent.ClusterID, now))
		}
	}
}

// isOffline 判断是否离线：状态已标为离线、从未有过心跳、或心跳超过阈值没更新
func (s *HeartbeatSource) isOffline(agent agentmodel.AgentInfo, now time.Time) bool {
	if agent.Status == agentmodel.StatusOffline || agent.LastHeartbeat.IsZero() {
		return true
	}
	return now.Sub(agent.LastHeartbeat) > s.config.OfflineAfter
}

func heartbeatOfflineAlert(agent agentmodel.AgentInfo, now time.Time) alert.Alert {
	details := map[string]string{}
	if !agent.LastHeartbeat.IsZero() {
		details[alert.DetailLastHeartbeat] = agent.LastHeartbeat.UTC().Format(time.RFC3339)
	}
	return alert.Alert{
		Fingerprint: alert.HeartbeatFingerprint(agent.ClusterID),
		Source:      alert.SourceHeartbeat,
		ClusterID:   agent.ClusterID,
		Severity:    alert.SeverityCritical,
		Status:      alert.StatusFiring,
		Kind:        alert.KindFiring,
		Title:       "Agent 离线",
		Summary:     "Agent 心跳超时，可能已离线或网络异常",
		Details:     details,
		StartsAt:    now,
	}
}

func heartbeatRecoveredAlert(clusterID string, now time.Time) alert.Alert {
	return alert.Alert{
		Fingerprint: alert.HeartbeatFingerprint(clusterID),
		Source:      alert.SourceHeartbeat,
		ClusterID:   clusterID,
		Severity:    alert.SeverityCritical,
		Status:      alert.StatusResolved,
		Kind:        alert.KindResolved,
		Title:       "Agent 离线",
		Summary:     "Agent 心跳已恢复",
		EndsAt:      &now,
	}
}

// runPeriodically 立即跑一次，之后按间隔跑，直到 stopCh 关闭；退出时关闭 done
func runPeriodically(interval time.Duration, stopCh <-chan struct{}, done chan<- struct{}, run func()) {
	defer close(done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	run()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			run()
		}
	}
}
