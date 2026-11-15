// atlhyper_master_v2/notifier/interfaces.go
// 通知模块对外的入口
package notifier

import (
	"context"
	"errors"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
)

// AlertManager 通知模块对外的唯一入口：装配并运行「告警来源 → 管道 → 发送」。
//
// 外部只需要它的四件事：启停、把事件变化交进来、给某个接收人发一条测试消息。
// 路由、分组、静默、重试等都在模块内部，调用方不感知
type AlertManager interface {
	// Start 从库里恢复通知状态并启动后台协程
	Start() error
	// Stop 停止全部后台协程并等它们退出
	Stop()
	// OnIncidentTransition AIOps 引擎的订阅回调。只入队、立即返回；队列满时丢弃并计数
	OnIncidentTransition(transition aiops.IncidentTransition)
	// TestReceiver 给指定接收人发一条测试消息（不管它是否启用）。
	// 接收人不存在 → ErrReceiverNotFound；配置不完整 → ErrReceiverConfigIncomplete；
	// 其余错误是发送失败的原因，不含 Webhook 地址与密码，可以直接展示
	TestReceiver(ctx context.Context, receiverID int64) error
}

var (
	// ErrReceiverNotFound 接收人不存在
	ErrReceiverNotFound = errors.New("接收人不存在")
	// ErrReceiverConfigIncomplete 接收人的配置缺少必填项（含配置解密失败的情况）
	ErrReceiverConfigIncomplete = channel.ErrReceiverConfigIncomplete
	// ErrUnsupportedReceiverType 没有对应发送器的接收人类型
	ErrUnsupportedReceiverType = channel.ErrUnsupportedReceiverType
)
