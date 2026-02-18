// atlhyper_master_v2/gateway/handler/slo_latency.go
// 入口延迟分布 API Handler
package handler

import (
	"math"
	"net/http"
	"sort"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/slo"
)

// LatencyDistribution GET /api/v2/slo/domains/latency
// 返回指定域名的延迟分布（bucket + method + status）
func (h *SLOHandler) LatencyDistribution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterIDs, err := h.repo.GetAllClusterIDs(r.Context())
		if err == nil && len(clusterIDs) > 0 {
			clusterID = clusterIDs[0]
		} else {
			clusterID = "default"
		}
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
	now := time.Now()
	start, end := getTimeRange(now, timeRange)

	// 获取域名下所有 service_key
	mappings, err := h.repo.GetRouteMappingsByDomain(ctx, clusterID, domain)
	if err != nil {
		sloLog.Error("获取路由映射失败", "domain", domain, "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 去重 service_key
	serviceKeys := make(map[string]bool)
	for _, m := range mappings {
		serviceKeys[m.ServiceKey] = true
	}

	// 合并所有 service 的数据
	var totalRequests, errorRequests int64
	var latencySum float64
	var latencyCount int64
	var mGet, mPost, mPut, mDelete, mOther int64
	var s2xx, s3xx, s4xx, s5xx int64
	allBuckets := make(map[float64]int64)

	for serviceKey := range serviceKeys {
		// 优先查 hourly
		hourlyRows, err := h.repo.GetHourlyMetrics(ctx, clusterID, serviceKey, start, end)
		if err != nil {
			sloLog.Debug("查询 hourly 失败", "key", serviceKey, "err", err)
		}

		if len(hourlyRows) > 0 {
			var latestHourStart time.Time
			for _, m := range hourlyRows {
				totalRequests += m.TotalRequests
				errorRequests += m.ErrorRequests
				mGet += m.MethodGet
				mPost += m.MethodPost
				mPut += m.MethodPut
				mDelete += m.MethodDelete
				mOther += m.MethodOther
				s2xx += m.Status2xx
				s3xx += m.Status3xx
				s4xx += m.Status4xx
				s5xx += m.Status5xx

				if b := slo.ParseJSONBuckets(m.LatencyBuckets); b != nil {
					for le, count := range b {
						allBuckets[le] += count
					}
				}

				// 用加权平均近似 latencySum/Count
				latencySum += float64(m.AvgLatencyMs) * float64(m.TotalRequests)
				latencyCount += m.TotalRequests

				if m.HourStart.After(latestHourStart) {
					latestHourStart = m.HourStart
				}
			}

			// 补充未聚合的当前时段 raw 数据
			rawStart := latestHourStart.Add(time.Hour)
			if rawStart.Before(end) {
				rawRows, rawErr := h.repo.GetRawMetrics(ctx, clusterID, serviceKey, rawStart, end)
				if rawErr == nil {
					for _, m := range rawRows {
						totalRequests += m.TotalRequests
						errorRequests += m.ErrorRequests
						latencySum += m.LatencySum
						latencyCount += m.LatencyCount
						mGet += m.MethodGet
						mPost += m.MethodPost
						mPut += m.MethodPut
						mDelete += m.MethodDelete
						mOther += m.MethodOther
						s2xx += m.Status2xx
						s3xx += m.Status3xx
						s4xx += m.Status4xx
						s5xx += m.Status5xx

						if b := slo.ParseJSONBuckets(m.LatencyBuckets); b != nil {
							for le, count := range b {
								allBuckets[le] += count
							}
						}
					}
				}
			}
		} else {
			// 回退到 raw
			rawRows, err := h.repo.GetRawMetrics(ctx, clusterID, serviceKey, start, end)
			if err != nil {
				sloLog.Debug("查询 raw 失败", "key", serviceKey, "err", err)
				continue
			}
			for _, m := range rawRows {
				totalRequests += m.TotalRequests
				errorRequests += m.ErrorRequests
				latencySum += m.LatencySum
				latencyCount += m.LatencyCount
				mGet += m.MethodGet
				mPost += m.MethodPost
				mPut += m.MethodPut
				mDelete += m.MethodDelete
				mOther += m.MethodOther
				s2xx += m.Status2xx
				s3xx += m.Status3xx
				s4xx += m.Status4xx
				s5xx += m.Status5xx

				if b := slo.ParseJSONBuckets(m.LatencyBuckets); b != nil {
					for le, count := range b {
						allBuckets[le] += count
					}
				}
			}
		}
	}

	// 计算分位数
	p50 := slo.CalculateQuantileMs(allBuckets, 0.50)
	p95 := slo.CalculateQuantileMs(allBuckets, 0.95)
	p99 := slo.CalculateQuantileMs(allBuckets, 0.99)

	var avgLatency int
	if latencyCount > 0 {
		avgLatency = int(latencySum / float64(latencyCount))
	}

	// 构建 bucket 响应
	buckets := buildLatencyBuckets(allBuckets)

	// 构建 method 分布
	methods := buildMethodBreakdown(mGet, mPost, mPut, mDelete, mOther)

	// 构建状态码分布
	statusCodes := buildStatusCodeBreakdown(s2xx, s3xx, s4xx, s5xx)

	resp := model.LatencyDistributionResponse{
		Domain:        domain,
		TotalRequests: totalRequests,
		P50LatencyMs:  p50,
		P95LatencyMs:  p95,
		P99LatencyMs:  p99,
		AvgLatencyMs:  avgLatency,
		Buckets:       buckets,
		Methods:       methods,
		StatusCodes:   statusCodes,
	}

	writeJSON(w, http.StatusOK, resp)
}

// buildLatencyBuckets 将 map[float64]int64 转换为 LatencyBucket 切片
// key 为秒值，输出为毫秒
func buildLatencyBuckets(buckets map[float64]int64) []model.LatencyBucket {
	if len(buckets) == 0 {
		return nil
	}

	// 排序
	les := make([]float64, 0, len(buckets))
	for le := range buckets {
		if !math.IsInf(le, 1) {
			les = append(les, le)
		}
	}
	sort.Float64s(les)

	// 转换为非累积计数（differential buckets）
	result := make([]model.LatencyBucket, 0, len(les))
	var prevCount int64
	for _, le := range les {
		cumulativeCount := buckets[le]
		count := cumulativeCount - prevCount
		if count < 0 {
			count = 0
		}
		ms := le * 1000
		result = append(result, model.LatencyBucket{
			LE:    ms,
			Count: count,
		})
		prevCount = cumulativeCount
	}

	return result
}

// buildMethodBreakdown 构建 HTTP 方法分布
func buildMethodBreakdown(get, post, put, del, other int64) []model.MethodBreakdown {
	var result []model.MethodBreakdown
	if get > 0 {
		result = append(result, model.MethodBreakdown{Method: "GET", Count: get})
	}
	if post > 0 {
		result = append(result, model.MethodBreakdown{Method: "POST", Count: post})
	}
	if put > 0 {
		result = append(result, model.MethodBreakdown{Method: "PUT", Count: put})
	}
	if del > 0 {
		result = append(result, model.MethodBreakdown{Method: "DELETE", Count: del})
	}
	if other > 0 {
		result = append(result, model.MethodBreakdown{Method: "OTHER", Count: other})
	}
	// 按 count 降序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result
}

// buildStatusCodeBreakdown 构建状态码分布
func buildStatusCodeBreakdown(s2xx, s3xx, s4xx, s5xx int64) []model.StatusCodeBreakdown {
	var result []model.StatusCodeBreakdown
	if s2xx > 0 {
		result = append(result, model.StatusCodeBreakdown{Code: "2xx", Count: s2xx})
	}
	if s3xx > 0 {
		result = append(result, model.StatusCodeBreakdown{Code: "3xx", Count: s3xx})
	}
	if s4xx > 0 {
		result = append(result, model.StatusCodeBreakdown{Code: "4xx", Count: s4xx})
	}
	if s5xx > 0 {
		result = append(result, model.StatusCodeBreakdown{Code: "5xx", Count: s5xx})
	}
	return result
}

