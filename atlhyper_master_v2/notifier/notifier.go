// atlhyper_master_v2/notifier/notifier.go
// 通知模块入口 - 接口定义
package notifier

import "context"

// AlertManager 告警管理器接口
// 外部调用者应依赖此接口，而非具体实现
//
// 创建实例使用 manager.NewAlertManager()
type AlertManager interface {
	// Start 启动告警管理器
	Start()

	// Stop 停止告警管理器
	Stop()

	// Send 发送告警（经过去重、聚合、限流）
	Send(ctx context.Context, alert *Alert) error

	// Test 测试指定渠道
	Test(ctx context.Context, channelType string) error
}
