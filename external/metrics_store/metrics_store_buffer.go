// metrics_store_buffer.go
package metrics_store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	model "NeuroController/model/metrics"
	"NeuroController/sync/center/http/master_metrics"
)

var (
	// 每个节点：时间升序的快照列表
	memBuf   = make(map[string][]*model.NodeMetricsSnapshot)
	memMu    sync.Mutex
	// 内存窗口，默认 30m，可用 METRICS_MEM_RETENTION 覆盖（如 "10m", "1h"）
	memRetention = parseIntervalFromEnv("METRICS_MEM_RETENTION", 30*time.Minute)
	// 可选：每节点最多保留多少条（0 表示不限制），用 METRICS_MEM_PER_NODE_CAP 配置
	perNodeCap = parsePerNodeCapFromEnv("METRICS_MEM_PER_NODE_CAP", 0)
)

// 供 StartMetricsSync 调用：拉取 → 解析 → 写入内存（不落盘）
func saveLatestSnapshotsOnce() error {
	raw, err := master_metrics.GetLatestNodeMetrics() // json.RawMessage
	if err != nil {
		return err
	}

	// 1) 先尝试数组形状：map[node][]*snapshot
	var asArray map[string][]*model.NodeMetricsSnapshot
	if err := json.Unmarshal(raw, &asArray); err == nil && len(asArray) > 0 {
		memAppendAndTrim(asArray)
		log.Printf("[MetricsSync] mem append OK (array), nodes=%d", len(asArray))
		return nil
	}

	// 2) 再尝试对象形状：map[node]*snapshot → 包成切片
	var asObject map[string]*model.NodeMetricsSnapshot
	if err := json.Unmarshal(raw, &asObject); err == nil && len(asObject) > 0 {
		arr := make(map[string][]*model.NodeMetricsSnapshot, len(asObject))
		for node, snap := range asObject {
			if snap != nil {
				arr[node] = []*model.NodeMetricsSnapshot{snap}
			}
		}
		memAppendAndTrim(arr)
		log.Printf("[MetricsSync] mem append OK (object), nodes=%d", len(asObject))
		return nil
	}

	return fmt.Errorf("decode /agent/dataapi/latest failed, body=%s",
		bytes.ReplaceAll(raw, []byte("\n"), []byte{}))
}

// —— 内存写入与裁剪 —— //

func memAppendAndTrim(in map[string][]*model.NodeMetricsSnapshot) {
	now := time.Now().UTC()
	cutoff := now.Add(-memRetention)

	memMu.Lock()
	defer memMu.Unlock()

	for node, snaps := range in {
		if len(snaps) == 0 {
			continue
		}
		// 只追加窗口内的数据（也可先追加再统一裁剪）
		for _, s := range snaps {
			if t, ok := tsAsTime(s.Timestamp); ok && (t.After(cutoff) || t.Equal(cutoff)) {
				memBuf[node] = append(memBuf[node], s)
			}
		}
		// 时间窗口裁剪（保留 >= cutoff）
		if len(memBuf[node]) > 0 {
			memBuf[node] = trimByCutoff(memBuf[node], cutoff)
		}
		// 条数上限（保留最新 N 条）
		if perNodeCap > 0 && len(memBuf[node]) > perNodeCap {
			memBuf[node] = memBuf[node][len(memBuf[node])-perNodeCap:]
		}
	}
}

// 将 Timestamp（time.Time 或 string）转为 UTC time.Time
func tsAsTime(ts any) (time.Time, bool) {
	switch v := ts.(type) {
	case time.Time:
		return v.UTC(), true
	case *time.Time:
		if v == nil {
			return time.Time{}, false
		}
		return v.UTC(), true
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t.UTC(), true
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC(), true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

// 在切片中找到第一个 >= cutoff 的下标并截断
func trimByCutoff(snaps []*model.NodeMetricsSnapshot, cutoff time.Time) []*model.NodeMetricsSnapshot {
	if len(snaps) == 0 {
		return snaps
	}
	// 如果最后一条都早于 cutoff，直接清空
	if t, ok := tsAsTime(snaps[len(snaps)-1].Timestamp); ok && t.Before(cutoff) {
		return nil
	}
	// 线性回扫（数据量不大时足够；如需更快可改成二分）
	idx := 0
	for i := len(snaps) - 1; i >= 0; i-- {
		t, ok := tsAsTime(snaps[i].Timestamp)
		if !ok {
			continue
		}
		if t.Before(cutoff) {
			idx = i + 1
			break
		}
	}
	if idx <= 0 {
		return snaps
	}
	if idx >= len(snaps) {
		return nil
	}
	return snaps[idx:]
}

// —— 配置解析 —— //

func parsePerNodeCapFromEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
		log.Printf("⚠️ METRICS: invalid %s=%q, fallback to %d", key, v, def)
	}
	return def
}
