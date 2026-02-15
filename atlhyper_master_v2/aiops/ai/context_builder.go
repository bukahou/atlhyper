// atlhyper_master_v2/aiops/ai/context_builder.go
// 将结构化事件数据转换为 LLM 可理解的文本描述
package ai

import (
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// IncidentContext LLM 输入上下文
type IncidentContext struct {
	IncidentSummary  string // 事件基本信息
	TimelineText     string // 时间线叙述
	AffectedEntities string // 受影响实体及其风险评分
	RootCauseEntity  string // 根因实体详情
	HistoricalContext string // 历史相似事件
}

// BuildIncidentContext 从结构化数据构建 LLM 上下文
func BuildIncidentContext(
	incident *database.AIOpsIncident,
	entities []*database.AIOpsIncidentEntity,
	timeline []*database.AIOpsIncidentTimeline,
	historical []*database.AIOpsIncident,
) *IncidentContext {
	return &IncidentContext{
		IncidentSummary:  buildSummary(incident),
		RootCauseEntity:  buildRootCause(entities),
		AffectedEntities: buildEntities(entities),
		TimelineText:     buildTimeline(timeline),
		HistoricalContext: buildHistorical(historical),
	}
}

// buildSummary 构建事件概要
func buildSummary(inc *database.AIOpsIncident) string {
	if inc == nil {
		return "事件概要: 无数据"
	}

	duration := formatDuration(inc.DurationS)
	resolved := "进行中"
	if inc.ResolvedAt != nil {
		resolved = fmt.Sprintf("已解决 (%s)", inc.ResolvedAt.Format("2006-01-02 15:04"))
	}

	return fmt.Sprintf(`事件概要:
  ID: %s
  状态: %s | 严重度: %s | 持续: %s
  集群: %s
  开始时间: %s
  解决状态: %s
  峰值风险: %.0f | 复发次数: %d`,
		inc.ID,
		inc.State, inc.Severity, duration,
		inc.ClusterID,
		inc.StartedAt.Format("2006-01-02 15:04:05"),
		resolved,
		inc.PeakRisk, inc.Recurrence,
	)
}

// buildRootCause 构建根因实体描述
func buildRootCause(entities []*database.AIOpsIncidentEntity) string {
	for _, e := range entities {
		if e.Role == "root_cause" {
			return fmt.Sprintf(`根因实体:
  %s (角色: root_cause)
  R_local: %.2f | R_final: %.2f`,
				e.EntityKey, e.RLocal, e.RFinal)
		}
	}
	return "根因实体: 未识别"
}

// buildEntities 构建受影响实体列表
func buildEntities(entities []*database.AIOpsIncidentEntity) string {
	if len(entities) == 0 {
		return "受影响实体: 无"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("受影响实体 (%d 个):\n", len(entities)))
	for i, e := range entities {
		b.WriteString(fmt.Sprintf("  %d. %-30s %-12s R=%.2f\n",
			i+1, e.EntityKey, e.Role, e.RFinal))
	}
	return b.String()
}

// buildTimeline 构建时间线描述
func buildTimeline(timeline []*database.AIOpsIncidentTimeline) string {
	if len(timeline) == 0 {
		return "时间线: 无记录"
	}

	var b strings.Builder
	b.WriteString("时间线:\n")
	for _, t := range timeline {
		ts := t.Timestamp.Format("15:04:05")
		detail := t.Detail
		if len(detail) > 200 {
			detail = detail[:200] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s [%s] %s %s\n",
			ts, t.EventType, t.EntityKey, detail))
	}
	return b.String()
}

// buildHistorical 构建历史事件描述
func buildHistorical(incidents []*database.AIOpsIncident) string {
	if len(incidents) == 0 {
		return "历史相似事件: 无"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("历史相似事件 (%d 个):\n", len(incidents)))
	for i, inc := range incidents {
		duration := formatDuration(inc.DurationS)
		b.WriteString(fmt.Sprintf("  %d. %s (%s) — %s, 持续 %s\n",
			i+1, inc.ID, inc.StartedAt.Format("2006-01-02"), inc.RootCause, duration))
	}
	return b.String()
}

// formatDuration 格式化持续时间（秒 → 人类可读）
func formatDuration(seconds int64) string {
	if seconds <= 0 {
		return "进行中"
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return fmt.Sprintf("%d 秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1f 小时", d.Hours())
}
