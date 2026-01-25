// atlhyper_master_v2/notifier/trigger/event.go
// K8s 事件告警触发器
package trigger

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/enrich"
	"AtlHyper/atlhyper_master_v2/notifier/template"
)

// EventConfig 事件告警配置
type EventConfig struct {
	CheckInterval time.Duration
}

// EventTrigger K8s 事件告警触发器
type EventTrigger struct {
	eventRepo  database.ClusterEventRepository
	enricher   *enrich.Enricher
	manager    notifier.AlertManager
	config     EventConfig
	lastSeenID int64

	stopCh chan struct{}
	wg     sync.WaitGroup
	mu     sync.Mutex
}

// NewEventTrigger 创建事件告警触发器
func NewEventTrigger(
	eventRepo database.ClusterEventRepository,
	query enrich.ResourceQuery,
	manager notifier.AlertManager,
	cfg EventConfig,
) *EventTrigger {
	return &EventTrigger{
		eventRepo: eventRepo,
		enricher:  enrich.NewEnricher(query),
		manager:   manager,
		config:    cfg,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动触发器
func (t *EventTrigger) Start() error {
	// 获取最新事件 ID
	latestID, err := t.eventRepo.GetLatestEventID(context.Background())
	if err != nil {
		log.Printf("[EventTrigger] 获取最新事件 ID 失败，从 0 开始: %v", err)
		latestID = 0
	}
	t.lastSeenID = latestID

	t.wg.Add(1)
	go t.loop()

	log.Printf("[EventTrigger] 启动: 间隔=%v, 起始ID=%d", t.config.CheckInterval, t.lastSeenID)
	return nil
}

// Stop 停止触发器
func (t *EventTrigger) Stop() error {
	close(t.stopCh)
	t.wg.Wait()
	log.Println("[EventTrigger] 已停止")
	return nil
}

// loop 轮询循环
func (t *EventTrigger) loop() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.processEvents()
		}
	}
}

// processEvents 处理新事件
func (t *EventTrigger) processEvents() {
	ctx := context.Background()

	events, err := t.eventRepo.GetEventsSince(ctx, t.lastSeenID)
	if err != nil {
		log.Printf("[EventTrigger] 查询事件失败: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	var sent int
	for _, event := range events {
		if err := t.triggerEvent(ctx, event); err != nil {
			log.Printf("[EventTrigger] 处理事件 %d 失败: %v", event.ID, err)
		} else {
			sent++
		}

		t.mu.Lock()
		if event.ID > t.lastSeenID {
			t.lastSeenID = event.ID
		}
		t.mu.Unlock()
	}

	log.Printf("[EventTrigger] 处理 %d 个事件, 发送 %d 个告警", len(events), sent)
}

// triggerEvent 触发单个事件告警
func (t *EventTrigger) triggerEvent(ctx context.Context, event *database.ClusterEvent) error {
	// 丰富数据
	enriched := t.enricher.EnrichByResource(ctx, event.ClusterID, event.InvolvedKind, event.InvolvedNamespace, event.InvolvedName)

	// 构建模板数据
	data := &template.AlertData{
		Title:     fmt.Sprintf("[%s] %s", event.ClusterID, event.Reason),
		Message:   event.Message,
		Severity:  "warning",
		Source:    "k8s_event",
		ClusterID: event.ClusterID,
		Resource:  fmt.Sprintf("%s/%s/%s", event.InvolvedKind, event.InvolvedNamespace, event.InvolvedName),
		Reason:    event.Reason,
		Timestamp: event.LastTimestamp,
		Fields: map[string]string{
			"count": fmt.Sprintf("%d", event.Count),
		},
		Enriched: enriched,
	}

	if err := t.manager.SendWithTemplate("k8s_event", data); err != nil {
		return fmt.Errorf("send alert: %w", err)
	}

	log.Printf("[EventTrigger] 告警已发送: cluster=%s, reason=%s, kind=%s, name=%s",
		event.ClusterID, event.Reason, event.InvolvedKind, event.InvolvedName)

	return nil
}
