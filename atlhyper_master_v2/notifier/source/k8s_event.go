// atlhyper_master_v2/notifier/source/k8s_event.go
// K8s Warning 事件 → 告警（旧的逐条事件告警）：受策略 k8sEventAlertsEnabled 控制，默认关；也走同一条管道
package source

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

const (
	// defaultK8sEventExpiry K8s 事件告警多久没有再出现就结束
	defaultK8sEventExpiry = 10 * time.Minute
	// maxEventSummaryRunes 事件消息写进告警摘要的长度上限
	maxEventSummaryRunes = 200
)

// K8sEventConfig K8s 事件告警配置
type K8sEventConfig struct {
	CheckInterval time.Duration // 轮询间隔；0 = 30 秒
	ExpireAfter   time.Duration // 多久没有再出现就结束；0 = 10 分钟
	Now           func() time.Time
}

// K8sEventSource 轮询持久化的 K8s Warning 事件。
//
//   - 新出现的事件（同一资源的同一 Reason 是一条）→ 一条 low 告警
//   - 告警 10 分钟没有再出现 → 安静结束，不发恢复通知
//   - 开关关着的时候不产生告警，期间出现的事件之后也不补报；已有的告警到点照样结束
type K8sEventSource struct {
	events    ClusterEventReader
	policy    PolicyReader
	alerts    FiringAlertReader
	submitter AlertSubmitter
	config    K8sEventConfig

	lastSeenID int64 // 已处理到的事件 id；只由检测协程读写

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	done    chan struct{}
}

// NewK8sEventSource 创建 K8sEventSource
func NewK8sEventSource(events ClusterEventReader, policy PolicyReader, alerts FiringAlertReader, submitter AlertSubmitter, cfg K8sEventConfig) *K8sEventSource {
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 30 * time.Second
	}
	if cfg.ExpireAfter <= 0 {
		cfg.ExpireAfter = defaultK8sEventExpiry
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &K8sEventSource{events: events, policy: policy, alerts: alerts, submitter: submitter, config: cfg}
}

// Start 从最新的事件之后开始轮询（启动前已有的事件不报）。重复调用无效果
func (s *K8sEventSource) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), sourceStoreTimeout)
	latestID, err := s.events.GetLatestEventID(ctx)
	cancel()
	if err != nil {
		log.Warn("获取最新事件 ID 失败，从 0 开始", "err", err)
		latestID = 0
	}
	s.lastSeenID = latestID

	s.running = true
	s.stopCh, s.done = make(chan struct{}), make(chan struct{})
	go runPeriodically(s.config.CheckInterval, s.stopCh, s.done, s.check)
	log.Info("K8s 事件告警来源已启动", "interval", s.config.CheckInterval.String(), "fromEventID", latestID)
	return nil
}

