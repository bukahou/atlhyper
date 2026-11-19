// atlhyper_master_v2/ai/errors.go
// AI 模块错误定义（上层用 errors.Is 识别，而不是匹配文案）
package ai

import (
	"errors"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

var (
	// ErrRoleNotAssigned 角色未分配 Provider（用户可在 AI 设置中修复）
	ErrRoleNotAssigned = errors.New("角色未分配 Provider")
	// ErrBudgetExhausted 角色预算已用尽且无可用的 fallback Provider
	ErrBudgetExhausted = errors.New("角色预算已用尽")
	// ErrConversationNotFound 会话不存在或不属于当前用户（两者不区分，避免借此探测别人的会话）
	ErrConversationNotFound = errors.New("对话不存在")
	// ErrEmptyMessage 既没有用户消息也没有要分析的告警
	ErrEmptyMessage = errors.New("消息为空")
)

// 错误码（网关响应、深度分析状态、对话流的错误分段共用；前端 i18n 按码给出可操作提示）
const (
	CodeRoleUnassigned  = "ai_role_unassigned"  // 去 AI 设置分配角色
	CodeBudgetExhausted = "ai_budget_exhausted" // 去 AI 设置调整预算
	CodeConfig          = "ai_config"           // 其他配置问题（客户端建不起来：提供商类型、地址、密钥格式）
	CodeAuth            = "ai_auth"             // 提供商拒绝了密钥或无权使用该模型（401 / 403）
	CodeRateLimited     = "ai_rate_limited"     // 提供商限流 / 配额用尽（429）
	CodeTimeout         = "ai_timeout"          // 超时
	CodeUnreachable     = "ai_unreachable"      // 连不上提供商（连接被拒、地址解析失败）
	CodeInterrupted     = "ai_interrupted"      // 回答在完成前中断（流没结束、连接中途断开）
	CodeRefused         = "ai_refused"          // 模型拒绝回答（安全审核）
	CodeUpstream        = "ai_upstream"         // 其余的模型调用失败
)

// ErrorCode 把错误归为稳定的错误码；nil → ""。
// 模型调用失败按 llm.Classify 的类别细分（只看错误类型与状态码，⛔ 不看文字）
func ErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrRoleNotAssigned):
		return CodeRoleUnassigned
	case errors.Is(err, ErrBudgetExhausted):
		return CodeBudgetExhausted
	}
	switch llm.Classify(err) {
	case llm.ErrKindAuth, llm.ErrKindForbidden:
		return CodeAuth
	case llm.ErrKindRateLimited:
		return CodeRateLimited
	case llm.ErrKindTimeout:
		return CodeTimeout
	case llm.ErrKindUnreachable:
		return CodeUnreachable
	case llm.ErrKindIncomplete, llm.ErrKindConnectionLost:
		return CodeInterrupted
	case llm.ErrKindRefused:
		return CodeRefused
	default:
		return CodeUpstream
	}
}

// configErrorCode 取 AI 配置 / 建客户端阶段的错误码：角色与预算问题照常细分，其余算配置问题
func configErrorCode(err error) string {
	switch code := ErrorCode(err); code {
	case CodeRoleUnassigned, CodeBudgetExhausted:
		return code
	default:
		return CodeConfig
	}
}

// errorSummary 错误码对应的简短中文说明：对话错误分段的兜底文案、AI 设置页的提供商状态。
// 原始错误（可能含内部地址、响应体、请求 ID）只进日志
func errorSummary(code string) string {
	switch code {
	case CodeRoleUnassigned:
		return "AI 角色未分配提供商"
	case CodeBudgetExhausted:
		return "AI 角色预算已用尽"
	case CodeConfig:
		return "AI 配置有误（提供商类型、地址或密钥格式）"
	case CodeAuth:
		return "认证失败，请检查 API Key 与模型权限"
	case CodeRateLimited:
		return "请求频率或配额超限"
	case CodeTimeout:
		return "请求超时"
	case CodeUnreachable:
		return "连不上模型服务"
	case CodeInterrupted:
		return "回答在完成前中断了"
	case CodeRefused:
		return "模型拒绝回答这个请求"
	default:
		return "调用模型失败"
	}
}
