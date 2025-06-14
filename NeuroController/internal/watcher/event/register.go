// =======================================================================================
// 📄 watcher/event/register.go
//
// ✨ Description:
//     Registers the EventWatcher with the controller-runtime manager to observe
//     all Event resources in the cluster. Encapsulates the watcher instance construction
//     and controller binding logic to decouple controller/main.go from watcher details.
//
// 🛠️ Features:
//     - NewEventWatcher(client.Client): Creates a watcher instance with injected client
//     - RegisterWatcher(mgr ctrl.Manager): Registers the watcher with controller-runtime
//
// 📦 Dependencies:
//     - controller-runtime
//     - event_watcher.go (contains reconciliation logic)
//     - utils/k8s_client.go (shared Kubernetes client utilities)
//
// 📍 Usage:
//     - Called in controller/main.go during watcher registration phase
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package event

import (
	"context"

	"NeuroController/internal/utils"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 将 EventWatcher 注册到 controller-runtime 的管理器中
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

	utils.Info(context.TODO(), "✅ EventWatcher 注册成功",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/event"),
	)
	return nil
}

// ✅ 工厂方法：使用注入的 client 创建新的 EventWatcher 实例
func NewEventWatcher(c client.Client) *EventWatcher {
	return &EventWatcher{client: c}
}
