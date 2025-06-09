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

// ✅ Registrar: Registers ServiceWatcher into controller-runtime
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	serviceWatcher := NewServiceWatcher(client)

	if err := serviceWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ Failed to register ServiceWatcher",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/service"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ Successfully registered ServiceWatcher",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/service"),
	)
	return nil
}

// ✅ Factory: Creates a new ServiceWatcher instance with injected client
func NewServiceWatcher(c client.Client) *ServiceWatcher {
	return &ServiceWatcher{client: c}
}
