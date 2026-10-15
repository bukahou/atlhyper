// atlhyper_master_v2/gateway/handler/aiops/ai_error.go
// AI 调用失败 → 稳定错误码（前端按码显示可操作的 i18n 文案）
package aiops

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
)

// writeAIError 写出 AI 失败响应（错误码由 ai.ErrorCode 统一分类，与深度分析状态共用）
// 配置类错误 → 409；超时 → 504；其余 → 502。⛔ 不把上游原始错误文本（RequestID 等）透传给页面，只记日志
func writeAIError(w http.ResponseWriter, op string, err error) {
	code := ai.ErrorCode(err)
	status, msg := http.StatusBadGateway, "AI 调用失败"
	switch code {
	case ai.CodeRoleUnassigned:
		status, msg = http.StatusConflict, "AI 角色未分配 Provider"
	case ai.CodeBudgetExhausted:
		status, msg = http.StatusConflict, "AI 角色预算已用尽"
	case ai.CodeTimeout:
		status, msg = http.StatusGatewayTimeout, "AI 调用超时"
		aiLog.Warn("AI 调用超时", "op", op, "err", err)
	default:
		aiLog.Warn("AI 调用失败", "op", op, "err", err)
	}
	handler.WriteJSON(w, status, map[string]string{"error": msg, "code": code})
}
