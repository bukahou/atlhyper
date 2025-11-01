// atlhyper_aiservice/service/diagnose/stage3_service.go
package diagnose

import (
	"AtlHyper/atlhyper_aiservice/client/ai"
	"AtlHyper/atlhyper_aiservice/service/diagnose/prompt"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RunStage3FinalDiagnosis —— 阶段三：最终综合诊断分析
// ------------------------------------------------------------
// 基于阶段一（AI 初步分析）与阶段二（Master 上下文资源），
// 进行最终的上下文一致性诊断，返回结构化结果。
func RunStage3FinalDiagnosis(clusterID string, stage1, stage2 map[string]interface{}) (map[string]interface{}, error) {
	ctx := context.Background()

	// 🧠 Step 1. 构造 Prompt（融合前两阶段输出）
	prompt := prompt.BuildStage3Prompt(clusterID, stage1, stage2)

	// ⚙️ Step 2. 调用通用 AI 接口（内部自动完成客户端初始化与关闭）
	out, err := ai.GenerateText(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI final diagnosis failed: %v", err)
	}

	// 🧩 Step 3. 尝试解析输出为 JSON（与前面阶段保持一致）
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		if idx := strings.Index(out, "{"); idx != -1 {
			_ = json.Unmarshal([]byte(out[idx:]), &parsed)
		}
	}

	// 🧱 Step 4. 若无法解析则保留原始输出
	if parsed == nil {
		parsed = map[string]interface{}{"raw": out}
	}

	// 🧾 Step 5. 返回统一结构
	return map[string]interface{}{
		"summary": fmt.Sprintf("✅ 阶段三诊断完成（cluster=%s）", clusterID),
		"prompt":  prompt,
		"ai_raw":  out,
		"ai_json": parsed,
	}, nil
}
// func buildStage3Prompt(clusterID string, stage1, stage2 map[string]interface{}) string {
// 	// 序列化前两阶段结果
// 	b, _ := json.MarshalIndent(stage1, "", "  ")
// 	f, _ := json.MarshalIndent(stage2, "", "  ")

// 	return fmt.Sprintf(`集群 ID: %s

// ========================
//  阶段一：AI 初步分析结果
// ========================
// 以下为 AI 对事件日志的初步推理输出（含可能的根因、影响、修复建议及资源需求清单）：
// %s

// ========================
//  阶段二：Master 上下文资源
// ========================
// 以下为 Master 根据 needResources 清单返回的真实资源数据，
// 包括 Pod / Deployment / Service / Node / Metrics 等结构化详情：
// %s

// ========================
//  任务说明
// ========================
// 请你结合「阶段一 AI 初判」与「阶段二 上下文资源数据」，
// 进行一次更全面、上下文一致的诊断分析。

// 要求：
// 1 请严格以 JSON 格式输出结果（不要额外解释文字）。
// 2 所有结论、推测、建议必须**基于上述上下文中的实际内容**。
// 3 若部分信息不足以确定，请在 JSON 中注明 "confidence": 低，而不是编造。
// 4 不要输出任何自然语言解释。

// ========================
//  输出 JSON 模板（必须完整填写）
// ========================
// {
//   "finalSummary": "string —— 对整个事件的总体概述（简明扼要）。",
//   "rootCause": "string —— 问题的主要原因分析，需结合上下文验证。",
//   "impact": "string —— 问题对集群和服务的影响范围。",
//   "confidence": 0.0 —— 数值型，范围 0~1，代表分析置信度。",
//   "immediateActions": [
//     "string —— 推荐的即时修复措施（可多条）"
//   ],
//   "furtherChecks": [
//     "string —— 建议后续进一步验证的方向（如日志、Metrics、ConfigMap、Pod 状态等）"
//   ]
// }

// ⚠️ 输出规则：
// - 仅输出 JSON，不要包含解释说明或文字分析。
// - 所有字段必须存在，即使内容为空字符串。
// - 若无法判断某项，请填写 "unknown"。
// `, clusterID, string(b), string(f))
// }
