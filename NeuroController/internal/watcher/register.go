// =======================================================================================
// 📄 watcher/register.go
//
// ✨ 功能说明：
//     集中注册所有资源监听器（Pod、Node、Service、Deployment、Event）到 controller-runtime。
//     封装统一入口函数 RegisterAllWatchers，供 controller/main.go 调用使用。
//     实现结构化模块加载，避免 main 函数中直接引用各子模块，提升可维护性与扩展性。
//
// 🛠️ 提供功能：
//     - RegisterAllWatchers(ctrl.Manager): 统一注册所有 Watcher 控制器
//
// 📦 依赖：
//     - watcher/pod
//     - watcher/node
//     - watcher/service
//     - watcher/deployment
//     - watcher/event
//
// 📍 使用场景：
//     - 在 controller/main.go 启动时仅调用本文件的 RegisterAllWatchers 即可加载所有插件监听器
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package watcher

import (
	"NeuroController/internal/utils"
	"NeuroController/internal/watcher/deployment"
	"NeuroController/internal/watcher/event"
	"NeuroController/internal/watcher/node"
	"NeuroController/internal/watcher/pod"
	"NeuroController/internal/watcher/service"

	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ✅ 批量注册所有 Watcher
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

		utils.Info(ctx, "✅ 成功注册 Watcher",
			utils.WithTraceID(ctx),
			zap.String("watcher", w.Name),
		)
	}
	return nil
}

// =======================================================================================
// ✅ 所有 Watcher 注册表（集中管理、便于扩展）
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
	// 未来添加新的 Watcher，只需添加一行：
	// {"PVCWatcher", pvc.RegisterWatcher},
}
