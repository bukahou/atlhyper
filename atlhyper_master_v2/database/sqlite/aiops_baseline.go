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
	return `INSERT INTO aiops_baseline_states (entity_key, metric_name, ema, variance, count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_key, metric_name) DO UPDATE SET
			ema = excluded.ema,
			variance = excluded.variance,
			count = excluded.count,
			updated_at = excluded.updated_at`,
		[]any{state.EntityKey, state.MetricName, state.EMA, state.Variance, state.Count, state.UpdatedAt}
}

// SelectAll 查询所有基线状态
func (d *aIOpsBaselineDialect) SelectAll() (string, []any) {
	return `SELECT entity_key, metric_name, ema, variance, count, updated_at FROM aiops_baseline_states`, nil
}

// SelectByEntity 按实体查询基线状态
func (d *aIOpsBaselineDialect) SelectByEntity(entityKey string) (string, []any) {
	return `SELECT entity_key, metric_name, ema, variance, count, updated_at
		FROM aiops_baseline_states WHERE entity_key = ?`, []any{entityKey}
}

// DeleteByEntity 按实体删除基线状态
func (d *aIOpsBaselineDialect) DeleteByEntity(entityKey string) (string, []any) {
	return `DELETE FROM aiops_baseline_states WHERE entity_key = ?`, []any{entityKey}
}

// ScanRow 扫描基线状态行
func (d *aIOpsBaselineDialect) ScanRow(rows *sql.Rows) (*database.AIOpsBaselineState, error) {
	s := &database.AIOpsBaselineState{}
	err := rows.Scan(&s.EntityKey, &s.MetricName, &s.EMA, &s.Variance, &s.Count, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}
