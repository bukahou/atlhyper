// atlhyper_master_v2/database/sync.go
// 配置同步服务
// 启动时将 config 中的配置同步到数据库
package database

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/config"
)

// InitAISettings 初始化 AI 全局设置
// 首次启动时从环境变量配置写入数据库，之后以数据库为准
func InitAISettings(ctx context.Context, db *DB, cfg *config.AIConfig) error {
	existing, _ := db.AISettings.Get(ctx)
	if existing != nil {
		log.Info("AI Settings 已存在，跳过初始化")
		return nil
	}

	toolTimeout := int(cfg.ToolTimeout.Seconds())
	if toolTimeout <= 0 {
		toolTimeout = 30
	}

	settings := &AISettings{
		ToolTimeout: toolTimeout,
		UpdatedAt:   time.Now(),
	}

	if err := db.AISettings.Update(ctx, settings); err != nil {
		return err
	}

	log.Info("AI Settings 已初始化", "timeoutSec", toolTimeout)
	return nil
}

// SeedAIProvider 从环境变量种子配置初始化 AI Provider
// 仅在 ai_providers 表无数据且 seed.Provider 非空时执行
// 用于部署时通过环境变量自动配置 Ollama 等本地 AI 服务
func SeedAIProvider(ctx context.Context, db *DB, seed *config.AISeed) error {
	if seed.Provider == "" {
		return nil
	}

	providers, _ := db.AIProvider.List(ctx)
	if len(providers) > 0 {
		log.Info("AI Provider 表已有数据，跳过种子初始化")
		return nil
	}

	name := seed.Name
	if name == "" {
		name = seed.Provider + " (seed)"
	}

	now := time.Now()
	newProvider := &AIProvider{
		Name:        name,
		Provider:    seed.Provider,
		APIKey:      seed.APIKey,
		Model:       seed.Model,
		BaseURL:     seed.BaseURL,
		Description: "环境变量种子配置自动创建",
		Status:      "unknown",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := db.AIProvider.Create(ctx, newProvider); err != nil {
		return fmt.Errorf("创建种子 AI Provider 失败: %w", err)
	}

	log.Info("AI Provider 种子初始化完成",
		"provider", seed.Provider,
		"model", seed.Model,
		"baseURL", seed.BaseURL,
	)
	return nil
}
