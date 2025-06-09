// =======================================================================================
// 📄 internal/bootstrap/manager.go
//
// ✨ Description:
//     Encapsulates the startup logic of controller-runtime's manager,
//     responsible for loading all Watchers and starting the control loop.
//     Acts as the core bootstrap module for cmd/neurocontroller/main.go,
//     decoupling the main function from registration logic.
//
// 📦 Provided Features:
//     - StartManager(): Starts the controller-runtime manager.
//
// 📍 Usage Scenario:
//     - Called by main.go as the unified entry point to launch controllers.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package bootstrap

import (
	"NeuroController/internal/utils"
	"NeuroController/internal/watcher"
	"context"
	"os"

	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ✅ Starts the controller manager (loads and runs all Watchers)
func StartManager() {
	// ✅ Create the controller-runtime manager
	cfg, err := resolveRestConfig()
	if err != nil {
		utils.Fatal(nil, "❌ Failed to load Kubernetes config", zap.Error(err))
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		// To support namespace filtering in the future. Currently watches the entire cluster.
		//Namespace: "default",
	})
	if err != nil {
		utils.Fatal(nil, "❌ Failed to initialize Controller Manager", zap.Error(err))
	}

	// ✅ Register all Watchers
	if err := watcher.RegisterAllWatchers(mgr); err != nil {
		utils.Fatal(nil, "❌ Failed to register Watcher modules", zap.Error(err))
	}

	// ✅ Start the controller loop (blocking call)
	utils.Info(nil, "🚀 Starting controller-runtime manager ...")
	if err := mgr.Start(context.Background()); err != nil {
		utils.Fatal(nil, "❌ Controller main loop exited with error", zap.Error(err))
	}
}

// ✅ Private helper: Automatically detects kubeconfig or in-cluster configuration
func resolveRestConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err == nil {
			utils.Info(context.TODO(), "✅ Using local kubeconfig")
			return cfg, nil
		}
		utils.Warn(context.TODO(), "⚠️ Failed to load local kubeconfig, trying in-cluster mode", zap.Error(err))
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		utils.Error(context.TODO(), "❌ Failed to load in-cluster configuration", zap.Error(err))
		return nil, err
	}

	utils.Info(context.TODO(), "✅ Using in-cluster configuration")
	return cfg, nil
}
