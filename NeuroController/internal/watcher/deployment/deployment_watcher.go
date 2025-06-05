// =======================================================================================
// 📄 watcher/deployment/deployment_watcher.go
//
// ✨ 功能说明：
//     实现 DeploymentWatcher 控制器的核心监听逻辑，负责监听 Deployment 状态变更事件，
//     判断是否存在副本异常（如 UnavailableReplica / ReadyReplicaMismatch / 超时）并记录日志。
//
// 🛠️ 提供功能：
//     - Reconcile(): controller-runtime 的回调函数，执行监听响应逻辑
//     - logDeploymentAbnormal(): 输出 Deployment 异常日志（已结构化）
//
// 📦 依赖：
//     - controller-runtime（控制器绑定与监听事件驱动）
//     - apps/v1.Deployment
//     - utils（日志系统 / trace 注入）
//     - abnormal（Deployment 异常识别与分类）
//
// 📍 使用场景：
//     - 在 watcher/deployment/register.go 中注册，通过 controller/main.go 启动时加载
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 🗓 创建时间：2025-06
// =======================================================================================

package deployment

import (
	"context"
	"time"

	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"go.uber.org/zap"
)

// =======================================================================================
// ✅ 结构体：DeploymentWatcher
//
// 封装 Kubernetes client，并作为 controller-runtime 的 Reconciler 使用。
type DeploymentWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 方法：绑定 controller-runtime 控制器
//
// 注册用于监听 Deployment 状态变更的 controller，并绑定过滤器（仅状态变更时触发）。
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

// =======================================================================================
// ✅ 方法：核心监听逻辑（Deployment 异常识别入口）
func (w *DeploymentWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var deploy appsv1.Deployment
	if err := w.client.Get(ctx, req.NamespacedName, &deploy); err != nil {
		utils.Warn(ctx, "获取 Deployment 失败",
			utils.WithTraceID(ctx),
			zap.String("deployment", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 提取异常原因（内部已判断冷却期）
	reason := abnormal.GetDeploymentAbnormalReason(deploy)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// ✅ 输出日志（封装）
	logDeploymentAbnormal(ctx, deploy, reason)

	// TODO: 可扩展自动缩容 / 邮件通知 / APM 上报
	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 函数：输出结构化 Deployment 异常日志
func logDeploymentAbnormal(ctx context.Context, deploy appsv1.Deployment, reason *abnormal.DeploymentAbnormalReason) {
	utils.Warn(ctx, "⚠️ 发现 Deployment 异常",
		utils.WithTraceID(ctx),
		zap.String("time", time.Now().Format(time.RFC3339)),
		zap.String("deployment", deploy.Name),
		zap.String("namespace", deploy.Namespace),
		zap.String("reason", reason.Code),
		zap.String("message", reason.Message),
		zap.String("severity", reason.Severity),
		zap.String("category", reason.Category),
	)
}
