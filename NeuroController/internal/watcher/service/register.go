// =======================================================================================
// 📄 watcher/service/register.go
//
// ✨ Description:
//     Registers the ServiceWatcher into the controller-runtime Manager, enabling
//     automatic monitoring of all Service object changes in the cluster.
//     Encapsulates the creation (NewServiceWatcher) and registration (SetupWithManager)
//     of the watcher to decouple the controller/main.go from internal logic.
//
// 🛠️ Features:
//     - NewServiceWatcher(client.Client): Factory function to instantiate a watcher
//     - RegisterWatcher(mgr ctrl.Manager): Registers the watcher to controller-runtime
//
// 📦 Dependencies:
//     - controller-runtime (Manager, controller construction)
//     - service_watcher.go (watch logic implementation)
//     - utils/k8s_client.go (provides global shared client instance)
//
// 📍 Usage:
//     - Called from controller/main.go to initialize the service watcher component
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package service

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 注册器：将 ServiceWatcher 注册到 controller-runtime
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	serviceWatcher := NewServiceWatcher(client)

	if err := serviceWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ 注册 ServiceWatcher 失败",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/service"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ ServiceWatcher 注册成功",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/service"),
	)
	return nil
}

// ✅ 工厂函数：使用注入的 client 创建 ServiceWatcher 实例
func NewServiceWatcher(c client.Client) *ServiceWatcher {
	return &ServiceWatcher{client: c}
}
