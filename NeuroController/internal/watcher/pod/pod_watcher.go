// =======================================================================================
// 📄 watcher/pod/pod_watcher.go
//
// ✨ Description:
//     Implements the core logic of the PodWatcher controller,
//     responsible for listening to Pod status changes in the cluster.
//     Automatically detects abnormal states (e.g., CrashLoopBackOff, ImagePullBackOff, OOMKilled),
//     and delegates decisions to the strategy module to determine whether to trigger actions.
//     Actual responses (e.g., scaling, alerting) are handled by the actuator and reporter modules.
//
// 🛠️ Features:
//     - Reconcile(): Callback triggered by controller-runtime upon Pod status changes
//     - isCrashLoopOrFailed(): Determines if the Pod is in an abnormal state
//
// 📦 Dependencies:
//     - controller-runtime (controller binding and event handling)
//     - strategy module (abnormal state detection and decision making)
//     - actuator module (replica control)
//     - reporter module (email alerting)
//     - utils (logging, K8s client utilities)
//
// 📍 Usage:
//     - Register in watcher/pod/register.go, initialized by controller/main.go
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package pod

import (
	"context"

	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// =======================================================================================
// ✅ Struct: PodWatcher
//
// Wraps the Kubernetes client and acts as a controller-runtime Reconciler.
type PodWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ Method: SetupWithManager
//
// Registers the PodWatcher with the controller-runtime manager,
// configured to watch only Pod status changes.
func (w *PodWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(w)
}

// =======================================================================================
// ✅ Method: Reconcile
//
// Core reconciliation logic triggered on Pod status changes.
// If an abnormal state is detected, it's recorded via the diagnosis module.
// Future extensions may include invoking actuator or reporter modules.
func (w *PodWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod
	err := w.client.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logPodDeleted(ctx, req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		logPodGetError(ctx, req.Namespace, req.Name, err)
		return ctrl.Result{}, err
	}

	// ✨ Detect abnormal states (includes cooldown check)
	reason := abnormal.GetPodAbnormalReason(pod)
	if reason == nil {
		// Optionally: fmt.Printf("✅ Pod is healthy: %s/%s\n", req.Namespace, req.Name)
		return ctrl.Result{}, nil
	}

	// Record abnormal event for further processing
	diagnosis.CollectPodAbnormalEvent(pod, reason)

	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ Helper: logPodDeleted
//
// Logs when a Pod has been deleted (often during rolling updates).
func logPodDeleted(ctx context.Context, namespace, name string) {
	utils.Info(ctx, "ℹ️ Pod has been deleted (possibly due to a rolling update)",
		utils.WithTraceID(ctx),
		zap.String("namespace", namespace),
		zap.String("pod", name),
	)
}

// =======================================================================================
// ✅ Helper: logPodGetError
//
// Logs when a Pod retrieval fails due to reasons other than NotFound.
func logPodGetError(ctx context.Context, namespace, name string, err error) {
	utils.Warn(ctx, "❌ Failed to retrieve Pod",
		utils.WithTraceID(ctx),
		zap.String("namespace", namespace),
		zap.String("pod", name),
		zap.String("error", err.Error()),
	)
}
