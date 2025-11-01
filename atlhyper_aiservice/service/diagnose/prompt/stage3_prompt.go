package prompt

import (
	"encoding/json"
	"fmt"
)

// BuildStage3Prompt —— 阶段3提示词构造：融合初判与上下文生成最终诊断输入
func BuildStage3Prompt(clusterID string, stage1, stage2 map[string]interface{}) string {
	b, _ := json.MarshalIndent(stage1, "", "  ")
	f, _ := json.MarshalIndent(stage2, "", "  ")

	return fmt.Sprintf(`集群 ID: %s

========================
 阶段一：AI 初步分析结果
========================
%s

========================
 阶段二：Master 上下文资源
========================
%s

========================
 任务说明
========================
请综合以上两部分内容，输出如下 JSON：
{
  "finalSummary": "...",
  "rootCause": "...",
  "impact": "...",
  "confidence": 0.0,
  "immediateActions": [],
  "furtherChecks": []
}

规则：
1. 严格 JSON 格式，不得包含解释性文字。
2. 字段必须完整存在。
3. 若不确定，请以 "unknown" 或 confidence 低值标注。
`, clusterID, string(b), string(f))
}
