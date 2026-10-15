// atlhyper_master_v2/ai/errors.go
// AI 模块错误定义（上层用 errors.Is 识别，而不是匹配文案）
package ai

import (
	"context"
	"errors"
)

var (
	// ErrRoleNotAssigned 角色未分配 Provider（用户可在 AI 设置中修复）
	ErrRoleNotAssigned = errors.New("角色未分配 Provider")
	// ErrBudgetExhausted 角色预算已用尽且无可用的 fallback Provider
	ErrBudgetExhausted = errors.New("角色预算已用尽")
)

// 错误码（网关响应、深度分析状态共用；前端 i18n 按码给出可操作提示）
const (
	CodeRoleUnassigned  = "ai_role_unassigned"  // 去 AI 设置分配角色
	CodeBudgetExhausted = "ai_budget_exhausted" // 去 AI 设置调整预算
	CodeTimeout         = "ai_timeout"          // 超过分析时限
	CodeUpstream        = "ai_upstream"         // 模型调用失败 / 返回不可用
)

// ErrorCode 把错误归为稳定的错误码；nil → ""
func ErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrRoleNotAssigned):
		return CodeRoleUnassigned
	case errors.Is(err, ErrBudgetExhausted):
		return CodeBudgetExhausted
	case errors.Is(err, context.DeadlineExceeded):
		return CodeTimeout
	default:
		return CodeUpstream
	}
}
