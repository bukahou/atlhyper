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

	"NeuroController/internal/utils"

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
// ✅ 方法：核心监听逻辑
//
// 当 Node 状态变更时由 controller-runtime 调用，判断是否为 NotReady / Unknown，
// 若异常则记录日志，后续可扩展为通知或策略响应。
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

	if isNodeAbnormal(node) {
		utils.Warn(ctx, "🚨 发现异常 Node",
			utils.WithTraceID(ctx),
			zap.String("node", node.Name),
			zap.Any("conditions", node.Status.Conditions),
		)
	}

	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 辅助函数：判断 Node 是否为异常状态（支持分类）
//
// 判断 Node 条件中是否存在严重异常（如 Ready=False/Unknown），
// 或资源类异常（如 Memory/Disk/PID/Network）。
func isNodeAbnormal(node corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if isFatalNodeCondition(cond) {
			return true
		}
		if isWarningNodeCondition(cond) {
			utils.Warn(context.TODO(), "⚠️ Node 次级资源异常",
				utils.WithTraceID(context.TODO()),
				zap.String("node", node.Name),
				zap.String("type", string(cond.Type)),
				zap.String("status", string(cond.Status)),
			)
		}
	}
	return false
}

// =======================================================================================
// ✅ 异常分类定义（Node）
//
// 严重异常类型（Fatal）：需要立即处理的条件（如 NotReady / Unknown）
// 资源压力类型（Warning）：可选记录或预警的条件（如资源异常）
var fatalNodeConditions = map[corev1.NodeConditionType]bool{
	corev1.NodeReady: true, // 节点状态异常（离线/通信失败）
}

var warningNodeConditions = map[corev1.NodeConditionType]bool{
	corev1.NodeMemoryPressure:     true, // 内存压力
	corev1.NodeDiskPressure:       true, // 磁盘空间不足
	corev1.NodePIDPressure:        true, // 可用进程数耗尽
	corev1.NodeNetworkUnavailable: true, // 网络不可用（如 CNI 未启动）
}

// =======================================================================================
// ✅ 判断是否为致命异常（Ready=False / Unknown）
func isFatalNodeCondition(cond corev1.NodeCondition) bool {
	return fatalNodeConditions[cond.Type] &&
		(cond.Status == corev1.ConditionFalse || cond.Status == corev1.ConditionUnknown)
}

// =======================================================================================
// ✅ 判断是否为资源压力异常（True 表示资源异常）
func isWarningNodeCondition(cond corev1.NodeCondition) bool {
	return warningNodeConditions[cond.Type] && cond.Status == corev1.ConditionTrue
}
