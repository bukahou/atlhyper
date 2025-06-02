// =======================================================================================
// 📄 watcher/service/register.go
//
// ✨ 功能说明：
//     注册 ServiceWatcher 到 controller-runtime 管理器中，实现自动监听 Service 变化。
//     封装监听器实例构造（NewServiceWatcher）与 controller 绑定（SetupWithManager）逻辑，
//     解耦 controller/main.go 与 watcher 具体实现细节。
//
// 🛠️ 提供功能：
//     - NewServiceWatcher(client.Client): 创建监听器实例（注入共享 client）
//     - RegisterWatcher(mgr ctrl.Manager): 注册监听器到 controller-runtime 管理器
//
// 📦 依赖：
//     - controller-runtime（Manager、控制器构造）
//     - service_watcher.go（监听逻辑定义）
//     - utils/k8s_client.go（获取全局共享 client 实例）
//
// 📍 使用场景：
//     - 在 controller/main.go 中统一加载 watcher/service 的注册器
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package service

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 注册器：注册 ServiceWatcher 到 controller-runtime
func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	serviceWatcher := NewServiceWatcher(client)

	if err := serviceWatcher.SetupWithManager(mgr); err != nil {
		utils.Error(
			context.TODO(),
			"❌ 注册 ServiceWatcher 失败",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/service"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(
		context.TODO(),
		"✅ 成功注册 ServiceWatcher",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/service"),
	)
	return nil
}

// ✅ 工厂方法：构造 ServiceWatcher 实例（注入 client）
func NewServiceWatcher(c client.Client) *ServiceWatcher {
	return &ServiceWatcher{client: c}
}
