// atlhyper_master_v2/aiops/baseline/state.go
// 基线状态管理器（内存缓存 + 定期 flush DB + 启动恢复）
package baseline

import (
	"context"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// StateManager 基线状态管理器
type StateManager struct {
	mu     sync.RWMutex
	states map[string]*aiops.BaselineState // key = entityKey + "|" + metricName
	dirty  map[string]bool                 // 需要 flush 的状态

	// 最新异常结果缓存（供 API 查询）
	anomalies map[string][]*aiops.AnomalyResult // key = entityKey

	repo database.AIOpsBaselineRepository
}

// NewStateManager 创建基线状态管理器
func NewStateManager(repo database.AIOpsBaselineRepository) *StateManager {
	return &StateManager{
		states:    make(map[string]*aiops.BaselineState),
		dirty:     make(map[string]bool),
		anomalies: make(map[string][]*aiops.AnomalyResult),
		repo:      repo,
	}
}

// Update 更新指标并检测异常
func (m *StateManager) Update(points []aiops.MetricDataPoint) []*aiops.AnomalyResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	var results []*aiops.AnomalyResult

	for _, p := range points {
		cacheKey := p.EntityKey + "|" + p.MetricName

		// 获取或创建状态
		state, ok := m.states[cacheKey]
		if !ok {
			state = &aiops.BaselineState{
				EntityKey:  p.EntityKey,
				MetricName: p.MetricName,
			}
			m.states[cacheKey] = state
		}

		// 执行异常检测
		_, result := Detect(state, p.Value, now)
		m.dirty[cacheKey] = true

		if result != nil {
			results = append(results, result)
			// 更新异常缓存
			m.anomalies[p.EntityKey] = appendOrReplace(m.anomalies[p.EntityKey], result)
		}
	}

	return results
}

// FlushToDB 将脏状态批量写入数据库
func (m *StateManager) FlushToDB(ctx context.Context) error {
	m.mu.Lock()
	dirtyStates := make([]*database.AIOpsBaselineState, 0, len(m.dirty))
	for key := range m.dirty {
		if state, ok := m.states[key]; ok {
			dirtyStates = append(dirtyStates, &database.AIOpsBaselineState{
				EntityKey:  state.EntityKey,
				MetricName: state.MetricName,
				EMA:        state.EMA,
				Variance:   state.Variance,
				Count:      state.Count,
				UpdatedAt:  state.UpdatedAt,
			})
		}
	}
	m.dirty = make(map[string]bool)
	m.mu.Unlock()

	if len(dirtyStates) == 0 {
		return nil
	}
	return m.repo.BatchUpsert(ctx, dirtyStates)
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
		cacheKey := s.EntityKey + "|" + s.MetricName
		m.states[cacheKey] = &aiops.BaselineState{
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

// GetStates 返回指定实体的所有基线状态
func (m *StateManager) GetStates(entityKey string) *aiops.EntityBaseline {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &aiops.EntityBaseline{
		EntityKey: entityKey,
		States:    make([]*aiops.BaselineState, 0),
		Anomalies: make([]*aiops.AnomalyResult, 0),
	}
	prefix := entityKey + "|"
	for key, state := range m.states {
		if strings.HasPrefix(key, prefix) {
			result.States = append(result.States, state)
		}
	}
	if anomalies, ok := m.anomalies[entityKey]; ok {
		result.Anomalies = anomalies
	}
	return result
}

// GetAllAnomalies 返回所有当前异常（供风险评分使用）
func (m *StateManager) GetAllAnomalies() []*aiops.AnomalyResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []*aiops.AnomalyResult
	for _, anomalies := range m.anomalies {
		all = append(all, anomalies...)
	}
	return all
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
