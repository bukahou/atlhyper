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

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
			},
		}).
		Complete(w)
}

// =======================================================================================
// ✅ 方法：核心监听逻辑
//
// 在字段变更被筛选器触发后执行，记录异常和可疑的 Service 变更日志。
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

	processServiceChange(ctx, &svc)
	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 辅助函数：处理变更字段，按严重性打印分类日志
func processServiceChange(ctx context.Context, svc *corev1.Service) {
	if len(svc.Spec.Selector) == 0 {
		utils.Warn(ctx, "🚨 Service 未关联任何 Pod（Selector 为空）",
			utils.WithTraceID(ctx),
			zap.String("service", svc.Name),
			zap.String("namespace", svc.Namespace),
		)
	}

	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		utils.Warn(ctx, "⚠️ 检测到 ExternalName 类型 Service",
			utils.WithTraceID(ctx),
			zap.String("service", svc.Name),
		)
	}

	if svc.Spec.ClusterIP == "None" || svc.Spec.ClusterIP == "" {
		utils.Warn(ctx, "⚠️ Service ClusterIP 异常（为空或 None）",
			utils.WithTraceID(ctx),
			zap.String("service", svc.Name),
		)
	}
}
