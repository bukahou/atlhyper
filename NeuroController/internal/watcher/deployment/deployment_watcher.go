// =======================================================================================
// 📄 watcher/deployment/deployment_watcher.go
//
// ✨ Description:
//     Implements the core controller logic for DeploymentWatcher,
//     responsible for watching Deployment status changes and identifying
//     replica-related abnormalities (e.g., UnavailableReplicas, mismatch in ReadyReplicas, timeout).
//
// 🛠️ Features:
//     - Reconcile(): Main controller-runtime callback that reacts to status changes
//     - logDeploymentAbnormal(): Emits structured log entries for abnormal Deployments
//
// 📦 Dependencies:
//     - controller-runtime (controller registration and event handling)
//     - apps/v1.Deployment
//     - utils (logging / trace injection)
//     - abnormal (Deployment abnormality detection and classification)
//
// 📍 Usage:
//     - Registered in watcher/deployment/register.go
//     - Loaded during controller startup via controller/main.go
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package deployment

import (
	"context"

	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
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
		utils.Warn(ctx, "获取 Deployment 失败",
			utils.WithTraceID(ctx),
			zap.String("deployment", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 🔍 分析是否存在异常（内部自动处理冷却时间逻辑）
	reason := abnormal.GetDeploymentAbnormalReason(deploy)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// 收集并上报 Deployment 异常事件
	diagnosis.CollectDeploymentAbnormalEvent(deploy, reason)

	// ✅ 如有需要可补充结构化日志输出：
	// logDeploymentAbnormal(ctx, deploy, reason)

	// TODO：后续可拓展为自动伸缩、邮件告警、APM 上报等
	return ctrl.Result{}, nil
}
