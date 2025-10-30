package insight

import (
	"AtlHyper/atlhyper_aiservice/client/ai"
	"AtlHyper/atlhyper_aiservice/prompt"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RunInsightAnalysis —— 通用运维 AI 洞察分析
// ------------------------------------------------------------
// 输入一段自然语言的系统摘要，输出结构化诊断结果。
// 适用于 CPU/Memory 异常、磁盘告警、网络中断、日志分析等场景。
func RunInsightAnalysis(summary string) (map[string]interface{}, error) {
	ctx := context.Background()
	promptText := prompt.BuildInsightPrompt(summary)

	// ✅ 单步调用（自动创建 & 关闭 AI 客户端）
	out, err := ai.GenerateText(ctx, promptText)
	if err != nil {
		return nil, fmt.Errorf("AI 调用失败: %v", err)
	}

	// 🧹 Step 1. 清理 Markdown 包裹（如 ```json ... ```）
	clean := strings.TrimSpace(out)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	// 🧩 Step 2. 尝试解析 JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(clean), &parsed); err != nil {
		if idx := strings.Index(clean, "{"); idx != -1 {
			_ = json.Unmarshal([]byte(clean[idx:]), &parsed)
		}
	}

	// 🧱 Step 3. 无法解析时直接返回原始输出
	if parsed == nil {
		return map[string]interface{}{"raw": out}, nil
	}

	return parsed, nil
}