// atlhyper_master_v2/ai/errors.go
// AI 模块错误定义（上层用 errors.Is 识别，而不是匹配文案）
package ai

import "errors"

var (
	// ErrRoleNotAssigned 角色未分配 Provider（用户可在 AI 设置中修复）
	ErrRoleNotAssigned = errors.New("角色未分配 Provider")
	// ErrBudgetExhausted 角色预算已用尽且无可用的 fallback Provider
	ErrBudgetExhausted = errors.New("角色预算已用尽")
)
