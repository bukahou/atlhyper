// atlhyper_master_v2/aiops/ai/enhancer.go
// AIOps AI 增强服务
// 独立于 AIOpsEngine（单向依赖: aiops/ai → aiops）
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps-AI")

// LLMClientMeta 客户端元数据（工厂函数返回）
type LLMClientMeta struct {
	ProviderID   int64
	ProviderName string
	Model        string
}

// LLMClientFactory 创建 LLM 客户端的工厂函数
// 每次调用返回新实例 + 有效上下文窗口(tokens) + Provider 元数据，调用方负责 Close
type LLMClientFactory func(ctx context.Context) (llm.LLMClient, int, *LLMClientMeta, error)

// SummarizeResponse 事件摘要响应
type SummarizeResponse struct {
	IncidentID       string           `json:"incidentId"`
	Summary          string           `json:"summary"`
	RootCauseAnalysis string          `json:"rootCauseAnalysis"`
	Recommendations  []Recommendation `json:"recommendations"`
	SimilarIncidents []SimilarMatch   `json:"similarIncidents"`
	GeneratedAt      int64            `json:"generatedAt"`
	ReportID         int64            `json:"reportId,omitempty"`
	FromCache        bool             `json:"fromCache"`
}

// Recommendation 处置建议
type Recommendation struct {
	Priority    int    `json:"priority"`
	Action      string `json:"action"`
	Reason      string `json:"reason"`
	Impact      string `json:"impact"`
	IsAutomatic bool   `json:"isAutomatic"`
}

// SimilarMatch 相似事件匹配
type SimilarMatch struct {
	IncidentID string  `json:"incidentId"`
	Similarity float64 `json:"similarity"`
	RootCause  string  `json:"rootCause"`
	Resolution string  `json:"resolution"`
	OccurredAt string  `json:"occurredAt"`
}

// Token 预估常量
const (
	MaxPromptChars  = 16000 // Prompt 最大字符数（~4000 tokens）
	WarnPromptChars = 12000 // 超过此值记录 warning 日志
)

// 默认配置
const (
	defaultCooldown        = 60 * time.Second // Rate Limit 冷却时间
	defaultMaxCache        = 200              // 最大缓存条目数
	defaultCacheTTL        = 24 * time.Hour   // 缓存 TTL
	rateLimitExpiry        = 1 * time.Hour    // Rate Limit 条目过期清理阈值
	maxConcurrentAnalysis  = 3                // 最多同时进行的后台分析数
)

// 可缓存的事件状态（已结束，数据不再变化）
var cacheableStates = map[string]bool{
	"recovery": true,
	"stable":   true,
}

// cachedResult 缓存的 AI 分析结果
type cachedResult struct {
	response *SummarizeResponse
	cachedAt time.Time
}

// RecordUsageFunc 记录 AI 调用消耗的回调函数
type RecordUsageFunc func(ctx context.Context, role string, providerID int64, inputTokens, outputTokens int)

// Enhancer AIOps AI 增強服务
type Enhancer struct {
	incidentRepo database.AIOpsIncidentRepository
	reportRepo   database.AIReportRepository // 报告持久化（可选）
	llmFactory   LLMClientFactory

	// 预算扣减回调（可选，由 master.go 注入）
	recordUsage RecordUsageFunc

	// 后台自动触发器（可选）
	bgTrigger *backgroundTrigger

	// 并发信号量：限制同时进行的后台分析数量
	concurrencySem chan struct{}

	// Rate Limit: 同一事件在 cooldown 内不允许重复调用 LLM
	rateMu    sync.Mutex
	lastCalls map[string]time.Time // incidentID → 上次调用时间
	cooldown  time.Duration

	// 缓存: 已完结事件的 AI 结果
	cacheMu  sync.RWMutex
	cache    map[string]*cachedResult
	maxCache int
	cacheTTL time.Duration
}

