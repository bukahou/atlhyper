// external/interfaces/metrics/metrics_snapshot.go
package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sort"

	"NeuroController/external/master_store"
	"NeuroController/model"
	"NeuroController/model/metrics"
)

const sourceMetrics = model.SourceMetricsSnapshot

// 与 Agent 上报保持一致：payload 是 {"snapshots":[...]}；也容忍直接传数组 [...]
type metricsPayload struct {
	Snapshots []metrics.NodeMetricsSnapshot `json:"snapshots"`
}

func decodeMetricsPayload(raw []byte) ([]metrics.NodeMetricsSnapshot, error) {
	// 1) 对象形式
	var obj metricsPayload
	if err := json.Unmarshal(raw, &obj); err == nil && len(obj.Snapshots) > 0 {
		return obj.Snapshots, nil
	}
	// 2) 直接数组
	var arr []metrics.NodeMetricsSnapshot
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	return nil, errors.New("payload 中未找到 snapshots")
}

// GetLatestNodeMetricsByCluster
// 返回：指定集群内“每个节点的最新一条快照”。
// 不接收时间参数；使用 master_store 的保留窗口（默认约 15 分钟）。
func GetLatestNodeMetricsByCluster(_ context.Context, clusterID string) (map[string]metrics.NodeMetricsSnapshot, error) {
	out := make(map[string]metrics.NodeMetricsSnapshot, 64)
	if clusterID == "" {
		return out, nil
	}

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceMetrics || r.ClusterID != clusterID {
			continue
		}
		snaps, err := decodeMetricsPayload(r.Payload)
		if err != nil {
			log.Printf("[metrics_iface] decode payload fail: cluster=%s err=%v", r.ClusterID, err)
			continue
		}
		for _, s := range snaps {
			if s.NodeName == "" {
				continue
			}
			prev, ok := out[s.NodeName]
			if !ok || s.Timestamp.After(prev.Timestamp) {
				out[s.NodeName] = s
			}
		}
	}
	return out, nil
}

// GetAllNodeMetricsByCluster
// 返回：指定集群内“每个节点在内存窗口内的全部快照（去重 + 时间升序）”。
// 去重策略：同一节点同一时间戳只保留一条（后写覆盖前写）。
func GetAllNodeMetricsByCluster(_ context.Context, clusterID string) (map[string][]metrics.NodeMetricsSnapshot, error) {
	out := make(map[string][]metrics.NodeMetricsSnapshot, 64)
	if clusterID == "" {
		return out, nil
	}

	// 先聚到 map[node]map[timestamp]snapshot 做去重，然后再落到有序切片
	tmp := make(map[string]map[int64]metrics.NodeMetricsSnapshot, 64)

	recs := master_store.Snapshot()
	for _, r := range recs {
		if r.Source != sourceMetrics || r.ClusterID != clusterID {
			continue
		}
		snaps, err := decodeMetricsPayload(r.Payload)
		if err != nil {
			log.Printf("[metrics_iface] decode payload fail: cluster=%s err=%v", r.ClusterID, err)
			continue
		}
		for _, s := range snaps {
			if s.NodeName == "" || s.Timestamp.IsZero() {
				continue
			}
			if _, ok := tmp[s.NodeName]; !ok {
				tmp[s.NodeName] = make(map[int64]metrics.NodeMetricsSnapshot, 64)
			}
			ts := s.Timestamp.UnixNano() // 用纳秒做 key，避免同秒冲突
			tmp[s.NodeName][ts] = s
		}
	}

	// 展平为按时间升序的切片
	for node, bucket := range tmp {
		list := make([]metrics.NodeMetricsSnapshot, 0, len(bucket))
		keys := make([]int64, 0, len(bucket))
		for k := range bucket {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, k := range keys {
			list = append(list, bucket[k])
		}
		out[node] = list
	}
	return out, nil
}
