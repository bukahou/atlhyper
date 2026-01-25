// service/config/slack_config.go
// Slack 配置服务
package config

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/repository"
)

// SlackConfigDTO Slack 配置 DTO（供 Handler 使用）
type SlackConfigDTO struct {
	ID          int64     `json:"id"`
	Enable      int       `json:"enable"`
	Webhook     string    `json:"webhook"`
	IntervalSec int64     `json:"intervalSec"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GetSlackConfigUI 返回 Slack 配置（给 Web UI 使用）
func GetSlackConfigUI(ctx context.Context) (SlackConfigDTO, error) {
	cfg, err := repository.Config.GetSlackConfig(ctx)
	if err != nil {
		return SlackConfigDTO{}, err
	}
	enable := 0
	if cfg.Enable {
		enable = 1
	}
	return SlackConfigDTO{
		ID:          cfg.ID,
		Enable:      enable,
		Webhook:     cfg.Webhook,
		IntervalSec: cfg.IntervalSec,
		UpdatedAt:   cfg.UpdatedAt,
	}, nil
}

// GetSlackConfigMasked 返回脱敏的 Slack 配置（低权限用户）
func GetSlackConfigMasked() SlackConfigDTO {
	return SlackConfigDTO{
		ID:          1,
		Enable:      0,
		Webhook:     "https://hooks.slack.com/services/***",
		IntervalSec: 5,
		UpdatedAt:   time.Time{},
	}
}

// Web 更新入参（全部可选，nil 表示不更新该字段）
type SlackUpdateReq struct {
	Enable      *int    `json:"enable,omitempty"`      // 0/1
	Webhook     *string `json:"webhook,omitempty"`     // 为空串也会写入
	IntervalSec *int64  `json:"intervalSec,omitempty"` // 秒
}

// UpdateSlackConfigUI UI 层更新
func UpdateSlackConfigUI(ctx context.Context, req SlackUpdateReq) error {
	// 先获取当前配置
	cfg, err := repository.Config.GetSlackConfig(ctx)
	if err != nil {
		return err
	}

	// 根据请求更新字段
	if req.Enable != nil {
		cfg.Enable = *req.Enable != 0
	}
	if req.Webhook != nil {
		cfg.Webhook = *req.Webhook
	}
	if req.IntervalSec != nil {
		cfg.IntervalSec = *req.IntervalSec
	}

	return repository.Config.UpdateSlackConfig(ctx, cfg)
}