// NewEnhancer 创建 AI 增强服务
func NewEnhancer(
	incidentRepo database.AIOpsIncidentRepository,
	reportRepo database.AIReportRepository,
	llmFactory LLMClientFactory,
) *Enhancer {
	return &Enhancer{
		incidentRepo:   incidentRepo,
		reportRepo:     reportRepo,
		llmFactory:     llmFactory,
		concurrencySem: make(chan struct{}, maxConcurrentAnalysis),
		lastCalls:      make(map[string]time.Time),
		cooldown:       defaultCooldown,
		cache:          make(map[string]*cachedResult),
		maxCache:       defaultMaxCache,
		cacheTTL:       defaultCacheTTL,
	}
}

// SetRecordUsage 设置预算扣减回调
func (e *Enhancer) SetRecordUsage(fn RecordUsageFunc) {
	e.recordUsage = fn
}

// GetRecordUsage 获取预算扣减回调（供 AnalysisConfig 使用）
func (e *Enhancer) GetRecordUsage() RecordUsageFunc {
	return e.recordUsage
}

// EnableBackgroundTrigger 启用后台自动触发器
func (e *Enhancer) EnableBackgroundTrigger(budgetRepo database.AIRoleBudgetRepository) {
	e.bgTrigger = newBackgroundTrigger(e, budgetRepo)
	log.Info("后台自动分析触发器已启用")
}

// NotifyIncidentEvent 通知事件创建/升级（供 AIOps 引擎回调）
func (e *Enhancer) NotifyIncidentEvent(incidentID, severity, trigger string) {
	if e.bgTrigger == nil {
		return
	}
	e.bgTrigger.Submit(incidentID, severity, trigger)
}

// SetAnalysisConfig 配置 analysis 深度分析（启用 Tool Calling 循环）
func (e *Enhancer) SetAnalysisConfig(cfg *AnalysisConfig) {
	if e.bgTrigger != nil {
		e.bgTrigger.SetAnalysisConfig(cfg)
		log.Info("analysis 深度分析已配置")
	}
}

// TriggerAnalysis 手动触发深度分析（异步，直接调用 RunAnalysis，不经过 background 流程）
func (e *Enhancer) TriggerAnalysis(incidentID string) {
	if e.bgTrigger == nil || e.bgTrigger.analysisCfg == nil {
		log.Warn("深度分析未配置", "incident", incidentID)
		return
	}

	log.Info("手动触发深度分析", "incident", incidentID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), analysisTimeout)
		defer cancel()

		if err := RunAnalysis(ctx, *e.bgTrigger.analysisCfg, incidentID, "manual"); err != nil {
			log.Warn("手动深度分析失败", "incident", incidentID, "err", err)
		} else {
			log.Info("手动深度分析完成", "incident", incidentID)
		}
	}()
}

// Stop 停止后台任务
func (e *Enhancer) Stop() {
	if e.bgTrigger != nil {
		e.bgTrigger.Stop()
	}
}

