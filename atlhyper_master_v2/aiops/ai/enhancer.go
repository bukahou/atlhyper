// atlhyper_master_v2/aiops/ai/enhancer.go
// AIOps AI 增强服务
// 独立于 AIOpsEngine（单向依赖: aiops/ai → aiops）
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps-AI")

// LLMClientFactory 创建 LLM 客户端的工厂函数
// 每次调用返回新实例，调用方负责 Close
type LLMClientFactory func(ctx context.Context) (llm.LLMClient, error)

// SummarizeResponse 事件摘要响应
type SummarizeResponse struct {
	IncidentID       string           `json:"incidentId"`
	Summary          string           `json:"summary"`
	RootCauseAnalysis string          `json:"rootCauseAnalysis"`
	Recommendations  []Recommendation `json:"recommendations"`
	SimilarIncidents []SimilarMatch   `json:"similarIncidents"`
	GeneratedAt      int64            `json:"generatedAt"`
}

// Recommendation 处置建议
type Recommendation struct {
	Priority    int    `json:"priority"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
	Impact      string `json:"impact"`
	IsAutomatic bool   `json:"isAutomatic"`
}

// SimilarMatch 相似事件匹配
type SimilarMatch struct {
	IncidentID string  `json:"incidentId"`
	Similarity float64 `json:"similarity"`
	RootCause  string  `json:"rootCause"`
	Resolution string  `json:"resolution"`
	OccurredAt string  `json:"occurredAt"`
}

// Enhancer AIOps AI 增强服务
type Enhancer struct {
	incidentRepo database.AIOpsIncidentRepository
	llmFactory   LLMClientFactory
}

// NewEnhancer 创建 AI 增强服务
func NewEnhancer(
	incidentRepo database.AIOpsIncidentRepository,
	llmFactory LLMClientFactory,
) *Enhancer {
	return &Enhancer{
		incidentRepo: incidentRepo,
		llmFactory:   llmFactory,
	}
}

// Summarize 生成事件 AI 摘要
func (e *Enhancer) Summarize(ctx context.Context, incidentID string) (*SummarizeResponse, error) {
	// 1. 查询事件数据
	incident, err := e.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}
	if incident == nil {
		return nil, fmt.Errorf("事件不存在: %s", incidentID)
	}

	entities, err := e.incidentRepo.GetEntities(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("查询受影响实体失败: %w", err)
	}

	timeline, err := e.incidentRepo.GetTimeline(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("查询时间线失败: %w", err)
	}

	// 2. 查询历史相似事件（根因实体的历史事件，90天内）
	var historical []*database.AIOpsIncident
	if incident.RootCause != "" {
		since := time.Now().Add(-90 * 24 * time.Hour)
		historical, _ = e.incidentRepo.ListByEntity(ctx, incident.RootCause, since)
	}

	// 3. 构建 LLM 上下文
	incidentCtx := BuildIncidentContext(incident, entities, timeline, historical)

	// 4. 生成 Prompt
	prompt := SummarizePrompt(incidentCtx)

	// 5. 调用 LLM
	client, err := e.llmFactory(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}
	defer client.Close()

	stream, err := client.ChatStream(ctx, &llm.Request{
		SystemPrompt: prompt.System,
		Messages:     []llm.Message{{Role: "user", Content: prompt.User}},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 6. 收集完整响应
	fullText := collectResponse(stream)

	log.Debug("LLM 响应", "incident", incidentID, "len", len(fullText))

	// 7. 解析结构化输出
	return parseResponse(fullText, incidentID, historical)
}

// collectResponse 收集流式响应为完整文本
func collectResponse(stream <-chan *llm.Chunk) string {
	var b strings.Builder
	for chunk := range stream {
		switch chunk.Type {
		case llm.ChunkText:
			b.WriteString(chunk.Content)
		case llm.ChunkError:
			if chunk.Error != nil {
				log.Warn("LLM 流错误", "err", chunk.Error)
			}
		case llm.ChunkDone:
			return b.String()
		}
	}
	return b.String()
}

// parseResponse 解析 LLM 响应为结构化结果
func parseResponse(raw string, incidentID string, historical []*database.AIOpsIncident) (*SummarizeResponse, error) {
	jsonStr := extractJSON(raw)

	var parsed struct {
		Summary           string `json:"summary"`
		RootCauseAnalysis string `json:"rootCauseAnalysis"`
		Recommendations   []struct {
			Priority int    `json:"priority"`
			Action   string `json:"action"`
			Reason   string `json:"reason"`
			Impact   string `json:"impact"`
		} `json:"recommendations"`
		SimilarPattern string `json:"similarPattern"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		// 降级：整段文本作为摘要
		return &SummarizeResponse{
			IncidentID:       incidentID,
			Summary:          raw,
			SimilarIncidents: buildSimilarMatches(historical),
			GeneratedAt:      time.Now().UnixMilli(),
		}, nil
	}

	recommendations := make([]Recommendation, len(parsed.Recommendations))
	for i, r := range parsed.Recommendations {
		recommendations[i] = Recommendation{
			Priority:    r.Priority,
			Action:      r.Action,
			Reason:      r.Reason,
			Impact:      r.Impact,
			IsAutomatic: false,
		}
	}

	return &SummarizeResponse{
		IncidentID:        incidentID,
		Summary:           parsed.Summary,
		RootCauseAnalysis: parsed.RootCauseAnalysis,
		Recommendations:   recommendations,
		SimilarIncidents:  buildSimilarMatches(historical),
		GeneratedAt:       time.Now().UnixMilli(),
	}, nil
}

// extractJSON 从 LLM 输出中提取 JSON 内容
// LLM 可能在 JSON 前后添加 markdown 代码块标记
func extractJSON(text string) string {
	// 尝试提取 ```json ... ``` 中的内容
	if start := strings.Index(text, "```json"); start != -1 {
		jsonStart := start + 7
		if end := strings.Index(text[jsonStart:], "```"); end != -1 {
			return strings.TrimSpace(text[jsonStart : jsonStart+end])
		}
	}

	// 尝试提取 ``` ... ``` 中的内容
	if start := strings.Index(text, "```"); start != -1 {
		codeStart := start + 3
		if end := strings.Index(text[codeStart:], "```"); end != -1 {
			candidate := strings.TrimSpace(text[codeStart : codeStart+end])
			if len(candidate) > 0 && candidate[0] == '{' {
				return candidate
			}
		}
	}

	// 尝试直接找 JSON 对象
	if start := strings.Index(text, "{"); start != -1 {
		if end := strings.LastIndex(text, "}"); end > start {
			return text[start : end+1]
		}
	}

	return text
}

// buildSimilarMatches 从历史事件构建相似事件列表
func buildSimilarMatches(historical []*database.AIOpsIncident) []SimilarMatch {
	if len(historical) == 0 {
		return []SimilarMatch{}
	}

	matches := make([]SimilarMatch, 0, len(historical))
	for i, inc := range historical {
		// 简单相似度：越近的事件相似度越高
		similarity := 0.9 - float64(i)*0.1
		if similarity < 0.3 {
			similarity = 0.3
		}
		matches = append(matches, SimilarMatch{
			IncidentID: inc.ID,
			Similarity: similarity,
			RootCause:  inc.RootCause,
			OccurredAt: inc.StartedAt.Format(time.RFC3339),
		})
	}
	return matches
}
