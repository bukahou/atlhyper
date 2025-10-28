package service

import (
	"AtlHyper/atlhyper_aiservice/client/ai"
	"AtlHyper/atlhyper_aiservice/config"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
)

func RunStage3FinalDiagnosis(clusterID string, stage1, stage2 map[string]interface{}) (map[string]interface{}, error) {
	ctx := context.Background()
	prompt := buildStage3Prompt(clusterID, stage1, stage2)

	cfg := config.GetGeminiConfig()
	c, err := ai.GetGeminiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gemini client failed: %v", err)
	}
	model := c.GenerativeModel(cfg.ModelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("AI second stage failed: %v", err)
	}

	out := ""
	for _, p := range resp.Candidates[0].Content.Parts {
		out += fmt.Sprintf("%v", p)
	}

	var parsed map[string]interface{}
	_ = json.Unmarshal([]byte(out), &parsed)

	return map[string]interface{}{
		"prompt":  prompt,
		"ai_raw":  out,
		"ai_json": parsed,
	}, nil
}

func buildStage3Prompt(clusterID string, stage1, stage2 map[string]interface{}) string {
	b, _ := json.MarshalIndent(stage1, "", "  ")
	f, _ := json.MarshalIndent(stage2, "", "  ")

	return fmt.Sprintf(`集群 ID: %s
【阶段一 AI 初判】:
%s

【阶段二 Master 上下文】:
%s

请综合分析，输出最终 JSON：
{
  "finalSummary": "...",
  "rootCause": "...",
  "impact": "...",
  "confidence": 0.0,
  "immediateActions": ["..."],
  "furtherChecks": ["..."]
}
仅输出 JSON，不要解释。`, clusterID, string(b), string(f))
}
