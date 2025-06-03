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
	"time"

	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

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

	// ✅ 获取异常主因（内部已判断冷却时间窗口）
	reason := abnormal.GetPodAbnormalReason(pod)
	if reason == nil {
		return ctrl.Result{}, nil // ✅ 无需处理
	}

	// ✅ 输出结构化异常日志
	utils.Warn(ctx, "🚨 发现异常 Pod",
		utils.WithTraceID(ctx),
		zap.String("time", time.Now().Format(time.RFC3339)),
		zap.String("name", pod.Name),
		zap.String("namespace", pod.Namespace),
		zap.String("phase", string(pod.Status.Phase)),
		zap.String("reason", reason.Code),
		zap.String("category", reason.Category),
		zap.String("severity", reason.Severity),
		zap.String("message", reason.Message),
	)

	// 🔧 后续可调用响应策略模块
	// actuator.ScaleDeploymentToZero(ctx, w.client, pod)
	// reporter.SendCrashAlert(ctx, pod, "触发默认异常响应：未使用策略模块")

	return ctrl.Result{}, nil
}
