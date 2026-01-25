// atlhyper_master_v2/tester/notifier.go
// 通知渠道测试器
package tester

import (
	"context"
	"errors"

	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/manager"
)

// NotifierTester 通知渠道测试器
type NotifierTester struct {
	alertManager *manager.AlertManager
}

// NewNotifierTester 创建通知渠道测试器
func NewNotifierTester(alertMgr *manager.AlertManager) *NotifierTester {
	return &NotifierTester{
		alertManager: alertMgr,
	}
}

// Name 返回测试器名称
func (t *NotifierTester) Name() string {
	return "notifier"
}

// Test 测试通知渠道
// target 是渠道类型：slack, email 等
func (t *NotifierTester) Test(ctx context.Context, target string) Result {
	if target == "" {
		return NewFailureResult("channel type required")
	}

	err := t.alertManager.Test(ctx, target)
	if err != nil {
		// 处理特定错误
		if errors.Is(err, notifier.ErrChannelNotFound) {
			return NewFailureResult("channel not found")
		}
		if errors.Is(err, notifier.ErrChannelDisabled) {
			return NewFailureResult("channel is disabled")
		}
		if errors.Is(err, notifier.ErrInvalidConfig) {
			return NewFailureResult("invalid channel configuration")
		}
		if errors.Is(err, notifier.ErrUnsupportedChannel) {
			return NewFailureResult("unsupported channel type")
		}
		return NewFailureResultWithDetails("failed to send test notification", map[string]any{
			"error": err.Error(),
		})
	}

	return NewSuccessResultWithDetails("test notification sent successfully", map[string]any{
		"channel": target,
	})
}
