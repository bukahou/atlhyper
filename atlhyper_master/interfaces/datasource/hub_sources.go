// interfaces/datasource/hub_sources.go
package datasource

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"AtlHyper/atlhyper_master/master_store"
	m "AtlHyper/model" // 含 Source 常量定义
)

// HubSources：统一数据源（Reader）的 Hub 实现
// -----------------------------------------------------------------------------
// - 无状态：零值即可使用，不持有连接，不做缓存。
// - 读取：所有数据均来自 master_store.Snapshot() 返回的副本。
// - 责任：只做“按 ClusterID+Source 过滤 → JSON 解码 → 原样返回模型”。
//   * 不做 UI/页面定制的裁剪或聚合（这些应在上层 handler/service 完成）。
type HubSources struct{}


// interfaces/datasource/hub_sources.go

func (HubSources) ListClusterIDs(ctx context.Context) ([]string, error) {
    snap := master_store.Snapshot()
    if len(snap) == 0 {
        return nil, nil
    }

    uniq := make(map[string]struct{}, 32)
    for _, r := range snap {
        cid := r.ClusterID
        if cid == "" {
            continue
        }
        uniq[cid] = struct{}{}
    }

    out := make([]string, 0, len(uniq))
    for cid := range uniq {
        out = append(out, cid)
    }
    sort.Strings(out)
    return out, nil
}




// hubFilter：按 clusterID+source 从 Hub 快照中过滤记录并按时间排序（升序）
// -----------------------------------------------------------------------------
// Snapshot() 返回只读副本；此处排序与筛选不会影响底层。
func hubFilter(clusterID, source string) []master_store.EnvelopeRecord {
	all := master_store.Snapshot()

	// 提示：为避免共享全局切片导致并发问题，生产中建议改成函数内局部切片：
	// out := make([]master_store.EnvelopeRecord, 0, 128)
	out := outPool[:0] // 轻量优化，可删除换成上面的局部切片

	for _, r := range all {
		if r.ClusterID == clusterID && r.Source == source {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EnqueuedAt.Before(out[j].EnqueuedAt) })
	return out
}

// 轻量优化的可复用缓冲；如有高并发，请改为函数内局部变量，避免共享可变状态。
var outPool = make([]master_store.EnvelopeRecord, 0, 128)

// ================= 事件 =================

// GetK8sEventsRecent：获取最近的 N 条事件（按入池时间倒序）
// -----------------------------------------------------------------------------
// 返回：底层模型 base.LogEvent（不裁剪，不聚合）。
// 说明：一条 EnvelopeRecord 的 Payload 中可能包含多条事件，因此需要逐条解码；
//      这里倒序遍历记录，再倒序遍历记录内事件，从最新开始收集到 limit 条为止。
func (HubSources) GetK8sEventsRecent(ctx context.Context, clusterID string, limit int) ([]LogEvent, error) {
	recs := hubFilter(clusterID, m.SourceK8sEvent)
	rows := make([]LogEvent, 0, limit)

	for i := len(recs) - 1; i >= 0 && len(rows) < limit; i-- { // 倒序：最新的记录在末尾
		evs, err := decodeEvents(recs[i].Payload) // hub_decode.go：[]LogEvent
		if err != nil {
			continue // 解码失败跳过该条记录
		}
		for j := len(evs) - 1; j >= 0 && len(rows) < limit; j-- { // 记录内也按倒序取
			rows = append(rows, evs[j])
		}
	}
	return rows, nil
}

// ================= 指标（metrics_snapshot） =================

