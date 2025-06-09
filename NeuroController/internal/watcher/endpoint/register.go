// =======================================================================================
// 📄 watcher/endpoint/register.go
//
// ✨ 功能说明：
//     注册 EndpointWatcher 到 controller-runtime 管理器中，实现自动监听所有 Endpoints 状态变化。
//     封装监听器实例构造（NewEndpointWatcher）与 controller 绑定（SetupWithManager）逻辑，
//     解耦 controller/main.go 与 watcher 具体实现细节。
// =======================================================================================

package endpoint

import (
	"NeuroController/internal/utils"
	"context"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RegisterWatcher(mgr ctrl.Manager) error {
	client := utils.GetClient()
	watcher := NewEndpointWatcher(client)

	if err := watcher.SetupWithManager(mgr); err != nil {
		utils.Error(context.TODO(), "❌ 注册 EndpointWatcher 失败",
			utils.WithTraceID(context.TODO()),
			zap.String("module", "watcher/endpoint"),
			zap.Error(err),
		)
		return err
	}

	utils.Info(context.TODO(), "✅ 成功注册 EndpointWatcher",
		utils.WithTraceID(context.TODO()),
		zap.String("module", "watcher/endpoint"),
	)
	return nil
}

func NewEndpointWatcher(c client.Client) *EndpointWatcher {
	return &EndpointWatcher{client: c}
}
