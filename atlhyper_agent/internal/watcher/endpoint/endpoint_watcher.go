package endpoint

import (
	"context"
	"log"

	"AtlHyper/atlhyper_agent/internal/diagnosis"
	"AtlHyper/atlhyper_agent/internal/watcher/abnormal"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ✅ 控制器结构体
type EndpointWatcher struct {
	client client.Client
}

// ✅ 将 EndpointWatcher 注册到 controller-runtime 的管理器中
func (w *EndpointWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		Complete(w)
}

// ✅ 核心逻辑：在 Endpoint 发生变更时触发
func (w *EndpointWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ep corev1.Endpoints
	if err := w.client.Get(ctx, req.NamespacedName, &ep); err != nil {
		log.Printf("❌ 获取 Endpoints 失败: %s/%s → %v", req.Namespace, req.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//  分析是否存在异常状态
	reason := abnormal.GetEndpointAbnormalReason(&ep)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	//  收集异常事件，供诊断或上报使用
	diagnosis.CollectEndpointAbnormalEvent(ep, reason)

	return ctrl.Result{}, nil
}
