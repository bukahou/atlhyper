// atlhyper_aiservice/llm/interface.go
package llm

import "context"

// LLMClient —— 通用大模型接口定义
type LLMClient interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
	GenerateJSON(ctx context.Context, prompt string) (map[string]interface{}, error)
	Close() error
}
