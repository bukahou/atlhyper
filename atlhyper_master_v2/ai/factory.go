// atlhyper_master_v2/ai/factory.go
// AIService 工厂函数
package ai

import (
	"time"

	_ "AtlHyper/atlhyper_master_v2/ai/llm/anthropic" // 注册 anthropic provider
	_ "AtlHyper/atlhyper_master_v2/ai/llm/gemini"    // 注册 gemini provider
	_ "AtlHyper/atlhyper_master_v2/ai/llm/ollama"    // 注册 ollama provider
	_ "AtlHyper/atlhyper_master_v2/ai/llm/openai"    // 注册 openai provider
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service/operations"
)

// NewService 创建 AIService
// AI 配置（API Key、Model 等）从数据库动态获取，支持热更新
func NewService(
	cfg ServiceConfig,
	ops *operations.CommandService,
	bus mq.Producer,
	providerRepo database.AIProviderRepository,
	activeRepo database.AIActiveConfigRepository,
	modelRepo database.AIProviderModelRepository,
	budgetRepo database.AIRoleBudgetRepository,
	convRepo database.AIConversationRepository,
	msgRepo database.AIMessageRepository,
) AIService {
	// Tool 超时默认 30s
	toolTimeout := cfg.ToolTimeout
	if toolTimeout == 0 {
		toolTimeout = 30 * time.Second
	}

	return &aiServiceImpl{
		providerRepo: providerRepo,
		activeRepo:   activeRepo,
		modelRepo:    modelRepo,
		budgetRepo:   budgetRepo,
		executor:     newToolExecutor(ops, bus, toolTimeout),
		convRepo:     convRepo,
		msgRepo:      msgRepo,
	}
}
