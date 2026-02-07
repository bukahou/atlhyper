// Package slo SLO 数据仓库
//
// snapshot.go - Stage 2: Per-pod Delta 计算
//
// snapshotManager 维护每个 Pod 级别的上一次采集值 (prev)，
// 用于计算 counter 累积值的增量 (delta = current - prev)。
//
// 为什么必须 per-pod delta:
//
//	场景: 3 个 Pod，其中 Pod-C 重启
//	        Pod-A    Pod-B    Pod-C    聚合值
//	t=0     100      200      300      600
//	t=1     110      210      0(重启)  320
//
//	错误方式 (先聚合再 delta):
//	  delta = 320 - 600 = -280 → 误判为重置 → 报 320 (错!)
//
//	正确方式 (先 delta 再聚合):
//	  Pod-A: 110-100 = 10, Pod-B: 210-200 = 10
//	  Pod-C: 0 < 300, 重置, delta = 0 (跳过)
//	  聚合 = 20 (正确!)
//
// Agent 重启时 prev 为空，首次采集所有 Pod 的 delta 都为 0，
// 不会产生异常数据。第二次采集开始正常。
package slo

import (
	"sync"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// snapshotManager 维护 per-pod 级别的 prev 值
type snapshotManager struct {
	mu sync.Mutex

	// Linkerd response per-pod
	// key: "pod|status_code|classification|direction" → prev value
	linkerdResponsePrev map[string]float64

	// Linkerd latency per-pod
	// key: "pod|direction|le" → prev bucket value
	linkerdBucketPrev map[string]float64
	// key: "pod|direction" → prev sum
	linkerdSumPrev map[string]float64
	// key: "pod|direction" → prev count
	linkerdCountPrev map[string]float64

	// Ingress per-service-key
	// key: "service_key|code|method" → prev value
	ingressRequestPrev map[string]float64
	// key: "service_key|le" → prev bucket
	ingressBucketPrev map[string]float64
	// key: "service_key" → prev sum
	ingressSumPrev map[string]float64
	// key: "service_key" → prev count
	ingressCountPrev map[string]float64

	// Edge (outbound per-pod)
	// key: "pod|dst_ns|dst_name" → prev latency sum
	edgeSumPrev map[string]float64
	// key: "pod|dst_ns|dst_name" → prev latency count
	edgeCountPrev map[string]float64
}

func newSnapshotManager() *snapshotManager {
	return &snapshotManager{
		linkerdResponsePrev: make(map[string]float64),
		linkerdBucketPrev:   make(map[string]float64),
		linkerdSumPrev:      make(map[string]float64),
		linkerdCountPrev:    make(map[string]float64),
		ingressRequestPrev:  make(map[string]float64),
		ingressBucketPrev:   make(map[string]float64),
		ingressSumPrev:      make(map[string]float64),
		ingressCountPrev:    make(map[string]float64),
		edgeSumPrev:         make(map[string]float64),
		edgeCountPrev:       make(map[string]float64),
	}
}

// deltaResult 所有类型的增量结果
type deltaResult struct {
	// Linkerd inbound
	inboundResponses []linkerdResponseDelta
	inboundBuckets   []linkerdBucketDelta
	inboundSums      []linkerdSumDelta
	inboundCounts    []linkerdCountDelta

	// Linkerd outbound (for edges)
	outboundResponses []linkerdResponseDelta
	outboundSums      []linkerdSumDelta
	outboundCounts    []linkerdCountDelta

	// Ingress
	ingressRequests []ingressRequestDelta
	ingressBuckets  []ingressBucketDelta
	ingressSums     []ingressSumDelta
	ingressCounts   []ingressCountDelta
}

// calcDeltas 计算所有指标的 per-pod delta
func (sm *snapshotManager) calcDeltas(raw *sdk.OTelRawMetrics) *deltaResult {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	result := &deltaResult{}

	// ---- Linkerd responses ----
	newResponsePrev := make(map[string]float64, len(raw.LinkerdResponses))
	for _, m := range raw.LinkerdResponses {
		key := m.Pod + "|" + m.StatusCode + "|" + m.Classification + "|" + m.Direction
		newResponsePrev[key] = m.Value

		prev, hasPrev := sm.linkerdResponsePrev[key]
		if !hasPrev {
			continue // 首次采集，跳过
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdResponseDelta{
			Namespace:      m.Namespace,
			Deployment:     m.Deployment,
			Pod:            m.Pod,
			Direction:      m.Direction,
			StatusCode:     m.StatusCode,
			Classification: m.Classification,
			TLS:            m.TLS,
			DstNamespace:   m.DstNamespace,
			DstDeployment:  m.DstDeployment,
			Delta:          delta,
		}

		if m.Direction == "inbound" {
			result.inboundResponses = append(result.inboundResponses, d)
		} else if m.Direction == "outbound" {
			result.outboundResponses = append(result.outboundResponses, d)
		}
	}
	sm.linkerdResponsePrev = newResponsePrev

	// ---- Linkerd latency buckets ----
	newBucketPrev := make(map[string]float64, len(raw.LinkerdLatencyBuckets))
	for _, m := range raw.LinkerdLatencyBuckets {
		key := m.Pod + "|" + m.Direction + "|" + m.Le
		newBucketPrev[key] = m.Value

		prev, hasPrev := sm.linkerdBucketPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdBucketDelta{
			Namespace:  m.Namespace,
			Deployment: m.Deployment,
			Pod:        m.Pod,
			Direction:  m.Direction,
			Le:         m.Le,
			Delta:      delta,
		}
		if m.Direction == "inbound" {
			result.inboundBuckets = append(result.inboundBuckets, d)
		}
		// outbound buckets 不用于 Edge（Edge 只用 sum/count）
	}
	sm.linkerdBucketPrev = newBucketPrev

	// ---- Linkerd latency sums ----
	newSumPrev := make(map[string]float64, len(raw.LinkerdLatencySums))
	for _, m := range raw.LinkerdLatencySums {
		key := m.Pod + "|" + m.Direction
		newSumPrev[key] = m.Value

		prev, hasPrev := sm.linkerdSumPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdSumDelta{
			Namespace:  m.Namespace,
			Deployment: m.Deployment,
			Pod:        m.Pod,
			Direction:  m.Direction,
			Delta:      delta,
		}
		if m.Direction == "inbound" {
			result.inboundSums = append(result.inboundSums, d)
		} else if m.Direction == "outbound" {
			result.outboundSums = append(result.outboundSums, d)
		}
	}
	sm.linkerdSumPrev = newSumPrev

	// ---- Linkerd latency counts ----
	newCountPrev := make(map[string]float64, len(raw.LinkerdLatencyCounts))
	for _, m := range raw.LinkerdLatencyCounts {
		key := m.Pod + "|" + m.Direction
		newCountPrev[key] = m.Value

		prev, hasPrev := sm.linkerdCountPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdCountDelta{
			Namespace:  m.Namespace,
			Deployment: m.Deployment,
			Pod:        m.Pod,
			Direction:  m.Direction,
			Delta:      delta,
		}
		if m.Direction == "inbound" {
			result.inboundCounts = append(result.inboundCounts, d)
		} else if m.Direction == "outbound" {
			result.outboundCounts = append(result.outboundCounts, d)
		}
	}
	sm.linkerdCountPrev = newCountPrev

	// ---- Edge outbound latency (from sums/counts with dst info) ----
	// Note: outbound latency sums/counts use the same snapshot as above
	// but we need to track per-pod+dst for edge aggregation.
	// The outbound sums/counts are already separated above.

	// ---- Ingress requests ----
	newIngressReqPrev := make(map[string]float64, len(raw.IngressRequests))
	for _, m := range raw.IngressRequests {
		key := m.ServiceKey + "|" + m.Code + "|" + m.Method
		newIngressReqPrev[key] = m.Value

		prev, hasPrev := sm.ingressRequestPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressRequests = append(result.ingressRequests, ingressRequestDelta{
			ServiceKey: m.ServiceKey,
			Code:       m.Code,
			Method:     m.Method,
			Delta:      delta,
		})
	}
	sm.ingressRequestPrev = newIngressReqPrev

	// ---- Ingress latency buckets ----
	newIngressBucketPrev := make(map[string]float64, len(raw.IngressLatencyBuckets))
	for _, m := range raw.IngressLatencyBuckets {
		key := m.ServiceKey + "|" + m.Le
		newIngressBucketPrev[key] = m.Value

		prev, hasPrev := sm.ingressBucketPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressBuckets = append(result.ingressBuckets, ingressBucketDelta{
			ServiceKey: m.ServiceKey,
			Le:         m.Le,
			Delta:      delta,
		})
	}
	sm.ingressBucketPrev = newIngressBucketPrev

	// ---- Ingress latency sums ----
	newIngressSumPrev := make(map[string]float64, len(raw.IngressLatencySums))
	for _, m := range raw.IngressLatencySums {
		key := m.ServiceKey
		newIngressSumPrev[key] = m.Value

		prev, hasPrev := sm.ingressSumPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressSums = append(result.ingressSums, ingressSumDelta{
			ServiceKey: m.ServiceKey,
			Delta:      delta,
		})
	}
	sm.ingressSumPrev = newIngressSumPrev

	// ---- Ingress latency counts ----
	newIngressCountPrev := make(map[string]float64, len(raw.IngressLatencyCounts))
	for _, m := range raw.IngressLatencyCounts {
		key := m.ServiceKey
		newIngressCountPrev[key] = m.Value

		prev, hasPrev := sm.ingressCountPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(m.Value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressCounts = append(result.ingressCounts, ingressCountDelta{
			ServiceKey: m.ServiceKey,
			Delta:      delta,
		})
	}
	sm.ingressCountPrev = newIngressCountPrev

	return result
}

// calcDelta 通用增量计算
// delta = current - prev; if delta < 0 → counter 重置, delta = 0 (跳过本周期)
func calcDelta(current, prev float64) float64 {
	delta := current - prev
	if delta < 0 {
		return 0
	}
	return delta
}
