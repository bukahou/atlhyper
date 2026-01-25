// repository/mem/reader.go
// 内存仓库读取实现
package mem

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/atlhyper_master/store/memory"
	"AtlHyper/model/transport"
)

// HubReader 基于内存 Hub 的 Reader 实现
type HubReader struct{}

// hubFilter 按 clusterID+source 过滤并按时间排序
func hubFilter(clusterID, source string) []memory.EnvelopeRecord {
	all := memory.Snapshot()
	out := make([]memory.EnvelopeRecord, 0, 128)

	for _, r := range all {
		if r.ClusterID == clusterID && r.Source == source {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EnqueuedAt.Before(out[j].EnqueuedAt) })
	return out
}

// latestPayload 返回某个 Source 的最新 Payload
func latestPayload(clusterID, source string) (json.RawMessage, bool) {
	recs := hubFilter(clusterID, source)
	if len(recs) == 0 {
		return nil, false
	}
	return recs[len(recs)-1].Payload, true
}

// ============================================================
// 事件
// ============================================================

func (HubReader) GetK8sEventsRecent(ctx context.Context, clusterID string, limit int) ([]repository.LogEvent, error) {
	recs := hubFilter(clusterID, transport.SourceK8sEvent)
	rows := make([]repository.LogEvent, 0, limit)

	for i := len(recs) - 1; i >= 0 && len(rows) < limit; i-- {
		evs, err := decodeEvents(recs[i].Payload)
		if err != nil {
			continue
		}
		for j := len(evs) - 1; j >= 0 && len(rows) < limit; j-- {
			rows = append(rows, evs[j])
		}
	}
	return rows, nil
}

// ============================================================
// 指标
// ============================================================

func (HubReader) GetClusterMetricsLatest(ctx context.Context, clusterID string) ([]repository.NodeMetricsSnapshot, error) {
	recs := hubFilter(clusterID, transport.SourceMetricsSnapshot)
	if len(recs) == 0 {
		return nil, nil
	}

	latest := recs[len(recs)-1]
	arr, err := decodeMetricsBatch(latest.Payload)
	if err != nil {
		return nil, err
	}

	if !latest.EnqueuedAt.IsZero() {
		for i := range arr {
			if arr[i].Timestamp.IsZero() {
				arr[i].Timestamp = latest.EnqueuedAt
			}
		}
	}
	return arr, nil
}

func (HubReader) GetClusterMetricsRange(ctx context.Context, clusterID string, since, until time.Time) ([]repository.NodeMetricsSnapshot, error) {
	recs := hubFilter(clusterID, transport.SourceMetricsSnapshot)
	out := make([]repository.NodeMetricsSnapshot, 0, 256)

	for _, r := range recs {
		if r.EnqueuedAt.Before(since) || !r.EnqueuedAt.Before(until) {
			continue
		}
		batch, err := decodeMetricsBatch(r.Payload)
		if err != nil {
			continue
		}
		for i := range batch {
			if batch[i].Timestamp.IsZero() {
				batch[i].Timestamp = r.EnqueuedAt
			}
		}
		out = append(out, batch...)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out, nil
}

// ============================================================
// 资源列表
// ============================================================

func (HubReader) GetPodListLatest(ctx context.Context, clusterID string) ([]repository.Pod, error) {
	raw, ok := latestPayload(clusterID, transport.SourcePodListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodePodList(raw)
}

func (HubReader) GetNodeListLatest(ctx context.Context, clusterID string) ([]repository.Node, error) {
	raw, ok := latestPayload(clusterID, transport.SourceNodeListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeNodeList(raw)
}

func (HubReader) GetServiceListLatest(ctx context.Context, clusterID string) ([]repository.Service, error) {
	raw, ok := latestPayload(clusterID, transport.SourceServiceListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeServiceList(raw)
}

func (HubReader) GetNamespaceListLatest(ctx context.Context, clusterID string) ([]repository.Namespace, error) {
	raw, ok := latestPayload(clusterID, transport.SourceNamespaceListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeNamespaceList(raw)
}

func (HubReader) GetIngressListLatest(ctx context.Context, clusterID string) ([]repository.Ingress, error) {
	raw, ok := latestPayload(clusterID, transport.SourceIngressListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeIngressList(raw)
}

func (HubReader) GetDeploymentListLatest(ctx context.Context, clusterID string) ([]repository.Deployment, error) {
	raw, ok := latestPayload(clusterID, transport.SourceDeploymentListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeDeploymentList(raw)
}

func (HubReader) GetConfigMapListLatest(ctx context.Context, clusterID string) ([]repository.ConfigMap, error) {
	raw, ok := latestPayload(clusterID, transport.SourceConfigMapListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeConfigMapList(raw)
}

// ============================================================
// 集群列表
// ============================================================

func (HubReader) ListClusterIDs(ctx context.Context) ([]string, error) {
	snap := memory.Snapshot()
	if len(snap) == 0 {
		return nil, nil
	}

	uniq := make(map[string]struct{}, 32)
	for _, r := range snap {
		if r.ClusterID != "" {
			uniq[r.ClusterID] = struct{}{}
		}
	}

	out := make([]string, 0, len(uniq))
	for cid := range uniq {
		out = append(out, cid)
	}
	sort.Strings(out)
	return out, nil
}
