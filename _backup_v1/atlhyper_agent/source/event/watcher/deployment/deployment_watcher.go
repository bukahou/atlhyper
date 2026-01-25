// source/watcher/deployment/deployment_watcher.go
// Deployment Watcher 实现
package deployment

import (
	"context"
	"log"

	"AtlHyper/atlhyper_agent/source/event/datahub"
	"AtlHyper/atlhyper_agent/source/event/abnormal"

	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// =======================================================================================
// ✅ 结构体：DeploymentWatcher
//
// 封装了 Kubernetes 客户端，并实现了 controller-runtime 的 Reconciler 接口。
type DeploymentWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 方法：SetupWithManager
//
// 将该控制器注册到 manager，用于监听 Deployment 资源。
// 默认只在状态变更时触发。
func (w *DeploymentWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(w)
}

// =======================================================================================
// ✅ 方法：Reconcile
//
// Deployment 状态变更时的核心处理逻辑。
// 利用 abnormal 模块检测异常情况，必要时触发诊断流程。
func (w *DeploymentWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var deploy appsv1.Deployment
	if err := w.client.Get(ctx, req.NamespacedName, &deploy); err != nil {
		log.Printf("❌ 获取 Deployment 失败: %s/%s → %v", req.Namespace, req.Name, err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 🔍 分析是否存在异常（内部自动处理冷却时间逻辑）
	reason := abnormal.GetDeploymentAbnormalReason(deploy)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// 收集并上报 Deployment 异常事件
	datahub.CollectDeploymentAbnormalEvent(deploy, reason)

	return ctrl.Result{}, nil
}
