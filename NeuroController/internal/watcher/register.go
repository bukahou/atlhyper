// =======================================================================================
// 📄 watcher/register.go
//
// ✨ Description:
//     Centralized registration of all resource watchers (Pod, Node, Service, Deployment, Event).
//     Provides a unified entry point RegisterAllWatchers for controller/main.go.
//     Enhances modularity, maintainability, and scalability by decoupling watcher imports.
//
// 🛠️ Features:
//     - RegisterAllWatchers(ctrl.Manager): Register all watcher controllers in a single call
//
// 📦 Dependencies:
//     - watcher/pod
//     - watcher/node
//     - watcher/service
//     - watcher/deployment
//     - watcher/event
//
// 📍 Usage:
//     - Simply call RegisterAllWatchers() from controller/main.go to register all watchers
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package watcher

import (
	"NeuroController/internal/utils"
	"NeuroController/internal/watcher/deployment"
	"NeuroController/internal/watcher/endpoint"
	"NeuroController/internal/watcher/event"
	"NeuroController/internal/watcher/node"
	"NeuroController/internal/watcher/pod"
	"NeuroController/internal/watcher/service"

	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ✅ Register all watchers to controller-runtime manager
//
// Iterates over the WatcherRegistry and invokes each module’s registration logic.
// If any watcher fails to register, the process will be aborted and an error returned.
func RegisterAllWatchers(mgr ctrl.Manager) error {
	ctx := context.TODO()

	for _, w := range WatcherRegistry {
		if err := w.Action(mgr); err != nil {
			utils.Error(ctx, "❌ Failed to register watcher",
				utils.WithTraceID(ctx),
				zap.String("watcher", w.Name),
				zap.Error(err),
			)
			return err
		}

		utils.Info(ctx, "✅ Successfully registered watcher",
			utils.WithTraceID(ctx),
			zap.String("watcher", w.Name),
		)
	}
	return nil
}

// =======================================================================================
// ✅ Watcher registry list (centralized and extendable)
//
// Simply add new watchers to this list for auto-registration.
// =======================================================================================
var WatcherRegistry = []struct {
	Name   string
	Action func(ctrl.Manager) error
}{
	{"PodWatcher", pod.RegisterWatcher},
	{"NodeWatcher", node.RegisterWatcher},
	{"ServiceWatcher", service.RegisterWatcher},
	{"DeploymentWatcher", deployment.RegisterWatcher},
	{"EventWatcher", event.RegisterWatcher},
	{"EndpointWatcher", endpoint.RegisterWatcher},
	// Future watchers can be added here:
	// {"PVCWatcher", pvc.RegisterWatcher},
}
