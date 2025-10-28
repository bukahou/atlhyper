// atlhyper_aiservice/service/gemini_service.go
package service

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_aiservice/client"
	"AtlHyper/atlhyper_aiservice/config"

	"github.com/google/generative-ai-go/genai"
)

// GenerateByGemini 仅负责业务逻辑：根据 prompt 获取生成文本
func GenerateByGemini(prompt string) (string, error) {
	cfg := config.GetGeminiConfig() // ✅ 直接取全局配置

	ctx := context.Background()
	c, err := client.GetGeminiClient(ctx)
	if err != nil {
		return "", err
	}

	model := c.GenerativeModel(cfg.ModelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("生成内容失败: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("无候选结果")
	}

	output := ""
	for _, p := range resp.Candidates[0].Content.Parts {
		output += fmt.Sprintf("%v", p)
	}

	return output, nil
}

