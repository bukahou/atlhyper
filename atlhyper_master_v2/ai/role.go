// atlhyper_master_v2/ai/role.go
// AI 角色常量、路由配置与解析
package ai

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
)

// 角色常量
const (
	RoleBackground = "background" // 后台分析（AIOps 事件摘要）
	RoleChat       = "chat"       // 交互对话（用户 Chat）
	RoleAnalysis   = "analysis"   // 深度分析（高危事件调查）
)

// ValidRoles 有效角色列表
var ValidRoles = []string{RoleBackground, RoleChat, RoleAnalysis}

// IsValidRole 检查角色是否有效
func IsValidRole(role string) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// RoleConfig 角色路由解析结果
type RoleConfig struct {
	llm.Config
	ContextWindow int    // 有效上下文窗口（模型默认值 or 用户覆盖值）
	ProviderID    int64  // Provider ID（用于统计）
	ProviderName  string // Provider 名称（用于日志/报告）
}

// loadAIConfigForRole 按角色加载 AI 配置
// 解析优先级:
//  1. 查找持有该角色的 Provider → 检查预算 → 返回
//  2. 预算耗尽 → 使用 fallback Provider
//  3. 无角色分配 → 退回 ai_active_config.provider_id（向后兼容）
func (s *aiServiceImpl) loadAIConfigForRole(ctx context.Context, role string) (*RoleConfig, error) {
	// 1. 检查 AI 总开关
	active, err := s.activeRepo.Get(ctx)
	if err != nil || active == nil || !active.Enabled {
		return nil, fmt.Errorf("AI 功能未启用")
	}

	// 2. 查找持有该角色的 Provider
	providers, _ := s.providerRepo.List(ctx)
	for _, p := range providers {
		if !containsRole(p.Roles, role) {
			continue
		}

		// 找到了持有该角色的 Provider → 检查预算
		if s.budgetRepo != nil {
			if budget, _ := s.budgetRepo.Get(ctx, role); budget != nil {
				if !checkBudget(budget) {
					// 预算耗尽 → 尝试 fallback
					if budget.FallbackProviderID != nil {
						fallback, err := s.providerRepo.GetByID(ctx, *budget.FallbackProviderID)
						if err == nil && fallback != nil {
							log.Warn("角色预算耗尽，使用降级 Provider",
								"role", role, "fallback", fallback.Name)
							return s.providerToRoleConfig(ctx, fallback), nil
						}
					}
					return nil, fmt.Errorf("角色 %s 每日预算已用尽", role)
				}
			}
		}

		return s.providerToRoleConfig(ctx, p), nil
	}

	// 3. 无角色分配 → 退回全局 (向后兼容)
	return s.loadAIConfigFallback(ctx, active)
}

// loadAIConfigFallback 使用 ai_active_config.provider_id 作为兜底
func (s *aiServiceImpl) loadAIConfigFallback(ctx context.Context, active *database.AIActiveConfig) (*RoleConfig, error) {
	if active.ProviderID == nil {
		return nil, fmt.Errorf("未设置 AI 提供商")
	}

	provider, err := s.providerRepo.GetByID(ctx, *active.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("获取 AI 提供商失败: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("AI 提供商不存在: %d", *active.ProviderID)
	}

	return s.providerToRoleConfig(ctx, provider), nil
}

// providerToRoleConfig 将 Provider 转换为 RoleConfig
func (s *aiServiceImpl) providerToRoleConfig(ctx context.Context, p *database.AIProvider) *RoleConfig {
	cfg := &RoleConfig{
		Config: llm.Config{
			Provider: p.Provider,
			APIKey:   p.APIKey,
			Model:    p.Model,
			BaseURL:  p.BaseURL,
		},
		ProviderID:   p.ID,
		ProviderName: p.Name,
	}

	// 解析有效上下文窗口
	cfg.ContextWindow = EffectiveContextWindow(ctx, p, s.modelRepo)
	return cfg
}

// EffectiveContextWindow 获取 Provider 的有效 context_window
func EffectiveContextWindow(ctx context.Context, provider *database.AIProvider, modelRepo database.AIProviderModelRepository) int {
	// 用户覆盖优先
	if provider.ContextWindowOverride > 0 {
		return provider.ContextWindowOverride
	}
	// 查模型默认值
	if modelRepo != nil {
		models, _ := modelRepo.ListByProvider(ctx, provider.Provider)
		for _, m := range models {
			if m.Model == provider.Model && m.ContextWindow > 0 {
				return m.ContextWindow
			}
		}
	}
	return 0
}

// checkBudget 检查预算是否还有余额
func checkBudget(budget *database.AIRoleBudget) bool {
	// 检查是否需要跨日重置
	if budget.DailyResetAt != nil {
		now := time.Now()
		resetDate := budget.DailyResetAt.Truncate(24 * time.Hour)
		today := now.Truncate(24 * time.Hour)
		if today.After(resetDate) {
			// 跨日了，视为有余额（调用方负责重置）
			return true
		}
	}

	// Token 限额
	if budget.DailyTokenLimit > 0 && budget.DailyTokensUsed >= budget.DailyTokenLimit {
		return false
	}
	// 调用次数限额
	if budget.DailyCallLimit > 0 && budget.DailyCallsUsed >= budget.DailyCallLimit {
		return false
	}
	return true
}

// containsRole 检查角色列表中是否包含指定角色
func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// MaxPromptCharsForContext 根据 context_window 计算 Prompt 最大字符数
// 供 AIOps Enhancer 使用
func MaxPromptCharsForContext(contextWindow int) int {
	if contextWindow <= 0 {
		return 16000 // 云端大模型: 保持默认值
	}
	// 上下文窗口的 50% 给 prompt（另 50% 给输出 + overhead）
	// 1 token ≈ 2.5 chars
	chars := contextWindow / 2 * 25 / 10
	if chars < 2000 {
		return 2000
	}
	return chars
}
