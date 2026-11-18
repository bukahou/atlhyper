// atlhyper_master_v2/database/repo/cluster.go
// ClusterRepository 实现（集群注册表）
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type clusterRepo struct {
	db      *sql.DB
	dialect database.ClusterDialect
}

func newClusterRepo(db *sql.DB, dialect database.ClusterDialect) *clusterRepo {
	return &clusterRepo{db: db, dialect: dialect}
}

func (r *clusterRepo) Create(ctx context.Context, cluster *database.Cluster) error {
	query, args := r.dialect.Insert(cluster, time.Now())
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		if r.dialect.IsDuplicateKey(err) {
			return database.ErrClusterExists
		}
		return err
	}
	return nil
}

func (r *clusterRepo) UpdateProfile(ctx context.Context, clusterID, name, description string) error {
	query, args := r.dialect.UpdateProfile(clusterID, name, description, time.Now())
	return r.execOnRegistered(ctx, query, args...)
}

func (r *clusterRepo) SetToken(ctx context.Context, clusterID, tokenHash, tokenPrefix string, issuedAt time.Time) error {
	query, args := r.dialect.UpdateToken(clusterID, tokenHash, tokenPrefix, issuedAt, time.Now())
	return r.execOnRegistered(ctx, query, args...)
}

func (r *clusterRepo) Delete(ctx context.Context, clusterID string) error {
	query, args := r.dialect.Delete(clusterID)
	return r.execOnRegistered(ctx, query, args...)
}

// execOnRegistered 执行一条针对单个集群的写语句；没有命中任何行说明集群未登记
func (r *clusterRepo) execOnRegistered(ctx context.Context, query string, args ...any) error {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return database.ErrClusterNotFound
	}
	return nil
}

func (r *clusterRepo) GetByID(ctx context.Context, clusterID string) (*database.Cluster, error) {
	query, args := r.dialect.SelectByID(clusterID)
	return r.queryOne(ctx, query, args...)
}

func (r *clusterRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*database.Cluster, error) {
	// 没签发令牌的集群哈希列是空串：空哈希不得匹配到它们
	if tokenHash == "" {
		return nil, nil
	}
	query, args := r.dialect.SelectByTokenHash(tokenHash)
	return r.queryOne(ctx, query, args...)
}

func (r *clusterRepo) List(ctx context.Context) ([]*database.Cluster, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clusters := []*database.Cluster{}
	for rows.Next() {
		cluster, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
	}
	return clusters, rows.Err()
}

func (r *clusterRepo) queryOne(ctx context.Context, query string, args ...any) (*database.Cluster, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, rows.Err()
	}
	return r.dialect.ScanRow(rows)
}

var _ database.ClusterRepository = (*clusterRepo)(nil)
