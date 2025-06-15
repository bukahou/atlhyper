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
	"NeuroController/internal/watcher/abnormal"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
)

// =======================================================================================
// ✅ 结构体：NodeWatcher
//
// 封装 Kubernetes 客户端，实现 controller-runtime 的 Reconciler 接口。
type NodeWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 方法：SetupWithManager
//
// 将 NodeWatcher 注册到 controller-runtime，用于监听 Node 状态变化。
// 默认只在状态变更时触发。
func (w *NodeWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(w)
}

// =======================================================================================
// ✅ 方法：Reconcile
//
// 节点异常检测的核心逻辑入口。
func (w *NodeWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var node corev1.Node
	if err := w.client.Get(ctx, req.NamespacedName, &node); err != nil {
		utils.Warn(ctx, "❌ 获取 Node 资源失败",
			utils.WithTraceID(ctx),
			zap.String("node", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 判断是否处于异常状态（内部已处理节流逻辑）
	reason := abnormal.GetNodeAbnormalReason(node)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// ➕ 将异常事件收集并传递给诊断模块
	diagnosis.CollectNodeAbnormalEvent(node, reason)
	// logNodeAbnormal(ctx, node, reason) // 可选结构化日志输出

	// TODO：后续可实现告警、自动扩缩容或 APM 上报等功能
	return ctrl.Result{}, nil
}
