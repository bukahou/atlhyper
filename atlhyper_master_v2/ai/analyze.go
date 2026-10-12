// atlhyper_master_v2/ai/analyze.go
// 通用多轮 Tool Calling 循环（非交互式，后台执行）
// 从 aiops/ai/analysis.go 提取的通用逻辑
package ai

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/prompts"
)

const (
	defaultAnalyzeMaxRounds = 8               // 默认最大调查轮次
	defaultAnalyzeTimeout   = 5 * time.Minute // 默认分析超时
	maxToolCallsPerAnalyze  = 5               // 每轮最多 Tool Call 数
)

// Analyze 执行非交互式多轮 Tool Calling 分析
func (s *aiServiceImpl) Analyze(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResult, error) {
	// 默认值
	maxRounds := req.MaxRounds
	if maxRounds <= 0 {
		maxRounds = defaultAnalyzeMaxRounds
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultAnalyzeTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 获取角色配置（Provider + 预算检查）
	role := req.Role
	if role == "" {
		role = RoleAnalysis
	}
	roleCfg, err := s.loadAIConfigForRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("AI 配置错误: %w", err)
	}

	// 创建 LLM 客户端
	llmClient, err := llm.NewLLMClient(roleCfg.Config)
	if err != nil {
		s.recordProviderError(ctx, roleCfg.ProviderID, fmt.Sprintf("创建客户端失败: %v", err))
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}
	defer llmClient.Close()

	log.Info("分析开始",
		"cluster", req.ClusterID,
		"role", role,
		"provider", roleCfg.ProviderName,
		"model", roleCfg.Config.Model,
		"maxRounds", maxRounds,
	)

	exec := func(ctx context.Context, tc *llm.ToolCall) (string, error) {
		return s.executor.Execute(ctx, req.ClusterID, tc)
	}
	out, err := runToolLoop(ctx, llmClient, exec, loopParams{
		SystemPrompt:  req.SystemPrompt,
		UserPrompt:    req.UserPrompt,
		MaxRounds:     maxRounds,
		Tools:         prompts.GetToolDefinitions(),
		MaxToolResult: toolResultMaxLen(roleCfg.ContextWindow),
	})
	s.RecordUsage(ctx, role, roleCfg.ProviderID, out.InputTokens, out.OutputTokens)
	if err != nil {
		s.recordProviderError(ctx, roleCfg.ProviderID, err.Error())
		return nil, err
	}
	if out.Response != "" {
		s.clearProviderError(ctx, roleCfg.ProviderID)
	}

	log.Info("分析完成",
		"cluster", req.ClusterID,
		"rounds", out.Rounds,
		"tools", out.ToolCalls,
		"tokens", out.InputTokens+out.OutputTokens,
	)
	return &AnalyzeResult{
		Response:     out.Response,
		ToolCalls:    out.ToolCalls,
		InputTokens:  out.InputTokens,
		OutputTokens: out.OutputTokens,
		ProviderName: roleCfg.ProviderName,
		Model:        roleCfg.Config.Model,
		Steps:        out.Steps,
	}, nil
}
