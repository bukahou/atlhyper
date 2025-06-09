// =======================================================================================
// 📄 watcher/service/service_watcher.go
//
// ✨ 功能说明：
//     实现 ServiceWatcher 控制器的核心监听逻辑，负责监听 Service 对象的变更，
//     可用于未来感知 Service 的配置漂移、端口变动、选择器变化等。
//
// 🛠️ 提供功能：
//     - Reconcile(): controller-runtime 的回调函数，执行监听响应逻辑
//
// 📦 依赖：
//     - controller-runtime（控制器绑定与监听事件驱动）
//     - corev1.Service
//     - utils（日志打印、client 工具等）
//
// 📍 使用场景：
//     - 在 watcher/service/register.go 中注册，通过 controller/main.go 启动时加载
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package service

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

// ✅ 结构体：ServiceWatcher
//
// 封装 Kubernetes client，并作为 controller-runtime 的 Reconciler 使用。
type ServiceWatcher struct {
	client client.Client
}

// ✅ 方法：绑定 controller-runtime 控制器
func (w *ServiceWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(w)
}

// =======================================================================================
// ✅ 方法：核心监听逻辑（Service 异常识别入口）
func (w *ServiceWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var svc corev1.Service
	if err := w.client.Get(ctx, req.NamespacedName, &svc); err != nil {
		utils.Warn(ctx, "❌ 获取 Service 失败",
			utils.WithTraceID(ctx),
			zap.String("service", req.Name),
			zap.Error(err),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 提取异常原因（内部已判断冷却期）
	reason := abnormal.GetServiceAbnormalReason(svc)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	diagnosis.CollectServiceAbnormalEvent(svc, reason)
	// logServiceAbnormal(ctx, svc, reason)

	// TODO: 后续动作（如通知、自动修复）
	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 函数：输出结构化 Service 异常日志
// func logServiceAbnormal(ctx context.Context, svc corev1.Service, reason *abnormal.ServiceAbnormalReason) {
// 	utils.Warn(ctx, "🚨 发现异常 Service",
// 		utils.WithTraceID(ctx),
// 		zap.String("time", time.Now().Format(time.RFC3339)),
// 		zap.String("service", svc.Name),
// 		zap.String("namespace", svc.Namespace),
// 		zap.String("reason", reason.Code),
// 		zap.String("message", reason.Message),
// 		zap.String("severity", reason.Severity),
// 	)
// }
