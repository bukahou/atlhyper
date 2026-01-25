// atlhyper_master_v2/notifier/channel/channel.go
// 通知渠道接口定义
package channel

import (
	"context"

	"AtlHyper/atlhyper_master_v2/notifier"
)

// Channel 通知渠道接口
// 所有通知渠道（Slack、Email、DingTalk 等）必须实现此接口
type Channel interface {
	// Send 发送通知
	Send(ctx context.Context, msg *notifier.Message) error

	// Type 返回渠道类型标识
	Type() string
}
