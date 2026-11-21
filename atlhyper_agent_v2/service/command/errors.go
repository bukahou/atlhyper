package command

import (
	"context"
	"errors"
	"net"
	"syscall"

	"AtlHyper/model_v3/command"
)

var (
	// errClickHouseNotConfigured 没有配置 ClickHouse 时，查询类指令无法执行
	errClickHouseNotConfigured = errors.New("ClickHouse not configured")
	// errInvalidParams 指令参数格式或取值不合法
	errInvalidParams = errors.New("invalid params")
)

// resultCode 把执行错误归到 Result.Code 的类别：按错误的类型与链判断，⛔ 不看错误文字
func resultCode(err error) string {
	var dnsErr *net.DNSError
	var netErr net.Error
	switch {
	case errors.Is(err, errClickHouseNotConfigured):
		return command.CodeNotConfigured
	case errors.Is(err, errInvalidParams):
		return command.CodeInvalidParams
	case errors.Is(err, context.DeadlineExceeded):
		return command.CodeTimeout
	case errors.As(err, &dnsErr):
		if dnsErr.IsTimeout {
			return command.CodeTimeout
		}
		return command.CodeUnavailable
	case errors.Is(err, syscall.ECONNREFUSED):
		return command.CodeUnavailable
	case errors.As(err, &netErr) && netErr.Timeout():
		return command.CodeTimeout
	}
	return command.CodeFailed
}
