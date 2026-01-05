package overview

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/model/collect"
	"AtlHyper/model/k8s"
	"AtlHyper/model/transport"
)

// 简单封装 datasource 调用，方便单测替换

func fetchPods(ctx context.Context, clusterID string) ([]k8s.Pod, error) {
	return repository.GetPodListLatest(ctx, clusterID)
}

func fetchNodes(ctx context.Context, clusterID string) ([]k8s.Node, error) {
	return repository.GetNodeListLatest(ctx, clusterID)
}

func fetchEvents(ctx context.Context, clusterID string, limit int) ([]transport.LogEvent, error) {
	return repository.GetK8sEventsRecent(ctx, clusterID, limit)
}

func fetchMetricsRange(ctx context.Context, clusterID string, since, until time.Time) ([]collect.NodeMetricsSnapshot, error) {
	return repository.GetClusterMetricsRange(ctx, clusterID, since, until)
}
