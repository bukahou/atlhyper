// =======================================================================================
// 📄 watcher/node/node_watcher.go
//
// ✨ 功能说明：
//     实现 NodeWatcher 控制器的核心监听逻辑，负责监听 Node 状态变更事件，
//     判断是否为 NotReady / Unknown 等异常状态，并进行日志记录与通知响应。
//
// 🛠️ 提供功能：
//     - Reconcile(): controller-runtime 的回调函数，执行监听响应逻辑
//     - isNodeAbnormal(): 判断 Node 是否为异常状态（如 NotReady）
//
// 📦 依赖：
//     - controller-runtime（控制器绑定与监听事件驱动）
//     - corev1.Node / NodeCondition
//     - utils（日志打印、client 工具等）
//
// 📍 使用场景：
//     - 在 watcher/node/register.go 中注册，通过 controller/main.go 启动时加载
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package node

import (
	"context"
	"time"

	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"go.uber.org/zap"
)

// =======================================================================================
// ✅ 结构体：NodeWatcher
//
// 封装 Kubernetes client，并作为 controller-runtime 的 Reconciler 使用。
type NodeWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 方法：绑定 controller-runtime 控制器
//
// 注册用于监听 Node 状态变更的 controller，并绑定过滤器（仅状态变更时触发）。
func (w *NodeWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
			},
		}).
		Complete(w)
}

// =======================================================================================
// ✅ 方法：核心监听逻辑（Node 异常识别入口）
func (w *NodeWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var node corev1.Node
	if err := w.client.Get(ctx, req.NamespacedName, &node); err != nil {
		utils.Warn(ctx, "❌ 获取 Node 失败",
			utils.WithTraceID(ctx),
			zap.String("node", req.Name),
			zap.String("error", err.Error()),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 提取异常原因（内部已判断冷却期）
	reason := abnormal.GetNodeAbnormalReason(node)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// ✅ 输出日志（封装）
	logNodeAbnormal(ctx, node, reason)

	// TODO: 后续执行动作（告警 / 缩容）
	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 函数：输出结构化 Node 异常日志
func logNodeAbnormal(ctx context.Context, node corev1.Node, reason *abnormal.NodeAbnormalReason) {
	utils.Warn(ctx, "🚨 发现异常 Node",
		utils.WithTraceID(ctx),
		zap.String("time", time.Now().Format(time.RFC3339)),
		zap.String("node", node.Name),
		zap.String("reason", reason.Code),
		zap.String("message", reason.Message),
		zap.String("severity", reason.Severity),
		zap.String("category", reason.Category),
	)
}
