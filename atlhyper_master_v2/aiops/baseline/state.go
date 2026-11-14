// atlhyper_master_v2/aiops/baseline/state.go
// 基线状态管理器（内存缓存 + 定期 flush DB + 启动恢复）
package baseline

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// baselineKey 一条基线的键：某个集群里某个实体的某个指标
type baselineKey struct {
	entity     aiops.EntityRef
	metricName string
}

// StateManager 基线状态管理器
type StateManager struct {
	mu     sync.RWMutex
	states map[baselineKey]*aiops.BaselineState
	dirty  map[baselineKey]bool // 需要 flush 的状态

	// 已从内存清掉、库里的行还没删的实体；下一次 flush 时删除
	removedEntities map[aiops.EntityRef]struct{}

	// 最新异常结果缓存（供 API 查询）
	anomalies map[aiops.EntityRef][]*aiops.AnomalyResult

	repo database.AIOpsBaselineRepository
}

// NewStateManager 创建基线状态管理器
func NewStateManager(repo database.AIOpsBaselineRepository) *StateManager {
	return &StateManager{
		states:          make(map[baselineKey]*aiops.BaselineState),
		dirty:           make(map[baselineKey]bool),
		removedEntities: make(map[aiops.EntityRef]struct{}),
		anomalies:       make(map[aiops.EntityRef][]*aiops.AnomalyResult),
		repo:            repo,
	}
}

// Update 用一个集群本轮的指标更新基线并检测异常
func (m *StateManager) Update(clusterID string, points []aiops.MetricDataPoint) []*aiops.AnomalyResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	var results []*aiops.AnomalyResult

	for _, p := range points {
		entity := aiops.EntityRef{ClusterID: clusterID, EntityKey: p.EntityKey}
		key := baselineKey{entity: entity, metricName: p.MetricName}

		// 获取或创建状态
		state, ok := m.states[key]
		if !ok {
			state = &aiops.BaselineState{
				ClusterID:  clusterID,
				EntityKey:  p.EntityKey,
				MetricName: p.MetricName,
			}
			m.states[key] = state
			// 实体在两次 flush 之间消失又出现：不再删它的行
			delete(m.removedEntities, entity)
		}

		// 执行异常检测
		_, result := Detect(state, p.Value, now)
		m.dirty[key] = true

		if result != nil {
			results = append(results, result)
			// 更新异常缓存
			m.anomalies[entity] = appendOrReplace(m.anomalies[entity], result)
		}
	}

	return results
}

// FlushToDB 把内存里的变化写入数据库：先删已消失实体的行，再写脏状态。
//
// 任一步失败时，对应的待办留在内存里，下一次 flush 重试。
func (m *StateManager) FlushToDB(ctx context.Context) error {
	m.mu.Lock()
	removedByCluster := make(map[string][]string)
	for entity := range m.removedEntities {
		removedByCluster[entity.ClusterID] = append(removedByCluster[entity.ClusterID], entity.EntityKey)
	}
	dirtyKeys := make([]baselineKey, 0, len(m.dirty))
	dirtyStates := make([]*database.AIOpsBaselineState, 0, len(m.dirty))
	for key := range m.dirty {
		state, ok := m.states[key]
		if !ok {
			continue
		}
		dirtyKeys = append(dirtyKeys, key)
		dirtyStates = append(dirtyStates, &database.AIOpsBaselineState{
			ClusterID:  state.ClusterID,
			EntityKey:  state.EntityKey,
			MetricName: state.MetricName,
			EMA:        state.EMA,
			Variance:   state.Variance,
			Count:      state.Count,
			UpdatedAt:  state.UpdatedAt,
		})
	}
	m.dirty = make(map[baselineKey]bool)
	m.mu.Unlock()

	for clusterID, entityKeys := range removedByCluster {
		if err := m.repo.DeleteEntities(ctx, clusterID, entityKeys); err != nil {
			m.markDirty(dirtyKeys)
			return err
		}
		m.forgetRemoved(clusterID, entityKeys)
	}

	if len(dirtyStates) == 0 {
		return nil
	}
	if err := m.repo.BatchUpsert(ctx, dirtyStates); err != nil {
		m.markDirty(dirtyKeys)
		return err
	}
	return nil
}

