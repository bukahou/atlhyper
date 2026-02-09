// Package slo SLO 数据仓库
//
// aggregate.go - Stage 3: 聚合
//
// 将 per-pod delta 聚合为 service/edge/ingress 级别的指标:
//   - 3a: aggregateServices — Linkerd inbound → ServiceMetrics[]
//   - 3b: extractEdges — Linkerd outbound → ServiceEdge[]
//   - 3c: aggregateIngress — Ingress delta → IngressMetrics[] (秒→毫秒)
package slo

import (
	"fmt"
	"strconv"

	"AtlHyper/model_v2"
)

// =============================================================================
// Stage 3a: Aggregate to Service (Linkerd inbound)
// =============================================================================

// serviceKey namespace + deployment
type serviceKey struct {
	Namespace  string
	Deployment string
}

func (k serviceKey) String() string {
	return k.Namespace + "/" + k.Deployment
}

// serviceAccumulator 单个 service 的聚合中间状态
type serviceAccumulator struct {
	// 请求计数: key = "status_code|classification"
	requests map[string]int64

	// 延迟直方图: key = le (ms string)
	latencyBuckets map[string]int64
	latencySum     float64
	latencyCount   int64

	// mTLS 聚合
	tlsRequestDelta   int64
	totalRequestDelta int64
}

// aggregateServices 将 inbound per-pod delta 聚合为 service 级别
func aggregateServices(
	responses []linkerdResponseDelta,
	buckets []linkerdBucketDelta,
	sums []linkerdSumDelta,
	counts []linkerdCountDelta,
) []model_v2.ServiceMetrics {
	accs := make(map[serviceKey]*serviceAccumulator)

	getAcc := func(ns, deploy string) *serviceAccumulator {
		key := serviceKey{Namespace: ns, Deployment: deploy}
		acc, ok := accs[key]
		if !ok {
			acc = &serviceAccumulator{
				requests:       make(map[string]int64),
				latencyBuckets: make(map[string]int64),
			}
			accs[key] = acc
		}
		return acc
	}

	// 聚合 response delta
	for _, d := range responses {
		acc := getAcc(d.Namespace, d.Deployment)
		reqKey := d.StatusCode + "|" + d.Classification
		acc.requests[reqKey] += int64(d.Delta)

		// mTLS 聚合
		acc.totalRequestDelta += int64(d.Delta)
		if d.TLS == "true" {
			acc.tlsRequestDelta += int64(d.Delta)
		}
	}

	// 聚合 latency buckets
	for _, d := range buckets {
		acc := getAcc(d.Namespace, d.Deployment)
		acc.latencyBuckets[d.Le] += int64(d.Delta)
	}

	// 聚合 latency sums
	for _, d := range sums {
		acc := getAcc(d.Namespace, d.Deployment)
		acc.latencySum += d.Delta
	}

	// 聚合 latency counts
	for _, d := range counts {
		acc := getAcc(d.Namespace, d.Deployment)
		acc.latencyCount += int64(d.Delta)
	}

	// 转换为 model_v2.ServiceMetrics
	result := make([]model_v2.ServiceMetrics, 0, len(accs))
	for key, acc := range accs {
		sm := model_v2.ServiceMetrics{
			Namespace:         key.Namespace,
			Name:              key.Deployment,
			LatencySum:        acc.latencySum,
			LatencyCount:      acc.latencyCount,
			TLSRequestDelta:   acc.tlsRequestDelta,
			TotalRequestDelta: acc.totalRequestDelta,
		}

		// 请求增量
		for reqKey, delta := range acc.requests {
			var statusCode, classification string
			// 解析 "status_code|classification"
			for i, c := range reqKey {
				if c == '|' {
					statusCode = reqKey[:i]
					classification = reqKey[i+1:]
					break
				}
			}
			sm.Requests = append(sm.Requests, model_v2.RequestDelta{
				StatusCode:     statusCode,
				Classification: classification,
				Delta:          delta,
			})
		}

		// 延迟直方图
		if len(acc.latencyBuckets) > 0 {
			sm.LatencyBuckets = acc.latencyBuckets
		}

		result = append(result, sm)
	}

	return result
}

// =============================================================================
// Stage 3b: Extract Edges (Linkerd outbound)
// =============================================================================

// edgeKey 源→目标
type edgeKey struct {
	SrcNamespace string
	SrcName      string
	DstNamespace string
	DstName      string
}

func (k edgeKey) String() string {
	return fmt.Sprintf("%s/%s→%s/%s", k.SrcNamespace, k.SrcName, k.DstNamespace, k.DstName)
}

// edgeAccumulator 单条边的聚合中间状态
type edgeAccumulator struct {
	requestDelta int64
	failureDelta int64
	latencySum   float64
	latencyCount int64
}

