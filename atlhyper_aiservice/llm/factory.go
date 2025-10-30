// atlhyper_aiservice/llm/factory.go
package llm

import (
	"AtlHyper/atlhyper_aiservice/config"
	"context"
	"fmt"
	"strings"
)

// NewClient —— 底层工厂函数，根据配置创建不同模型客户端
func NewClient(ctx context.Context) (LLMClient, error) {
	cfg := config.GetGeminiConfig()
	model := cfg.ModelName

	switch {
	case strings.HasPrefix(model, "gemini"):
		return newGeminiClient(ctx)
	// case strings.HasPrefix(model, "gpt"):
	//     return newOpenAIClient(ctx)
	default:
		return nil, fmt.Errorf("unsupported model: %s", model)
	}
}
