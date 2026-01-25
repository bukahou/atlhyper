// atlhyper_master_v2/notifier/manager/manager.go
// 告警管理器核心
package manager

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
)

// 配置常量
const (
	DedupTTL        = 10 * time.Minute // 去重 TTL
	AggregateWindow = 30 * time.Second // 聚合窗口
	AggregateMax    = 100              // 最大缓冲
	RateLimitPerMin = 5                // 每分钟限制
	MaxAlertsInMsg  = 15               // 消息内最多展示
)

// AlertManager 告警管理器
type AlertManager struct {
	channelRepo database.NotifyChannelRepository

	// 内部组件
	dedup   *dedupCache
	buffer  *aggregateBuffer
	limiter *rateLimiter

	// 状态
	running bool
	stopCh  chan struct{}
	mu      sync.Mutex
}

// NewAlertManager 创建告警管理器
func NewAlertManager(repo database.NotifyChannelRepository) *AlertManager {
	m := &AlertManager{
		channelRepo: repo,
		dedup:       newDedupCache(DedupTTL),
		limiter:     newRateLimiter(RateLimitPerMin),
		stopCh:      make(chan struct{}),
	}
	// buffer 的 flush 回调
	m.buffer = newAggregateBuffer(AggregateWindow, AggregateMax, m.flush)
	return m
}

// Start 启动告警管理器
func (m *AlertManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}
	m.running = true
	log.Println("[AlertManager] 启动告警管理器")
}

// Stop 停止告警管理器
func (m *AlertManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}
	m.running = false

	// 停止缓冲区（会 flush 剩余告警）
	m.buffer.Stop()
	m.dedup.Stop()

	close(m.stopCh)
	log.Println("[AlertManager] 停止告警管理器")
}

// Send 发送告警
// 告警会经过去重 → 缓冲 → 限流 → 发送
func (m *AlertManager) Send(ctx context.Context, alert *notifier.Alert) error {
	m.mu.Lock()
	running := m.running
	m.mu.Unlock()

	if !running {
		return nil
	}

	// 设置时间戳
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Layer 1: 去重
	key := alert.DedupKey()
	if m.dedup.IsDuplicate(key) {
		log.Printf("[AlertManager] 告警已去重，跳过: %s (%s)", alert.Title, alert.ClusterID)
		return nil
	}

	log.Printf("[AlertManager] 收到告警: %s (%s)", alert.Title, alert.ClusterID)

	// Layer 2: 加入缓冲区
	// Critical 或缓冲满会立即 flush
	m.buffer.Add(alert)

	return nil
}

// Test 测试发送
// 发送一条测试告警到指定渠道
func (m *AlertManager) Test(ctx context.Context, channelType string) error {
	// 获取渠道配置
	ch, err := m.channelRepo.GetByType(ctx, channelType)
	if err != nil {
		return err
	}
	if ch == nil {
		return notifier.ErrChannelNotFound
	}
	if !ch.Enabled {
		return notifier.ErrChannelDisabled
	}

	// 创建 channel
	sender, err := m.createChannel(ch)
	if err != nil {
		return err
	}

	// 发送测试消息
	testMsg := &notifier.Message{
		Title:    "AtlHyper 测试通知",
		Content:  "这是一条测试消息，如果您收到此消息，说明通知配置正确。",
		Severity: notifier.SeverityInfo,
		Fields: map[string]string{
			"渠道类型": channelType,
			"发送时间": time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	return sender.Send(ctx, testMsg)
}

// flush 缓冲区 flush 回调
func (m *AlertManager) flush(alerts []*notifier.Alert) {
	if len(alerts) == 0 {
		return
	}

	log.Printf("[AlertManager] 缓冲区 flush: %d 条告警", len(alerts))

	// Layer 3: 限流
	if !m.limiter.Allow() {
		waitTime := m.limiter.WaitTime()
		log.Printf("[AlertManager] 限流触发，等待 %v 后重试", waitTime)

		// 延迟重试
		time.AfterFunc(waitTime, func() {
			m.flush(alerts)
		})
		return
	}

	// 构建摘要
	summary := m.buildSummary(alerts)

	// 分发到各渠道
	m.dispatch(context.Background(), summary)
}

// buildSummary 构建告警摘要
func (m *AlertManager) buildSummary(alerts []*notifier.Alert) *notifier.AlertSummary {
	summary := &notifier.AlertSummary{
		Total:       len(alerts),
		BySeverity:  make(map[string]int),
		GeneratedAt: time.Now(),
	}

	clusterSet := make(map[string]struct{})
	nsSet := make(map[string]struct{})

	for _, alert := range alerts {
		// 统计级别
		summary.BySeverity[alert.Severity]++

		// 收集集群
		if alert.ClusterID != "" {
			clusterSet[alert.ClusterID] = struct{}{}
		}

		// 从 Resource 提取命名空间 (格式: Kind/Namespace/Name)
		if ns := extractNamespace(alert.Resource); ns != "" {
			nsSet[ns] = struct{}{}
		}
	}

	// 转换为列表
	for c := range clusterSet {
		summary.Clusters = append(summary.Clusters, c)
	}
	for ns := range nsSet {
		summary.Namespaces = append(summary.Namespaces, ns)
	}

	// 限制展示条数
	if len(alerts) > MaxAlertsInMsg {
		summary.Alerts = alerts[:MaxAlertsInMsg]
		summary.HasMore = true
		summary.MoreCount = len(alerts) - MaxAlertsInMsg
	} else {
		summary.Alerts = alerts
	}

	return summary
}

// dispatch 分发到各渠道
func (m *AlertManager) dispatch(ctx context.Context, summary *notifier.AlertSummary) {
	// 获取所有启用的渠道
	channels, err := m.channelRepo.ListEnabled(ctx)
	if err != nil {
		log.Printf("[AlertManager] 获取渠道列表失败: %v", err)
		return
	}

	if len(channels) == 0 {
		log.Println("[AlertManager] 无可用通知渠道，告警未发送")
		return
	}

	// 构建通用消息
	msg := m.buildMessage(summary)

	// 发送到各渠道
	for _, ch := range channels {
		sender, err := m.createChannel(ch)
		if err != nil {
			log.Printf("[AlertManager] 创建 %s channel 失败: %v", ch.Type, err)
			continue
		}

		if err := sender.Send(ctx, msg); err != nil {
			log.Printf("[%s] 发送失败: %v", ch.Type, err)
		} else {
			log.Printf("[%s] 发送成功: %d 条告警", ch.Type, summary.Total)
		}
	}
}

// createChannel 根据渠道配置创建 channel
func (m *AlertManager) createChannel(ch *database.NotifyChannel) (channel.Channel, error) {
	switch ch.Type {
	case "slack":
		var cfg database.SlackConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return nil, err
		}
		if cfg.WebhookURL == "" {
			return nil, notifier.ErrInvalidConfig
		}
		return channel.NewSlackChannel(channel.SlackConfig{WebhookURL: cfg.WebhookURL}), nil

	case "email":
		var cfg database.EmailConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return nil, err
		}
		if cfg.SMTPHost == "" || len(cfg.ToAddresses) == 0 {
			return nil, notifier.ErrInvalidConfig
		}
		return channel.NewEmailChannel(channel.EmailConfig{
			SMTPHost:     cfg.SMTPHost,
			SMTPPort:     cfg.SMTPPort,
			SMTPUser:     cfg.SMTPUser,
			SMTPPassword: cfg.SMTPPassword,
			SMTPTLS:      cfg.SMTPTLS,
			FromAddress:  cfg.FromAddress,
			ToAddresses:  cfg.ToAddresses,
		}), nil

	default:
		return nil, notifier.ErrUnsupportedChannel
	}
}

