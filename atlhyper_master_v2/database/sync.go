// atlhyper_master_v2/database/sync.go
// 配置同步服务
// 启动时将 config 中的配置同步到数据库
package database

import (
	"context"
	"log"
	"strconv"
	"time"

	"AtlHyper/atlhyper_master_v2/config"
)

// InitAIActiveConfig 初始化 AI 激活配置
// 首次启动时从环境变量配置写入数据库，之后以数据库为准
func InitAIActiveConfig(ctx context.Context, db *DB, cfg *config.AIConfig) error {
	// 检查是否已有配置
	existing, _ := db.AIActive.Get(ctx)
	if existing != nil {
		log.Println("[Sync] AI Active Config 已存在，跳过初始化")
		return nil
	}

	// 从配置初始化
	now := time.Now()
	toolTimeout := int(cfg.ToolTimeout.Seconds())
	if toolTimeout <= 0 {
		toolTimeout = 30
	}

	activeConfig := &AIActiveConfig{
		Enabled:     cfg.Enabled,
		ToolTimeout: toolTimeout,
		UpdatedAt:   now,
	}

	if err := db.AIActive.Update(ctx, activeConfig); err != nil {
		return err
	}

	log.Printf("[Sync] AI Active Config 已初始化 (enabled=%v, timeout=%ds)", cfg.Enabled, toolTimeout)
	return nil
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

// MigrateOldAIConfig 迁移旧 Settings 表中的 AI 配置到新的 ai_providers 表
// 策略：如果 ai_providers 表已有数据则跳过；否则从 settings 表迁移
func MigrateOldAIConfig(ctx context.Context, db *DB) error {
	// 检查新表是否已有数据
	providers, _ := db.AIProvider.List(ctx)
	if len(providers) > 0 {
		log.Println("[Migrate] AI Provider 表已有数据，跳过迁移")
		return nil
	}

	// 检查旧配置是否存在
	apiKey, _ := db.Settings.Get(ctx, "ai.api_key")
	if apiKey == nil || apiKey.Value == "" {
		log.Println("[Migrate] 无旧 AI 配置，跳过迁移")
		return nil
	}

	// 读取旧配置
	enabled, _ := db.Settings.Get(ctx, "ai.enabled")
	provider, _ := db.Settings.Get(ctx, "ai.provider")
	model, _ := db.Settings.Get(ctx, "ai.model")
	timeout, _ := db.Settings.Get(ctx, "ai.tool_timeout")

	providerValue := getValue(provider)
	if providerValue == "" {
		providerValue = "gemini"
	}
	modelValue := getValue(model)
	if modelValue == "" {
		modelValue = "gemini-2.0-flash"
	}

	// 创建新 Provider
	now := time.Now()
	newProvider := &AIProvider{
		Name:        "默认配置（迁移）",
		Provider:    providerValue,
		APIKey:      apiKey.Value,
		Model:       modelValue,
		Description: "从旧配置迁移",
		Status:      "unknown",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := db.AIProvider.Create(ctx, newProvider); err != nil {
		log.Printf("[Migrate] 创建 AI Provider 失败: %v", err)
		return err
	}

	// 更新激活配置
	activeConfig := &AIActiveConfig{
		Enabled:     stringToBool(getValue(enabled)),
		ProviderID:  &newProvider.ID,
		ToolTimeout: stringToInt(getValue(timeout)),
		UpdatedAt:   now,
	}
	if activeConfig.ToolTimeout <= 0 {
		activeConfig.ToolTimeout = 30
	}

	if err := db.AIActive.Update(ctx, activeConfig); err != nil {
		log.Printf("[Migrate] 更新 AI Active Config 失败: %v", err)
		return err
	}

	log.Printf("[Migrate] AI 配置已迁移到新表 (provider=%s, model=%s)", providerValue, modelValue)
	return nil
}
