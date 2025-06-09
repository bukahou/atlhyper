// =======================================================================================
// 📄 watcher/endpoint/endpoint_watcher.go
//
// ✨ 功能说明：
//     实现 EndpointWatcher 控制器的核心监听逻辑，负责监听 Endpoints 对象状态变化，
//     检查是否出现无可用后端 / Subsets 为空等异常情况，并进行结构化日志输出。
//
// 🛠️ 提供功能：
//     - Reconcile(): controller-runtime 的回调函数，执行监听响应逻辑
//     - logEndpointAbnormal(): 异常日志输出封装
//
// 📍 使用场景：
//     - 在 watcher/endpoint/register.go 中注册，通过 controller/main.go 启动时加载
// =======================================================================================

package endpoint

import (
	"context"

	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
)

type EndpointWatcher struct {
	client client.Client
}

func (w *EndpointWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		Complete(w)
}

func (w *EndpointWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var ep corev1.Endpoints
	if err := w.client.Get(ctx, req.NamespacedName, &ep); err != nil {
		utils.Warn(ctx, "❌ 获取 Endpoints 失败",
			utils.WithTraceID(ctx),
			zap.String("endpoint", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 检查是否异常
	reason := abnormal.GetEndpointAbnormalReason(&ep)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	diagnosis.CollectEndpointAbnormalEvent(ep, reason)
	// 输出结构化日志
	// logEndpointAbnormal(ctx, ep, reason)

	// TODO: 后续响应操作
	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 函数：输出结构化 Endpoints 异常日志
// func logEndpointAbnormal(ctx context.Context, ep corev1.Endpoints, reason *abnormal.EndpointAbnormalReason) {
// 	utils.Warn(ctx, "🚨 发现异常 Endpoints",
// 		utils.WithTraceID(ctx),
// 		zap.String("time", time.Now().Format(time.RFC3339)),
// 		zap.String("endpoint", ep.Name),
// 		zap.String("namespace", ep.Namespace),
// 		zap.String("reason", reason.Code),
// 		zap.String("message", reason.Message),
// 		zap.String("severity", reason.Severity),
// 	)
// }
