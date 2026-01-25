// atlhyper_master_v2/database/sync.go
// 配置同步服务
// 启动时将 config 中的配置同步到数据库
package database

import (
	"context"
	"encoding/json"
	"log"

	"AtlHyper/atlhyper_master_v2/config"
)

// SyncNotifyChannels 同步通知渠道配置到数据库
// 规则：数据库无记录时插入，有记录时跳过（保留用户修改）
func SyncNotifyChannels(ctx context.Context, db *DB, cfg *config.NotifierConfig) error {
	// 同步 Slack
	if err := syncSlackChannel(ctx, db, &cfg.Slack); err != nil {
		log.Printf("[Sync] Slack 配置同步失败: %v", err)
	}

	// 同步 Email
	if err := syncEmailChannel(ctx, db, &cfg.Email); err != nil {
		log.Printf("[Sync] Email 配置同步失败: %v", err)
	}

	return nil
}

// syncSlackChannel 同步 Slack 渠道配置
func syncSlackChannel(ctx context.Context, db *DB, cfg *config.SlackChannelConfig) error {
	// 检查是否已存在
	existing, _ := db.Notify.GetByType(ctx, "slack")
	if existing != nil {
		log.Printf("[Sync] Slack 配置已存在 (ID=%d)，跳过同步", existing.ID)
		return nil
	}

	// 如果未启用且无配置，跳过
	if !cfg.Enabled && cfg.WebhookURL == "" {
		log.Println("[Sync] Slack 未配置，跳过")
		return nil
	}

	// 构建配置 JSON
	configJSON, err := json.Marshal(SlackConfig{
		WebhookURL: cfg.WebhookURL,
	})
	if err != nil {
		return err
	}

	// 创建记录
	channel := &NotifyChannel{
		Type:    "slack",
		Name:    "Slack",
		Enabled: cfg.Enabled,
		Config:  string(configJSON),
	}

	if err := db.Notify.Create(ctx, channel); err != nil {
		return err
	}

	log.Printf("[Sync] Slack 配置已同步到数据库 (enabled=%v)", cfg.Enabled)
	return nil
}

// syncEmailChannel 同步 Email 渠道配置
func syncEmailChannel(ctx context.Context, db *DB, cfg *config.EmailChannelConfig) error {
	// 检查是否已存在
	existing, _ := db.Notify.GetByType(ctx, "email")
	if existing != nil {
		log.Printf("[Sync] Email 配置已存在 (ID=%d)，跳过同步", existing.ID)
		return nil
	}

	// 如果未启用且无配置，跳过
	if !cfg.Enabled && cfg.SMTPHost == "" {
		log.Println("[Sync] Email 未配置，跳过")
		return nil
	}

	// 构建配置 JSON
	configJSON, err := json.Marshal(EmailConfig{
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUser:     cfg.SMTPUser,
		SMTPPassword: cfg.SMTPPassword,
		SMTPTLS:      cfg.SMTPTLS,
		FromAddress:  cfg.FromAddress,
		ToAddresses:  cfg.ToAddresses,
	})
	if err != nil {
		return err
	}

	// 创建记录
	channel := &NotifyChannel{
		Type:    "email",
		Name:    "Email",
		Enabled: cfg.Enabled,
		Config:  string(configJSON),
	}

	if err := db.Notify.Create(ctx, channel); err != nil {
		return err
	}

	log.Printf("[Sync] Email 配置已同步到数据库 (enabled=%v)", cfg.Enabled)
	return nil
}
