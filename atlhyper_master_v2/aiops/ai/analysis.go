// atlhyper_master_v2/aiops/ai/analysis.go
// analysis 角色：深度分析循环（多轮 Tool Calling）
// 复用 chat 的 Tool 基础设施，以非交互模式后台执行
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/prompts"
	"AtlHyper/atlhyper_master_v2/database"
)

const (
	maxAnalysisRounds       = 8    // 最大调查轮次
	maxToolCallsPerAnalysis = 5    // 每轮最多 Tool Call 数
	analysisTimeout         = 5 * time.Minute
)

// ToolExecuteFunc Tool 执行函数（由 ai 包注入，避免循环依赖）
type ToolExecuteFunc func(ctx context.Context, clusterID string, tc *llm.ToolCall) (string, error)

// AnalysisConfig 深度分析配置
type AnalysisConfig struct {
	LLMFactory     LLMClientFactory
	ToolExecute    ToolExecuteFunc
	ToolDefs       []llm.ToolDefinition
	ReportRepo     database.AIReportRepository
	IncidentRepo   database.AIOpsIncidentRepository
	BudgetRepo     database.AIRoleBudgetRepository // 预算检查（可选）
	RecordUsage    RecordUsageFunc                  // 预算扣减回调（可选）
}

// InvestigationStep 调查步骤记录
type InvestigationStep struct {
	Round     int                  `json:"round"`
	Thinking  string               `json:"thinking"`
	ToolCalls []InvestigationTool  `json:"toolCalls"`
}

// InvestigationTool 调查中的 Tool 调用记录
type InvestigationTool struct {
	Tool          string `json:"tool"`
	Params        string `json:"params"`
	ResultSummary string `json:"resultSummary"`
}

// RunAnalysis 执行深度分析（后台静默，无 SSE）
func RunAnalysis(ctx context.Context, cfg AnalysisConfig, incidentID, trigger string) error {
	// 0. 预算前置检查（防止自动触发导致成本失控）
	if cfg.BudgetRepo != nil {
		budget, _ := cfg.BudgetRepo.Get(ctx, "analysis")
		if budget != nil && !isBudgetAvailable(budget) {
			return fmt.Errorf("analysis 角色预算已用尽")
		}
	}

	ctx, cancel := context.WithTimeout(ctx, analysisTimeout)
	defer cancel()

	// 1. 查询事件数据
	incident, err := cfg.IncidentRepo.GetByID(ctx, incidentID)
	if err != nil || incident == nil {
		return fmt.Errorf("查询事件失败: %w", err)
	}

	entities, _ := cfg.IncidentRepo.GetEntities(ctx, incidentID)
	timeline, _ := cfg.IncidentRepo.GetTimeline(ctx, incidentID)

	// 2. 创建 LLM 客户端
	client, contextWindow, meta, err := cfg.LLMFactory(ctx)
	if err != nil {
		return fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}
	defer client.Close()

	// 3. 构建初始上下文
	incidentCtx := BuildIncidentContext(incident, entities, timeline, nil)
	systemPrompt := prompts.BuildAnalysisPrompt()
	userPrompt := prompts.BuildAnalysisUserPrompt(incidentCtx)

	messages := []llm.Message{
		{Role: "user", Content: userPrompt},
	}

	// 4. 创建报告记录（先创建，每轮追加 investigation_steps）
	report := &database.AIReport{
		IncidentID:         incidentID,
		ClusterID:          incident.ClusterID,
		Role:               "analysis",
		Trigger:            trigger,
		InvestigationSteps: "[]",
		CreatedAt:          time.Now(),
	}
	if meta != nil {
		report.ProviderName = meta.ProviderName
		report.Model = meta.Model
	}
	if cfg.ReportRepo != nil {
		if err := cfg.ReportRepo.Create(ctx, report); err != nil {
			log.Warn("创建分析报告失败", "err", err)
		}
	}

	var steps []InvestigationStep
	var totalInputTokens, totalOutputTokens int
	maxToolResult := toolResultMaxLen(contextWindow)
	startTime := time.Now()

	// 5. 多轮 Tool Calling 循环
	for round := 1; round <= maxAnalysisRounds; round++ {
		remaining := maxAnalysisRounds - round + 1
		roundHint := fmt.Sprintf(
			"\n\n[系统提示] 剩余调查轮次: %d/%d。如果信息已足够，请返回最终分析报告，不要调用 Tool。",
			remaining, maxAnalysisRounds,
		)

		llmReq := &llm.Request{
			SystemPrompt: systemPrompt + roundHint,
			Messages:     messages,
			Tools:        cfg.ToolDefs,
		}

		// 最后一轮不提供 tools，强制输出
		if round == maxAnalysisRounds {
			llmReq.SystemPrompt = systemPrompt + "\n\n[系统提示] 这是最后一轮。请基于已有信息输出最终分析报告。不要再调用 Tool。"
			llmReq.Tools = nil
		}

		stream, err := client.ChatStream(ctx, llmReq)
		if err != nil {
			return fmt.Errorf("第 %d 轮 LLM 调用失败: %w", round, err)
		}

		// 收集响应
		text, toolCalls, usage := collectAnalysisResponse(stream)

		// 累计 token 使用量
		if usage != nil {
			totalInputTokens += usage.InputTokens
			totalOutputTokens += usage.OutputTokens
		}

		log.Debug("analysis 轮次完成",
			"incident", incidentID, "round", round,
			"text_len", len(text), "tool_calls", len(toolCalls))

		// 没有 Tool Call → 分析结束
		if len(toolCalls) == 0 {
			// 最终文本作为报告
			finalReport := parseAnalysisReport(text, incidentID)
			if cfg.ReportRepo != nil && report.ID > 0 {
				saveAnalysisResult(ctx, cfg.ReportRepo, report.ID, finalReport, steps, startTime, totalInputTokens, totalOutputTokens)
			}
			// 扣减预算
			providerID := int64(0)
			if meta != nil {
				providerID = meta.ProviderID
			}
			if cfg.RecordUsage != nil {
				cfg.RecordUsage(ctx, "analysis", providerID, totalInputTokens, totalOutputTokens)
			}
			log.Info("深度分析完成",
				"incident", incidentID, "rounds", round-1,
				"tokens", totalInputTokens+totalOutputTokens,
				"duration", time.Since(startTime).Round(time.Second))
			return nil
		}

		// 执行 Tool Calls
		if len(toolCalls) > maxToolCallsPerAnalysis {
			toolCalls = toolCalls[:maxToolCallsPerAnalysis]
		}

		step := InvestigationStep{
			Round:    round,
			Thinking: text,
		}

		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   text,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			result, err := cfg.ToolExecute(ctx, incident.ClusterID, &tc)
			if err != nil {
				result = fmt.Sprintf("执行失败: %v", err)
			}

			step.ToolCalls = append(step.ToolCalls, InvestigationTool{
				Tool:          tc.Name,
				Params:        tc.Params,
				ResultSummary: truncateForStep(result, 500),
			})

			messages = append(messages, llm.Message{
				Role: "tool",
				ToolResult: &llm.ToolResult{
					CallID:  tc.ID,
					Name:    tc.Name,
					Content: truncateForStep(result, maxToolResult),
				},
			})
		}

		steps = append(steps, step)

		// 每轮写 DB（防止中途崩溃丢失已有步骤）
		if cfg.ReportRepo != nil && report.ID > 0 {
			stepsJSON, _ := json.Marshal(steps)
			cfg.ReportRepo.UpdateInvestigationSteps(ctx, report.ID, string(stepsJSON))
		}
	}

	// 循环用尽后扣减预算
	providerID := int64(0)
	if meta != nil {
		providerID = meta.ProviderID
	}
	if cfg.RecordUsage != nil {
		cfg.RecordUsage(ctx, "analysis", providerID, totalInputTokens, totalOutputTokens)
	}

	return nil
}