// Summarize 生成事件 AI 摘要（用户手动触发）
// A4: 优先查询已有报告，无报告时才调 LLM
func (e *Enhancer) Summarize(ctx context.Context, incidentID string) (*SummarizeResponse, error) {
	// 优先查询已有 background 报告
	if e.reportRepo != nil {
		reports, err := e.reportRepo.ListByIncident(ctx, incidentID)
		if err == nil && len(reports) > 0 {
			// 取最新一条 background 报告
			for _, r := range reports {
				if r.Role == "background" {
					resp := reportToSummarizeResponse(r)
					return resp, nil
				}
			}
		}
	}

	// 无已有报告，调 LLM 生成
	result, incident, usage, meta, err := e.summarizeCore(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	e.saveReport(ctx, incident, result, "manual", meta, usage)
	// 扣减预算（手动触发也计入 background 角色）
	if e.recordUsage != nil && usage != nil {
		providerID := int64(0)
		if meta != nil {
			providerID = meta.ProviderID
		}
		e.recordUsage(ctx, "background", providerID, usage.InputTokens, usage.OutputTokens)
	}
	return result, nil
}

// reportToSummarizeResponse 将 DB 报告转换为 SummarizeResponse
func reportToSummarizeResponse(r *database.AIReport) *SummarizeResponse {
	resp := &SummarizeResponse{
		IncidentID:        r.IncidentID,
		Summary:           r.Summary,
		RootCauseAnalysis: r.RootCauseAnalysis,
		GeneratedAt:       r.CreatedAt.UnixMilli(),
		ReportID:          r.ID,
		FromCache:         true,
	}

	// 解析 JSON 字段
	if r.Recommendations != "" {
		var recs []Recommendation
		if err := json.Unmarshal([]byte(r.Recommendations), &recs); err == nil {
			resp.Recommendations = recs
		}
	}
	if resp.Recommendations == nil {
		resp.Recommendations = []Recommendation{}
	}

	if r.SimilarIncidents != "" {
		var sims []SimilarMatch
		if err := json.Unmarshal([]byte(r.SimilarIncidents), &sims); err == nil {
			resp.SimilarIncidents = sims
		}
	}
	if resp.SimilarIncidents == nil {
		resp.SimilarIncidents = []SimilarMatch{}
	}

	return resp
}

// SummarizeBackground 后台自动分析（指定 trigger 类型）
func (e *Enhancer) SummarizeBackground(ctx context.Context, incidentID, trigger string) (*SummarizeResponse, error) {
	// 并发上限检查
	select {
	case e.concurrencySem <- struct{}{}:
		defer func() { <-e.concurrencySem }()
	default:
		return nil, fmt.Errorf("后台分析并发上限（%d），跳过", maxConcurrentAnalysis)
	}

	result, incident, usage, meta, err := e.summarizeCore(ctx, incidentID)
	if err != nil {
		return nil, err
	}
	e.saveReport(ctx, incident, result, trigger, meta, usage)
	// 扣减预算
	if e.recordUsage != nil && usage != nil {
		providerID := int64(0)
		if meta != nil {
			providerID = meta.ProviderID
		}
		e.recordUsage(ctx, "background", providerID, usage.InputTokens, usage.OutputTokens)
	}
	return result, nil
}

// summarizeCore 核心分析流程
// 流程: 缓存查询 → Rate Limit → 查数据 → 构建上下文 → Token 预估 → LLM → 写缓存
// 返回: 结果, 事件, usage, meta, 错误
func (e *Enhancer) summarizeCore(ctx context.Context, incidentID string) (*SummarizeResponse, *database.AIOpsIncident, *llm.Usage, *LLMClientMeta, error) {
	// 1. 查缓存（命中则直接返回，不计 rate limit）
	if cached := e.getCache(incidentID); cached != nil {
		log.Debug("缓存命中", "incident", incidentID)
		return cached, nil, nil, nil, nil
	}

	// 2. Rate Limit 检查
	if err := e.checkRateLimit(incidentID); err != nil {
		return nil, nil, nil, nil, err
	}

	// 3. 查询事件数据
	incident, err := e.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("查询事件失败: %w", err)
	}
	if incident == nil {
		return nil, nil, nil, nil, fmt.Errorf("事件不存在: %s", incidentID)
	}

	entities, err := e.incidentRepo.GetEntities(ctx, incidentID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("查询受影响实体失败: %w", err)
	}

	timeline, err := e.incidentRepo.GetTimeline(ctx, incidentID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("查询时间线失败: %w", err)
	}

	// 4. 查询历史相似事件（根因实体的历史事件，90天内）
	var historical []*database.AIOpsIncident
	if incident.RootCause != "" {
		since := time.Now().Add(-90 * 24 * time.Hour)
		historical, _ = e.incidentRepo.ListByEntity(ctx, incident.RootCause, since)
	}

	// 5. 调用 LLM（先获取 client 和 context_window，再构建 prompt）
	client, contextWindow, meta, err := e.llmFactory(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}
	defer client.Close()

	// 6. 构建 Prompt + Token 预估截断（感知模型上下文窗口）
	prompt := e.buildPromptWithTruncation(incident, entities, timeline, historical, contextWindow)

	stream, err := client.ChatStream(ctx, &llm.Request{
		SystemPrompt: prompt.System,
		Messages:     []llm.Message{{Role: "user", Content: prompt.User}},
	})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 7. 收集完整响应
	fullText, usage := collectResponse(stream)

	log.Debug("LLM 响应", "incident", incidentID, "len", len(fullText))

	// 8. 解析结构化输出
	result, err := parseResponse(fullText, incidentID, historical)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// 9. 更新 rate limit 时间戳
	e.recordCall(incidentID)

	// 10. 若事件已完结（recovery/stable），写入缓存
	if cacheableStates[incident.State] {
		e.setCache(incidentID, result)
	}

	return result, incident, usage, meta, nil
}

