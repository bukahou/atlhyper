// =======================================================================================
// 📄 watcher/endpoint/endpoint_watcher.go
//
// ✨ Description:
//     Implements the core logic of the EndpointWatcher controller, responsible for
//     monitoring the state changes of Endpoints objects in the cluster.
//     Detects abnormal conditions such as missing backend pods or empty Subsets,
//     and logs structured diagnostic information.
//
// 🛠️ Features:
//     - Reconcile(): The main controller-runtime callback that triggers on changes
//     - logEndpointAbnormal(): Wrapper for structured abnormal event logging
//
// 📍 Usage:
//     - Registered via watcher/endpoint/register.go and loaded from controller/main.go
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package endpoint

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

// ✅ Controller structure
type EndpointWatcher struct {
	client client.Client
}

// ✅ Bind EndpointWatcher to controller-runtime manager
func (w *EndpointWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		Complete(w)
}

// ✅ Core logic: triggered on Endpoint change events
func (w *EndpointWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ep corev1.Endpoints
	if err := w.client.Get(ctx, req.NamespacedName, &ep); err != nil {
		utils.Warn(ctx, "❌ Failed to fetch Endpoints",
			utils.WithTraceID(ctx),
			zap.String("endpoint", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 🚨 Analyze for abnormal condition
	reason := abnormal.GetEndpointAbnormalReason(&ep)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// 🧠 Collect abnormal event for diagnosis/reporting
	diagnosis.CollectEndpointAbnormalEvent(ep, reason)

	// 📝 Optional: log structured details
	// logEndpointAbnormal(ctx, ep, reason)

	// 🔧 TODO: Add response actions (e.g., alerts, scaling)
	return ctrl.Result{}, nil
}
