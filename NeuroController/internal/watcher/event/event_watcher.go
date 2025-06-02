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

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

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
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return e.ObjectOld.GetResourceVersion() != e.ObjectNew.GetResourceVersion()
			},
		}).
		Complete(w)
}

// =======================================================================================
// ✅ 控制器回调：监听 Event 变更 → 筛选异常 → 执行处理
func (w *EventWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ev corev1.Event

	if err := w.client.Get(ctx, req.NamespacedName, &ev); err != nil {
		utils.Warn(ctx, "❌ 获取 Event 失败",
			utils.WithTraceID(ctx),
			zap.String("event", req.Name),
			zap.Error(err),
		)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	processEventIfNeeded(ctx, ev)
	return ctrl.Result{}, nil
}

// =======================================================================================
// ✅ 事件处理逻辑（封装判断 & 日志输出）
//
// 仅处理类型为 Warning 且属于预定义 Reason 列表的异常事件。
func processEventIfNeeded(ctx context.Context, ev corev1.Event) {
	if !isAbnormalEvent(ev) {
		return
	}

	utils.Warn(ctx, "⚠️ 捕捉到异常 Event",
		utils.WithTraceID(ctx),
		zap.String("reason", ev.Reason),
		zap.String("message", ev.Message),
		zap.String("kind", ev.InvolvedObject.Kind),
		zap.String("name", ev.InvolvedObject.Name),
		zap.String("namespace", ev.InvolvedObject.Namespace),
	)
}

// =======================================================================================
// ✅ 判断函数：是否为异常 Event（类型 + 原因）
func isAbnormalEvent(ev corev1.Event) bool {
	return ev.Type == corev1.EventTypeWarning &&
		abnormalEventReasons[ev.Reason]
}

// =======================================================================================
// ✅ 异常事件原因映射表（用于识别需重点关注的 Warning Event）
//
// Event.Reason 字段常用于描述事件发生的根本原因，以下为常见异常类型：
// 可根据生产环境中频率和严重性适当增减。
var abnormalEventReasons = map[string]bool{
	"FailedScheduling":       true, // Pod 调度失败（如资源不足 / 节点亲和性不满足）
	"BackOff":                true, // 容器启动失败后进入退避重试状态（如主进程持续崩溃）
	"ErrImagePull":           true, // 镜像拉取失败（如镜像不存在 / 网络异常）
	"ImagePullBackOff":       true, // 镜像拉取失败 + 退避中（ErrImagePull 后进入该状态）
	"FailedCreatePodSandBox": true, // Pod 沙箱创建失败（如 CNI 问题 / runtime 异常）
	"FailedMount":            true, // 卷挂载失败（如路径不存在 / 权限不足）
	"FailedAttachVolume":     true, // 卷附加失败（多见于 PVC / PV / 云盘等）
	"FailedMapVolume":        true, // 卷映射失败（如挂载点配置错误）
	"Unhealthy":              true, // 容器健康检查失败（如 readiness/liveness probe 检测未通过）
	"FailedKillPod":          true, // 无法终止 Pod（可能由进程卡死 / runtime 异常引起）
	"Failed":                 true, // 通用失败（不属于其他细分类的错误原因）
}
