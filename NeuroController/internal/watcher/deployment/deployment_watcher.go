// =======================================================================================
// 📄 watcher/deployment/deployment_watcher.go
//
// ✨ 功能说明：
//     实现 DeploymentWatcher 控制器的核心监听逻辑，负责监听 Deployment 对象的状态变更，
//     根据 Replica 数量差异、指定未 Ready 对象、残待等待列等做记录和告警。
//
// 🛠️ 提供功能：
//     - Reconcile(): controller-runtime 的回调函数，执行监听响应逻辑
//
// 📦 依赖：
//     - controller-runtime
//     - apps/v1.Deployment
//     - utils
//
// 📌 使用场景：
//     - 在 watcher/deployment/register.go 中注册
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 🗓 创建时间：2025-06
// =======================================================================================

package deployment

import (
	"context"

	"NeuroController/internal/utils"

	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"go.uber.org/zap"
)

// ✅ 结构体：DeploymentWatcher

type DeploymentWatcher struct {
	client client.Client
}

// ✅ 方法：绑定 controller-runtime 控制器

func (w *DeploymentWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
			},
		}).
		Complete(w)
}

// ✅ 方法：核心监听逻辑

func (w *DeploymentWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var deploy appsv1.Deployment
	if err := w.client.Get(ctx, req.NamespacedName, &deploy); err != nil {
		utils.Warn(ctx, "❌ 获取 Deployment 失败",
			utils.WithTraceID(ctx),
			zap.String("deployment", req.Name),
			zap.Error(err),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	processDeploymentStatus(ctx, &deploy)
	return ctrl.Result{}, nil
}

// ✅ 辅助函数：处理 Deployment 应用状态
func processDeploymentStatus(ctx context.Context, deploy *appsv1.Deployment) {
	if deploy.Status.ReadyReplicas < *deploy.Spec.Replicas {
		utils.Warn(ctx, "🚨 Deployment Ready Replica 不足",
			utils.WithTraceID(ctx),
			zap.String("deployment", deploy.Name),
			zap.Int32("desired", *deploy.Spec.Replicas),
			zap.Int32("ready", deploy.Status.ReadyReplicas),
		)
	}

	if deploy.Status.UnavailableReplicas > 0 {
		utils.Warn(ctx, "⚠️ Deployment 包含 Unavailable Replica",
			utils.WithTraceID(ctx),
			zap.String("deployment", deploy.Name),
			zap.Int32("unavailable", deploy.Status.UnavailableReplicas),
		)
	}
}
