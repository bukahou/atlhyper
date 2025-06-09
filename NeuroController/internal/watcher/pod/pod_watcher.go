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
	err := w.client.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logPodDeleted(ctx, req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		logPodGetError(ctx, req.Namespace, req.Name, err)
		return ctrl.Result{}, err
	}

	// ✨ 异常识别（包含冷却判断）
	reason := abnormal.GetPodAbnormalReason(pod)
	if reason == nil {
		// 可选加：fmt.Printf("✅ Pod 正常，无需处理：%s/%s\n", req.Namespace, req.Name)
		return ctrl.Result{}, nil
	}

	diagnosis.CollectPodAbnormalEvent(pod, reason)
	// logPodAbnormal(ctx, pod, reason)

	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 函数：输出结构化 Pod 异常日志
// func logPodAbnormal(ctx context.Context, pod corev1.Pod, reason *abnormal.PodAbnormalReason) {
// 	utils.Warn(ctx, "🚨 发现异常 Pod",
// 		utils.WithTraceID(ctx),
// 		zap.String("time", time.Now().Format(time.RFC3339)),
// 		zap.String("name", pod.Name),
// 		zap.String("namespace", pod.Namespace),
// 		zap.String("phase", string(pod.Status.Phase)),
// 		zap.String("reason", reason.Code),
// 		zap.String("category", reason.Category),
// 		zap.String("severity", reason.Severity),
// 		zap.String("message", reason.Message),
// 	)
// }

// =======================================================================================
// ✅ 函数：输出 Pod 被删除的 Info 日志（用于 CI/CD 场景识别）
// =======================================================================================
func logPodDeleted(ctx context.Context, namespace, name string) {
	utils.Info(ctx, "ℹ️ Pod 已被删除（可能为正常滚动更新）",
		utils.WithTraceID(ctx),
		zap.String("namespace", namespace),
		zap.String("pod", name),
	)
}

// =======================================================================================
// ✅ 函数：输出 Pod 获取失败日志（非 NotFound 情况）
// =======================================================================================
func logPodGetError(ctx context.Context, namespace, name string, err error) {
	utils.Warn(ctx, "❌ 获取 Pod 失败",
		utils.WithTraceID(ctx),
		zap.String("namespace", namespace),
		zap.String("pod", name),
		zap.String("error", err.Error()),
	)
}
