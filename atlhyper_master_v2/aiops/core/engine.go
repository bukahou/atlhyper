// atlhyper_master_v2/aiops/core/engine.go
// AIOps 引擎核心: OnSnapshot 编排图更新 + 基线检测 + 风险评分 + 状态机评估
package core

import (
	"context"
	"sort"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/baseline"
	"AtlHyper/atlhyper_master_v2/aiops/correlator"
	"AtlHyper/atlhyper_master_v2/aiops/incident"
	"AtlHyper/atlhyper_master_v2/aiops/risk"
	"AtlHyper/atlhyper_master_v2/aiops/statemachine"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps")

// engine AIOps 引擎实现
// 同时实现 statemachine.TransitionCallback 接口
type engine struct {
	store          datahub.Store
	corr           *correlator.Correlator
	stateManager   *baseline.StateManager
	scorer         *risk.Scorer
	sm             *statemachine.StateMachine
	incidentStore  *incident.Store
	graphRepo      database.AIOpsGraphRepository
	sloServiceRepo database.SLOServiceRepository
	sloRepo        database.SLORepository

	// 异常结果缓存（供风险详情查询）
	anomalyCache map[string][]*aiops.AnomalyResult // clusterID -> anomalies
	anomalyMu    sync.RWMutex

	flushInterval time.Duration
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// OnSnapshot 快照更新时触发
func (e *engine) OnSnapshot(clusterID string) {
	snap, err := e.store.GetSnapshot(clusterID)
	if err != nil || snap == nil {
		return
	}

	// 1. 构建并更新依赖图
	graph := correlator.BuildFromSnapshot(clusterID, snap)
	e.corr.Update(clusterID, graph)

	// 持久化图快照（异步，不阻塞主流程）
	go func() {
		data, err := correlator.Serialize(graph)
		if err != nil {
			log.Error("序列化依赖图失败", "cluster", clusterID, "err", err)
			return
		}
		if err := e.graphRepo.Save(context.Background(), clusterID, data); err != nil {
			log.Error("持久化依赖图失败", "cluster", clusterID, "err", err)
		}
	}()

	// 2. 提取指标并进行基线检测
	points := baseline.ExtractMetrics(clusterID, snap, e.sloServiceRepo, e.sloRepo)
	if len(points) == 0 {
		return
	}

	results := e.stateManager.Update(points)

	// 缓存异常结果
	e.anomalyMu.Lock()
	e.anomalyCache[clusterID] = results
	e.anomalyMu.Unlock()

	if len(results) > 0 {
		anomalyCount := 0
		for _, r := range results {
			if r.IsAnomaly {
				anomalyCount++
			}
		}
		if anomalyCount > 0 {
			log.Debug("检测到异常", "cluster", clusterID, "anomalies", anomalyCount)
		}

		// 3. 风险评分
		clusterRisk := e.scorer.Calculate(clusterID, graph, results, e.buildSLOContext(clusterID))
		if clusterRisk != nil {
			log.Debug("集群风险评分", "cluster", clusterID,
				"risk", clusterRisk.Risk, "level", clusterRisk.Level,
				"anomalies", clusterRisk.AnomalyCount)
		}

		// 4. 状态机评估
		if e.sm != nil {
			entityRisks := e.scorer.GetEntityRiskMap(clusterID)
			e.sm.Evaluate(context.Background(), clusterID, entityRisks, clusterRisk)
		}
	}
}

// ==================== TransitionCallback 实现 ====================

// OnWarningCreated 创建 Warning 事件
func (e *engine) OnWarningCreated(ctx context.Context, clusterID, entityKey string, risk *aiops.EntityRisk, now time.Time) string {
	return e.incidentStore.Create(ctx, clusterID, entityKey, risk, now)
}

// OnStateEscalated 事件升级
func (e *engine) OnStateEscalated(ctx context.Context, incidentID string, state aiops.EntityState, risk *aiops.EntityRisk, now time.Time) {
	e.incidentStore.UpdateState(ctx, incidentID, state, risk, now)
}

// OnRecoveryStarted 开始恢复
func (e *engine) OnRecoveryStarted(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time) {
	e.incidentStore.UpdateState(ctx, incidentID, aiops.StateRecovery, risk, now)
}

// OnRecurrence 事件复发
func (e *engine) OnRecurrence(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time) {
	e.incidentStore.IncrementRecurrence(ctx, incidentID, risk, now)
	e.incidentStore.UpdateState(ctx, incidentID, aiops.StateWarning, risk, now)
}

// OnStable 事件稳定（关闭）
func (e *engine) OnStable(ctx context.Context, incidentID string, entityKey string, now time.Time) {
	e.incidentStore.Resolve(ctx, incidentID, entityKey, now)
}

// ==================== 查询方法 ====================

// GetGraph 获取指定集群的依赖图
func (e *engine) GetGraph(clusterID string) *aiops.DependencyGraph {
	return e.corr.GetGraph(clusterID)
}

// GetGraphTrace 追踪指定实体的上下游链路
func (e *engine) GetGraphTrace(clusterID, fromKey, direction string, maxDepth int) *aiops.TraceResult {
	return e.corr.Trace(clusterID, fromKey, direction, maxDepth)
}

// GetBaseline 获取指定实体的基线状态
func (e *engine) GetBaseline(entityKey string) *aiops.EntityBaseline {
	return e.stateManager.GetStates(entityKey)
}

// GetClusterRisk 获取集群风险评分
func (e *engine) GetClusterRisk(clusterID string) *aiops.ClusterRisk {
	return e.scorer.GetClusterRisk(clusterID)
}

// GetEntityRisks 获取实体风险列表
func (e *engine) GetEntityRisks(clusterID, sortBy string, limit int) []*aiops.EntityRisk {
	return e.scorer.GetEntityRisks(clusterID, sortBy, limit)
}

// GetEntityRisk 获取单个实体的风险详情
func (e *engine) GetEntityRisk(clusterID, entityKey string) *aiops.EntityRiskDetail {
	entityRisk := e.scorer.GetEntityRisk(clusterID, entityKey)
	if entityRisk == nil {
		return nil
	}

	// 获取该实体的异常指标
	var metrics []*aiops.AnomalyResult
	e.anomalyMu.RLock()
	for _, a := range e.anomalyCache[clusterID] {
		if a.EntityKey == entityKey {
			metrics = append(metrics, a)
		}
	}
	e.anomalyMu.RUnlock()

	// 获取传播路径
	propagation := e.scorer.GetPropagationPaths(clusterID, entityKey)

	// 构建因果链（按时间排序）
	var causalChain []*aiops.CausalEntry
	e.anomalyMu.RLock()
	for _, a := range e.anomalyCache[clusterID] {
		if a.IsAnomaly {
			causalChain = append(causalChain, &aiops.CausalEntry{
				EntityKey:  a.EntityKey,
				MetricName: a.MetricName,
				Deviation:  a.Deviation,
				DetectedAt: a.DetectedAt,
			})
		}
	}
	e.anomalyMu.RUnlock()
	sort.Slice(causalChain, func(i, j int) bool {
		return causalChain[i].DetectedAt < causalChain[j].DetectedAt
	})

	return &aiops.EntityRiskDetail{
		EntityRisk:  *entityRisk,
		Metrics:     metrics,
		Propagation: propagation,
		CausalChain: causalChain,
	}
}

// GetIncidents 查询事件列表
func (e *engine) GetIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error) {
	return e.incidentStore.GetIncidents(ctx, opts)
}

// GetIncidentDetail 获取事件详情
func (e *engine) GetIncidentDetail(ctx context.Context, incidentID string) *aiops.IncidentDetail {
	return e.incidentStore.GetIncident(ctx, incidentID)
}

// GetIncidentStats 获取事件统计
func (e *engine) GetIncidentStats(ctx context.Context, clusterID string, since time.Time) *aiops.IncidentStats {
	return e.incidentStore.GetStats(ctx, clusterID, since)
}

// GetIncidentPatterns 获取历史事件模式
func (e *engine) GetIncidentPatterns(ctx context.Context, entityKey string, since time.Time) []*aiops.IncidentPattern {
	return e.incidentStore.GetPatterns(ctx, entityKey, since)
}

// ==================== 生命周期 ====================

// Start 启动引擎
func (e *engine) Start(ctx context.Context) error {
	// 1. 从数据库恢复基线状态
	if err := e.stateManager.LoadFromDB(ctx); err != nil {
		log.Warn("加载基线状态失败", "err", err)
	}

	// 2. 从数据库恢复依赖图
	clusterIDs, err := e.graphRepo.ListClusterIDs(ctx)
	if err != nil {
		log.Warn("加载依赖图集群列表失败", "err", err)
	}
	for _, cid := range clusterIDs {
		data, err := e.graphRepo.Load(ctx, cid)
		if err != nil || data == nil {
			continue
		}
		graph, err := correlator.Deserialize(data)
		if err != nil {
			log.Warn("反序列化依赖图失败", "cluster", cid, "err", err)
			continue
		}
		e.corr.Update(cid, graph)
	}
	log.Info("依赖图恢复完成", "clusters", len(clusterIDs))

	// 3. 启动定期 flush + Recovery→Stable 检查 goroutine
	bgCtx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	e.wg.Add(1)
	go e.flushLoop(bgCtx)

	if e.sm != nil {
		e.wg.Add(1)
		go e.recoveryCheckLoop(bgCtx)
	}

	log.Info("AIOps 引擎已启动", "flushInterval", e.flushInterval)
	return nil
}

// Stop 停止引擎
func (e *engine) Stop() error {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()

	// 最终 flush
	if err := e.stateManager.FlushToDB(context.Background()); err != nil {
		log.Error("最终 flush 基线状态失败", "err", err)
		return err
	}

	log.Info("AIOps 引擎已停止")
	return nil
}

// ==================== 内部方法 ====================

// buildSLOContext 从 SLO 仓库构建 SLO 上下文
func (e *engine) buildSLOContext(clusterID string) *risk.SLOContext {
	if e.sloRepo == nil {
		return nil
	}

	ctx := context.Background()

	// 获取集群所有 SLO 目标
	targets, err := e.sloRepo.GetTargets(ctx, clusterID)
	if err != nil || len(targets) == 0 {
		return nil
	}

	now := time.Now()
	var maxBurnRate float64
	var latestErrorRate, previousErrorRate float64
	hasData := false

	for _, t := range targets {
		// 获取最近 10 分钟的原始指标
		recent, err := e.sloRepo.GetRawMetrics(ctx, clusterID, t.Host, now.Add(-10*time.Minute), now)
		if err != nil || len(recent) == 0 {
			continue
		}

		// 汇总最近指标
		var totalReqs, errorReqs int64
		for _, r := range recent {
			totalReqs += r.TotalRequests
			errorReqs += r.ErrorRequests
		}
		if totalReqs == 0 {
			continue
		}

		actualErrorRate := float64(errorReqs) / float64(totalReqs)
		errorBudget := 1.0 - t.AvailabilityTarget/100.0
		if errorBudget <= 0 {
			errorBudget = 0.001
		}

		burnRate := actualErrorRate / errorBudget
		if burnRate > maxBurnRate {
			maxBurnRate = burnRate
		}

		if !hasData {
			latestErrorRate = actualErrorRate
			hasData = true
		}

		// 获取前一个 10 分钟窗口的数据计算增长率
		prev, err := e.sloRepo.GetRawMetrics(ctx, clusterID, t.Host, now.Add(-20*time.Minute), now.Add(-10*time.Minute))
		if err != nil || len(prev) == 0 {
			continue
		}
		var prevTotal, prevError int64
		for _, r := range prev {
			prevTotal += r.TotalRequests
			prevError += r.ErrorRequests
		}
		if prevTotal > 0 {
			previousErrorRate = float64(prevError) / float64(prevTotal)
		}
	}

	if !hasData {
		return nil
	}

	var errorGrowthRate float64
	if previousErrorRate > 0 {
		errorGrowthRate = (latestErrorRate - previousErrorRate) / previousErrorRate
	}

	return &risk.SLOContext{
		MaxBurnRate:     maxBurnRate,
		ErrorGrowthRate: errorGrowthRate,
	}
}

// flushLoop 定期将脏状态写入数据库
func (e *engine) flushLoop(ctx context.Context) {
	defer e.wg.Done()
	ticker := time.NewTicker(e.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.stateManager.FlushToDB(ctx); err != nil {
				log.Error("定期 flush 基线状态失败", "err", err)
			}
		}
	}
}

// recoveryCheckLoop 定期检查 Recovery 状态的实体是否可以转为 Stable
func (e *engine) recoveryCheckLoop(ctx context.Context) {
	defer e.wg.Done()
	ticker := time.NewTicker(10 * time.Minute) // 每 10 分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.sm.CheckRecoveryToStable(ctx)
		}
	}
}
