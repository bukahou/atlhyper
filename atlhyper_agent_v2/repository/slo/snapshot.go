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

// responseAggKey Linkerd response 的聚合元数据
type responseAggKey struct {
	Namespace      string
	Deployment     string
	Pod            string
	Direction      string
	StatusCode     string
	Classification string
	TLS            string
	DstNamespace   string
	DstDeployment  string
}

// latencyAggKey Linkerd latency 的聚合元数据
type latencyAggKey struct {
	Namespace  string
	Deployment string
	Pod        string
	Direction  string
	Le         string // bucket only
}

// calcDeltas 计算所有指标的 per-pod delta
//
// OTel Collector 输出的同一个 pod+status_code+classification+direction 组合
// 可能有多条指标行（因 client_id、srv_port、target_port 等标签不同）。
// 这些行代表不同来源的请求，counter 值需要先累加到同一个 delta key，
// 然后再与上一次的聚合值做 delta。
func (sm *snapshotManager) calcDeltas(raw *sdk.OTelRawMetrics) *deltaResult {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	result := &deltaResult{}

	// ---- Linkerd responses ----
	// Step 1: 按 key 聚合当前值
	type responseAgg struct {
		value float64
		meta  responseAggKey
	}
	responseAggs := make(map[string]*responseAgg)
	for _, m := range raw.LinkerdResponses {
		key := m.Pod + "|" + m.StatusCode + "|" + m.Classification + "|" + m.Direction
		agg, ok := responseAggs[key]
		if !ok {
			responseAggs[key] = &responseAgg{
				value: m.Value,
				meta: responseAggKey{
					Namespace: m.Namespace, Deployment: m.Deployment,
					Pod: m.Pod, Direction: m.Direction,
					StatusCode: m.StatusCode, Classification: m.Classification,
					TLS: m.TLS, DstNamespace: m.DstNamespace, DstDeployment: m.DstDeployment,
				},
			}
		} else {
			agg.value += m.Value
			// 保留有 dst 信息的元数据（outbound 用）
			if m.DstNamespace != "" && agg.meta.DstNamespace == "" {
				agg.meta.DstNamespace = m.DstNamespace
				agg.meta.DstDeployment = m.DstDeployment
			}
		}
	}

	// Step 2: 对聚合后的值做 delta
	newResponsePrev := make(map[string]float64, len(responseAggs))
	for key, agg := range responseAggs {
		newResponsePrev[key] = agg.value

		prev, hasPrev := sm.linkerdResponsePrev[key]
		if !hasPrev {
			continue // 首次采集，跳过
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdResponseDelta{
			Namespace:      agg.meta.Namespace,
			Deployment:     agg.meta.Deployment,
			Pod:            agg.meta.Pod,
			Direction:      agg.meta.Direction,
			StatusCode:     agg.meta.StatusCode,
			Classification: agg.meta.Classification,
			TLS:            agg.meta.TLS,
			DstNamespace:   agg.meta.DstNamespace,
			DstDeployment:  agg.meta.DstDeployment,
			Delta:          delta,
		}

		if agg.meta.Direction == "inbound" {
			result.inboundResponses = append(result.inboundResponses, d)
		} else if agg.meta.Direction == "outbound" {
			result.outboundResponses = append(result.outboundResponses, d)
		}
	}
	sm.linkerdResponsePrev = newResponsePrev

	// ---- Linkerd latency buckets ----
	type bucketAgg struct {
		value float64
		meta  latencyAggKey
	}
	bucketAggs := make(map[string]*bucketAgg)
	for _, m := range raw.LinkerdLatencyBuckets {
		key := m.Pod + "|" + m.Direction + "|" + m.Le
		agg, ok := bucketAggs[key]
		if !ok {
			bucketAggs[key] = &bucketAgg{
				value: m.Value,
				meta:  latencyAggKey{Namespace: m.Namespace, Deployment: m.Deployment, Pod: m.Pod, Direction: m.Direction, Le: m.Le},
			}
		} else {
			agg.value += m.Value
		}
	}

	newBucketPrev := make(map[string]float64, len(bucketAggs))
	for key, agg := range bucketAggs {
		newBucketPrev[key] = agg.value

		prev, hasPrev := sm.linkerdBucketPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdBucketDelta{
			Namespace:  agg.meta.Namespace,
			Deployment: agg.meta.Deployment,
			Pod:        agg.meta.Pod,
			Direction:  agg.meta.Direction,
			Le:         agg.meta.Le,
			Delta:      delta,
		}
		if agg.meta.Direction == "inbound" {
			result.inboundBuckets = append(result.inboundBuckets, d)
		}
		// outbound buckets 不用于 Edge（Edge 只用 sum/count）
	}
	sm.linkerdBucketPrev = newBucketPrev

	// ---- Linkerd latency sums ----
	type sumAgg struct {
		value float64
		meta  latencyAggKey
	}
	sumAggs := make(map[string]*sumAgg)
	for _, m := range raw.LinkerdLatencySums {
		key := m.Pod + "|" + m.Direction
		agg, ok := sumAggs[key]
		if !ok {
			sumAggs[key] = &sumAgg{
				value: m.Value,
				meta:  latencyAggKey{Namespace: m.Namespace, Deployment: m.Deployment, Pod: m.Pod, Direction: m.Direction},
			}
		} else {
			agg.value += m.Value
		}
	}

	newSumPrev := make(map[string]float64, len(sumAggs))
	for key, agg := range sumAggs {
		newSumPrev[key] = agg.value

		prev, hasPrev := sm.linkerdSumPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdSumDelta{
			Namespace:  agg.meta.Namespace,
			Deployment: agg.meta.Deployment,
			Pod:        agg.meta.Pod,
			Direction:  agg.meta.Direction,
			Delta:      delta,
		}
		if agg.meta.Direction == "inbound" {
			result.inboundSums = append(result.inboundSums, d)
		} else if agg.meta.Direction == "outbound" {
			result.outboundSums = append(result.outboundSums, d)
		}
	}
	sm.linkerdSumPrev = newSumPrev

	// ---- Linkerd latency counts ----
	type countAgg struct {
		value float64
		meta  latencyAggKey
	}
	countAggs := make(map[string]*countAgg)
	for _, m := range raw.LinkerdLatencyCounts {
		key := m.Pod + "|" + m.Direction
		agg, ok := countAggs[key]
		if !ok {
			countAggs[key] = &countAgg{
				value: m.Value,
				meta:  latencyAggKey{Namespace: m.Namespace, Deployment: m.Deployment, Pod: m.Pod, Direction: m.Direction},
			}
		} else {
			agg.value += m.Value
		}
	}

	newCountPrev := make(map[string]float64, len(countAggs))
	for key, agg := range countAggs {
		newCountPrev[key] = agg.value

		prev, hasPrev := sm.linkerdCountPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		d := linkerdCountDelta{
			Namespace:  agg.meta.Namespace,
			Deployment: agg.meta.Deployment,
			Pod:        agg.meta.Pod,
			Direction:  agg.meta.Direction,
			Delta:      delta,
		}
		if agg.meta.Direction == "inbound" {
			result.inboundCounts = append(result.inboundCounts, d)
		} else if agg.meta.Direction == "outbound" {
			result.outboundCounts = append(result.outboundCounts, d)
		}
	}
	sm.linkerdCountPrev = newCountPrev

	// ---- Ingress requests ----
	// Ingress 指标通常不会有同 key 多行问题（Traefik/Nginx 按 service+code+method 唯一），
	// 但为安全起见也做先聚合再 delta
	type ingressReqAgg struct {
		value      float64
		serviceKey string
		code       string
		method     string
	}
	ingressReqAggs := make(map[string]*ingressReqAgg)
	for _, m := range raw.IngressRequests {
		key := m.ServiceKey + "|" + m.Code + "|" + m.Method
		agg, ok := ingressReqAggs[key]
		if !ok {
			ingressReqAggs[key] = &ingressReqAgg{value: m.Value, serviceKey: m.ServiceKey, code: m.Code, method: m.Method}
		} else {
			agg.value += m.Value
		}
	}

	newIngressReqPrev := make(map[string]float64, len(ingressReqAggs))
	for key, agg := range ingressReqAggs {
		newIngressReqPrev[key] = agg.value

		prev, hasPrev := sm.ingressRequestPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressRequests = append(result.ingressRequests, ingressRequestDelta{
			ServiceKey: agg.serviceKey,
			Code:       agg.code,
			Method:     agg.method,
			Delta:      delta,
		})
	}
	sm.ingressRequestPrev = newIngressReqPrev

	// ---- Ingress latency buckets ----
	type ingressBucketAgg struct {
		value      float64
		serviceKey string
		le         string
	}
	ingressBucketAggs := make(map[string]*ingressBucketAgg)
	for _, m := range raw.IngressLatencyBuckets {
		key := m.ServiceKey + "|" + m.Le
		agg, ok := ingressBucketAggs[key]
		if !ok {
			ingressBucketAggs[key] = &ingressBucketAgg{value: m.Value, serviceKey: m.ServiceKey, le: m.Le}
		} else {
			agg.value += m.Value
		}
	}

	newIngressBucketPrev := make(map[string]float64, len(ingressBucketAggs))
	for key, agg := range ingressBucketAggs {
		newIngressBucketPrev[key] = agg.value

		prev, hasPrev := sm.ingressBucketPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressBuckets = append(result.ingressBuckets, ingressBucketDelta{
			ServiceKey: agg.serviceKey,
			Le:         agg.le,
			Delta:      delta,
		})
	}
	sm.ingressBucketPrev = newIngressBucketPrev

	// ---- Ingress latency sums ----
	type ingressSumAgg struct {
		value      float64
		serviceKey string
	}
	ingressSumAggs := make(map[string]*ingressSumAgg)
	for _, m := range raw.IngressLatencySums {
		key := m.ServiceKey
		agg, ok := ingressSumAggs[key]
		if !ok {
			ingressSumAggs[key] = &ingressSumAgg{value: m.Value, serviceKey: m.ServiceKey}
		} else {
			agg.value += m.Value
		}
	}

	newIngressSumPrev := make(map[string]float64, len(ingressSumAggs))
	for key, agg := range ingressSumAggs {
		newIngressSumPrev[key] = agg.value

		prev, hasPrev := sm.ingressSumPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressSums = append(result.ingressSums, ingressSumDelta{
			ServiceKey: agg.serviceKey,
			Delta:      delta,
		})
	}
	sm.ingressSumPrev = newIngressSumPrev

	// ---- Ingress latency counts ----
	type ingressCountAgg struct {
		value      float64
		serviceKey string
	}
	ingressCountAggs := make(map[string]*ingressCountAgg)
	for _, m := range raw.IngressLatencyCounts {
		key := m.ServiceKey
		agg, ok := ingressCountAggs[key]
		if !ok {
			ingressCountAggs[key] = &ingressCountAgg{value: m.Value, serviceKey: m.ServiceKey}
		} else {
			agg.value += m.Value
		}
	}

	newIngressCountPrev := make(map[string]float64, len(ingressCountAggs))
	for key, agg := range ingressCountAggs {
		newIngressCountPrev[key] = agg.value

		prev, hasPrev := sm.ingressCountPrev[key]
		if !hasPrev {
			continue
		}
		delta := calcDelta(agg.value, prev)
		if delta <= 0 {
			continue
		}

		result.ingressCounts = append(result.ingressCounts, ingressCountDelta{
			ServiceKey: agg.serviceKey,
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
