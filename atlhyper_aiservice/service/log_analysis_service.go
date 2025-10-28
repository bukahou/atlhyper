// atlhyper_aiservice/service/log_analysis_service.go
package service

import (
	"AtlHyper/atlhyper_aiservice/client"
	"AtlHyper/atlhyper_aiservice/config"
	m "AtlHyper/model/event"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/generative-ai-go/genai"
)

func DiagnoseEvents(clusterID string, events []m.EventLog) (map[string]interface{}, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("no events to analyze")
	}

	// 1️⃣ 按严重级别分组
	grouped := map[string][]m.EventLog{}
	for _, e := range events {
		key := e.Severity
		if key == "" {
			key = "Unknown"
		}
		grouped[key] = append(grouped[key], e)
	}
	severities := make([]string, 0, len(grouped))
	for sev := range grouped {
		severities = append(severities, sev)
	}
	sort.Strings(severities)

	// 2️⃣ 构造 prompt
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("集群 ID: %s\n", clusterID))
	sb.WriteString("以下是最新检测到的 Kubernetes 事件（按严重性分组）：\n\n")
	for _, sev := range severities {
		sb.WriteString(fmt.Sprintf("【%s 级事件】\n", sev))
		for _, e := range grouped[sev] {
			sb.WriteString(fmt.Sprintf(
				"- [%s] %s/%s → %s: %s\n  ↳ Message: %s (Time: %s)\n",
				e.Kind, e.Namespace, e.Node, e.Reason, e.Message, e.Category, e.EventTime))
		}
		sb.WriteString("\n")
	}

	// ✅ 明确要求严格 JSON 输出
	sb.WriteString("请严格以 JSON 格式回答，结构如下：\n")
	sb.WriteString(`{
  "summary": "一句话结论",
  "rootCause": "主要根因说明",
  "impact": "可能影响",
  "recommendation": "后续建议",
  "relatedResources": ["Pod", "Node", "Service"]
}
`)
	sb.WriteString("仅输出 JSON，不要包含解释或额外文字。\n")

	prompt := sb.String()

	// 3️⃣ 调用 Gemini
	cfg := config.GetGeminiConfig()
	ctx := context.Background()
	c, err := client.GetGeminiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Gemini 客户端失败: %v", err)
	}
	model := c.GenerativeModel(cfg.ModelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("AI 分析失败: %v", err)
	}
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("AI 无响应")
	}

	out := ""
	for _, p := range resp.Candidates[0].Content.Parts {
		out += fmt.Sprintf("%v", p)
	}

	// 4️⃣ 尝试解析 JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		// 容错：AI 可能带前缀文字，尝试截取第一个 { 开始部分
		if idx := strings.Index(out, "{"); idx != -1 {
			cut := out[idx:]
			_ = json.Unmarshal([]byte(cut), &parsed)
		}
	}
	if parsed == nil {
		parsed = map[string]interface{}{"raw": out}
	}

	// 5️⃣ 返回统一结构
	return map[string]interface{}{
		"summary": fmt.Sprintf("✅ 初步分析完成（cluster=%s）", clusterID),
		"prompt":  prompt,
		"ai_json": parsed, // 结构化输出
		"ai_raw":  out,    // 原始文本，便于调试
	}, nil
}
