// logic/pusher/interface.go
// 推送器接口定义
package pusher

import "context"

// DataSource 数据源接口
type DataSource interface {
	// Name 数据源名称
	Name() string
	// Fetch 获取数据
	Fetch(ctx context.Context) (any, error)
}

// Pusher 推送器接口
type Pusher interface {
	// Name 推送器名称
	Name() string
	// Push 执行一次推送
	Push(ctx context.Context) error
	// Start 启动定时推送
	Start(ctx context.Context)
	// Stop 停止推送
	Stop()
}
