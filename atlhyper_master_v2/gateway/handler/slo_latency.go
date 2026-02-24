// atlhyper_master_v2/gateway/handler/slo_latency.go
// 入口延迟分布 API Handler — OTelSnapshot 直读模式
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/model"
	slomodel "AtlHyper/model_v3/slo"
)

// LatencyDistribution GET /api/v2/slo/domains/latency
// 返回指定域名的延迟分布（优先从 SLOWindows 获取，回退到 SLOIngress）
func (h *SLOHandler) LatencyDistribution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeError(w, http.StatusBadRequest, "domain required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()

	// 获取 OTelSnapshot
	otel, err := h.querySvc.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 OTelSnapshot 失败", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 获取域名下所有 service_key
	mappings, err := h.sloRepo.GetRouteMappingsByDomain(ctx, clusterID, domain)
	if err != nil {
		sloLog.Error("获取路由映射失败", "domain", domain, "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	serviceKeys := make(map[string]bool)
	for _, m := range mappings {
		serviceKeys[m.ServiceKey] = true
	}

	// 路由映射为空时，直接以 domain 参数作为 ServiceKey 匹配
	if len(serviceKeys) == 0 {
		serviceKeys[domain] = true
	}

	// 优先从 SLOWindows[timeRange] 获取 IngressSLO（含 LatencyBuckets + Methods）
	var ingressList []slomodel.IngressSLO
	if otel != nil && otel.SLOWindows != nil {
		if w, ok := otel.SLOWindows[timeRange]; ok {
			ingressList = w.Current
		}
	}
	// 回退到 5min SLOIngress
	if len(ingressList) == 0 && otel != nil {
		ingressList = otel.SLOIngress
	}

	// 合并所有 service 的 IngressSLO 数据
	var totalRequests int64
	var totalRPS float64
	var weightedP50, weightedP95, weightedP99, weightedAvg float64
	var methods []model.MethodBreakdown
	var statusCodes []model.StatusCodeBreakdown
	var buckets []model.LatencyBucket

	// 聚合 map
	statusMap := make(map[string]int64)
	methodMap := make(map[string]int64)
	bucketMap := make(map[float64]int64) // le → count

	for _, ing := range ingressList {
		if !serviceKeys[ing.ServiceKey] {
			continue
		}

		totalRequests += ing.TotalRequests
		totalRPS += ing.RPS
		weightedP50 += ing.P50Ms * float64(ing.TotalRequests)
		p95 := ing.P95Ms
		if p95 == 0 {
			p95 = ing.P90Ms
		}
		weightedP95 += p95 * float64(ing.TotalRequests)
		weightedP99 += ing.P99Ms * float64(ing.TotalRequests)
		weightedAvg += ing.AvgMs * float64(ing.TotalRequests)

		// 聚合状态码
		for _, sc := range ing.StatusCodes {
			statusMap[sc.Code] += sc.Count
		}

		// 聚合延迟桶
		for _, b := range ing.LatencyBuckets {
			bucketMap[b.LE] += b.Count
		}

		// 聚合方法分布
		for _, m := range ing.Methods {
			methodMap[m.Method] += m.Count
		}
	}

	// 计算加权平均
	var p50, p95, p99, avg int
	if totalRequests > 0 {
		p50 = int(weightedP50 / float64(totalRequests))
		p95 = int(weightedP95 / float64(totalRequests))
		p99 = int(weightedP99 / float64(totalRequests))
		avg = int(weightedAvg / float64(totalRequests))
	}

	// 构建状态码分布
	for code, count := range statusMap {
		if count > 0 {
			statusCodes = append(statusCodes, model.StatusCodeBreakdown{Code: code, Count: count})
		}
	}

	// 构建延迟分布桶
	for le, count := range bucketMap {
		buckets = append(buckets, model.LatencyBucket{LE: le, Count: count})
	}

	// 构建方法分布
	for method, count := range methodMap {
		methods = append(methods, model.MethodBreakdown{Method: method, Count: count})
	}

	resp := model.LatencyDistributionResponse{
		Domain:        domain,
		TotalRequests: totalRequests,
		P50LatencyMs:  p50,
		P95LatencyMs:  p95,
		P99LatencyMs:  p99,
		AvgLatencyMs:  avg,
		Buckets:       buckets,
		Methods:       methods,
		StatusCodes:   statusCodes,
	}

	writeJSON(w, http.StatusOK, resp)
}
