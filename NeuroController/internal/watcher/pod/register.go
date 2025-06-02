// =======================================================================================
// 📄 watcher/pod/register.go
//
// ✨ 功能说明：
//     注册 PodWatcher 到 controller-runtime 管理器中，实现自动监听所有 Pod 的状态变更。
//     封装监听器实例构造（NewPodWatcher）与 controller 绑定（SetupWithManager）逻辑，
//     解耦 controller/main.go 与 watcher 具体实现细节。
//
// 🛠️ 提供功能：
//     - NewPodWatcher(client.Client): 创建监听器实例（注入共享 client）
//     - RegisterWatcher(mgr ctrl.Manager): 注册监听器到 controller-runtime 管理器
//
// 📦 依赖：
//     - controller-runtime（Manager、控制器构造）
//     - pod_watcher.go（监听逻辑定义）
//     - utils/k8s_client.go（获取全局共享 client 实例）
//
// 📍 使用场景：
//     - 在 controller/main.go 中统一加载 watcher/pod 的注册器
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package pod

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 注册器：注册 PodWatcher 到 controller-runtime
//
// 获取共享 K8s client → 构造监听器实例 → 注册到 controller-runtime 管理器。
// 若注册失败，将记录日志并返回错误。
func RegisterWatcher(mgr ctrl.Manager) error {
	// 获取共享 K8s client（从 utils 封装中注入）
	client := utils.GetClient()

	// 创建监听器实例（封装监听逻辑）
	podWatcher := NewPodWatcher(client)

	// 注册到 controller-runtime 管理器
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
		"✅ 成功注册 PodWatcher",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/pod"),
	)

	return nil
}

// ✅ 工厂方法：构造 PodWatcher 实例（注入 client）
func NewPodWatcher(c client.Client) *PodWatcher {
	return &PodWatcher{client: c}
}