// ==================== Rate Limit ====================

// checkRateLimit 检查同一事件的调用冷却时间
func (e *Enhancer) checkRateLimit(incidentID string) error {
	e.rateMu.Lock()
	defer e.rateMu.Unlock()

	now := time.Now()

	// 惰性清理过期条目（> 1h）
	for id, t := range e.lastCalls {
		if now.Sub(t) > rateLimitExpiry {
			delete(e.lastCalls, id)
		}
	}

	if last, ok := e.lastCalls[incidentID]; ok {
		remaining := e.cooldown - now.Sub(last)
		if remaining > 0 {
			return fmt.Errorf("请等待 %d 秒后再试", int(remaining.Seconds())+1)
		}
	}

	return nil
}

// recordCall 记录调用时间戳（LLM 调用成功后）
func (e *Enhancer) recordCall(incidentID string) {
	e.rateMu.Lock()
	defer e.rateMu.Unlock()
	e.lastCalls[incidentID] = time.Now()
}

// ==================== 结果缓存 ====================

// getCache 查询缓存（TTL 过期则淘汰）
func (e *Enhancer) getCache(incidentID string) *SummarizeResponse {
	e.cacheMu.RLock()
	entry, ok := e.cache[incidentID]
	e.cacheMu.RUnlock()

	if !ok {
		return nil
	}

	// TTL 过期
	if time.Since(entry.cachedAt) > e.cacheTTL {
		e.cacheMu.Lock()
		delete(e.cache, incidentID)
		e.cacheMu.Unlock()
		return nil
	}

	return entry.response
}

// setCache 写入缓存（满时淘汰最旧条目）
func (e *Enhancer) setCache(incidentID string, resp *SummarizeResponse) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()

	// 满时淘汰最旧条目
	if len(e.cache) >= e.maxCache {
		var oldestID string
		var oldestTime time.Time
		for id, entry := range e.cache {
			if oldestID == "" || entry.cachedAt.Before(oldestTime) {
				oldestID = id
				oldestTime = entry.cachedAt
			}
		}
		if oldestID != "" {
			delete(e.cache, oldestID)
		}
	}

	e.cache[incidentID] = &cachedResult{
		response: resp,
		cachedAt: time.Now(),
	}
}

// ==================== Token 预估 ====================

