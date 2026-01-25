// source/watcher/endpoint/endpoint_watcher.go
// Endpoint Watcher 实现 (使用 EndpointSlice API)
package endpoint

import (
	"context"
	"log"

	"AtlHyper/atlhyper_agent/source/event/datahub"
	"AtlHyper/atlhyper_agent/source/event/abnormal"

	discoveryv1 "k8s.io/api/discovery/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// =======================================================================================
// ✅ 控制器：EndpointSliceWatcher
//
// 说明：
// - Kubernetes v1.33+ 已弃用 core/v1 Endpoints。
// - 本控制器改为监听 discovery.k8s.io/v1 EndpointSlice。
// - 每当服务端点变化（例如 Pod IP、Ready 状态更新）时触发 Reconcile。
// =======================================================================================
type EndpointWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 注册到 controller-runtime 管理器
// =======================================================================================
func (w *EndpointWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&discoveryv1.EndpointSlice{}).
		Complete(w)
}

// =======================================================================================
// ✅ Reconcile —— 当 EndpointSlice 发生变更时触发
// =======================================================================================
func (w *EndpointWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var slice discoveryv1.EndpointSlice
	if err := w.client.Get(ctx, req.NamespacedName, &slice); err != nil {
		log.Printf("❌ 获取 EndpointSlice 失败: %s/%s → %v", req.Namespace, req.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 分析是否存在异常状态（例如 Ready=false 的 endpoint 数过多等）
	reason := abnormal.GetEndpointAbnormalReason(&slice)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// 📦 收集异常事件，供诊断或上报模块使用
	datahub.CollectEndpointAbnormalEvent(slice, reason)

	return ctrl.Result{}, nil
}
