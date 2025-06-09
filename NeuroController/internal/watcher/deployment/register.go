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
	"context"

	"NeuroController/internal/utils"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ Factory method: create a DeploymentWatcher instance with the shared client
func NewDeploymentWatcher(c client.Client) *DeploymentWatcher {
	return &DeploymentWatcher{client: c}
}

// ✅ Registrar: bind DeploymentWatcher to controller-runtime manager
//
// Retrieves the shared client → builds the watcher instance → registers to the manager.
// Logs error and returns if registration fails.
func RegisterWatcher(mgr ctrl.Manager) error {
	// Get global shared client from utils
	client := utils.GetClient()

	// Create watcher instance
	deploymentWatcher := NewDeploymentWatcher(client)

	// Register controller
	if err := deploymentWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ Failed to register DeploymentWatcher",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/deployment"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ DeploymentWatcher registered successfully",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/deployment"),
	)

	return nil
}
