// =======================================================================================
// 📄 event_api.go（interfaces/ui_api）
//
// ✨ 文件功能说明：
//     提供 Kubernetes Event 查询相关的逻辑封装接口，供 HTTP handler 层调用：
//     - 查询全集群事件
//     - 查询指定命名空间事件
//     - 查询指定资源关联事件（Kind + Name + Namespace）
//     - 聚合事件类型数量（如 Warning / Normal）
//
// 📦 依赖模块：
//     - internal/query/event
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package uiapi

import (
	"context"

	"NeuroController/internal/query/event"

	corev1 "k8s.io/api/core/v1"
)

// GetAllEvents 返回所有命名空间的事件
func GetAllEvents(ctx context.Context) ([]corev1.Event, error) {
	return event.ListAllEvents(ctx)
}

// GetEventsByNamespace 返回指定命名空间的事件
func GetEventsByNamespace(ctx context.Context, ns string) ([]corev1.Event, error) {
	return event.ListEventsByNamespace(ctx, ns)
}

// GetEventsByInvolvedObject 返回某资源对象关联的事件（如 Pod、Deployment 等）
func GetEventsByInvolvedObject(ctx context.Context, namespace, kind, name string) ([]corev1.Event, error) {
	return event.ListEventsByInvolvedObject(ctx, namespace, kind, name)
}

// GetEventTypeCounts 返回事件类型分布统计（用于 UI 概览）
func GetEventTypeCounts(ctx context.Context) (map[string]int, error) {
	return event.CountEventsByType(ctx)
}

// GetPersistedEventLogs 查询最近 N 天的结构化日志
// func GetPersistedEventLogs(days int) ([]types.LogEvent, error) {
// 	return logger.GetRecentEventLogs(days)
// }
