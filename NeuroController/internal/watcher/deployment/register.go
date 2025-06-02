// =======================================================================================
// 📄 watcher/deployment/register.go
//
// ✨ 功能说明：
//     注册 DeploymentWatcher 到 controller-runtime 管理器中，实现自动监听所有 Deployment 状态变化。
//     封装监听器实例构造（NewDeploymentWatcher）与 controller 绑定（SetupWithManager）逻辑，
//     解耦 controller/main.go 与 watcher 具体实现细节。
//
// 🛠️ 提供功能：
//     - NewDeploymentWatcher(client.Client): 创建监听器实例（注入共享 client）
//     - RegisterWatcher(mgr ctrl.Manager): 注册监听器到 controller-runtime 管理器
//
// 📦 依赖：
//     - controller-runtime（Manager、控制器构造）
//     - deployment_watcher.go（监听逻辑定义）
//     - utils/k8s_client.go（获取全局共享 client 实例）
//
// 📍 使用场景：
//     - 在 controller/main.go 中统一加载 watcher/deployment 的注册器
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package deployment

import (
	"context"

	"NeuroController/internal/utils"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 工厂方法：构造 DeploymentWatcher 实例（注入 client）
func NewDeploymentWatcher(c client.Client) *DeploymentWatcher {
	return &DeploymentWatcher{client: c}
}

// ✅ 注册器：注册 DeploymentWatcher 到 controller-runtime
//
// 获取共享 K8s client → 构造监听器实例 → 注册到 controller-runtime 管理器。
// 若注册失败，将记录日志并返回错误。
func RegisterWatcher(mgr ctrl.Manager) error {
	// 获取共享 client（从 utils 中封装）
	client := utils.GetClient()

	// 构造监听器实例
	deploymentWatcher := NewDeploymentWatcher(client)

	// 注册控制器
	if err := deploymentWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ 注册 DeploymentWatcher 失败",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/deployment"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ 成功注册 DeploymentWatcher",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/deployment"),
	)

	return nil
}
