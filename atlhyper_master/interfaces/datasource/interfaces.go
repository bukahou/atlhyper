// interfaces/datasource/interfaces.go
package datasource

import (
	"context"
	"time"
)

// Reader 定义统一的数据读取接口（通用型，返回底层模型，不做 UI 裁剪）
// -----------------------------------------------------------------------------
// 背景：
//   - 数据可能来自内存 Hub、数据库、远端 HTTP/gRPC 等多种来源。
//   - 通过 Reader 抽象统一访问方式，屏蔽底层实现差异，上层只关心“我要什么数据”。
//   - 未来可无缝替换实现（HubSources/DBSources/HttpSources），调用方无需改动。
// 约定：
//   - 事件返回 LogEvent（= model/event.LogEvent）。
//   - 指标返回 NodeMetricsSnapshot（= model/metrics.NodeMetricsSnapshot）。
//   - 资源列表返回底层模型别名（Pod/Node/Service/...）。
//   - 不做聚合/分桶/裁剪，这些留给上层 handler/service 处理。
type Reader interface {
	// ================== 事件 ==================
	// 获取最近 N 条事件（按时间倒序）
	GetK8sEventsRecent(ctx context.Context, clusterID string, limit int) ([]LogEvent, error)

	 // ================== 指标（集群级） ==================
    // 最新一次上报的全量节点快照
    GetClusterMetricsLatest(ctx context.Context, clusterID string) ([]NodeMetricsSnapshot, error)
    // 区间内的全量节点快照（扁平列表，按时间升序）
    GetClusterMetricsRange(ctx context.Context, clusterID string, since, until time.Time) ([]NodeMetricsSnapshot, error)

	// ================== 各类 *ListSnapshot（最新一次快照） ==================
	// 注意：这些方法只返回“最新一次快照”，因为这些资源是按快照上报的
	GetPodListLatest(ctx context.Context, clusterID string) ([]Pod, error)
	GetNodeListLatest(ctx context.Context, clusterID string) ([]Node, error)
	GetServiceListLatest(ctx context.Context, clusterID string) ([]Service, error)
	GetNamespaceListLatest(ctx context.Context, clusterID string) ([]Namespace, error)
	GetIngressListLatest(ctx context.Context, clusterID string) ([]Ingress, error)
	GetDeploymentListLatest(ctx context.Context, clusterID string) ([]Deployment, error)
	GetConfigMapListLatest(ctx context.Context, clusterID string) ([]ConfigMap, error)

	// 新增：返回去重后的 ClusterID 列表
    ListClusterIDs(ctx context.Context) ([]string, error)
}

// -----------------------------------------------------------------------------
// 全局注入点：impl
// -----------------------------------------------------------------------------
// - 默认实现为 HubSources（基于 master_store.Snapshot()）。
// - 若要切换到 DB/HTTP 等实现，启动时调用 SetReader 注入即可。
var impl Reader = &HubSources{}

// SetReader 允许在启动时替换实现（例如换成 DBSources/HttpSources）
func SetReader(r Reader) { impl = r }

// ================== 对外导出的读取函数（转发到 impl） ==================
// -----------------------------------------------------------------------------
// 这些函数是上层 handler 的统一入口；它们只做转发，保证调用层与底层解耦。
// -----------------------------------------------------------------------------

// 事件
func GetK8sEventsRecent(ctx context.Context, clusterID string, limit int) ([]LogEvent, error) {
	return impl.GetK8sEventsRecent(ctx, clusterID, limit)
}

// 指标
func GetClusterMetricsLatest(ctx context.Context, clusterID string) ([]NodeMetricsSnapshot, error) {
	return impl.GetClusterMetricsLatest(ctx, clusterID)
}
func GetClusterMetricsRange(ctx context.Context, clusterID string, since, until time.Time) ([]NodeMetricsSnapshot, error) {
	return impl.GetClusterMetricsRange(ctx, clusterID, since, until)
}

// 资源列表（快照型，只取最新一次）
func GetPodListLatest(ctx context.Context, clusterID string) ([]Pod, error) {
	return impl.GetPodListLatest(ctx, clusterID)
}
func GetNodeListLatest(ctx context.Context, clusterID string) ([]Node, error) {
	return impl.GetNodeListLatest(ctx, clusterID)
}
func GetServiceListLatest(ctx context.Context, clusterID string) ([]Service, error) {
	return impl.GetServiceListLatest(ctx, clusterID)
}
func GetNamespaceListLatest(ctx context.Context, clusterID string) ([]Namespace, error) {
	return impl.GetNamespaceListLatest(ctx, clusterID)
}
func GetIngressListLatest(ctx context.Context, clusterID string) ([]Ingress, error) {
	return impl.GetIngressListLatest(ctx, clusterID)
}
func GetDeploymentListLatest(ctx context.Context, clusterID string) ([]Deployment, error) {
	return impl.GetDeploymentListLatest(ctx, clusterID)
}
func GetConfigMapListLatest(ctx context.Context, clusterID string) ([]ConfigMap, error) {
	return impl.GetConfigMapListLatest(ctx, clusterID)
}

// ListClusterIDs 返回去重后的 ClusterID 列表
func ListClusterIDs(ctx context.Context) ([]string, error) {
    return impl.ListClusterIDs(ctx)
}