// buildMessage 构建通知消息
func (m *AlertManager) buildMessage(summary *notifier.AlertSummary) *notifier.Message {
	// 构建内容
	content := m.buildContent(summary)

	return &notifier.Message{
		Title:    m.buildTitle(summary),
		Content:  content,
		Severity: m.determineSeverity(summary),
		Fields:   m.buildFields(summary),
	}
}

// buildTitle 构建标题
func (m *AlertManager) buildTitle(summary *notifier.AlertSummary) string {
	return "集群告警汇总"
}

// buildContent 构建内容
func (m *AlertManager) buildContent(summary *notifier.AlertSummary) string {
	var content string

	// 级别统计
	content += "📊 级别分布\n"
	if c := summary.BySeverity[notifier.SeverityCritical]; c > 0 {
		content += "🔴 Critical: " + itoa(c) + "  "
	}
	if c := summary.BySeverity[notifier.SeverityWarning]; c > 0 {
		content += "🟠 Warning: " + itoa(c) + "  "
	}
	if c := summary.BySeverity[notifier.SeverityInfo]; c > 0 {
		content += "🔵 Info: " + itoa(c)
	}
	content += "\n\n"

	// 告警列表
	for i, alert := range summary.Alerts {
		emoji := severityEmoji(alert.Severity)
		content += emoji + " " + alert.Resource + "\n"
		content += "   " + alert.Reason + " | " + alert.Message + "\n"
		if i < len(summary.Alerts)-1 {
			content += "\n"
		}
	}

	if summary.HasMore {
		content += "\n... 还有 " + itoa(summary.MoreCount) + " 条告警\n"
	}

	return content
}

// buildFields 构建扩展字段
func (m *AlertManager) buildFields(summary *notifier.AlertSummary) map[string]string {
	fields := make(map[string]string)
	fields["告警总数"] = itoa(summary.Total)

	if len(summary.Clusters) > 0 {
		fields["涉及集群"] = joinStrings(summary.Clusters, ", ")
	}
	if len(summary.Namespaces) > 0 {
		fields["涉及命名空间"] = joinStrings(summary.Namespaces, ", ")
	}

	return fields
}

// determineSeverity 确定整体严重级别
func (m *AlertManager) determineSeverity(summary *notifier.AlertSummary) string {
	if summary.BySeverity[notifier.SeverityCritical] > 0 {
		return notifier.SeverityCritical
	}
	if summary.BySeverity[notifier.SeverityWarning] > 0 {
		return notifier.SeverityWarning
	}
	return notifier.SeverityInfo
}

// 辅助函数

func extractNamespace(resource string) string {
	// 格式: Kind/Namespace/Name 或 Kind/Name
	parts := strings.Split(resource, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func severityEmoji(severity string) string {
	switch severity {
	case notifier.SeverityCritical:
		return "🔴"
	case notifier.SeverityWarning:
		return "🟠"
	default:
		return "🔵"
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func joinStrings(s []string, sep string) string {
	return strings.Join(s, sep)
}
