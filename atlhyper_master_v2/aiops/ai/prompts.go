// atlhyper_master_v2/aiops/ai/prompts.go
// AIOps 专用 Prompt 模板
package ai

import "fmt"

// SystemPrompt AIOps 事件分析系统提示词
const SystemPrompt = `你是 AtlHyper 平台的 AIOps 分析引擎。你的任务是分析 Kubernetes 集群的运维事件，
提供根因分析、处置建议和历史模式匹配。

要求:
1. 根因分析必须基于提供的数据，不要臆测
2. 处置建议必须具体可执行，按优先级排列
3. 如果有历史相似事件，指出模式和趋势
4. 使用技术精确的语言，避免模糊表述
5. 输出格式严格遵循 JSON Schema

输出格式:
{
  "summary": "一段话概述事件（什么时间，什么实体，什么问题，什么影响）",
  "rootCauseAnalysis": "详细分析根因链路（从源头到影响面）",
  "recommendations": [
    {
      "priority": 1,
      "action": "具体操作步骤",
      "reason": "为什么这样做",
      "impact": "预期效果"
    }
  ],
  "similarPattern": "如果有历史相似事件，描述模式和建议"
}`

// UserPromptTemplate 用户消息模板
const UserPromptTemplate = `请分析以下 Kubernetes 集群事件:

%s

请按照指定的 JSON 格式输出分析结果。`

// PromptPair 系统提示词 + 用户消息
type PromptPair struct {
	System string
	User   string
}

// SummarizePrompt 组装完整 Prompt
func SummarizePrompt(ctx *IncidentContext) *PromptPair {
	userContent := fmt.Sprintf(UserPromptTemplate,
		ctx.IncidentSummary+"\n\n"+
			ctx.RootCauseEntity+"\n\n"+
			ctx.AffectedEntities+"\n\n"+
			ctx.TimelineText+"\n\n"+
			ctx.HistoricalContext,
	)
	return &PromptPair{
		System: SystemPrompt,
		User:   userContent,
	}
}
