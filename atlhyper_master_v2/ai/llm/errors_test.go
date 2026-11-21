package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"testing"
)

type statusErr int

func (s statusErr) Error() string       { return "status" }
func (s statusErr) HTTPStatusCode() int { return int(s) }

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

// 按类型 / 状态码分类，⛔ 不看错误文字：文字里写着 401 也不算认证失败
func TestClassify(t *testing.T) {
	wrap := func(err error) error { return fmt.Errorf("调用模型: %w", err) }
	cases := []struct {
		name string
		err  error
		want ErrorKind
	}{
		{"401", wrap(&HTTPError{Provider: "openai", StatusCode: 401}), ErrKindAuth},
		{"403", &HTTPError{StatusCode: 403}, ErrKindForbidden},
		{"429", &HTTPError{StatusCode: 429}, ErrKindRateLimited},
		{"400", &HTTPError{StatusCode: 400}, ErrKindBadRequest},
		{"413", &HTTPError{StatusCode: 413}, ErrKindBadRequest},
		{"500", &HTTPError{StatusCode: 500}, ErrKindProviderError},
		{"503", &HTTPError{StatusCode: 503}, ErrKindProviderError},
		{"504", &HTTPError{StatusCode: 504}, ErrKindTimeout},
		{"SDK 自带的状态码接口", wrap(statusErr(429)), ErrKindRateLimited},
		{"适配层判定好的类别", wrap(&KindError{Kind: ErrKindRefused, Err: errors.New("x")}), ErrKindRefused},
		{"流没结束", wrap(ErrStreamIncomplete), ErrKindIncomplete},
		{"取消", wrap(context.Canceled), ErrKindCanceled},
		{"超时", wrap(context.DeadlineExceeded), ErrKindTimeout},
		{"网络超时", wrap(&net.OpError{Op: "read", Err: timeoutErr{}}), ErrKindTimeout},
		{"连接被拒", wrap(&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}), ErrKindUnreachable},
		{"地址解析失败", wrap(&net.DNSError{Err: "no such host", Name: "api.example.com"}), ErrKindUnreachable},
		{"中途断开", wrap(io.ErrUnexpectedEOF), ErrKindConnectionLost},
		{"连接被重置", wrap(&net.OpError{Op: "read", Err: syscall.ECONNRESET}), ErrKindConnectionLost},
		{"文字里有 401 也不算", errors.New("unexpected 401 in body"), ErrKindUnknown},
		{"nil", nil, ""},
	}
	for _, c := range cases {
		if got := Classify(c.err); got != c.want {
			t.Errorf("%s: Classify = %q, want %q", c.name, got, c.want)
		}
	}
}

// HTTPError 保留原始错误链（日志里看得到响应体摘要）
func TestHTTPError_Unwrap(t *testing.T) {
	cause := errors.New("body: invalid key")
	err := &HTTPError{Provider: "ollama", StatusCode: 401, Err: cause}
	if !errors.Is(err, cause) {
		t.Error("应能 Unwrap 到原始错误")
	}
}
