// atlhyper_master_v2/database/repo/aiops_graph.go
// AIOps 依赖图 Repository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

// aiopsGraphRepo AIOps 依赖图 Repository 实现
type aiopsGraphRepo struct {
	db      *sql.DB
	dialect database.AIOpsGraphDialect
}

// newAIOpsGraphRepo 创建 AIOps 依赖图 Repository
func newAIOpsGraphRepo(db *sql.DB, dialect database.AIOpsGraphDialect) *aiopsGraphRepo {
	return &aiopsGraphRepo{db: db, dialect: dialect}
}

// Save 保存图快照（覆盖式更新）
func (r *aiopsGraphRepo) Save(ctx context.Context, clusterID string, snapshot []byte) error {
	query, args := r.dialect.Upsert(clusterID, snapshot)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// Load 加载图快照
func (r *aiopsGraphRepo) Load(ctx context.Context, clusterID string) ([]byte, error) {
	query, args := r.dialect.SelectByCluster(clusterID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		_, data, err := r.dialect.ScanSnapshot(rows)
		return data, err
	}
	return nil, nil
}

// ListClusterIDs 列出所有有图快照的集群 ID
func (r *aiopsGraphRepo) ListClusterIDs(ctx context.Context) ([]string, error) {
	query, args := r.dialect.SelectAllClusterIDs()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

// 确保实现了接口
var _ database.AIOpsGraphRepository = (*aiopsGraphRepo)(nil)