// collectAnalysisResponse 收集流式响应（不推送 SSE）
func collectAnalysisResponse(stream <-chan *llm.Chunk) (string, []llm.ToolCall, *llm.Usage) {
	var text strings.Builder
	var toolCalls []llm.ToolCall

	for chunk := range stream {
		switch chunk.Type {
		case llm.ChunkText:
			text.WriteString(chunk.Content)
		case llm.ChunkToolCall:
			if chunk.ToolCall != nil {
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}
		case llm.ChunkDone:
			return text.String(), toolCalls, chunk.Usage
		case llm.ChunkError:
			if chunk.Error != nil {
				log.Warn("analysis LLM 流错误", "err", chunk.Error)
			}
			return text.String(), toolCalls, nil
		}
	}
	return text.String(), toolCalls, nil
}

// parseAnalysisReport 从 LLM 输出中解析分析报告
func parseAnalysisReport(raw, incidentID string) *SummarizeResponse {
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
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return &SummarizeResponse{
			IncidentID:  incidentID,
			Summary:     raw,
			GeneratedAt: time.Now().UnixMilli(),
		}
	}

	recs := make([]Recommendation, len(parsed.Recommendations))
	for i, r := range parsed.Recommendations {
		recs[i] = Recommendation{
			Priority: r.Priority,
			Action:   r.Action,
			Reason:   r.Reason,
			Impact:   r.Impact,
		}
	}

	return &SummarizeResponse{
		IncidentID:        incidentID,
		Summary:           parsed.Summary,
		RootCauseAnalysis: parsed.RootCauseAnalysis,
		Recommendations:   recs,
		GeneratedAt:       time.Now().UnixMilli(),
	}
}

// saveAnalysisResult 保存分析结果到报告（完整更新所有字段）
func saveAnalysisResult(ctx context.Context, repo database.AIReportRepository, reportID int64, result *SummarizeResponse, steps []InvestigationStep, startTime time.Time, inputTokens, outputTokens int) {
	report, err := repo.GetByID(ctx, reportID)
	if err != nil || report == nil {
		return
	}

	report.Summary = result.Summary
	report.RootCauseAnalysis = result.RootCauseAnalysis
	report.DurationMs = time.Since(startTime).Milliseconds()
	report.InputTokens = inputTokens
	report.OutputTokens = outputTokens

	recsJSON, _ := json.Marshal(result.Recommendations)
	report.Recommendations = string(recsJSON)

	stepsJSON, _ := json.Marshal(steps)
	report.InvestigationSteps = string(stepsJSON)

	// 完整更新报告（包括 summary、recommendations、tokens 等）
	if err := repo.UpdateResult(ctx, reportID, report); err != nil {
		log.Warn("更新分析报告失败", "id", reportID, "err", err)
	}
}

// truncateForStep 截断结果用于调查步骤记录
func truncateForStep(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// toolResultMaxLen 根据上下文窗口计算 Tool 结果最大长度
// 从 context.go 中导出供 analysis 使用
func toolResultMaxLen(contextWindow int) int {
	if contextWindow <= 0 {
		return 32000
	}
	if contextWindow <= 8192 {
		return 2000
	}
	if contextWindow <= 32768 {
		return 8000
	}
	return 32000
}
