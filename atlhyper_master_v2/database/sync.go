// atlhyper_master_v2/database/sync.go
// 配置同步服务
// 启动时将 config 中的配置同步到数据库
package database

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

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

// SyncAIConfig 同步 AI 配置到数据库
// 策略：数据库优先 - 首次部署从环境变量同步，后续以数据库为准
// Settings Key 约定：
//   - ai.enabled: bool (true/false)
//   - ai.provider: string (gemini/openai/anthropic)
//   - ai.api_key: string (加密存储)
//   - ai.model: string (模型名称)
//   - ai.tool_timeout: int (秒)
func SyncAIConfig(ctx context.Context, db *DB, cfg *config.AIConfig) error {
	// 检查是否已有 AI 配置（以 api_key 为判断依据）
	existing, _ := db.Settings.Get(ctx, "ai.api_key")
	if existing != nil {
		log.Println("[Sync] AI 配置已存在，使用数据库配置")
		return nil
	}

	// 如果环境变量没有配置 API Key，跳过同步
	if cfg.APIKey == "" {
		log.Println("[Sync] AI 未配置 (无 API Key)，跳过同步")
		return nil
	}

	// 从环境变量同步到数据库
	settings := []*Setting{
		{Key: "ai.enabled", Value: boolToString(cfg.Enabled), Description: "AI 功能总开关"},
		{Key: "ai.provider", Value: cfg.Provider, Description: "AI 提供商 (gemini/openai/anthropic)"},
		{Key: "ai.api_key", Value: cfg.APIKey, Description: "AI API Key"},
		{Key: "ai.model", Value: cfg.Model, Description: "AI 模型名称"},
		{Key: "ai.tool_timeout", Value: intToString(int(cfg.ToolTimeout.Seconds())), Description: "Tool 调用超时(秒)"},
	}

	for _, s := range settings {
		if err := db.Settings.Set(ctx, s); err != nil {
			log.Printf("[Sync] AI 配置同步失败 (%s): %v", s.Key, err)
			return err
		}
	}

	log.Printf("[Sync] AI 配置已同步到数据库 (provider=%s, model=%s)", cfg.Provider, cfg.Model)
	return nil
}

// LoadAIConfigFromDB 从数据库加载 AI 配置
// 返回 AIConfig 结构体，如果数据库无配置则返回 nil
func LoadAIConfigFromDB(ctx context.Context, db *DB) *config.AIConfig {
	// 尝试加载所有 AI 配置
	apiKey, _ := db.Settings.Get(ctx, "ai.api_key")
	if apiKey == nil || apiKey.Value == "" {
		return nil
	}

	enabled, _ := db.Settings.Get(ctx, "ai.enabled")
	provider, _ := db.Settings.Get(ctx, "ai.provider")
	model, _ := db.Settings.Get(ctx, "ai.model")
	timeout, _ := db.Settings.Get(ctx, "ai.tool_timeout")

	cfg := &config.AIConfig{
		Enabled:     stringToBool(getValue(enabled)),
		Provider:    getValue(provider),
		APIKey:      apiKey.Value,
		Model:       getValue(model),
		ToolTimeout: secondsToDuration(stringToInt(getValue(timeout))),
	}

	return cfg
}

// 辅助函数
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func stringToBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}

func intToString(i int) string {
	return strconv.Itoa(i)
}

func stringToInt(s string) int {
	var i int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			i = i*10 + int(c-'0')
		}
	}
	return i
}

func getValue(s *Setting) string {
	if s == nil {
		return ""
	}
	return s.Value
}

func secondsToDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 30 * time.Second // 默认 30 秒
	}
	return time.Duration(seconds) * time.Second
}
