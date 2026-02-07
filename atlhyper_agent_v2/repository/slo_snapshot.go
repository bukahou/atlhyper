// Package repository 数据访问层
//
// slo_snapshot.go - SLO Counter 快照管理
//
// 维护上一次采集的 Counter 累计值，用于计算增量。
// Counter 类型指标 (如 requests_total) 是累计值，
// 需要 delta = current - previous 才能得到时间段内的增量。
//
// 快照管理器还处理 Counter 重置检测:
// 如果 delta < 0，说明 Ingress Controller 重启导致 Counter 重置，
// 此时直接使用当前值作为增量。
package repository

import (
	"sort"
	"strings"
	"sync"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// CounterDelta Counter 增量数据
type CounterDelta struct {
	Host       string
	Status     string
	MetricType string
	Delta      int64 // 增量值 (current - previous)
}

// HistogramDelta Histogram 增量数据
type HistogramDelta struct {
	Host         string
	BucketDeltas map[string]int64 // le -> delta
	SumDelta     float64
	CountDelta   int64
}

// SLOSnapshotManager Counter 快照管理器
type SLOSnapshotManager struct {
	mu   sync.RWMutex
	prev map[string]int64   // counter key -> previous value
	hPrev map[string]int64  // histogram bucket key -> previous value
	sPrev map[string]float64 // histogram sum key -> previous value
	cPrev map[string]int64   // histogram count key -> previous value
}

// NewSLOSnapshotManager 创建快照管理器
func NewSLOSnapshotManager() *SLOSnapshotManager {
	return &SLOSnapshotManager{
		prev:  make(map[string]int64),
		hPrev: make(map[string]int64),
		sPrev: make(map[string]float64),
		cPrev: make(map[string]int64),
	}
}

// CalculateCounterDeltas 计算 Counter 增量
func (m *SLOSnapshotManager) CalculateCounterDeltas(counters []sdk.IngressCounterMetric) []CounterDelta {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deltas []CounterDelta
	newPrev := make(map[string]int64)

	for _, c := range counters {
		key := counterKey(c.Host, c.Status, c.MetricType)
		newPrev[key] = c.Value

		prevVal, hasPrev := m.prev[key]
		delta := c.Value
		if hasPrev {
			delta = c.Value - prevVal
			if delta < 0 {
				delta = c.Value // Counter 重置
			}
		}

		if delta > 0 {
			deltas = append(deltas, CounterDelta{
				Host:       c.Host,
				Status:     c.Status,
				MetricType: c.MetricType,
				Delta:      delta,
			})
		}
	}

	m.prev = newPrev
	return deltas
}

// CalculateHistogramDeltas 计算 Histogram 增量
func (m *SLOSnapshotManager) CalculateHistogramDeltas(histograms []sdk.IngressHistogramMetric) []HistogramDelta {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deltas []HistogramDelta
	newHPrev := make(map[string]int64)
	newSPrev := make(map[string]float64)
	newCPrev := make(map[string]int64)

	for _, h := range histograms {
		hd := HistogramDelta{
			Host:         h.Host,
			BucketDeltas: make(map[string]int64),
		}

		// Bucket 增量
		for le, count := range h.Buckets {
			bKey := bucketKey(h.Host, le)
			newHPrev[bKey] = count

			prevVal, hasPrev := m.hPrev[bKey]
			delta := count
			if hasPrev {
				delta = count - prevVal
				if delta < 0 {
					delta = count
				}
			}
			if delta > 0 {
				hd.BucketDeltas[le] = delta
			}
		}

		// Sum 增量
		sKey := h.Host + "|sum"
		newSPrev[sKey] = h.Sum
		prevSum, hasPrev := m.sPrev[sKey]
		hd.SumDelta = h.Sum
		if hasPrev {
			hd.SumDelta = h.Sum - prevSum
			if hd.SumDelta < 0 {
				hd.SumDelta = h.Sum
			}
		}

		// Count 增量
		cKey := h.Host + "|count"
		newCPrev[cKey] = h.Count
		prevCount, hasPrev := m.cPrev[cKey]
		hd.CountDelta = h.Count
		if hasPrev {
			hd.CountDelta = h.Count - prevCount
			if hd.CountDelta < 0 {
				hd.CountDelta = h.Count
			}
		}

		deltas = append(deltas, hd)
	}

	m.hPrev = newHPrev
	m.sPrev = newSPrev
	m.cPrev = newCPrev
	return deltas
}

// counterKey 生成 Counter 唯一键
func counterKey(host, status, metricType string) string {
	return host + "|" + status + "|" + metricType
}

// bucketKey 生成 Bucket 唯一键
func bucketKey(host, le string) string {
	return host + "|" + le
}

// sortedLabels 生成排序后的标签字符串（用于通用 key 生成）
func sortedLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ",")
}
