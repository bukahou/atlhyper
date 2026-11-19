// atlhyper_master_v2/ai/role.go
// AI 角色常量、路由配置与解析
package ai

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master_v2/ai/budget"
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
//  3. 无角色分配 → 返回错误
func (s *aiServiceImpl) loadAIConfigForRole(ctx context.Context, role string) (*RoleConfig, error) {
	// 1. 查找持有该角色的 Provider
	providers, err := s.providerRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取 AI 提供商: %w", err)
	}
	for _, p := range providers {
		if !containsRole(p.Roles, role) {
			continue
		}

		// 找到了持有该角色的 Provider → 检查预算（跨日 / 跨月清零在 budget.Ledger 里）
		if s.budgets != nil {
			current, err := s.budgets.Current(ctx, role)
			if err != nil {
				return nil, err
			}
			if !budget.HasRemaining(current) {
				// 预算耗尽 → 尝试 fallback
				if current.FallbackProviderID != nil {
					fallback, err := s.providerRepo.GetByID(ctx, *current.FallbackProviderID)
					if err == nil && fallback != nil {
						log.Warn("角色预算耗尽，使用降级 Provider",
							"role", role, "fallback", fallback.Name)
						return s.providerToRoleConfig(ctx, fallback), nil
					}
				}
				return nil, fmt.Errorf("角色 %s: %w", role, ErrBudgetExhausted)
			}
		}

		return s.providerToRoleConfig(ctx, p), nil
	}

	// 3. 无角色分配 → 严格模式：返回错误
	return nil, fmt.Errorf("角色 %s: %w，请在 AI 设置中为该角色指定提供商", role, ErrRoleNotAssigned)
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
	if p.Provider == "bedrock" {
		cfg.Config.Region = s.bedrockRegion // 区域是部署级配置，不随 provider 存库
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

// RecordUsage 记录 AI 调用消耗（预算扣减 + Provider 统计更新）
func (s *aiServiceImpl) RecordUsage(ctx context.Context, role string, providerID int64, inputTokens, outputTokens int) {
	// 1. 扣减角色预算（日 + 月同时扣减）
	if s.budgetRepo != nil {
		if err := s.budgetRepo.IncrementUsage(ctx, role, inputTokens, outputTokens); err != nil {
			log.Warn("扣减角色预算失败", "role", role, "err", err)
		}
	}
	// 2. 累加 Provider 统计
	totalTokens := int64(inputTokens + outputTokens)
	if err := s.providerRepo.IncrementUsage(ctx, providerID, 1, totalTokens, 0); err != nil {
		log.Warn("更新 Provider 统计失败", "provider", providerID, "err", err)
	}
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
