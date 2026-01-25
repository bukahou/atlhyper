// atlhyper_master_v2/notifier/interface.go
// 告警管理器接口
package notifier

import (
	"context"
	"errors"

	"AtlHyper/atlhyper_master_v2/notifier/template"
)

// AlertManager 告警管理器接口
type AlertManager interface {
	// SendWithTemplate 使用模板发送告警
	// templateName: heartbeat_offline, heartbeat_recovery, k8s_event
	SendWithTemplate(templateName string, data *template.AlertData) error

	// Test 测试指定渠道
	Test(ctx context.Context, channelType string) error

	// Start 启动
	Start() error

	// Stop 停止
	Stop()
}

// 错误定义
var (
	ErrChannelNotFound    = errors.New("channel not found")
	ErrChannelDisabled    = errors.New("channel is disabled")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrUnsupportedChannel = errors.New("unsupported channel type")
)
