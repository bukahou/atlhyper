// atlhyper_aiservice/client/ai/interface.go
package ai

import (
	"AtlHyper/atlhyper_aiservice/llm"
	"context"
	"fmt"
)

// GenerateText —— 单入口函数：创建客户端、调用模型、关闭客户端
func GenerateText(ctx context.Context, prompt string) (string, error) {
	client, err := NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("init AI client failed: %v", err)
	}
	defer client.Close()

	out, err := client.GenerateText(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("AI generation failed: %v", err)
	}
	return out, nil
}


// NewClient —— 工厂：创建通用 LLM 客户端
func NewClient(ctx context.Context) (llm.LLMClient, error) {
	client, err := llm.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create LLM client failed: %v", err)
	}
	return client, nil
}
