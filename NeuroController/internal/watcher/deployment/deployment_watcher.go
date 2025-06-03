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
// 📌 控制器结构体定义，作为 controller-runtime 的 Reconciler 实现体
type DeploymentWatcher struct {
	client client.Client // controller-runtime 提供的通用 Client 接口（用于访问 K8s API 资源）
}

// ✅ 方法：绑定 controller-runtime 控制器
// 📌 将 DeploymentWatcher 注册为 Deployment 类型的控制器
//   - 使用 WithEventFilter 过滤掉无变更事件，减少不必要的 Reconcile 调用
func (w *DeploymentWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}). // 👀 监听对象为 apps/v1.Deployment 资源
		WithEventFilter(predicate.Funcs{
			// ⚙️ 只在资源版本变更时触发（资源内容有更新）
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
			},
		}).
		Complete(w)
}

// ✅ 方法：核心监听逻辑
// 📌 controller-runtime 的核心入口函数，每当 Deployment 状态变更时调用
func (w *DeploymentWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var deploy appsv1.Deployment

	// 🔍 根据 NamespacedName 查询当前变更的 Deployment 对象
	if err := w.client.Get(ctx, req.NamespacedName, &deploy); err != nil {
		// ❌ 获取失败（可能被删除或网络故障）
		utils.Warn(ctx, "❌ 获取 Deployment 失败",
			utils.WithTraceID(ctx),             // 🔗 注入 traceID（用于链路追踪）
			zap.String("deployment", req.Name), // 📎 打印变更对象名称
			zap.Error(err),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err) // ✅ 忽略 404 错误（资源不存在）
	}

	// 🧠 调用处理函数判断副本状态
	processDeploymentStatus(ctx, &deploy)
	return ctrl.Result{}, nil
}

// ✅ 辅助函数：处理 Deployment 应用状态
// 📌 对比 Deployment 的期望副本数（Spec.Replicas）与当前副本状态（Status）
func processDeploymentStatus(ctx context.Context, deploy *appsv1.Deployment) {
	// 🚨 检查 Ready 副本是否小于期望值（正常副本不足）
	if deploy.Status.ReadyReplicas < *deploy.Spec.Replicas {
		utils.Warn(ctx, "🚨 Deployment Ready Replica 不足",
			utils.WithTraceID(ctx),
			zap.String("deployment", deploy.Name),
			zap.Int32("desired", *deploy.Spec.Replicas),     // 期望副本数
			zap.Int32("ready", deploy.Status.ReadyReplicas), // 实际就绪副本数
		)
	}

	// ⚠️ 检查是否有不可用副本（例如崩溃重启）
	if deploy.Status.UnavailableReplicas > 0 {
		utils.Warn(ctx, "⚠️ Deployment 包含 Unavailable Replica",
			utils.WithTraceID(ctx),
			zap.String("deployment", deploy.Name),
			zap.Int32("unavailable", deploy.Status.UnavailableReplicas), // 当前不可用副本数
		)
	}
}
