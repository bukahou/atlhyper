// atlhyper_master_v2/database/sqlite/aiops_baseline.go
// AIOps 基线状态 SQLite 方言实现
package sqlite

import (
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

// aIOpsBaselineDialect AIOps 基线 SQLite 方言
type aIOpsBaselineDialect struct{}

// Upsert 插入或更新基线状态
func (d *aIOpsBaselineDialect) Upsert(state *database.AIOpsBaselineState) (string, []any) {
	return `INSERT INTO aiops_baseline_states (cluster_id, entity_key, metric_name, ema, variance, count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, entity_key, metric_name) DO UPDATE SET
			ema = excluded.ema,
			variance = excluded.variance,
			count = excluded.count,
			updated_at = excluded.updated_at`,
		[]any{state.ClusterID, state.EntityKey, state.MetricName, state.EMA, state.Variance, state.Count, state.UpdatedAt}
}

// SelectAll 查询所有基线状态
func (d *aIOpsBaselineDialect) SelectAll() (string, []any) {
	return `SELECT cluster_id, entity_key, metric_name, ema, variance, count, updated_at FROM aiops_baseline_states`, nil
}

// DeleteEntity 删除某个集群里一个实体的全部基线行
func (d *aIOpsBaselineDialect) DeleteEntity(clusterID, entityKey string) (string, []any) {
	return `DELETE FROM aiops_baseline_states WHERE cluster_id = ? AND entity_key = ?`, []any{clusterID, entityKey}
}

// ScanRow 扫描基线状态行
func (d *aIOpsBaselineDialect) ScanRow(rows *sql.Rows) (*database.AIOpsBaselineState, error) {
	s := &database.AIOpsBaselineState{}
	err := rows.Scan(&s.ClusterID, &s.EntityKey, &s.MetricName, &s.EMA, &s.Variance, &s.Count, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}
