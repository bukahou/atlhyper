package command

// 指令失败的类别（Result.Code）。Master 按它决定状态码与界面提示，⛔ 不比较 Error 文字；
// Error 是执行方的原始错误，只进日志。
const (
	CodeNotConfigured = "not_configured" // 执行方缺少所需的数据源（如没有接入 ClickHouse）
	CodeInvalidParams = "invalid_params" // 参数格式或取值不合法
	CodeTimeout       = "timeout"        // 执行超时
	CodeUnavailable   = "unavailable"    // 数据源连不上（拒绝连接、域名解析失败）
	CodeRejected      = "rejected"       // 执行方守门拒绝（来源、集群或目标不被允许），未执行
	CodeFailed        = "failed"         // 其他失败
)

// IsKnownResultCode 是否为已定义的失败类别。旧版 Agent 不带类别，不认识的类别都按 CodeFailed 处理
func IsKnownResultCode(code string) bool {
	switch code {
	case CodeNotConfigured, CodeInvalidParams, CodeTimeout, CodeUnavailable, CodeRejected, CodeFailed:
		return true
	}
	return false
}
