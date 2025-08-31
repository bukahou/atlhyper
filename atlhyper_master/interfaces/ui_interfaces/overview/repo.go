package overview

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	event "AtlHyper/model/event"
	"AtlHyper/model/metrics"
	"AtlHyper/model/node"
	"AtlHyper/model/pod"
)

// 简单封装 datasource 调用，方便单测替换

func fetchPods(ctx context.Context, clusterID string) ([]pod.Pod, error) {
	return datasource.GetPodListLatest(ctx, clusterID)
}

func fetchNodes(ctx context.Context, clusterID string) ([]node.Node, error) {
	return datasource.GetNodeListLatest(ctx, clusterID)
}

func fetchEvents(ctx context.Context, clusterID string, limit int) ([]event.LogEvent, error) {
	return datasource.GetK8sEventsRecent(ctx, clusterID, limit)
}

func fetchMetricsRange(ctx context.Context, clusterID string, since, until time.Time) ([]metrics.NodeMetricsSnapshot, error) {
	return datasource.GetClusterMetricsRange(ctx, clusterID, since, until)
}
