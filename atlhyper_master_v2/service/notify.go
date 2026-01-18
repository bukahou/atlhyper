// atlhyper_master_v2/service/notify.go
// 通知服务
// 负责判断通知渠道是否生效，以及发送通知
package service

import (
	"context"
	"encoding/json"
	"log"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

// NotifyService 通知服务
type NotifyService struct {
	channelRepo repository.NotifyChannelRepository
	// notifiers 将在 notifier 包实现后注入
}

// NewNotifyService 创建服务
func NewNotifyService(channelRepo repository.NotifyChannelRepository) *NotifyService {
	return &NotifyService{
		channelRepo: channelRepo,
	}
}

// IsEffective 判断渠道是否真正生效
// 规则：enabled=1 且配置有效
func (s *NotifyService) IsEffective(ch *repository.NotifyChannel) bool {
	if !ch.Enabled {
		return false
	}

	switch ch.Type {
	case "slack":
		var cfg repository.SlackConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return false
		}
		return cfg.WebhookURL != ""

	case "email":
		var cfg repository.EmailConfig
		if err := json.Unmarshal([]byte(ch.Config), &cfg); err != nil {
			return false
		}
		return cfg.SMTPHost != "" && len(cfg.ToAddresses) > 0

	default:
		return false
	}
}

// GetEffectiveChannels 获取所有生效的渠道
func (s *NotifyService) GetEffectiveChannels(ctx context.Context) ([]*repository.NotifyChannel, error) {
	channels, err := s.channelRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}

	var effective []*repository.NotifyChannel
	for _, ch := range channels {
		if s.IsEffective(ch) {
			effective = append(effective, ch)
		}
	}
	return effective, nil
}

// SendAlert 发送告警通知
// TODO: 实现具体发送逻辑
func (s *NotifyService) SendAlert(ctx context.Context, alert *Alert) error {
	channels, err := s.GetEffectiveChannels(ctx)
	if err != nil {
		return err
	}

	if len(channels) == 0 {
		log.Printf("[Notify] 无可用通知渠道，告警未发送: %s", alert.Title)
		return nil
	}

	for _, ch := range channels {
		// TODO: 调用对应的 notifier 发送
		log.Printf("[Notify] 将发送告警到 %s (%s): %s", ch.Name, ch.Type, alert.Title)
	}

	return nil
}

// Alert 告警信息
type Alert struct {
	Title     string
	Message   string
	Severity  string // info / warning / critical
	ClusterID string
	Resource  string
}
