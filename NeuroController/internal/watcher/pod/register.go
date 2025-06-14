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

// ✅ 注册器：将 PodWatcher 注册到 controller-runtime
//
// 获取全局 Kubernetes 客户端 → 构造 watcher 实例 →
// 注册到 controller-runtime 的管理器中。
// 若注册失败，则记录错误日志。
func RegisterWatcher(mgr ctrl.Manager) error {
	// 获取共享的 Kubernetes 客户端（通过 utils 封装）
	client := utils.GetClient()

	// 注入客户端并构造 PodWatcher 实例
	podWatcher := NewPodWatcher(client)

	// 注册到管理器
	if err := podWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ 注册 PodWatcher 失败",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/pod"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ PodWatcher 注册成功",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/pod"),
	)

	return nil
}

// ✅ 工厂函数：使用注入的 client 创建 PodWatcher 实例
func NewPodWatcher(c client.Client) *PodWatcher {
	return &PodWatcher{client: c}
}
