package service

import (
	"AtlHyper/atlhyper_aiservice/client/ai"
	"AtlHyper/atlhyper_aiservice/config"
	m "AtlHyper/model/event"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

func RunStage1Analysis(clusterID string, events []m.EventLog) (map[string]interface{}, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events to analyze")
	}

	// ① 构造 prompt
	prompt := buildStage1Prompt(clusterID, events)

	// ② 调 Gemini
	cfg := config.GetGeminiConfig()
	ctx := context.Background()
	c, err := ai.GetGeminiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gemini client failed: %v", err)
	}
	model := c.GenerativeModel(cfg.ModelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %v", err)
	}

	out := ""
	for _, p := range resp.Candidates[0].Content.Parts {
		out += fmt.Sprintf("%v", p)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		if idx := strings.Index(out, "{"); idx != -1 {
			_ = json.Unmarshal([]byte(out[idx:]), &parsed)
		}
	}
	if parsed == nil {
		parsed = map[string]interface{}{"raw": out}
	}

	return map[string]interface{}{
		"summary": fmt.Sprintf("✅ 初步分析完成（cluster=%s）", clusterID),
		"prompt":  prompt,
		"ai_json": parsed,
		"ai_raw":  out,
	}, nil
}

func buildStage1Prompt(clusterID string, events []m.EventLog) string {
	grouped := map[string][]m.EventLog{}
	for _, e := range events {
		key := e.Severity
		if key == "" {
			key = "Unknown"
		}
		grouped[key] = append(grouped[key], e)
	}
	sevs := make([]string, 0, len(grouped))
	for k := range grouped {
		sevs = append(sevs, k)
	}
	sort.Strings(sevs)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("集群 ID: %s\n", clusterID))
	sb.WriteString("以下是最新检测到的 Kubernetes 事件（按严重性分组）：\n\n")
	for _, s := range sevs {
		sb.WriteString(fmt.Sprintf("【%s 级事件】\n", s))
		for _, e := range grouped[s] {
			sb.WriteString(fmt.Sprintf("- [%s] %s/%s → %s: %s\n", e.Kind, e.Namespace, e.Node, e.Reason, e.Message))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("请严格输出 JSON，结构如下：{ summary, rootCause, impact, recommendation, needResources }\n")
	return sb.String()
}
