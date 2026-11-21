package gemini

import (
	"net/url"
	"strings"
)

// redactedError 错误文字里的 API Key 已抹掉；Unwrap 保留原错误链，只供 errors.Is / As 分类。
// ⛔ 不要打印 Unwrap 出来的原错误：里面仍带着密钥
type redactedError struct {
	msg string
	err error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.err }

// redactAPIKey 抹掉错误文字里的 API Key。SDK 的 REST 传输把它放在请求网址的 key 参数里，
// 传输层错误（连不上、超时、拿到响应前被取消）会带出完整网址，往上交、写日志之前都要先抹掉。
// 按已知的密钥值替换（含网址转义后的写法），不靠解析网址格式
func redactAPIKey(err error, apiKey string) error {
	if err == nil || apiKey == "" {
		return err
	}
	msg := err.Error()
	redacted := strings.ReplaceAll(msg, apiKey, "<redacted>")
	if escaped := url.QueryEscape(apiKey); escaped != apiKey {
		redacted = strings.ReplaceAll(redacted, escaped, "<redacted>")
	}
	if redacted == msg {
		return err
	}
	return &redactedError{msg: redacted, err: err}
}