// extractEdges 从 outbound per-pod delta 提取拓扑
func extractEdges(
	responses []linkerdResponseDelta,
	sums []linkerdSumDelta,
	counts []linkerdCountDelta,
) []model_v2.ServiceEdge {
	accs := make(map[edgeKey]*edgeAccumulator)

	getAcc := func(srcNs, srcName, dstNs, dstName string) *edgeAccumulator {
		key := edgeKey{
			SrcNamespace: srcNs,
			SrcName:      srcName,
			DstNamespace: dstNs,
			DstName:      dstName,
		}
		acc, ok := accs[key]
		if !ok {
			acc = &edgeAccumulator{}
			accs[key] = acc
		}
		return acc
	}

	// 聚合 outbound response delta
	for _, d := range responses {
		if d.DstNamespace == "" || d.DstDeployment == "" {
			continue
		}
		acc := getAcc(d.Namespace, d.Deployment, d.DstNamespace, d.DstDeployment)
		acc.requestDelta += int64(d.Delta)
		if d.Classification == "failure" {
			acc.failureDelta += int64(d.Delta)
		}
	}

	// 聚合 outbound latency sums（OTel Collector 导出的 dst_* 标签可用）
	for _, d := range sums {
		if d.DstNamespace == "" || d.DstDeployment == "" {
			continue
		}
		acc := getAcc(d.Namespace, d.Deployment, d.DstNamespace, d.DstDeployment)
		acc.latencySum += d.Delta
	}

	// 聚合 outbound latency counts
	for _, d := range counts {
		if d.DstNamespace == "" || d.DstDeployment == "" {
			continue
		}
		acc := getAcc(d.Namespace, d.Deployment, d.DstNamespace, d.DstDeployment)
		acc.latencyCount += int64(d.Delta)
	}

	// 转换为 model_v2.ServiceEdge
	result := make([]model_v2.ServiceEdge, 0, len(accs))
	for key, acc := range accs {
		if acc.requestDelta == 0 {
			continue
		}
		result = append(result, model_v2.ServiceEdge{
			SrcNamespace: key.SrcNamespace,
			SrcName:      key.SrcName,
			DstNamespace: key.DstNamespace,
			DstName:      key.DstName,
			RequestDelta: acc.requestDelta,
			FailureDelta: acc.failureDelta,
			LatencySum:   acc.latencySum,
			LatencyCount: acc.latencyCount,
		})
	}

	return result
}

// =============================================================================
// Stage 3c: Aggregate Ingress (Controller 无关)
// =============================================================================

// ingressAccumulator 单个 service-key 的聚合中间状态
type ingressAccumulator struct {
	// 请求计数: key = "code|method"
	requests map[string]int64

	// 延迟直方图: key = le (毫秒 string)
	latencyBuckets map[string]int64
	latencySum     float64 // 毫秒
	latencyCount   int64
}

// aggregateIngress 聚合入口指标，同时将秒转为毫秒
func aggregateIngress(
	requests []ingressRequestDelta,
	buckets []ingressBucketDelta,
	sums []ingressSumDelta,
	counts []ingressCountDelta,
) []model_v2.IngressMetrics {
	accs := make(map[string]*ingressAccumulator)

	getAcc := func(serviceKey string) *ingressAccumulator {
		acc, ok := accs[serviceKey]
		if !ok {
			acc = &ingressAccumulator{
				requests:       make(map[string]int64),
				latencyBuckets: make(map[string]int64),
			}
			accs[serviceKey] = acc
		}
		return acc
	}

	// 请求计数
	for _, d := range requests {
		if d.ServiceKey == "" {
			continue
		}
		acc := getAcc(d.ServiceKey)
		reqKey := d.Code + "|" + d.Method
		acc.requests[reqKey] += int64(d.Delta)
	}

	// 延迟 buckets (秒→毫秒)
	for _, d := range buckets {
		if d.ServiceKey == "" {
			continue
		}
		acc := getAcc(d.ServiceKey)
		msLe := secondsToMillisLe(d.Le)
		acc.latencyBuckets[msLe] += int64(d.Delta)
	}

	// 延迟 sum (秒→毫秒)
	for _, d := range sums {
		if d.ServiceKey == "" {
			continue
		}
		acc := getAcc(d.ServiceKey)
		acc.latencySum += d.Delta * 1000 // 秒→毫秒
	}

	// 延迟 count
	for _, d := range counts {
		if d.ServiceKey == "" {
			continue
		}
		acc := getAcc(d.ServiceKey)
		acc.latencyCount += int64(d.Delta)
	}

	// 转换为 model_v2.IngressMetrics
	result := make([]model_v2.IngressMetrics, 0, len(accs))
	for serviceKey, acc := range accs {
		im := model_v2.IngressMetrics{
			ServiceKey:   serviceKey,
			LatencySum:   acc.latencySum,
			LatencyCount: acc.latencyCount,
		}

		// 请求增量
		for reqKey, delta := range acc.requests {
			var code, method string
			for i, c := range reqKey {
				if c == '|' {
					code = reqKey[:i]
					method = reqKey[i+1:]
					break
				}
			}
			im.Requests = append(im.Requests, model_v2.IngressRequestDelta{
				Code:   code,
				Method: method,
				Delta:  delta,
			})
		}

		// 延迟直方图
		if len(acc.latencyBuckets) > 0 {
			im.LatencyBuckets = acc.latencyBuckets
		}

		result = append(result, im)
	}

	return result
}

// secondsToMillisLe 将秒单位的 le 转为毫秒字符串
// "0.1" → "100", "0.3" → "300", "5" → "5000", "+Inf" → "+Inf"
func secondsToMillisLe(le string) string {
	if le == "+Inf" {
		return "+Inf"
	}
	sec, err := strconv.ParseFloat(le, 64)
	if err != nil {
		return le // 无法解析，原样返回
	}
	ms := sec * 1000
	// 整数时不带小数点
	if ms == float64(int64(ms)) {
		return strconv.FormatInt(int64(ms), 10)
	}
	return strconv.FormatFloat(ms, 'f', -1, 64)
}
