// =======================================================================================
// 📄 watcher/endpoint/register.go
//
// ✨ Description:
//     Registers the EndpointWatcher to the controller-runtime manager,
//     enabling automatic monitoring of all Endpoints status changes in the cluster.
//     Encapsulates the construction of the watcher instance (NewEndpointWatcher)
//     and the binding logic (SetupWithManager) to decouple implementation
//     from controller/main.go.
//
// 🛠️ Features:
//     - NewEndpointWatcher(client.Client): Creates a new watcher instance
//     - RegisterWatcher(mgr ctrl.Manager): Registers the watcher with the controller manager
//
// 📍 Usage:
//     - Called from controller/main.go to activate the Endpoints monitoring logic
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package endpoint

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 将 EndpointWatcher 注册到 controller-runtime 管理器中
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	watcher := NewEndpointWatcher(client)

	if err := watcher.SetupWithManager(mgr); err != nil {
		utils.Error(context.TODO(), "❌ 注册 EndpointWatcher 失败",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/endpoint"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(context.TODO(), "✅ EndpointWatcher 注册成功",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/endpoint"),
	)
	return nil
}

// ✅ 构造一个新的 EndpointWatcher 实例
func NewEndpointWatcher(c client.Client) *EndpointWatcher {
	return &EndpointWatcher{client: c}
}
