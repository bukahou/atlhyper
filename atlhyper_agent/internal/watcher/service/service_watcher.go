package service

import (
	"context"
	"log"

	"AtlHyper/atlhyper_agent/internal/diagnosis"
	"AtlHyper/atlhyper_agent/internal/watcher/abnormal"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 结构体：ServiceWatcher
//
// 封装 Kubernetes 客户端，作为 controller-runtime 的 Reconciler 使用。
type ServiceWatcher struct {
	client client.Client
}

// ✅ 方法：将 ServiceWatcher 绑定到 controller-runtime 的管理器中
func (w *ServiceWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(w)
}

// =======================================================================================
// ✅ 方法：Service 对象的核心调和逻辑
//
// 当 Service 被创建或更新时，该方法将由 controller-runtime 触发。
// 若检测到异常状态，将被收集并传递给 diagnosis 模块处理。
func (w *ServiceWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var svc corev1.Service
	if err := w.client.Get(ctx, req.NamespacedName, &svc); err != nil {
		log.Printf("❌ 获取 Service 失败: %s/%s → %v", req.Namespace, req.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 分析是否存在已知异常模式（内部已处理冷却时间）
	reason := abnormal.GetServiceAbnormalReason(svc)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// 上报异常事件到诊断模块
	diagnosis.CollectServiceAbnormalEvent(svc, reason)
	// logServiceAbnormal(ctx, svc, reason) // 可选结构化日志

	// TODO：后续可添加通知、自动修复等增强功能
	return ctrl.Result{}, nil
}
