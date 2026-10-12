// atlhyper_master_v2/gateway/handler/aiops/ai_error.go
// AI 调用失败 → 稳定错误码（前端按码显示可操作的 i18n 文案）
package aiops

import (
	"errors"
	"net/http"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
)

// 错误码（与前端 i18n aiops.ai.errors 的键一一对应）
const (
	aiErrRoleUnassigned  = "ai_role_unassigned"  // 去 AI 设置分配角色
	aiErrBudgetExhausted = "ai_budget_exhausted" // 去 AI 设置调整预算
	aiErrUpstream        = "ai_upstream"         // 模型调用失败，看 provider 状态 / 稍后重试
)

// writeAIError 写出 AI 失败响应
// 配置类错误 → 409 + 可操作的错误码；其余 → 502。⛔ 不把上游原始错误文本（RequestID 等）透传给页面，只记日志
func writeAIError(w http.ResponseWriter, op string, err error) {
	status, code, msg := http.StatusBadGateway, aiErrUpstream, "AI 调用失败"
	switch {
	case errors.Is(err, ai.ErrRoleNotAssigned):
		status, code, msg = http.StatusConflict, aiErrRoleUnassigned, "AI 角色未分配 Provider"
	case errors.Is(err, ai.ErrBudgetExhausted):
		status, code, msg = http.StatusConflict, aiErrBudgetExhausted, "AI 角色预算已用尽"
	default:
		aiLog.Warn("AI 调用失败", "op", op, "err", err)
	}
	handler.WriteJSON(w, status, map[string]string{"error": msg, "code": code})
}
