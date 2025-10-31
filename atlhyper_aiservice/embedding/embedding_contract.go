package embedding

import (
	"context"
)

// EmbeddingClient —— 通用向量生成接口
// 未来可以兼容 Gemini、OpenAI、Claude 等不同模型
type EmbeddingClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	Close() error
}