// buildPromptWithTruncation 构建 Prompt，超限时逐步截断
// contextWindow: Provider 上下文窗口 (tokens), 0 = 使用默认限制
func (e *Enhancer) buildPromptWithTruncation(
	incident *database.AIOpsIncident,
	entities []*database.AIOpsIncidentEntity,
	timeline []*database.AIOpsIncidentTimeline,
	historical []*database.AIOpsIncident,
	contextWindow int,
) *PromptPair {
	// 根据 context_window 动态调整 Prompt 最大字符数
	maxChars := MaxPromptChars
	if contextWindow > 0 {
		maxChars = maxPromptCharsForContext(contextWindow)
	}
	warnChars := maxChars * 3 / 4

	incidentCtx := BuildIncidentContext(incident, entities, timeline, historical)
	prompt := SummarizePrompt(incidentCtx)
	totalChars := len(prompt.System) + len(prompt.User)

	if totalChars > warnChars {
		log.Warn("Prompt 较长", "chars", totalChars, "warn_threshold", warnChars)
	}

	if totalChars <= maxChars {
		return prompt
	}

	// 超限 → 逐步截断: historical → timeline → entities
	log.Warn("Prompt 超限，开始截断", "chars", totalChars, "max", maxChars)

	truncSteps := []struct {
		hist int
		tl   int
		ent  int
	}{
		{len(historical) / 2, len(timeline), len(entities)},       // 砍半 historical
		{len(historical) / 2, len(timeline) / 2, len(entities)},   // 再砍半 timeline
		{len(historical) / 2, len(timeline) / 2, len(entities) / 2}, // 再砍半 entities
		{0, len(timeline) / 2, len(entities) / 2},                  // 清空 historical
		{0, 0, len(entities) / 2},                                   // 清空 timeline
		{0, 0, 0},                                                   // 全清（兜底）
	}

	for _, step := range truncSteps {
		h := truncateSlice(historical, step.hist)
		t := truncateSlice(timeline, step.tl)
		en := truncateSlice(entities, step.ent)

		rebuilt := BuildIncidentContext(incident, en, t, h)
		prompt = SummarizePrompt(rebuilt)
		totalChars = len(prompt.System) + len(prompt.User)

		if totalChars <= maxChars {
			log.Info("Prompt 截断成功", "chars", totalChars,
				"historical", len(h), "timeline", len(t), "entities", len(en))
			return prompt
		}
	}

	log.Warn("Prompt 截断后仍超限（兜底返回）", "chars", totalChars)
	return prompt
}

