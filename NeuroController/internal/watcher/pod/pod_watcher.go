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
	"log"

	"NeuroController/internal/diagnosis"
	"NeuroController/internal/watcher/abnormal"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// =======================================================================================
// ✅ 结构体：PodWatcher
//
// 封装 Kubernetes 客户端，实现 controller-runtime 的 Reconciler 接口。
type PodWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 方法：SetupWithManager
//
// 将 PodWatcher 注册到 controller-runtime 的管理器中，
// 并配置为仅在 Pod 状态变化时触发。
func (w *PodWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(w)
}

// =======================================================================================
// ✅ 方法：Reconcile
//
// Pod 状态变更时触发的核心处理逻辑。
// 若检测到异常状态，则通过 diagnosis 模块记录该异常。
// 后续可扩展为调用执行器或上报模块。
func (w *PodWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod
	err := w.client.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logPodDeleted(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		logPodGetError(req.Namespace, req.Name, err)
		return ctrl.Result{}, err
	}

	// ✨ 检测是否为异常状态（已内置冷却判断）
	reason := abnormal.GetPodAbnormalReason(pod)
	if reason == nil {
		return ctrl.Result{}, nil
	}
	// 记录异常事件，供后续处理
	diagnosis.CollectPodAbnormalEvent(pod, reason)

	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 辅助函数：logPodDeleted
func logPodDeleted(namespace, name string) {
	log.Printf("🧹 Pod 已被删除: %s/%s", namespace, name)
}

// =======================================================================================
// ✅ 辅助函数：logPodGetError
func logPodGetError(namespace, name string, err error) {
	log.Printf("❌ 获取 Pod 失败: %s/%s → %v", namespace, name, err)
}
