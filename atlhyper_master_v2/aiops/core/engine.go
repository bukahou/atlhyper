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
	"AtlHyper/atlhyper_master_v2/aiops/subject"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	slocalc "AtlHyper/atlhyper_master_v2/slo"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/slo"
)

var log = logger.Module("AIOps")

// engine AIOps 引擎实现
// 同时实现 statemachine.TransitionHandler 接口
type engine struct {
	store         datahub.Store
	corr          *correlator.Correlator
	stateManager  *baseline.StateManager
	scorer        *risk.Scorer
	sm            *statemachine.StateMachine
	incidentStore *incident.Store
	graphRepo     database.AIOpsGraphRepository
	graphSaver    *graphSnapshotSaver // Start 时创建
	sloRepo       database.SLORepository

	// 事件进行中累计证据的节流（同一事件每分钟最多一次）
	signalThrottle signalThrottle

	// 事件变化的订阅者（Start 之前注册，之后只读）
	transitionSubscribers []func(aiops.IncidentTransition)

	// 拓扑归并：各集群当前有效的关联与关联变化的订阅者
	links           linkState
	linkSubscribers []func(aiops.IncidentLink)

	// 异常结果缓存（供风险详情查询）
	anomalyCache map[string][]*aiops.AnomalyResult // clusterID -> anomalies
	anomalyMu    sync.RWMutex

	flushInterval time.Duration
	bgCtx         context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// SubscribeTransitions 订阅事件变化（须在 Start 之前调用）
func (e *engine) SubscribeTransitions(fn func(aiops.IncidentTransition)) {
	if fn == nil {
		return
	}
	e.transitionSubscribers = append(e.transitionSubscribers, fn)
}

// publishTransition 把一次事件变化发给全部订阅者
func (e *engine) publishTransition(t aiops.IncidentTransition) {
	for _, fn := range e.transitionSubscribers {
		fn(t)
	}
}

// OnSnapshot 快照更新时触发
func (e *engine) OnSnapshot(clusterID string) {
	snap, err := e.store.GetSnapshot(clusterID)
	if err != nil || snap == nil {
		return
	}

	// 补上此前没写成的事件写入（写锁被占住时积压的；平时队列为空）
	if e.incidentStore != nil && e.bgCtx != nil {
		e.incidentStore.FlushPending(e.bgCtx)
	}

	// 获取 OTel 数据（直接从 snap.OTel 读取，非 Ring Buffer）
	otel := snap.OTel

	// 1. 构建并更新依赖图
	graph := correlator.BuildFromSnapshot(clusterID, snap, otel)
	e.corr.Update(clusterID, graph)

	// 持久化图快照（异步，不阻塞主流程；拓扑没变不写）
	if e.graphSaver != nil {
		e.graphSaver.SaveIfChanged(clusterID, graph)
	}

	// otel 已在上方从 snap.OTel 获取（直接读取，非 Ring Buffer）

	// 2. 清理已不存在于快照中的实体基线状态（防止滚动更新后旧 Pod 状态残留）
	activeKeys := extractActiveEntityKeys(snap, otel)
	if removed := e.stateManager.CleanupStaleEntities(clusterID, activeKeys); removed > 0 {
		log.Debug("清理过期基线状态", "cluster", clusterID, "removed", removed)
	}

	// 3. 提取指标并进行基线检测（路径 A: EMA+3σ）
	points := baseline.ExtractMetrics(clusterID, snap, otel)
	if len(points) == 0 {
		return
	}

	results := e.stateManager.Update(clusterID, points)

	// 路径 B: 确定性异常直注（绕过冷启动）
	deterministicResults := baseline.ExtractDeterministicAnomalies(snap)
	// Enhanced: OTel 确定性异常（APM 高错误率/高延迟、日志错误尖峰）
	// otel==nil 时函数内部直接 return nil，不影响 Basic 层
	otelDeterministic := baseline.ExtractOTelDeterministicAnomalies(otel)
	deterministicResults = append(deterministicResults, otelDeterministic...)
	results = mergeAnomalyResults(results, deterministicResults)

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

		// 4. 状态机评估（以主体为单位）
		if e.sm != nil {
			e.evaluateSubjects(clusterID, snap, e.scorer.GetEntityRiskMap(clusterID), clusterRisk)
		}
	}
}

