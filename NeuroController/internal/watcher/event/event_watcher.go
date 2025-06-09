// =======================================================================================
// 📄 watcher/event/event_watcher.go
//
// ✨ 功能说明：
//     实现 EventWatcher 控制器，用于监听 Kubernetes 中的核心事件（Event），
//     如拉取失败、挂载失败、调度失败等，并筛选出 Warning 级别进行处理。
//
// 🛠️ 提供功能：
//     - 监听 Event 类型资源
//     - 仅处理 Type="Warning" 的事件
//
// 📦 依赖：
//     - controller-runtime
//     - corev1.Event
//     - utils 日志模块
//
// 📍 使用场景：
//     - watcher/event/register.go 注册后，controller/main.go 加载启动
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
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
// ✅ 控制器结构体
type EventWatcher struct {
	client client.Client
}

// =======================================================================================
// ✅ 绑定 Controller 到 Manager
func (w *EventWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Event{}).
		Complete(w)
}

// =======================================================================================
// ✅ 控制器回调：监听 Event 变更 → 筛选异常 → 执行处理
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

	// ✨ 提取异常原因（内部已判断冷却期）
	reason := abnormal.GetEventAbnormalReason(ev)
	if reason == nil {
		return ctrl.Result{}, nil
	}

	diagnosis.CollectEventAbnormalEvent(ev, reason)
	// logAbnormalEvent(ctx, ev, reason)

	// TODO: 后续执行动作（告警 / 缩容）
	return ctrl.Result{}, nil
}
