package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
)

// ErrorKind 调用模型失败的类别：前端按它出文案、提供商状态按它记录。
// 程序按类别分支，⛔ 不比对错误文字。
type ErrorKind string

const (
	ErrKindAuth           ErrorKind = "auth"            // 密钥无效（401）
	ErrKindForbidden      ErrorKind = "forbidden"       // 无权使用该模型（403）
	ErrKindRateLimited    ErrorKind = "rate_limited"    // 限流 / 配额用尽（429）
	ErrKindBadRequest     ErrorKind = "bad_request"     // 请求被拒（其余 4xx：上下文过长、参数不对、模型不存在）
	ErrKindProviderError  ErrorKind = "provider_error"  // 提供商自身出错（5xx）
	ErrKindTimeout        ErrorKind = "timeout"         // 超时（含 504）
	ErrKindUnreachable    ErrorKind = "unreachable"     // 连不上：连接被拒、地址解析失败
	ErrKindConnectionLost ErrorKind = "connection_lost" // 连接中途断开
	ErrKindIncomplete     ErrorKind = "incomplete"      // 模型的流在完成前结束
	ErrKindRefused        ErrorKind = "refused"         // 模型拒绝回答（安全审核）
	ErrKindCanceled       ErrorKind = "canceled"        // 调用方取消
	ErrKindUnknown        ErrorKind = "unknown"
)

// HTTPError 提供商返回了非 2xx。手写 HTTP 的适配层直接返回它；SDK 的错误类型由各自适配层转成它或 KindError。
type HTTPError struct {
	Provider   string
	StatusCode int
	Err        error // 原始错误（含响应体摘要），只进日志
}

func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s API error: %d: %v", e.Provider, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("%s API error: %d", e.Provider, e.StatusCode)
}

func (e *HTTPError) Unwrap() error { return e.Err }

// HTTPStatusCode 状态码（与 AWS SDK 的响应错误同一个方法名，Classify 一并识别）
func (e *HTTPError) HTTPStatusCode() int { return e.StatusCode }

// KindError 适配层已经判定好类别的错误（厂商特有的错误类型只在各自适配层识别）
type KindError struct {
	Kind ErrorKind
	Err  error
}

func (e *KindError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Kind)
}

func (e *KindError) Unwrap() error { return e.Err }

// LLMErrorKind 类别
func (e *KindError) LLMErrorKind() ErrorKind { return e.Kind }

// Classify 按错误类型与状态码判定类别；认不出的为 unknown，nil 为空
func Classify(err error) ErrorKind {
	if err == nil {
		return ""
	}
	var kinded interface{ LLMErrorKind() ErrorKind }
	if errors.As(err, &kinded) {
		return kinded.LLMErrorKind()
	}
	var withStatus interface{ HTTPStatusCode() int }
	if errors.As(err, &withStatus) {
		if kind := kindOfStatus(withStatus.HTTPStatusCode()); kind != "" {
			return kind
		}
	}
	switch {
	case errors.Is(err, ErrStreamIncomplete):
		return ErrKindIncomplete
	case errors.Is(err, context.Canceled):
		return ErrKindCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return ErrKindTimeout
	case errors.Is(err, syscall.ECONNREFUSED):
		return ErrKindUnreachable
	case errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, io.EOF), errors.Is(err, syscall.ECONNRESET), errors.Is(err, syscall.EPIPE):
		return ErrKindConnectionLost
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrKindUnreachable
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrKindTimeout
	}
	return ErrKindUnknown
}

func kindOfStatus(code int) ErrorKind {
	switch {
	case code == http.StatusUnauthorized:
		return ErrKindAuth
	case code == http.StatusForbidden:
		return ErrKindForbidden
	case code == http.StatusTooManyRequests:
		return ErrKindRateLimited
	case code == http.StatusRequestTimeout || code == http.StatusGatewayTimeout:
		return ErrKindTimeout
	case code >= 400 && code < 500:
		return ErrKindBadRequest
	case code >= 500:
		return ErrKindProviderError
	}
	return ""
}
