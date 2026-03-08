// atlhyper_master_v2/aiops/ai/background.go
// 后台自动分析触发器
// 监听事件创建/升级，根据严重度阈值自动触发 AI 分析
package ai

import (
	"context"
	"sync"
	"time"

	db "AtlHyper/atlhyper_master_v2/database"
)

// severityOrder 严重度排序（越高越严重）
var severityOrder = map[string]int{
	"low":      1,
	"medium":   2,
	"high":     3,
	"critical": 4,
}

// backgroundTrigger 后台分析触发器
type backgroundTrigger struct {
	enhancer       *Enhancer
	budgetRepo     db.AIRoleBudgetRepository
	analysisCfg    *AnalysisConfig // analysis 配置（可选，设置后启用深度分析）

	queue chan triggerEvent
	seen  map[string]time.Time // incidentID → 最后触发时间
	mu    sync.Mutex
	done  chan struct{}
}

// triggerEvent 触发事件
type triggerEvent struct {
	IncidentID string
	Severity   string
	Trigger    string // "incident_created" / "state_escalated"
}

// newBackgroundTrigger 创建后台触发器
func newBackgroundTrigger(enhancer *Enhancer, budgetRepo db.AIRoleBudgetRepository) *backgroundTrigger {
	bt := &backgroundTrigger{
		enhancer:   enhancer,
		budgetRepo: budgetRepo,
		queue:      make(chan triggerEvent, 64),
		seen:       make(map[string]time.Time),
		done:       make(chan struct{}),
	}
	go bt.worker()
	return bt
}

// Submit 提交触发事件（非阻塞）
func (bt *backgroundTrigger) Submit(incidentID, severity, trigger string) {
	select {
	case bt.queue <- triggerEvent{
		IncidentID: incidentID,
		Severity:   severity,
		Trigger:    trigger,
	}:
	default:
		log.Warn("后台分析队列已满，丢弃事件", "incident", incidentID)
	}
}

// Stop 停止后台触发器
func (bt *backgroundTrigger) Stop() {
	close(bt.done)
}

// worker 后台处理协程
func (bt *backgroundTrigger) worker() {
	for {
		select {
		case <-bt.done:
			return
		case evt := <-bt.queue:
			bt.process(evt)
		}
	}
}

// process 处理单个触发事件
func (bt *backgroundTrigger) process(evt triggerEvent) {
	// 1. 去重：同一事件 5 分钟内不重复触发
	bt.mu.Lock()
	if last, ok := bt.seen[evt.IncidentID]; ok && time.Since(last) < 5*time.Minute {
		bt.mu.Unlock()
		log.Info("后台分析去重跳过（5分钟内已触发）", "incident", evt.IncidentID)
		return
	}
	bt.seen[evt.IncidentID] = time.Now()
	// 清理过期条目
	for id, t := range bt.seen {
		if time.Since(t) > 30*time.Minute {
			delete(bt.seen, id)
		}
	}
	bt.mu.Unlock()

	// 2. 查询预算配置中的最低严重度
	if bt.budgetRepo != nil {
		budget, err := bt.budgetRepo.Get(context.Background(), "background")
		if err == nil && budget != nil {
			minSeverity := budget.AutoTriggerMinSeverity
			if minSeverity == "off" {
				log.Info("后台分析已禁用（配置=off）", "incident", evt.IncidentID)
				return
			}
			if minSeverity != "" && !meetsMinSeverity(evt.Severity, minSeverity) {
				log.Info("严重度不足，跳过后台分析",
					"incident", evt.IncidentID,
					"severity", evt.Severity,
					"min", minSeverity)
				return
			}
		}
	}

	// 3. 预算前置检查（防止自动触发导致成本失控）
	if bt.budgetRepo != nil {
		budget, err := bt.budgetRepo.Get(context.Background(), "background")
		if err == nil && budget != nil && !isBudgetAvailable(budget) {
			log.Warn("后台分析预算已用尽，跳过", "incident", evt.IncidentID)
			return
		}
	}

	// 4. 触发 background 分析
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Info("后台自动分析触发", "incident", evt.IncidentID, "trigger", evt.Trigger)

	result, err := bt.enhancer.SummarizeBackground(ctx, evt.IncidentID, evt.Trigger)
	if err != nil {
		log.Warn("后台分析失败", "incident", evt.IncidentID, "err", err)
		return
	}

	log.Info("后台分析完成", "incident", evt.IncidentID, "summary_len", len(result.Summary))

	// 5. 检查是否需要触发 analysis 深度分析
	bt.maybeTriggerAnalysis(evt)
}

// SetAnalysisConfig 设置 analysis 配置（启用深度分析）
func (bt *backgroundTrigger) SetAnalysisConfig(cfg *AnalysisConfig) {
	bt.analysisCfg = cfg
}

// maybeTriggerAnalysis 检查是否需要触发 analysis 深度分析
func (bt *backgroundTrigger) maybeTriggerAnalysis(evt triggerEvent) {
	if bt.analysisCfg == nil {
		return
	}
	if bt.budgetRepo == nil {
		return
	}

	// 读取 analysis 角色的自动触发阈值
	budget, err := bt.budgetRepo.Get(context.Background(), "analysis")
	if err != nil || budget == nil {
		return
	}

	minSeverity := budget.AutoTriggerMinSeverity
	if minSeverity == "" || minSeverity == "off" {
		return
	}

	if !meetsMinSeverity(evt.Severity, minSeverity) {
		return
	}

	// analysis 预算前置检查
	if !isBudgetAvailable(budget) {
		log.Warn("analysis 预算已用尽，跳过深度分析", "incident", evt.IncidentID)
		return
	}

	log.Info("触发 analysis 深度分析",
		"incident", evt.IncidentID,
		"severity", evt.Severity,
		"min_severity", minSeverity)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), analysisTimeout)
		defer cancel()

		if err := RunAnalysis(ctx, *bt.analysisCfg, evt.IncidentID, "auto_escalation"); err != nil {
			log.Warn("analysis 深度分析失败", "incident", evt.IncidentID, "err", err)
		}
	}()
}

// meetsMinSeverity 检查事件严重度是否达到最低阈值
func meetsMinSeverity(severity, minSeverity string) bool {
	return severityOrder[severity] >= severityOrder[minSeverity]
}

// isBudgetAvailable 检查预算是否还有余额（多维度：input/output × 日/月）
// 0 = 无限制的语义保留
func isBudgetAvailable(budget *db.AIRoleBudget) bool {
	if budget == nil {
		return true
	}
	// 日限额检查
	if budget.DailyInputTokenLimit > 0 && budget.DailyInputTokensUsed >= budget.DailyInputTokenLimit {
		return false
	}
	if budget.DailyOutputTokenLimit > 0 && budget.DailyOutputTokensUsed >= budget.DailyOutputTokenLimit {
		return false
	}
	if budget.DailyCallLimit > 0 && budget.DailyCallsUsed >= budget.DailyCallLimit {
		return false
	}
	// 月限额检查
	if budget.MonthlyInputTokenLimit > 0 && budget.MonthlyInputTokensUsed >= budget.MonthlyInputTokenLimit {
		return false
	}
	if budget.MonthlyOutputTokenLimit > 0 && budget.MonthlyOutputTokensUsed >= budget.MonthlyOutputTokenLimit {
		return false
	}
	if budget.MonthlyCallLimit > 0 && budget.MonthlyCallsUsed >= budget.MonthlyCallLimit {
		return false
	}
	return true
}
