package node

import (
	"context"
	"log"

	"AtlHyper/atlhyper_agent/internal/diagnosis"
	"AtlHyper/atlhyper_agent/internal/watcher/abnormal"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		log.Printf("❌ 获取 Node 失败: %s → %v", req.NamespacedName.String(), err)
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
