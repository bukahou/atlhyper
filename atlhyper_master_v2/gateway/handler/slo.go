// atlhyper_master_v2/gateway/handler/slo.go
// SLO API Handler
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/slo"
	"AtlHyper/common/logger"
)

var sloLog = logger.Module("SLO-Handler")

// SLOHandler SLO API Handler
type SLOHandler struct {
	repo        database.SLORepository
	serviceRepo database.SLOServiceRepository
	aggregator  *slo.Aggregator
}

// NewSLOHandler 创建 SLOHandler
func NewSLOHandler(repo database.SLORepository, serviceRepo database.SLOServiceRepository, aggregator *slo.Aggregator) *SLOHandler {
	return &SLOHandler{
		repo:        repo,
		serviceRepo: serviceRepo,
		aggregator:  aggregator,
	}
}

// ==================== 域名列表 ====================

// Domains GET /api/v2/slo/domains
func (h *SLOHandler) Domains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		// 自动获取第一个有数据的集群
		clusterIDs, err := h.repo.GetAllClusterIDs(r.Context())
		if err == nil && len(clusterIDs) > 0 {
			clusterID = clusterIDs[0]
		} else {
			clusterID = "default"
		}
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()

	// 获取所有 hosts
	hosts, err := h.repo.GetAllHosts(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 hosts 失败", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 计算时间范围
	now := time.Now()
	start, end := getTimeRange(now, timeRange)
	prevStart, prevEnd := getPreviousTimeRange(start, end)

	// 获取所有目标配置
	targets, err := h.repo.GetTargets(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 targets 失败", "err", err)
	}
	targetMap := buildTargetMap(targets)

	// 构建响应
	var domains []model.DomainSLO
	var totalAvail, totalBudget, totalRPS float64
	var healthyCount, warningCount, criticalCount int

	for _, host := range hosts {
		domain := h.buildDomainSLO(ctx, clusterID, host, start, end, prevStart, prevEnd, timeRange, targetMap)
		domains = append(domains, domain)

		// 统计
		if domain.Current != nil {
			totalAvail += domain.Current.Availability
			totalRPS += domain.Current.RequestsPerSec
		}
		totalBudget += domain.ErrorBudget

		switch domain.Status {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
		case "critical":
			criticalCount++
		}
	}

	// 计算平均值
	var avgAvail, avgBudget float64
	if len(domains) > 0 {
		avgAvail = totalAvail / float64(len(domains))
		avgBudget = totalBudget / float64(len(domains))
	}

	resp := model.SLODomainsResponse{
		Domains: domains,
		Summary: model.SLOSummary{
			TotalDomains:    len(domains),
			HealthyCount:    healthyCount,
			WarningCount:    warningCount,
			CriticalCount:   criticalCount,
			AvgAvailability: avgAvail,
			AvgErrorBudget:  avgBudget,
			TotalRPS:        totalRPS,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// buildDomainSLO 构建单个域名的 SLO 信息
func (h *SLOHandler) buildDomainSLO(ctx context.Context, clusterID, host string, start, end, prevStart, prevEnd time.Time, timeRange string, targetMap map[string]map[string]*database.SLOTarget) model.DomainSLO {
	domain := model.DomainSLO{
		Host:    host,
		Targets: make(map[string]*model.SLOTargetSpec),
		Status:  "unknown",
		Trend:   "stable",
	}

	// 获取元信息（从路由映射表）
	mapping, _ := h.repo.GetRouteMappingByServiceKey(ctx, clusterID, host)
	if mapping != nil {
		domain.IngressName = mapping.IngressName
		domain.Namespace = mapping.Namespace
		domain.TLS = mapping.TLS
	}

	// 获取当前周期的 hourly 数据
	hourlyMetrics, err := h.repo.GetHourlyMetrics(ctx, clusterID, host, start, end)
	if err != nil {
		sloLog.Error("获取 hourly 数据失败", "host", host, "err", err)
	}

	if len(hourlyMetrics) > 0 {
		// 使用 hourly 数据
		domain.Current = aggregateHourlyMetrics(hourlyMetrics)
	} else {
		// 回退到 raw 数据（hourly 还未聚合时）
		rawMetrics, err := h.repo.GetRawMetrics(ctx, clusterID, host, start, end)
		if err != nil {
			sloLog.Error("获取 raw 数据失败", "host", host, "err", err)
		}
		if len(rawMetrics) > 0 {
			domain.Current = aggregateRawMetrics(rawMetrics)
		}
	}

	// 获取上一周期数据
	prevHourlyMetrics, err := h.repo.GetHourlyMetrics(ctx, clusterID, host, prevStart, prevEnd)
	if err == nil && len(prevHourlyMetrics) > 0 {
		domain.Previous = aggregateHourlyMetrics(prevHourlyMetrics)
	}

	// 设置目标
	if hostTargets, ok := targetMap[host]; ok {
		for tr, t := range hostTargets {
			domain.Targets[tr] = &model.SLOTargetSpec{
				Availability: t.AvailabilityTarget,
				P95Latency:   t.P95LatencyTarget,
			}
		}
	}

	// 如果没有目标，使用默认值
	if len(domain.Targets) == 0 {
		domain.Targets["1d"] = &model.SLOTargetSpec{Availability: 95.0, P95Latency: 300}
		domain.Targets["7d"] = &model.SLOTargetSpec{Availability: 96.0, P95Latency: 280}
		domain.Targets["30d"] = &model.SLOTargetSpec{Availability: 97.0, P95Latency: 250}
	}

	// 计算状态
	if domain.Current != nil {
		target := domain.Targets[timeRange]
		if target == nil {
			target = domain.Targets["1d"]
		}
		if target != nil {
			domain.Status = slo.DetermineStatus(domain.Current.Availability, target.Availability, domain.Current.P95Latency, target.P95Latency)
			domain.ErrorBudget = slo.CalculateErrorBudgetRemaining(domain.Current.Availability, target.Availability)
		}

		// 计算趋势
		if domain.Previous != nil {
			domain.Trend = slo.CalculateTrend(domain.Current.Availability, domain.Previous.Availability)
		}
	}

	return domain
}

// ==================== V2: 按真实域名分组 ====================

// DomainsV2 GET /api/v2/slo/domains/v2
// 按真实域名分组返回 SLO 数据（使用 IngressRoute 映射）
func (h *SLOHandler) DomainsV2(w http.ResponseWriter, r *http.Request) {
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

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()

	// 获取所有真实域名
	domains, err := h.repo.GetAllDomains(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取域名列表失败", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 计算时间范围
	now := time.Now()
	start, end := getTimeRange(now, timeRange)
	prevStart, prevEnd := getPreviousTimeRange(start, end)

	// 获取所有目标配置
	targets, _ := h.repo.GetTargets(ctx, clusterID)
	targetMap := buildTargetMap(targets)

	// 构建响应
	var domainResponses []model.DomainSLOResponseV2
	var totalAvail, totalBudget, totalRPS float64
	var healthyCount, warningCount, criticalCount int

	for _, domain := range domains {
		domainResp := h.buildDomainSLOV2(ctx, clusterID, domain, start, end, prevStart, prevEnd, timeRange, targetMap)
		domainResponses = append(domainResponses, domainResp)

		// 统计
		if domainResp.Summary != nil {
			totalAvail += domainResp.Summary.Availability
			totalRPS += domainResp.Summary.RequestsPerSec
		}
		totalBudget += domainResp.ErrorBudgetRemaining

		switch domainResp.Status {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
		case "critical":
			criticalCount++
		}
	}

	// 计算平均值
	var avgAvail, avgBudget float64
	if len(domainResponses) > 0 {
		avgAvail = totalAvail / float64(len(domainResponses))
		avgBudget = totalBudget / float64(len(domainResponses))
	}

	// 查询 Linkerd meshed 服务总数
	var totalServices int
	if h.serviceRepo != nil {
		totalServices, _ = h.serviceRepo.CountDistinctServices(ctx, clusterID, start, end)
	}

	resp := model.SLODomainsResponseV2{
		Domains: domainResponses,
		Summary: model.SLOSummary{
			TotalServices:   totalServices,
			TotalDomains:    len(domainResponses),
			HealthyCount:    healthyCount,
			WarningCount:    warningCount,
			CriticalCount:   criticalCount,
			AvgAvailability: avgAvail,
			AvgErrorBudget:  avgBudget,
			TotalRPS:        totalRPS,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// buildDomainSLOV2 构建单个域名的 V2 SLO 信息
func (h *SLOHandler) buildDomainSLOV2(ctx context.Context, clusterID, domain string, start, end, prevStart, prevEnd time.Time, timeRange string, targetMap map[string]map[string]*database.SLOTarget) model.DomainSLOResponseV2 {
	resp := model.DomainSLOResponseV2{
		Domain: domain,
		TLS:    true, // 默认启用
		Status: "unknown",
	}

	// 获取该域名下的所有路由映射
	mappings, err := h.repo.GetRouteMappingsByDomain(ctx, clusterID, domain)
	if err != nil {
		sloLog.Error("获取路由映射失败", "domain", domain, "err", err)
		return resp
	}

	if len(mappings) > 0 {
		resp.TLS = mappings[0].TLS
	}

	// 按 service_key 分组路由映射
	// 因为 Traefik/nginx-ingress 的 metrics 是按 service 级别聚合的，
	// 同一个 service 的多个路径应该合并到一个 ServiceSLO
	serviceKeyGroups := make(map[string][]*database.SLORouteMapping)
	for _, mapping := range mappings {
		serviceKeyGroups[mapping.ServiceKey] = append(serviceKeyGroups[mapping.ServiceKey], mapping)
	}

	// 用于汇总域名级别数据
	var totalRequests, errorRequests int64
	var totalRPS float64
	var weightedP95Sum, weightedP99Sum float64
	serviceCount := 0

	// 按 service_key 构建 ServiceSLO
	for serviceKey, groupMappings := range serviceKeyGroups {
		// 使用第一个映射作为基础
		primaryMapping := groupMappings[0]

		// 收集该 service 的所有路径
		var paths []string
		for _, m := range groupMappings {
			paths = append(paths, m.PathPrefix)
		}

		// 构建 ServiceSLO
		serviceSLO := h.buildServiceSLO(ctx, clusterID, serviceKey, primaryMapping, paths, start, end, prevStart, prevEnd, timeRange, targetMap)
		resp.Services = append(resp.Services, serviceSLO)

		// 汇总数据（每个 service_key 只计算一次）
		if serviceSLO.Current != nil {
			totalRequests += serviceSLO.Current.TotalRequests
			errorRequests += int64(serviceSLO.Current.ErrorRate * float64(serviceSLO.Current.TotalRequests) / 100)
			totalRPS += serviceSLO.Current.RequestsPerSec
			// 按请求量加权累加 P95/P99
			weightedP95Sum += float64(serviceSLO.Current.P95Latency) * float64(serviceSLO.Current.TotalRequests)
			weightedP99Sum += float64(serviceSLO.Current.P99Latency) * float64(serviceSLO.Current.TotalRequests)
			serviceCount++
		}

		// 统计最差状态
		if serviceSLO.Status == "critical" && resp.Status != "critical" {
			resp.Status = "critical"
		} else if serviceSLO.Status == "warning" && resp.Status != "critical" && resp.Status != "warning" {
			resp.Status = "warning"
		} else if serviceSLO.Status == "healthy" && resp.Status == "unknown" {
			resp.Status = "healthy"
		}
	}

	// 域名级别目标：使用 domain 自身的目标配置，回退到默认值
	if domainTargets, ok := targetMap[domain]; ok {
		resp.Targets = make(map[string]*model.SLOTargetSpec)
		for tr, t := range domainTargets {
			resp.Targets[tr] = &model.SLOTargetSpec{
				Availability: t.AvailabilityTarget,
				P95Latency:   t.P95LatencyTarget,
			}
		}
	}
	if len(resp.Targets) == 0 {
		resp.Targets = map[string]*model.SLOTargetSpec{
			"1d": {Availability: 95.0, P95Latency: 300},
		}
	}

	// 计算域名级别汇总
	if serviceCount > 0 {
		var p95, p99 int
		if totalRequests > 0 {
			p95 = int(weightedP95Sum / float64(totalRequests))
			p99 = int(weightedP99Sum / float64(totalRequests))
		}
		resp.Summary = &model.SLOMetrics{
			Availability:   slo.CalculateAvailability(totalRequests, errorRequests),
			P95Latency:     p95,
			P99Latency:     p99,
			ErrorRate:      slo.CalculateErrorRate(totalRequests, errorRequests),
			RequestsPerSec: totalRPS,
			TotalRequests:  totalRequests,
		}

		// 使用当前时间范围的目标计算错误预算
		target := resp.Targets[timeRange]
		if target == nil {
			target = resp.Targets["1d"]
		}
		availTarget := 95.0
		if target != nil {
			availTarget = target.Availability
		}
		resp.ErrorBudgetRemaining = slo.CalculateErrorBudgetRemaining(resp.Summary.Availability, availTarget)
	}

	return resp
}

// buildServiceSLO 构建单个后端服务的 SLO 信息
func (h *SLOHandler) buildServiceSLO(ctx context.Context, clusterID, serviceKey string, mapping *database.SLORouteMapping, paths []string, start, end, prevStart, prevEnd time.Time, timeRange string, targetMap map[string]map[string]*database.SLOTarget) model.ServiceSLO {
	svc := model.ServiceSLO{
		ServiceKey:  serviceKey,
		ServiceName: mapping.ServiceName,
		ServicePort: mapping.ServicePort,
		Namespace:   mapping.Namespace,
		Paths:       paths,
		IngressName: mapping.IngressName,
		Targets:     make(map[string]*model.SLOTargetSpec),
		Status:      "unknown",
	}

	// 使用 service key (host) 查询指标数据
	host := serviceKey

	// 获取当前周期的 hourly 数据
	hourlyMetrics, err := h.repo.GetHourlyMetrics(ctx, clusterID, host, start, end)
	if err != nil {
		sloLog.Debug("获取 hourly 数据失败", "host", host, "err", err)
	}

	if len(hourlyMetrics) > 0 {
		svc.Current = aggregateHourlyMetrics(hourlyMetrics)
	} else {
		// 回退到 raw 数据
		rawMetrics, err := h.repo.GetRawMetrics(ctx, clusterID, host, start, end)
		if err != nil {
			sloLog.Debug("获取 raw 数据失败", "host", host, "err", err)
		}
		if len(rawMetrics) > 0 {
			svc.Current = aggregateRawMetrics(rawMetrics)
		}
	}

	// 获取上一周期数据
	prevHourlyMetrics, err := h.repo.GetHourlyMetrics(ctx, clusterID, host, prevStart, prevEnd)
	if err == nil && len(prevHourlyMetrics) > 0 {
		svc.Previous = aggregateHourlyMetrics(prevHourlyMetrics)
	}

	// 设置目标
	if hostTargets, ok := targetMap[host]; ok {
		for tr, t := range hostTargets {
			svc.Targets[tr] = &model.SLOTargetSpec{
				Availability: t.AvailabilityTarget,
				P95Latency:   t.P95LatencyTarget,
			}
		}
	}

	// 默认目标
	if len(svc.Targets) == 0 {
		svc.Targets["1d"] = &model.SLOTargetSpec{Availability: 95.0, P95Latency: 300}
	}

	// 计算状态
	if svc.Current != nil {
		target := svc.Targets[timeRange]
		if target == nil {
			target = svc.Targets["1d"]
		}
		if target != nil {
			svc.Status = slo.DetermineStatus(svc.Current.Availability, target.Availability, svc.Current.P95Latency, target.P95Latency)
			svc.ErrorBudget = slo.CalculateErrorBudgetRemaining(svc.Current.Availability, target.Availability)
		}
	}

	return svc
}

// ==================== 域名详情 ====================

// DomainDetail GET /api/v2/slo/domains/:host
func (h *SLOHandler) DomainDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 从 URL 路径提取 host
	host := r.URL.Query().Get("host")
	if host == "" {
		writeError(w, http.StatusBadRequest, "host required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = "default"
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()
	now := time.Now()
	start, end := getTimeRange(now, timeRange)
	prevStart, prevEnd := getPreviousTimeRange(start, end)

	targets, _ := h.repo.GetTargets(ctx, clusterID)
	targetMap := buildTargetMap(targets)

	domain := h.buildDomainSLO(ctx, clusterID, host, start, end, prevStart, prevEnd, timeRange, targetMap)

	writeJSON(w, http.StatusOK, domain)
}

// ==================== 域名历史 ====================

// DomainHistory GET /api/v2/slo/domains/:host/history
func (h *SLOHandler) DomainHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	host := r.URL.Query().Get("host")
	if host == "" {
		writeError(w, http.StatusBadRequest, "host required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = "default"
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()
	now := time.Now()
	start, end := getTimeRange(now, timeRange)

	hourlyMetrics, err := h.repo.GetHourlyMetrics(ctx, clusterID, host, start, end)
	if err != nil {
		sloLog.Error("获取历史数据失败", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	history := make([]model.SLODomainHistoryItem, 0, len(hourlyMetrics))
	for _, m := range hourlyMetrics {
		history = append(history, model.SLODomainHistoryItem{
			Timestamp:    m.HourStart.Format(time.RFC3339),
			Availability: m.Availability,
			P95Latency:   m.P95LatencyMs,
			P99Latency:   m.P99LatencyMs,
			RPS:          m.AvgRPS,
			ErrorRate:    slo.CalculateErrorRate(m.TotalRequests, m.ErrorRequests),
		})
	}

	resp := model.SLODomainHistoryResponse{
		Host:    host,
		History: history,
	}

	writeJSON(w, http.StatusOK, resp)
}

// ==================== 目标管理 ====================

// Targets 处理 /api/v2/slo/targets
// GET: 获取目标, PUT: 更新目标
func (h *SLOHandler) Targets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getTargets(w, r)
	case http.MethodPut:
		h.updateTarget(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getTargets 获取目标配置
func (h *SLOHandler) getTargets(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = "default"
	}

	targets, err := h.repo.GetTargets(r.Context(), clusterID)
	if err != nil {
		sloLog.Error("获取 targets 失败", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, targets)
}

// updateTarget 更新目标配置
func (h *SLOHandler) updateTarget(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateSLOTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Host == "" || req.TimeRange == "" {
		writeError(w, http.StatusBadRequest, "host and time_range required")
		return
	}

	if req.ClusterID == "" {
		req.ClusterID = "default"
	}

	now := time.Now()
	target := &database.SLOTarget{
		ClusterID:          req.ClusterID,
		Host:               req.Host,
		TimeRange:          req.TimeRange,
		AvailabilityTarget: req.AvailabilityTarget,
		P95LatencyTarget:   req.P95LatencyTarget,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := h.repo.UpsertTarget(r.Context(), target); err != nil {
		sloLog.Error("更新 target 失败", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ==================== 状态历史 ====================

// StatusHistory GET /api/v2/slo/status-history
func (h *SLOHandler) StatusHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = "default"
	}

	host := r.URL.Query().Get("host")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	history, err := h.repo.GetStatusHistory(r.Context(), clusterID, host, limit)
	if err != nil {
		sloLog.Error("获取状态历史失败", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]model.SLOStatusHistoryItem, 0, len(history))
	for _, h := range history {
		items = append(items, model.SLOStatusHistoryItem{
			Host:                 h.Host,
			TimeRange:            h.TimeRange,
			OldStatus:            h.OldStatus,
			NewStatus:            h.NewStatus,
			Availability:         h.Availability,
			P95Latency:           h.P95Latency,
			ErrorBudgetRemaining: h.ErrorBudgetRemaining,
			ChangedAt:            h.ChangedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, items)
}

// ==================== 辅助函数 ====================

// getTimeRange 根据时间范围字符串计算起止时间
func getTimeRange(now time.Time, timeRange string) (start, end time.Time) {
	end = now
	switch timeRange {
	case "1d":
		start = now.Add(-24 * time.Hour)
	case "7d":
		start = now.Add(-7 * 24 * time.Hour)
	case "30d":
		start = now.Add(-30 * 24 * time.Hour)
	default:
		start = now.Add(-24 * time.Hour)
	}
	return
}

// getPreviousTimeRange 获取上一个周期的时间范围
func getPreviousTimeRange(start, end time.Time) (prevStart, prevEnd time.Time) {
	duration := end.Sub(start)
	prevEnd = start
	prevStart = start.Add(-duration)
	return
}

// buildTargetMap 构建目标配置 map
func buildTargetMap(targets []*database.SLOTarget) map[string]map[string]*database.SLOTarget {
	result := make(map[string]map[string]*database.SLOTarget)
	for _, t := range targets {
		if result[t.Host] == nil {
			result[t.Host] = make(map[string]*database.SLOTarget)
		}
		result[t.Host][t.TimeRange] = t
	}
	return result
}

// aggregateHourlyMetrics 聚合多个小时的数据
func aggregateHourlyMetrics(metrics []*database.SLOMetricsHourly) *model.SLOMetrics {
	if len(metrics) == 0 {
		return nil
	}

	var totalRequests, errorRequests int64
	var totalRPS float64
	var weightedP95, weightedP99 float64

	for _, m := range metrics {
		totalRequests += m.TotalRequests
		errorRequests += m.ErrorRequests
		totalRPS += m.AvgRPS
		// 加权平均分位数（按请求量加权）
		weightedP95 += float64(m.P95LatencyMs) * float64(m.TotalRequests)
		weightedP99 += float64(m.P99LatencyMs) * float64(m.TotalRequests)
	}

	var p95, p99 int
	if totalRequests > 0 {
		p95 = int(weightedP95 / float64(totalRequests))
		p99 = int(weightedP99 / float64(totalRequests))
	}

	return &model.SLOMetrics{
		Availability:   slo.CalculateAvailability(totalRequests, errorRequests),
		P95Latency:     p95,
		P99Latency:     p99,
		ErrorRate:      slo.CalculateErrorRate(totalRequests, errorRequests),
		RequestsPerSec: totalRPS / float64(len(metrics)),
		TotalRequests:  totalRequests,
	}
}

// aggregateRawMetrics 聚合 raw 数据（hourly 未就绪时的回退方案）
func aggregateRawMetrics(metrics []*database.SLOMetricsRaw) *model.SLOMetrics {
	if len(metrics) == 0 {
		return nil
	}

	var totalRequests, errorRequests int64
	var totalLatencySum float64
	var totalLatencyCount int64
	var allBuckets []map[float64]int64

	for _, m := range metrics {
		totalRequests += m.TotalRequests
		errorRequests += m.ErrorRequests
		totalLatencySum += m.LatencySum
		totalLatencyCount += m.LatencyCount
		if b := slo.ParseJSONBuckets(m.LatencyBuckets); b != nil {
			allBuckets = append(allBuckets, b)
		}
	}

	// 计算 RPS（假设每条 raw 是 10 秒间隔）
	durationSeconds := float64(len(metrics)) * 10.0
	avgRPS := float64(totalRequests) / durationSeconds

	// 使用合并 bucket 计算精确分位数
	merged := slo.MergeBuckets(allBuckets...)
	p95 := slo.CalculateQuantileMs(merged, 0.95)
	p99 := slo.CalculateQuantileMs(merged, 0.99)

	// bucket 为空时回退到平均延迟
	if p95 == 0 && totalLatencyCount > 0 {
		avg := int(totalLatencySum / float64(totalLatencyCount))
		p95 = avg
		p99 = avg
	}

	return &model.SLOMetrics{
		Availability:   slo.CalculateAvailability(totalRequests, errorRequests),
		P95Latency:     p95,
		P99Latency:     p99,
		ErrorRate:      slo.CalculateErrorRate(totalRequests, errorRequests),
		RequestsPerSec: avgRPS,
		TotalRequests:  totalRequests,
	}
}