// markDirty 写库失败后把状态重新标脏，留给下一次 flush
func (m *StateManager) markDirty(keys []baselineKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		if _, ok := m.states[key]; ok {
			m.dirty[key] = true
		}
	}
}

// forgetRemoved 库里的行已删，从待删集合里拿掉。
// 删除期间重新出现的实体已被 Update 拿掉，这里再删一次无影响；
// 它重新出现后写下的状态带着脏标记，会在随后的写入里补回库中。
func (m *StateManager) forgetRemoved(clusterID string, entityKeys []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entityKey := range entityKeys {
		delete(m.removedEntities, aiops.EntityRef{ClusterID: clusterID, EntityKey: entityKey})
	}
}

// LoadFromDB 启动时从数据库恢复状态
func (m *StateManager) LoadFromDB(ctx context.Context) error {
	dbStates, err := m.repo.ListAll(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range dbStates {
		key := baselineKey{
			entity:     aiops.EntityRef{ClusterID: s.ClusterID, EntityKey: s.EntityKey},
			metricName: s.MetricName,
		}
		m.states[key] = &aiops.BaselineState{
			ClusterID:  s.ClusterID,
			EntityKey:  s.EntityKey,
			MetricName: s.MetricName,
			EMA:        s.EMA,
			Variance:   s.Variance,
			Count:      s.Count,
			UpdatedAt:  s.UpdatedAt,
		}
	}
	return nil
}

// GetStates 返回某个集群里指定实体的所有基线状态
func (m *StateManager) GetStates(clusterID, entityKey string) *aiops.EntityBaseline {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entity := aiops.EntityRef{ClusterID: clusterID, EntityKey: entityKey}
	result := &aiops.EntityBaseline{
		EntityKey: entityKey,
		States:    make([]*aiops.BaselineState, 0),
		Anomalies: make([]*aiops.AnomalyResult, 0),
	}
	for key, state := range m.states {
		if key.entity == entity {
			result.States = append(result.States, state)
		}
	}
	if anomalies, ok := m.anomalies[entity]; ok {
		result.Anomalies = anomalies
	}
	return result
}

// CleanupStaleEntities 清理某个集群里已不存在的实体的基线状态和异常缓存。
//
// activeKeys 是该集群当前快照里所有实体的实体键。只动 clusterID 这一个集群：
// 其他集群的实体不在这份快照里，不能据此判定它们已消失。
// 返回清掉的基线条数；对应的库内行在下一次 FlushToDB 时删除。
func (m *StateManager) CleanupStaleEntities(clusterID string, activeKeys map[string]bool) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0
	for key := range m.states {
		if key.entity.ClusterID != clusterID || activeKeys[key.entity.EntityKey] {
			continue
		}
		delete(m.states, key)
		delete(m.dirty, key)
		m.removedEntities[key.entity] = struct{}{}
		removed++
	}

	for entity := range m.anomalies {
		if entity.ClusterID == clusterID && !activeKeys[entity.EntityKey] {
			delete(m.anomalies, entity)
		}
	}

	return removed
}

// appendOrReplace 追加或替换同一指标的异常结果
func appendOrReplace(existing []*aiops.AnomalyResult, newResult *aiops.AnomalyResult) []*aiops.AnomalyResult {
	for i, r := range existing {
		if r.MetricName == newResult.MetricName {
			if newResult.IsAnomaly {
				existing[i] = newResult
			} else {
				// 异常消失，移除
				existing = append(existing[:i], existing[i+1:]...)
			}
			return existing
		}
	}
	if newResult.IsAnomaly {
		return append(existing, newResult)
	}
	return existing
}
