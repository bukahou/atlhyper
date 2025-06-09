// =======================================================================================
// 📄 watcher/pod/register.go
//
// ✨ Description:
//     Registers the PodWatcher into the controller-runtime Manager to automatically
//     monitor all changes in Pod status across the cluster.
//     Encapsulates both the creation of the PodWatcher instance (NewPodWatcher)
//     and its registration with the Manager (SetupWithManager).
//     Decouples controller/main.go from the watcher internals.
//
// 🛠️ Features:
//     - NewPodWatcher(client.Client): Factory function to instantiate a PodWatcher
//     - RegisterWatcher(mgr ctrl.Manager): Register the watcher into controller-runtime
//
// 📦 Dependencies:
//     - controller-runtime (Manager, controller binding)
//     - pod_watcher.go (core watcher logic)
//     - utils/k8s_client.go (global shared client instance)
//
// 📍 Usage:
//     - Called from controller/main.go to initialize the pod watcher component
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package pod

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ Registrar: Registers PodWatcher into controller-runtime
//
// Retrieves the global Kubernetes client → builds the watcher instance →
// registers it into the controller-runtime Manager.
// Logs error if registration fails.
func RegisterWatcher(mgr ctrl.Manager) error {
	// Retrieve shared Kubernetes client (from utils wrapper)
	client := utils.GetClient()

	// Instantiate watcher with client injection
	podWatcher := NewPodWatcher(client)

	// Register watcher to the manager
	if err := podWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ Failed to register PodWatcher",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/pod"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ Successfully registered PodWatcher",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/pod"),
	)

	return nil
}

// ✅ Factory: Create a new PodWatcher instance with injected client
func NewPodWatcher(c client.Client) *PodWatcher {
	return &PodWatcher{client: c}
}
