// atlhyper_master_v2/database/repo/node_metrics.go
// NodeMetrics Repository 实现
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// nodeMetricsRepo NodeMetrics Repository 实现
type nodeMetricsRepo struct {
	db      *sql.DB
	dialect database.NodeMetricsDialect
}

// newNodeMetricsRepo 创建 NodeMetrics Repository
func newNodeMetricsRepo(db *sql.DB, dialect database.NodeMetricsDialect) *nodeMetricsRepo {
	return &nodeMetricsRepo{db: db, dialect: dialect}
}

// UpsertLatest 更新或插入实时数据
func (r *nodeMetricsRepo) UpsertLatest(ctx context.Context, m *database.NodeMetricsLatest) error {
	query, args := r.dialect.UpsertLatest(m)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetLatest 获取单个节点实时数据
func (r *nodeMetricsRepo) GetLatest(ctx context.Context, clusterID, nodeName string) (*database.NodeMetricsLatest, error) {
	query, args := r.dialect.SelectLatest(clusterID, nodeName)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return r.dialect.ScanLatest(rows)
	}
	return nil, nil
}

// ListLatest 获取集群所有节点实时数据
func (r *nodeMetricsRepo) ListLatest(ctx context.Context, clusterID string) ([]*database.NodeMetricsLatest, error) {
	query, args := r.dialect.SelectAllLatest(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.NodeMetricsLatest
	for rows.Next() {
		m, err := r.dialect.ScanLatest(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// DeleteLatest 删除节点实时数据
func (r *nodeMetricsRepo) DeleteLatest(ctx context.Context, clusterID, nodeName string) error {
	query, args := r.dialect.DeleteLatest(clusterID, nodeName)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// InsertHistory 插入历史数据
func (r *nodeMetricsRepo) InsertHistory(ctx context.Context, m *database.NodeMetricsHistory) error {
	query, args := r.dialect.InsertHistory(m)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// GetHistory 获取历史数据
func (r *nodeMetricsRepo) GetHistory(ctx context.Context, clusterID, nodeName string, start, end time.Time) ([]*database.NodeMetricsHistory, error) {
	query, args := r.dialect.SelectHistory(clusterID, nodeName, start, end)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*database.NodeMetricsHistory
	for rows.Next() {
		m, err := r.dialect.ScanHistory(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// DeleteHistoryBefore 清理历史数据
func (r *nodeMetricsRepo) DeleteHistoryBefore(ctx context.Context, before time.Time) (int64, error) {
	query, args := r.dialect.DeleteHistoryBefore(before)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetAllNodeNames 获取所有节点名
func (r *nodeMetricsRepo) GetAllNodeNames(ctx context.Context, clusterID string) ([]string, error) {
	query, args := r.dialect.SelectAllNodeNames(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var nodeName string
		if err := rows.Scan(&nodeName); err != nil {
			return nil, err
		}
		result = append(result, nodeName)
	}
	return result, rows.Err()
}

// 确保实现了接口
var _ database.NodeMetricsRepository = (*nodeMetricsRepo)(nil)