// GetClusterMetricsLatest：获取“最新一次上报”的全量节点指标快照（所有节点）
// -----------------------------------------------------------------------------
// 返回：[]NodeMetricsSnapshot（底层完整模型；每个元素是一个节点的快照）
// 说明：如果某些快照的 Timestamp 为空，用该批次的 EnqueuedAt 兜底填充。
func (HubSources) GetClusterMetricsLatest(ctx context.Context, clusterID string) ([]NodeMetricsSnapshot, error) {
	recs := hubFilter(clusterID, m.SourceMetricsSnapshot)
	if len(recs) == 0 {
		return nil, nil
	}

	latest := recs[len(recs)-1]
	arr, err := decodeMetricsBatch(latest.Payload) // []NodeMetricsSnapshot
	if err != nil {
		return nil, err
	}
	// 兜底时间
	if !latest.EnqueuedAt.IsZero() {
		for i := range arr {
			if arr[i].Timestamp.IsZero() {
				arr[i].Timestamp = latest.EnqueuedAt
			}
		}
	}
	return arr, nil
}

// GetClusterMetricsRange：获取区间内的“全量节点指标快照”扁平列表
// -----------------------------------------------------------------------------
// 过滤逻辑：先用 EnvelopeRecord.EnqueuedAt 过滤批次（[since, until)），
// 再把该批次内的所有节点快照都放进结果；若快照无时间则用该批次 EnqueuedAt。
// 返回：[]NodeMetricsSnapshot（扁平列表；包含多个时间点×多个节点；按时间升序）
func (HubSources) GetClusterMetricsRange(ctx context.Context, clusterID string, since, until time.Time) ([]NodeMetricsSnapshot, error) {
	recs := hubFilter(clusterID, m.SourceMetricsSnapshot)
	out := make([]NodeMetricsSnapshot, 0, 256)

	for _, r := range recs {
		if r.EnqueuedAt.Before(since) || !r.EnqueuedAt.Before(until) {
			continue
		}
		batch, err := decodeMetricsBatch(r.Payload) 
		if err != nil {
			continue
		}
		// 兜底时间
		for i := range batch {
			if batch[i].Timestamp.IsZero() {
				batch[i].Timestamp = r.EnqueuedAt
			}
		}
		out = append(out, batch...)
	}

	// 按时间升序
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	return out, nil
}

// ================= 各类 *ListSnapshot（只取最新一次） =================

// latestPayload：返回某个 Source 的最新一条 Payload
func latestPayload(clusterID, source string) (json.RawMessage, bool) {
	recs := hubFilter(clusterID, source)
	if len(recs) == 0 {
		return nil, false
	}
	return recs[len(recs)-1].Payload, true
}

// 下列函数均解码“最新一次快照”，直接返回底层模型类型（通用，不裁剪）

func (HubSources) GetPodListLatest(ctx context.Context, clusterID string) ([]Pod, error) {
	raw, ok := latestPayload(clusterID, m.SourcePodListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodePodList(raw)
}

func (HubSources) GetNodeListLatest(ctx context.Context, clusterID string) ([]Node, error) {
	raw, ok := latestPayload(clusterID, m.SourceNodeListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeNodeList(raw)
}

func (HubSources) GetServiceListLatest(ctx context.Context, clusterID string) ([]Service, error) {
	raw, ok := latestPayload(clusterID, m.SourceServiceListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeServiceList(raw)
}

func (HubSources) GetNamespaceListLatest(ctx context.Context, clusterID string) ([]Namespace, error) {
	raw, ok := latestPayload(clusterID, m.SourceNamespaceListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeNamespaceList(raw)
}

func (HubSources) GetIngressListLatest(ctx context.Context, clusterID string) ([]Ingress, error) {
	raw, ok := latestPayload(clusterID, m.SourceIngressListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeIngressList(raw)
}

func (HubSources) GetDeploymentListLatest(ctx context.Context, clusterID string) ([]Deployment, error) {
	raw, ok := latestPayload(clusterID, m.SourceDeploymentListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeDeploymentList(raw)
}

func (HubSources) GetConfigMapListLatest(ctx context.Context, clusterID string) ([]ConfigMap, error) {
	raw, ok := latestPayload(clusterID, m.SourceConfigMapListSnapshot)
	if !ok {
		return nil, nil
	}
	return decodeConfigMapList(raw)
}
