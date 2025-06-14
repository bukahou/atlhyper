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

// ✅ 工厂方法：使用共享客户端创建 DeploymentWatcher 实例
func NewDeploymentWatcher(c client.Client) *DeploymentWatcher {
	return &DeploymentWatcher{client: c}
}

// ✅ 注册器：将 DeploymentWatcher 绑定到 controller-runtime 的管理器中
//
// 获取全局共享 client → 构建 watcher 实例 → 注册到 manager 中。
// 如果注册失败则记录错误日志并返回错误。
func RegisterWatcher(mgr ctrl.Manager) error {
	// 从 utils 获取全局共享 client
	client := utils.GetClient()

	// 创建 watcher 实例
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
		"✅ DeploymentWatcher 注册成功",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/deployment"),
	)

	return nil
}
