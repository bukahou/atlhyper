// atlhyper_master_v2/notifier/errors.go
// 错误定义
package notifier

import "errors"

var (
	// ErrChannelNotFound 渠道不存在
	ErrChannelNotFound = errors.New("notification channel not found")

	// ErrChannelDisabled 渠道已禁用
	ErrChannelDisabled = errors.New("notification channel is disabled")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid channel configuration")

	// ErrUnsupportedChannel 不支持的渠道类型
	ErrUnsupportedChannel = errors.New("unsupported channel type")
)
