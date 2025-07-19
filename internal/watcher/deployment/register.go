// =======================================================================================
// 📄 watcher/deployment/register.go
//
// ✨ Description:
//     Registers the DeploymentWatcher with the controller-runtime manager,
//     enabling automatic observation of all Deployment status changes.
//     This module encapsulates the watcher instantiation (NewDeploymentWatcher)
//     and registration (SetupWithManager), decoupling it from controller/main.go.
//
// 🛠️ Features:
//     - NewDeploymentWatcher(client.Client): Constructs a new watcher instance
//     - RegisterWatcher(mgr ctrl.Manager): Registers the watcher to the controller manager
//
// 📦 Dependencies:
//     - controller-runtime (Manager, controller registration)
//     - deployment_watcher.go (watch logic)
//     - utils/k8s_client.go (shared client access)
//
// 📍 Usage:
//     - Called from controller/main.go to load deployment watcher
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package deployment

import (
	"NeuroController/internal/utils"
	"log"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 工厂方法：使用共享客户端创建 DeploymentWatcher 实例
func NewDeploymentWatcher(c client.Client) *DeploymentWatcher {
	return &DeploymentWatcher{client: c}
}

func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	deploymentWatcher := NewDeploymentWatcher(client)

	if err := deploymentWatcher.SetupWithManager(mgr); err != nil {
		log.Printf("❌ 注册 DeploymentWatcher 失败: %v", err)
		return err
	}

	return nil
}