// Stop 停止轮询并等协程退出。重复调用无效果
func (s *K8sEventSource) Stop() error {
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

// check 一轮轮询
func (s *K8sEventSource) check() {
	ctx, cancel := context.WithTimeout(context.Background(), sourceStoreTimeout)
	defer cancel()

	policy, err := s.policy.Get(ctx)
	if err != nil {
		log.Error("读取通知策略失败，本轮跳过", "err", err)
		return
	}
	s.reportNewEvents(ctx, policy.K8sEventAlertsEnabled)
	s.expireStaleAlerts(ctx, policy.K8sEventAlertsEnabled)
}

// reportNewEvents 把上次之后新出现的事件变成告警。
// 开关关着也照样推进游标：关着的期间出现的事件，之后打开开关不补报
func (s *K8sEventSource) reportNewEvents(ctx context.Context, enabled bool) {
	events, err := s.events.GetEventsSince(ctx, s.lastSeenID)
	if err != nil {
		log.Error("查询新事件失败", "err", err)
		return
	}
	for _, event := range events {
		if event.ID > s.lastSeenID {
			s.lastSeenID = event.ID
		}
		if enabled {
			s.submitter.Submit(K8sEventAlert(event))
		}
	}
}

// expireStaleAlerts 结束久未再出现的 K8s 事件告警。
//
// 同一条事件再次发生时 K8s 只更新原来那一行的时间与次数，不会成为「新事件」；
// 所以到点时先看这条事件最近是否还在发生 —— 还在就给告警续期，不在才结束
func (s *K8sEventSource) expireStaleAlerts(ctx context.Context, enabled bool) {
	firing, err := s.alerts.ListFiring(ctx)
	if err != nil {
		log.Error("读取持续中的告警失败", "err", err)
		return
	}
	now := s.config.Now()
	for _, row := range firing {
		if row.Source != string(alert.SourceK8sEvent) || now.Sub(row.UpdatedAt) < s.config.ExpireAfter {
			continue
		}
		if enabled {
			if recurring := s.recentOccurrence(ctx, row, now); recurring != nil {
				s.submitter.Submit(K8sEventAlert(recurring)) // 内容相同：管道只刷新更新时间，不另发通知
				continue
			}
		}
		s.submitter.Submit(alert.Alert{
			Fingerprint: row.Fingerprint, Source: alert.SourceK8sEvent, ClusterID: row.ClusterID,
			Status: alert.StatusResolved, Kind: alert.KindResolved, EndsAt: &now,
			Quiet: true, // 只是不再出现，并不代表问题解决了：不发恢复通知
		})
	}
}

// recentOccurrence 这条告警对应的事件在过期窗口内是否还发生过；是则返回那条事件
func (s *K8sEventSource) recentOccurrence(ctx context.Context, row *database.NotifyAlert, now time.Time) *database.ClusterEvent {
	kind, name, reason := row.Details[alert.DetailEventKind], row.Details[alert.DetailEventName], row.Details[alert.DetailEventReason]
	events, err := s.events.ListByInvolvedResource(ctx, row.ClusterID, kind, row.Labels[alert.LabelNamespace], name)
	if err != nil {
		log.Error("查询事件是否仍在发生失败，按不再发生处理", "err", err)
		return nil
	}
	for _, event := range events {
		if event.Reason == reason && now.Sub(event.LastTimestamp) < s.config.ExpireAfter {
			return event
		}
	}
	return nil
}

// K8sEventAlert 一条 K8s Warning 事件 → 告警
func K8sEventAlert(event *database.ClusterEvent) alert.Alert {
	kind, namespace, name := event.InvolvedKind, event.InvolvedNamespace, event.InvolvedName
	subjectNamespace := namespace
	if subjectNamespace == "" {
		subjectNamespace = clusterScopeNamespace
	}
	subjectType := strings.ToLower(kind)

	resource := name
	if namespace != "" {
		resource = namespace + "/" + name
	}
	return alert.Alert{
		Fingerprint: alert.K8sEventFingerprint(event.ClusterID, kind, namespace, name, event.Reason),
		Source:      alert.SourceK8sEvent,
		ClusterID:   event.ClusterID,
		Severity:    alert.SeverityLow,
		Status:      alert.StatusFiring,
		Kind:        alert.KindFiring,
		Title:       strings.TrimSpace(kind + " " + resource + " " + event.Reason),
		Summary:     truncateRunes(event.Message, maxEventSummaryRunes),
		Labels: map[string]string{
			alert.LabelNamespace:   namespace,
			alert.LabelSubject:     subjectNamespace + "/" + subjectType + "/" + name,
			alert.LabelSubjectType: subjectType,
		},
		Details: map[string]string{
			alert.DetailEventKind:   kind,
			alert.DetailEventName:   name,
			alert.DetailEventReason: event.Reason,
			alert.DetailEventCount:  strconv.Itoa(int(event.Count)),
		},
	}
}

// truncateRunes 按字符数截断，被截断时以省略号结尾
func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}
