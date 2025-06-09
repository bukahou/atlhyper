// =======================================================================================
// 📄 watcher/service/service_watcher.go
//
// ✨ Description:
//     Implements the core logic for the ServiceWatcher controller, responsible for
//     monitoring Service object changes. This may include detecting drift in service
//     configuration, port changes, or selector modifications in future extensions.
//
// 🛠️ Features:
//     - Reconcile(): Reconciliation function invoked by controller-runtime
//
// 📦 Dependencies:
//     - controller-runtime (controller binding and event triggers)
//     - corev1.Service (Kubernetes API object)
//     - utils (logging and client tools)
//
// 📍 Usage:
//     - Registered via watcher/service/register.go
//     - Loaded and started in controller/main.go
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package service

import (
	"context"

	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
)

// ✅ Struct: ServiceWatcher
//
// Encapsulates the Kubernetes client and serves as a Reconciler for controller-runtime.
type ServiceWatcher struct {
	client client.Client
}

// ✅ Method: Bind ServiceWatcher to controller-runtime manager
func (w *ServiceWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(w)
}

// =======================================================================================
// ✅ Method: Core reconciliation logic for Service object changes
//
// When a Service is created or updated, this method will be triggered by the controller-runtime.
// If an abnormal status is detected, it will be collected and passed to the diagnosis module.
func (w *ServiceWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var svc corev1.Service
	if err := w.client.Get(ctx, req.NamespacedName, &svc); err != nil {
		utils.Warn(ctx, "❌ Failed to fetch Service object",
			utils.WithTraceID(ctx),
			zap.String("service", req.Name),
			zap.Error(err),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ Analyze service for known abnormal patterns (with cooldown check)
	reason := abnormal.GetServiceAbnormalReason(svc)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	diagnosis.CollectServiceAbnormalEvent(svc, reason)
	// logServiceAbnormal(ctx, svc, reason)

	// TODO: Future enhancements (e.g. notifications, auto-healing)
	return ctrl.Result{}, nil
}
