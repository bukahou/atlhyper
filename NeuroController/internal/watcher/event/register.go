// =======================================================================================
// 📄 watcher/event/register.go
//
// ✨ 功能说明：
//     注册 EventWatcher 到 controller-runtime 管理器中，实现监听集群中所有 Event 事件。
//     封装监听器实例构造（NewEventWatcher）与 controller 绑定逻辑，
//     解耦 controller/main.go 与具体监听逻辑。
//
// 🛠️ 提供功能：
//     - NewEventWatcher(client.Client): 创建监听器实例（注入共享 client）
//     - RegisterWatcher(mgr ctrl.Manager): 注册到 controller-runtime 管理器
//
// 📦 依赖：
//     - controller-runtime
//     - event_watcher.go（监听逻辑）
//     - utils/k8s_client.go（共享 client 工具）
//
// 📍 使用场景：
//     - 在 controller/main.go 中统一注册
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package event

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 注册器：注册 EventWatcher 到 controller-runtime
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	eventWatcher := NewEventWatcher(client)

	if err := eventWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(context.TODO(), "❌ 注册 EventWatcher 失败",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/event"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(context.TODO(), "✅ 成功注册 EventWatcher",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/event"),
	)
	return nil
}

// ✅ 工厂方法：构造 EventWatcher 实例
func NewEventWatcher(c client.Client) *EventWatcher {
	return &EventWatcher{client: c}
}
