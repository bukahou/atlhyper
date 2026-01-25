// source/watcher/event/event_watcher.go
// Event Watcher 实现
package event

import (
	"context"
	"log"

	"AtlHyper/atlhyper_agent/source/event/datahub"
	"AtlHyper/atlhyper_agent/source/event/abnormal"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// =======================================================================================
// ✅ 结构体：EventWatcher
//
// 封装了 Kubernetes 客户端，用于 controller-runtime 中的事件监听器
type EventWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 控制器注册方法
//
// 将 EventWatcher 注册到 controller-runtime 中，监听 Kubernetes 的 Event 资源
func (w *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(w)
}

// =======================================================================================
// ✅ EventWatcher 的 Reconcile 逻辑
//
// 在 Event 资源发生变更时触发。
// 仅处理类型为 "Warning" 的事件，并进行异常检测。
func (w *EventWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ev corev1.Event
	if err := w.client.Get(ctx, req.NamespacedName, &ev); err != nil {
		if !errors.IsNotFound(err) {
			log.Printf("❌ 获取 Event 失败: %s/%s → %v", req.Namespace, req.Name, err)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 检测是否为异常事件（内部已处理节流逻辑）
	reason := abnormal.GetEventAbnormalReason(ev)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// 收集并持久化该异常事件
	datahub.CollectEventAbnormalEvent(ev, reason)

	// TODO：触发后续处理逻辑（如告警、自动扩缩容等）
	return ctrl.Result{}, nil
}
