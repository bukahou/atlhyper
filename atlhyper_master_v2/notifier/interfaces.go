// atlhyper_master_v2/notifier/interfaces.go
// 通知发送接口定义
package notifier

import "context"

// Notifier 通知发送接口
type Notifier interface {
	// Send 发送通知
	Send(ctx context.Context, msg *Message) error

	// Type 返回通知类型
	Type() string
}

// Message 通知消息
type Message struct {
	Title    string            // 标题
	Content  string            // 内容
	Severity string            // 严重程度: info / warning / critical
	Fields   map[string]string // 额外字段
}

// Result 发送结果
type Result struct {
	Success bool
	Error   string
}
