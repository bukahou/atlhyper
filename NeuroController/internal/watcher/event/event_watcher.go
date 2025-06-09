// =======================================================================================
// 📄 watcher/event/event_watcher.go
//
// ✨ Description:
//     Implements the EventWatcher controller to monitor core Kubernetes events (Event),
//     such as image pull failure, volume mount failure, scheduling issues, etc.
//     Only processes events with Type = "Warning".
//
// 🛠️ Features:
//     - Watches corev1.Event resources
//     - Filters and handles only "Warning" type events
//
// 📦 Dependencies:
//     - controller-runtime (Kubernetes controller framework)
//     - corev1.Event (Kubernetes Event type)
//     - utils (logging utilities)
//
// 📍 Usage:
//     - Register in watcher/event/register.go
//     - Called and started by controller/main.go
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package event

import (
	"context"

	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
)

// =======================================================================================
// ✅ Struct: EventWatcher
//
// Encapsulates Kubernetes client for use with controller-runtime
type EventWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ Setup the controller with the manager
//
// Registers the EventWatcher with controller-runtime to watch Event resources
func (w *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(w)
}

// =======================================================================================
// ✅ Reconcile logic for EventWatcher
//
// Triggered on changes to Event resources.
// Filters "Warning" type events and processes them.
func (w *EventWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ev corev1.Event
	if err := w.client.Get(ctx, req.NamespacedName, &ev); err != nil {
		if !errors.IsNotFound(err) {
			utils.Warn(ctx, "❌ Failed to retrieve Event",
				utils.WithTraceID(ctx),
				zap.String("event", req.Name),
				zap.Error(err),
			)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ Check for abnormal conditions (cooldown already handled internally)
	reason := abnormal.GetEventAbnormalReason(ev)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// ⛑️ Collect and persist the abnormal event
	diagnosis.CollectEventAbnormalEvent(ev, reason)

	// TODO: Trigger follow-up actions (alerts, autoscaling, etc.)
	return ctrl.Result{}, nil
}