// truncateSlice 安全截断切片到指定长度
func truncateSlice[T any](s []T, maxLen int) []T {
	if maxLen <= 0 || len(s) == 0 {
		return nil
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// maxPromptCharsForContext 根据 context_window 计算 Prompt 最大字符数
func maxPromptCharsForContext(contextWindow int) int {
	if contextWindow <= 0 {
		return MaxPromptChars
	}
	// 上下文窗口的 50% 给 prompt（另 50% 给输出 + overhead）
	// 1 token ≈ 2.5 chars
	chars := contextWindow / 2 * 25 / 10
	if chars < 2000 {
		return 2000
	}
	return chars
}

// collectResponse 收集流式响应为完整文本和 usage 信息
func collectResponse(stream <-chan *llm.Chunk) (string, *llm.Usage) {
	var b strings.Builder
	for chunk := range stream {
		switch chunk.Type {
		case llm.ChunkText:
			b.WriteString(chunk.Content)
		case llm.ChunkError:
			if chunk.Error != nil {
				log.Warn("LLM 流错误", "err", chunk.Error)
			}
		case llm.ChunkDone:
			return b.String(), chunk.Usage
		}
	}
	return b.String(), nil
}

// parseResponse 解析 LLM 响应为结构化结果
func parseResponse(raw string, incidentID string, historical []*database.AIOpsIncident) (*SummarizeResponse, error) {
	jsonStr := extractJSON(raw)

	var parsed struct {
		Summary           string `json:"summary"`
		RootCauseAnalysis string `json:"rootCauseAnalysis"`
		Recommendations   []struct {
			Priority int    `json:"priority"`
			Action   string `json:"action"`
			Reason   string `json:"reason"`
			Impact   string `json:"impact"`
		} `json:"recommendations"`
		SimilarPattern string `json:"similarPattern"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		// 降级：整段文本作为摘要
		return &SummarizeResponse{
			IncidentID:       incidentID,
			Summary:          raw,
			SimilarIncidents: buildSimilarMatches(historical),
			GeneratedAt:      time.Now().UnixMilli(),
		}, nil
	}

	recommendations := make([]Recommendation, len(parsed.Recommendations))
	for i, r := range parsed.Recommendations {
		recommendations[i] = Recommendation{
			Priority:    r.Priority,
			Action:      r.Action,
			Reason:      r.Reason,
			Impact:      r.Impact,
			IsAutomatic: false,
		}
	}

	return &SummarizeResponse{
		IncidentID:        incidentID,
		Summary:           parsed.Summary,
		RootCauseAnalysis: parsed.RootCauseAnalysis,
		Recommendations:   recommendations,
		SimilarIncidents:  buildSimilarMatches(historical),
		GeneratedAt:       time.Now().UnixMilli(),
	}, nil
}

// extractJSON 从 LLM 输出中提取 JSON 内容
// LLM 可能在 JSON 前后添加 markdown 代码块标记
func extractJSON(text string) string {
	// 尝试提取 ```json ... ``` 中的内容
	if start := strings.Index(text, "```json"); start != -1 {
		jsonStart := start + 7
		if end := strings.Index(text[jsonStart:], "```"); end != -1 {
			return strings.TrimSpace(text[jsonStart : jsonStart+end])
		}
	}

	// 尝试提取 ``` ... ``` 中的内容
	if start := strings.Index(text, "```"); start != -1 {
		codeStart := start + 3
		if end := strings.Index(text[codeStart:], "```"); end != -1 {
			candidate := strings.TrimSpace(text[codeStart : codeStart+end])
			if len(candidate) > 0 && candidate[0] == '{' {
				return candidate
			}
		}
	}

	// 尝试直接找 JSON 对象
	if start := strings.Index(text, "{"); start != -1 {
		if end := strings.LastIndex(text, "}"); end > start {
			return text[start : end+1]
		}
	}

	return text
}

// saveReport 持久化 AI 分析报告（fire-and-forget，不影响主流程）
func (e *Enhancer) saveReport(ctx context.Context, incident *database.AIOpsIncident, result *SummarizeResponse, trigger string, meta *LLMClientMeta, usage *llm.Usage) {
	if e.reportRepo == nil {
		return
	}

	recsJSON, _ := json.Marshal(result.Recommendations)
	similarsJSON, _ := json.Marshal(result.SimilarIncidents)

	report := &database.AIReport{
		IncidentID:        incident.ID,
		ClusterID:         incident.ClusterID,
		Role:              "background",
		Trigger:           trigger,
		Summary:           result.Summary,
		RootCauseAnalysis: result.RootCauseAnalysis,
		Recommendations:   string(recsJSON),
		SimilarIncidents:  string(similarsJSON),
		CreatedAt:         time.Now(),
	}

	// 填充 Provider 元数据
	if meta != nil {
		report.ProviderName = meta.ProviderName
		report.Model = meta.Model
	}
	if usage != nil {
		report.InputTokens = usage.InputTokens
		report.OutputTokens = usage.OutputTokens
	}

	if err := e.reportRepo.Create(ctx, report); err != nil {
		log.Warn("保存 AI 报告失败", "incident", incident.ID, "err", err)
	}
}

// buildSimilarMatches 从历史事件构建相似事件列表
func buildSimilarMatches(historical []*database.AIOpsIncident) []SimilarMatch {
	if len(historical) == 0 {
		return []SimilarMatch{}
	}

	matches := make([]SimilarMatch, 0, len(historical))
	for i, inc := range historical {
		// 简单相似度：越近的事件相似度越高
		similarity := 0.9 - float64(i)*0.1
		if similarity < 0.3 {
			similarity = 0.3
		}
		matches = append(matches, SimilarMatch{
			IncidentID: inc.ID,
			Similarity: similarity,
			RootCause:  inc.RootCause,
			OccurredAt: inc.StartedAt.Format(time.RFC3339),
		})
	}
	return matches
}
