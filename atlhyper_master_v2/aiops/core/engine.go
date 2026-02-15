// atlhyper_master_v2/aiops/core/engine.go
// AIOps 引擎核心: OnSnapshot 编排图更新 + 基线检测
package core

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/baseline"
	"AtlHyper/atlhyper_master_v2/aiops/correlator"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps")

// engine AIOps 引擎实现
type engine struct {
	store          datahub.Store
	corr           *correlator.Correlator
	stateManager   *baseline.StateManager
	graphRepo      database.AIOpsGraphRepository
	sloServiceRepo database.SLOServiceRepository
	sloRepo        database.SLORepository

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
	}
}

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

	// 3. 启动定期 flush goroutine
	flushCtx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.wg.Add(1)
	go e.flushLoop(flushCtx)

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
