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
// ✅ Struct: DeploymentWatcher
//
// Wraps a Kubernetes client and implements controller-runtime's Reconciler interface.
type DeploymentWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ Method: SetupWithManager
//
// Registers the controller with the manager to watch Deployment resources.
// Automatically filters and only triggers on status changes.
func (w *DeploymentWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(w)
}

// =======================================================================================
// ✅ Method: Reconcile
//
// Core event handler for Deployment changes.
// Detects abnormalities using the abnormal module and triggers diagnostics if needed.
func (w *DeploymentWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var deploy appsv1.Deployment
	if err := w.client.Get(ctx, req.NamespacedName, &deploy); err != nil {
		utils.Warn(ctx, "Failed to fetch Deployment",
			utils.WithTraceID(ctx),
			zap.String("deployment", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 🔍 Analyze for abnormalities (cooldown logic handled internally)
	reason := abnormal.GetDeploymentAbnormalReason(deploy)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	diagnosis.CollectDeploymentAbnormalEvent(deploy, reason)
	// ✅ Structured log output can be added if needed:
	// logDeploymentAbnormal(ctx, deploy, reason)

	// TODO: Extend with autoscaling, email alerts, or APM reporting
	return ctrl.Result{}, nil
}
