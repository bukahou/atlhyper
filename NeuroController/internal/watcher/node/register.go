// =======================================================================================
// 📄 watcher/node/register.go
//
// ✨ Description:
//     Registers the NodeWatcher to the controller-runtime Manager to enable automatic
//     monitoring of all Node status changes in the cluster.
//     This file encapsulates the watcher instance construction (NewNodeWatcher)
//     and controller binding (SetupWithManager) to decouple logic from controller/main.go.
//
// 🛠️ Features:
//     - NewNodeWatcher(client.Client): Instantiates a NodeWatcher with injected client
//     - RegisterWatcher(mgr ctrl.Manager): Registers the watcher to the controller-runtime Manager
//
// 📦 Dependencies:
//     - controller-runtime (Manager and controller builder)
//     - node_watcher.go (watch logic implementation)
//     - utils/k8s_client.go (shared Kubernetes client provider)
//
// 📍 Usage:
//     - Called from controller/main.go to load and register node watchers
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package node

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ Registrar: Register NodeWatcher with controller-runtime
//
// Step-by-step:
// 1. Retrieve shared Kubernetes client from utils
// 2. Create the watcher instance
// 3. Register it to the controller-runtime manager
// Logs error if registration fails.
func RegisterWatcher(mgr ctrl.Manager) error {
	// Retrieve shared Kubernetes client
	client := utils.GetClient()

	// Construct watcher instance
	nodeWatcher := NewNodeWatcher(client)

	// Register to controller-runtime manager
	if err := nodeWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ Failed to register NodeWatcher",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/node"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ NodeWatcher registered successfully",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/node"),
	)

	return nil
}

// ✅ Factory method: Construct a NodeWatcher with the injected client
func NewNodeWatcher(c client.Client) *NodeWatcher {
	return &NodeWatcher{client: c}
}
