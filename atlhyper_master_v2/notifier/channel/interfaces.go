// atlhyper_master_v2/notifier/channel/interfaces.go
// 通知渠道：消息与发送器接口。发送器按接收人创建，见 receiver.go
package channel

import "context"

// Message 渲染后的消息
type Message struct {
	Subject string // 标题（邮件主题 / Slack 的纯文本摘要）
	Body    string // 正文
	Format  string // 格式：text / markdown
}

// Notifier 发送器：把一条消息发到一个具体的地址
type Notifier interface {
	Name() string
	Send(ctx context.Context, msg *Message) error
}
