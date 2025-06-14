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

// ✅ 注册所有 Watcher 到 controller-runtime 的管理器中
//
// 遍历 WatcherRegistry 并调用每个模块的注册方法。
// 如果任意模块注册失败，则终止流程并返回错误。
func RegisterAllWatchers(mgr ctrl.Manager) error {
	ctx := context.TODO()

	for _, w := range WatcherRegistry {
		if err := w.Action(mgr); err != nil {
			utils.Error(ctx, "❌ 注册 Watcher 失败",
				utils.WithTraceID(ctx),
				zap.String("watcher", w.Name),
				zap.Error(err),
			)
			return err
		}

		utils.Info(ctx, "✅ Watcher 注册成功",
			utils.WithTraceID(ctx),
			zap.String("watcher", w.Name),
		)
	}
	return nil
}

// =======================================================================================
// ✅ Watcher 注册表（集中管理、支持扩展）
//
// 只需将新的 Watcher 模块添加到该列表中，即可实现自动注册。
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
	// 未来可扩展更多模块，例如：
	// {"PVCWatcher", pvc.RegisterWatcher},
}
