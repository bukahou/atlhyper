// atlhyper_aiservice/client/embedding/interface.go“
package embedding

import (
	"AtlHyper/atlhyper_aiservice/embedding"
	"context"
	"fmt"
)

// 定义选择使用的模型家族名字
const family = "gemini"

// GenerateEmbedding —— 使用“默认家族”生成向量
func GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return GenerateEmbeddingFor(ctx, family, text)
}

// GenerateEmbeddingFor —— 指定“家族/厂商标识”生成向量
func GenerateEmbeddingFor(ctx context.Context, family string, text string) ([]float32, error) {
	client, err := embedding.NewClientByFamily(family)
	if err != nil {
		return nil, fmt.Errorf("create embedding client failed: %v", err)
	}
	defer client.Close()

	vec, err := client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embedding generation failed: %v", err)
	}
	return vec, nil
}