// evaluateSubjects 把实体风险按主体聚合后交给状态机。
// 风险计算、基线、依赖图仍以实体为单位；只有状态机与事件以主体为单位。
func (e *engine) evaluateSubjects(
	clusterID string,
	snap *cluster.ClusterSnapshot,
	entityRisks map[string]*aiops.EntityRisk,
	clusterRisk *aiops.ClusterRisk,
) {
	subjects := subject.ResolveSubjects(snap)
	subjectRisks := subject.AggregateBySubject(entityRisks, subjects)
	e.sm.Evaluate(e.bgCtx, clusterID, subjectRisks, clusterRisk)
	now := time.Now()
	e.recordActiveSignals(clusterID, subjectRisks, now)
	e.updateLinks(clusterID, subjects, now)
}

// ==================== TransitionHandler 实现 ====================

// OnTransition 状态机的每一次事件变化：先落库，再发给订阅者。
// 顺序不能反：订阅者（AI 后台分析、通知）拿到变化后会回头读这条事件。
func (e *engine) OnTransition(ctx context.Context, transition aiops.IncidentTransition, risk *aiops.SubjectRisk) {
	e.incidentStore.ApplyTransition(ctx, transition, risk)
	e.publishTransition(transition)
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
func (e *engine) GetBaseline(clusterID, entityKey string) *aiops.EntityBaseline {
	return e.stateManager.GetStates(clusterID, entityKey)
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

	// 构建因果树：以查询实体为中心，BFS 依赖图，收集关联异常实体
	causalTree := e.buildCausalTree(clusterID, entityKey)

	return &aiops.EntityRiskDetail{
		EntityRisk:  *entityRisk,
		Metrics:     metrics,
		Propagation: propagation,
		CausalChain: causalChain,
		CausalTree:  causalTree,
	}
}

// buildCausalTree 以查询实体为中心，利用依赖图构建因果树
// 上游：谁指向我（reverse adjacency），下游：我指向谁（adjacency）
// 总深度 ≤ 2（邻居 + 邻居的邻居），仅包含有异常指标的实体
func (e *engine) buildCausalTree(clusterID, entityKey string) []*aiops.CausalTreeNode {
	graph := e.corr.GetGraph(clusterID)
	if graph == nil {
		return nil
	}

	// 构建实体 → 异常指标索引
	anomalyIndex := make(map[string][]*aiops.AnomalyResult)
	e.anomalyMu.RLock()
	for _, a := range e.anomalyCache[clusterID] {
		if a.IsAnomaly {
			anomalyIndex[a.EntityKey] = append(anomalyIndex[a.EntityKey], a)
		}
	}
	e.anomalyMu.RUnlock()

	// 获取边类型索引：from|to → edgeType
	edgeTypeIndex := make(map[string]string)
	for _, edge := range graph.Edges {
		edgeTypeIndex[edge.From+"|"+edge.To] = edge.Type
	}

	entityRiskMap := e.scorer.GetEntityRiskMap(clusterID)
	visited := map[string]bool{entityKey: true}

	// 辅助函数：为邻居创建节点
	buildNode := func(neighborKey, edgeType, direction string, depth int) *aiops.CausalTreeNode {
		metrics := anomalyIndex[neighborKey]
		if len(metrics) == 0 && depth > 0 {
			return nil // 非直接邻居且无异常则跳过
		}

		node := &aiops.CausalTreeNode{
			EntityKey:  neighborKey,
			EntityType: aiops.ExtractEntityType(neighborKey),
			EdgeType:   edgeType,
			Direction:  direction,
			Metrics:    metrics,
		}
		if r := entityRiskMap[neighborKey]; r != nil {
			node.RFinal = r.RFinal
		}
		return node
	}

	var result []*aiops.CausalTreeNode

	// 上游：谁指向我（reverse adjacency）
	for _, upKey := range graph.Reverse()[entityKey] {
		if visited[upKey] {
			continue
		}
		visited[upKey] = true
		edgeType := edgeTypeIndex[upKey+"|"+entityKey]
		node := buildNode(upKey, edgeType, "upstream", 0)
		if node == nil {
			continue
		}

		// 上游的上游（depth=1）
		for _, upUpKey := range graph.Reverse()[upKey] {
			if visited[upUpKey] {
				continue
			}
			visited[upUpKey] = true
			et := edgeTypeIndex[upUpKey+"|"+upKey]
			child := buildNode(upUpKey, et, "upstream", 1)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}
		result = append(result, node)
	}

	// 下游：我指向谁（adjacency）
	for _, downKey := range graph.Adjacency()[entityKey] {
		if visited[downKey] {
			continue
		}
		visited[downKey] = true
		edgeType := edgeTypeIndex[entityKey+"|"+downKey]
		node := buildNode(downKey, edgeType, "downstream", 0)
		if node == nil {
			continue
		}

		// 下游的下游（depth=1）
		for _, downDownKey := range graph.Adjacency()[downKey] {
			if visited[downDownKey] {
				continue
			}
			visited[downDownKey] = true
			et := edgeTypeIndex[downKey+"|"+downDownKey]
			child := buildNode(downDownKey, et, "downstream", 1)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}
		result = append(result, node)
	}

	return result
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
func (e *engine) GetIncidentPatterns(ctx context.Context, clusterID, entityKey string, since time.Time) []*aiops.IncidentPattern {
	return e.incidentStore.GetPatterns(ctx, clusterID, entityKey, since)
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

	// 3. 从数据库恢复活跃事件到状态机
	if e.sm != nil {
		e.reloadActiveIncidents(ctx)
	}

	// 4. 启动定期 flush + Recovery→Stable 检查 goroutine
	e.bgCtx, e.cancel = context.WithCancel(context.Background())
	e.graphSaver = newGraphSnapshotSaver(e.bgCtx, e.graphRepo)

	// 5. 恢复拓扑归并的有效关联（订阅者据此重建抑制）
	e.reloadLinks()

	e.wg.Add(1)
	go e.flushLoop(e.bgCtx)

	if e.sm != nil {
		e.wg.Add(1)
		go e.recoveryCheckLoop(e.bgCtx)
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
	if e.graphSaver != nil {
		e.graphSaver.wait()
	}

	// 退出前再试一次积压的事件写入；仍写不进去的只能丢弃（队列只在内存里）
	if e.incidentStore != nil {
		e.incidentStore.FlushPending(context.Background())
		if left := e.incidentStore.PendingWrites(); left > 0 {
			log.Error("退出时仍有事件写入未完成，已丢弃", "pending", left)
		}
	}

	// 最终 flush
	if err := e.stateManager.FlushToDB(context.Background()); err != nil {
		log.Error("最终 flush 基线状态失败", "err", err)
		return err
	}

	log.Info("AIOps 引擎已停止")
	return nil
}

// ==================== 内部方法 ====================

// buildSLOContext 从 OTelSnapshot 构建 SLO 上下文
func (e *engine) buildSLOContext(clusterID string) *risk.SLOContext {
	if e.sloRepo == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(e.bgCtx, 5*time.Second)
	defer cancel()

	// 获取集群所有 SLO 目标
	targets, err := e.sloRepo.GetTargets(ctx, clusterID)
	if err != nil || len(targets) == 0 {
		return nil
	}

	// 获取最新 OTelSnapshot 中的 Ingress 数据
	timeline, _ := e.store.GetOTelTimeline(clusterID, time.Now().Add(-30*time.Second))
	if len(timeline) == 0 || timeline[len(timeline)-1].Snapshot == nil {
		return nil
	}
	otel := timeline[len(timeline)-1].Snapshot
	if len(otel.SLOIngress) == 0 {
		return nil
	}

	burn, hasData := maxBurnRate(targets, otel.SLOIngress)
	if !hasData {
		return nil
	}

	return &risk.SLOContext{
		MaxBurnRate: burn,
	}
}

// maxBurnRate 各 SLO 目标的错误预算燃烧率取最大值（没有任何目标对得上入口数据时 ok=false）。
// 目标按「域名」登记，入口按 IngressSLO.MatchesHost 找（与 SLO 页面同一规则）；
// 燃烧率用 slo.CalculateBurnRate（错误率与可用性目标都是百分数）
func maxBurnRate(targets []*database.SLOTarget, ingresses []slo.IngressSLO) (float64, bool) {
	var maxRate float64
	hasData := false
	for _, t := range targets {
		for i := range ingresses {
			if !ingresses[i].MatchesHost(t.Host) {
				continue
			}
			if burn := slocalc.CalculateBurnRate(ingresses[i].ErrorRate, t.AvailabilityTarget); burn > maxRate {
				maxRate = burn
			}
			hasData = true
		}
	}
	return maxRate, hasData
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

// mergeAnomalyResults 合并 EMA 路径和确定性路径的异常结果
// 同一 EntityKey+MetricName 时取 Score 更高者
func mergeAnomalyResults(emaResults []*aiops.AnomalyResult, deterministicResults []*aiops.AnomalyResult) []*aiops.AnomalyResult {
	if len(deterministicResults) == 0 {
		return emaResults
	}

	// 用 entityKey|metricName 做索引
	index := make(map[string]int, len(emaResults))
	for i, r := range emaResults {
		key := r.EntityKey + "|" + r.MetricName
		index[key] = i
	}

	for _, dr := range deterministicResults {
		key := dr.EntityKey + "|" + dr.MetricName
		if idx, exists := index[key]; exists {
			// 同指标：取 Score 更高者
			if dr.Score > emaResults[idx].Score {
				emaResults[idx] = dr
			}
		} else {
			emaResults = append(emaResults, dr)
			index[key] = len(emaResults) - 1
		}
	}

	return emaResults
}

// reloadActiveIncidents 从 DB 加载未关闭事件，按主体恢复到状态机。
// 条目里随事件走的各项（是否升级过、峰值、严重度、复发次数、发生时刻、观察期起点）都从库里取回
func (e *engine) reloadActiveIncidents(ctx context.Context) {
	incidents, _, err := e.incidentStore.GetIncidents(ctx, aiops.IncidentQueryOpts{})
	if err != nil {
		log.Warn("加载活跃事件失败", "err", err)
		return
	}

	count := 0
	now := time.Now()
	for _, inc := range incidents {
		if inc.State == aiops.StateStable {
			continue
		}
		if existing := e.sm.GetEntry(inc.ClusterID, inc.Subject); existing != nil {
			continue // 列表按开始时间倒序：同一主体只恢复最新的那条
		}
		entry := &aiops.StateMachineEntry{
			ClusterID:       inc.ClusterID,
			SubjectKey:      inc.Subject,
			CurrentState:    inc.State,
			IncidentID:      inc.ID,
			LastEvaluatedAt: now.Unix(),
			ReachedIncident: inc.EscalatedAt != nil,
			PeakRisk:        inc.PeakRisk,
			Severity:        inc.Severity,
			Recurrence:      inc.Recurrence,
			RootCause:       inc.RootCause,
			OccurrenceTimes: e.incidentStore.OccurrenceTimes(ctx, inc.ID, inc.StartedAt),
		}
		// 这一轮异常的起点 = 最近一次「发生」（建事件或复发）；时间线里取全量，不受 24 小时窗口影响
		if n := len(entry.OccurrenceTimes); n > 0 {
			entry.EpisodeStartedAt = entry.OccurrenceTimes[n-1]
		}
		if inc.State == aiops.StateRecovery {
			// 观察期的起点只在内存里，重启后从时间线取回；取不到时由状态机从现在起计
			if startedAt, ok := e.incidentStore.RecoveryStartedAt(ctx, inc.ID); ok {
				entry.RecoveryStartedAt = startedAt.Unix()
			}
		}
		e.sm.RestoreEntry(entry)
		count++
	}
	log.Info("活跃事件恢复到状态机", "count", count)
}

// extractActiveEntityKeys 从快照中提取当前所有活跃实体的 entityKey 集合
func extractActiveEntityKeys(snap *cluster.ClusterSnapshot, otel *cluster.OTelSnapshot) map[string]bool {
	keys := make(map[string]bool, len(snap.Pods)+len(snap.Nodes)+len(snap.Services))

	for i := range snap.Pods {
		pod := &snap.Pods[i]
		keys[aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)] = true
	}
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		keys[aiops.EntityKey(aiops.ClusterScope, "node", node.GetName())] = true
	}
	for i := range snap.Services {
		svc := &snap.Services[i]
		keys[aiops.EntityKey(svc.Summary.Namespace, "service", svc.Summary.Name)] = true
	}
	for i := range snap.Ingresses {
		ing := &snap.Ingresses[i]
		keys[aiops.EntityKey(ing.Summary.Namespace, "ingress", ing.Summary.Name)] = true
	}
	if otel != nil {
		for _, ing := range otel.SLOIngress {
			keys[aiops.IngressEntityKey(ing.ServiceKey)] = true
		}
		// Enhanced: APM 服务实体
		for _, svc := range otel.APMServices {
			keys[aiops.EntityKey(svc.Namespace, "service", svc.Name)] = true
		}
		// Enhanced: logs 虚拟实体
		if otel.LogsSummary != nil {
			keys[aiops.EntityKey(aiops.ClusterScope, "logs", "global")] = true
		}
	}
	return keys
}

// recoveryCheckLoop 定期关闭观察期已满的事件，并把失联的主体按「失联」关闭
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
			e.sm.CleanupStaleEntries(ctx, 30*time.Minute)
		}
	}
}
