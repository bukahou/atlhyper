// =======================================================================================
// 📄 watcher/pod/pod_watcher.go
//
// ✨ 功能说明：
//     实现 PodWatcher 控制器的核心监听逻辑，负责接收集群中 Pod 状态变更事件，
//     自动识别 CrashLoopBackOff、Failed 等异常状态，并调用策略模块判断是否触发响应动作。
//     最终由 actuator 和 reporter 模块执行具体操作（如缩容、告警）。
//
// 🛠️ 提供功能：
//     - Reconcile(): controller-runtime 的回调函数，执行具体监听响应逻辑
//     - isCrashLoopOrFailed(): 判定 Pod 是否为异常状态
//
// 📦 依赖：
//     - controller-runtime（控制器绑定与监听事件驱动）
//     - strategy 模块（异常识别与响应决策）
//     - actuator 模块（副本数控制）
//     - reporter 模块（邮件报警推送）
//     - utils（日志打印、client 工具等）
//
// 📍 使用场景：
//     - 在 watcher/pod/register.go 中进行注册，通过 controller/main.go 启动时加载
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package pod

import (
	"context"

	"NeuroController/internal/utils"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// =======================================================================================
// ✅ 结构体：PodWatcher
//
//	用于封装 Kubernetes client，并作为 controller-runtime 的 Reconciler 使用。
type PodWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 方法：绑定 controller-runtime 控制器
//
// 注册用于监听 Pod 状态变更的 controller，并为其绑定过滤器（仅在状态变更时触发）。
func (w *PodWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// 仅在 Pod 实际状态变化时触发（避免重复 Reconcile）
				return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
			},
		}).
		Complete(w)

}

// =======================================================================================
// ✅ 方法：核心监听逻辑
//
// 当 Pod 状态变更时由 controller-runtime 调用该方法进行处理，
// 若发现异常状态（如 CrashLoopBackOff、ImagePullBackOff、OOMKilled 等），
// 则交由策略模块判断并触发 actuator/reporter。
func (w *PodWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	var pod corev1.Pod
	if err := w.client.Get(ctx, req.NamespacedName, &pod); err != nil {
		utils.Warn(ctx, "❌ 获取 Pod 失败",
			utils.WithTraceID(ctx),
			zap.String("namespace", req.Namespace),
			zap.String("pod", req.Name),
			zap.String("error", err.Error()),
		)

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✅ 检查是否为异常状态（包含 Phase、Waiting、Terminated）
	if isPodAbnormal(pod) {
		utils.Warn(ctx, "🚨 发现异常 Pod",
			utils.WithTraceID(ctx),
			zap.String("name", pod.Name),
			zap.String("namespace", pod.Namespace),
			zap.String("phase", string(pod.Status.Phase)),
		)

		// ⚠️ 暂时跳过策略模块，默认启用所有操作（后续用策略替换）

		// ⚙️ 缩容
		//actuator.ScaleDeploymentToZero(ctx, w.client, pod)

		// 📧 发送报警通知
		//reporter.SendCrashAlert(ctx, pod, "触发默认异常响应：未使用策略模块")

	}

	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 辅助函数：判断 Pod 是否为异常状态
//
// 包含 Phase 为 Failed/Unknown 或 Container 状态为 CrashLoopBackOff。
func isPodAbnormal(pod corev1.Pod) bool {
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
		return true
	}
	for _, cs := range pod.Status.ContainerStatuses {
		// 检查 Waiting 状态
		if cs.State.Waiting != nil {
			if isAbnormalWaitingReason(cs.State.Waiting.Reason) {
				return true
			}
		}
		// 检查 Terminated 状态
		if cs.State.Terminated != nil {
			if isAbnormalTerminatedReason(cs.State.Terminated.Reason) {
				return true
			}
		}
	}
	return false
}

// =======================================================================================
// ✅ 异常原因映射表（Waiting 状态）
//
// 定义所有被视为异常的 Pod Container 等待状态原因，
// 例如镜像拉取失败、容器创建失败等。
var abnormalWaitingReasons = map[string]bool{
	"CrashLoopBackOff":     true, // 容器反复崩溃重启
	"ImagePullBackOff":     true, // 镜像拉取失败并进入退避状态
	"ErrImagePull":         true, // 镜像拉取错误
	"CreateContainerError": true, // 容器创建失败
}

// ✅ 异常原因映射表（Terminated 状态）
//
// 定义所有被视为异常的已终止状态的原因，例如 OOMKilled 等。
var abnormalTerminatedReasons = map[string]bool{
	"OOMKilled": true, // 容器因超出内存限制被杀死
	"Error":     true, // 通用错误退出状态
}

// =======================================================================================
// ✅ 方法：判断是否为异常的 Waiting 状态原因
//
// 用于检查 ContainerStatus.State.Waiting.Reason 是否属于预定义的异常列表。
func isAbnormalWaitingReason(reason string) bool {
	return abnormalWaitingReasons[reason]
}

// ✅ 方法：判断是否为异常的 Terminated 状态原因
//
// 用于检查 ContainerStatus.State.Terminated.Reason 是否属于预定义的异常列表。
func isAbnormalTerminatedReason(reason string) bool {
	return abnormalTerminatedReasons[reason]
}
