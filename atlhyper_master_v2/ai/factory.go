// atlhyper_master_v2/ai/factory.go
// AIService 工厂函数
package ai

import (
	"AtlHyper/atlhyper_master_v2/ai/budget"
	"AtlHyper/atlhyper_master_v2/ai/llm"
	_ "AtlHyper/atlhyper_master_v2/ai/llm/anthropic" // 注册 anthropic provider
	_ "AtlHyper/atlhyper_master_v2/ai/llm/bedrock"   // 注册 bedrock provider
	_ "AtlHyper/atlhyper_master_v2/ai/llm/gemini"    // 注册 gemini provider
	_ "AtlHyper/atlhyper_master_v2/ai/llm/ollama"    // 注册 ollama provider
	_ "AtlHyper/atlhyper_master_v2/ai/llm/openai"    // 注册 openai provider
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/database"
)

// NewService 创建 AIService
// AI 配置（API Key、Model 等）从数据库动态获取，支持热更新
func NewService(
	cfg ServiceConfig,
	providerRepo database.AIProviderRepository,
	modelRepo database.AIProviderModelRepository,
	budgetRepo database.AIRoleBudgetRepository,
	convRepo database.AIConversationRepository,
	msgRepo database.AIMessageRepository,
) AIService {
	var budgets *budget.Ledger
	if budgetRepo != nil {
		budgets = budget.NewLedger(budgetRepo)
	}

	return &aiServiceImpl{
		providerRepo: providerRepo,
		modelRepo:    modelRepo,
		budgetRepo:   budgetRepo,
		budgets:      budgets,
		tools:        toolkit.NewRegistry(),
		convRepo:     convRepo,
		msgRepo:      msgRepo,

		bedrockRegion: cfg.BedrockRegion,
		newClient:     llm.NewLLMClient,
	}
}
