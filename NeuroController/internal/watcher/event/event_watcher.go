// =======================================================================================
// 📄 watcher/event/event_watcher.go
//
// ✨ Description:
//     Implements the EventWatcher controller to monitor core Kubernetes events (Event),
//     such as image pull failure, volume mount failure, scheduling issues, etc.
//     Only processes events with Type = "Warning".
//
// 🛠️ Features:
//     - Watches corev1.Event resources
//     - Filters and handles only "Warning" type events
//
// 📦 Dependencies:
//     - controller-runtime (Kubernetes controller framework)
//     - corev1.Event (Kubernetes Event type)
//     - utils (logging utilities)
//
// 📍 Usage:
//     - Register in watcher/event/register.go
//     - Called and started by controller/main.go
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 🗓 Created: 2025-06
// =======================================================================================

package event

import (
	"context"

	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"
	"NeuroController/internal/utils/abnormal"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
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
			utils.Warn(ctx, "❌ 获取 Event 失败",
				utils.WithTraceID(ctx),
				zap.String("event", req.Name),
				zap.Error(err),
			)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ✨ 检测是否为异常事件（内部已处理节流逻辑）
	reason := abnormal.GetEventAbnormalReason(ev)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	// 收集并持久化该异常事件
	diagnosis.CollectEventAbnormalEvent(ev, reason)

	// TODO：触发后续处理逻辑（如告警、自动扩缩容等）
	return ctrl.Result{}, nil
}
