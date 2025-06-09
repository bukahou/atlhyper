// =======================================================================================
// 📄 watcher/node/node_watcher.go
//
// ✨ Description:
//     Implements the core logic of the NodeWatcher controller, responsible for observing
//     Node status changes and identifying abnormal conditions such as NotReady or Unknown.
//     Logs critical changes and triggers diagnosis routines.
//
// 🛠️ Features:
//     - Reconcile(): Callback method for controller-runtime, handles update logic
//     - isNodeAbnormal(): Determines if a Node is in an abnormal state (e.g., NotReady)
//
// 📦 Dependencies:
//     - controller-runtime (controller binding and event-driven updates)
//     - corev1.Node / NodeCondition (Kubernetes API types)
//     - utils (logging and Kubernetes client utilities)
//
// 📍 Usage:
//     - Registered in watcher/node/register.go, initialized from controller/main.go
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package node

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

// =======================================================================================
// ✅ Struct: NodeWatcher
//
// Wraps a Kubernetes client and acts as a controller-runtime Reconciler.
type NodeWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ Method: SetupWithManager
//
// Registers a controller with controller-runtime to monitor Node changes,
// triggering only on state transitions.
func (w *NodeWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(w)
}

// =======================================================================================
// ✅ Method: Reconcile
//
// Core logic entry point for Node abnormality detection.
func (w *NodeWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var node corev1.Node
	if err := w.client.Get(ctx, req.NamespacedName, &node); err != nil {
		utils.Warn(ctx, "❌ Failed to retrieve Node",
			utils.WithTraceID(ctx),
			zap.String("node", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ Identify abnormal state (internal cooldown handled)
	reason := abnormal.GetNodeAbnormalReason(node)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// ➕ Collect abnormal event for diagnosis module
	diagnosis.CollectNodeAbnormalEvent(node, reason)
	// logNodeAbnormal(ctx, node, reason) // optional logging

	// TODO: Implement alerting, auto-scaling, or APM reporting
	return ctrl.Result{}, nil
}
