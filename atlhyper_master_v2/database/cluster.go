// atlhyper_master_v2/database/cluster.go
// ClusterRepository 实现
package database

import (
	"context"
	"database/sql"
)

type clusterRepo struct {
	db      *sql.DB
	dialect ClusterDialect
}

func newClusterRepo(db *sql.DB, dialect ClusterDialect) *clusterRepo {
	return &clusterRepo{db: db, dialect: dialect}
}

func (r *clusterRepo) Create(ctx context.Context, cluster *Cluster) error {
	query, args := r.dialect.Insert(cluster)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	cluster.ID = id
	return nil
}

func (r *clusterRepo) Update(ctx context.Context, cluster *Cluster) error {
	query, args := r.dialect.Update(cluster)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *clusterRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *clusterRepo) GetByID(ctx context.Context, id int64) (*Cluster, error) {
	query, args := r.dialect.SelectByID(id)
	return r.queryOne(ctx, query, args...)
}

func (r *clusterRepo) GetByUID(ctx context.Context, uid string) (*Cluster, error) {
	query, args := r.dialect.SelectByUID(uid)
	return r.queryOne(ctx, query, args...)
}

func (r *clusterRepo) List(ctx context.Context) ([]*Cluster, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []*Cluster
	for rows.Next() {
		c, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, rows.Err()
}

func (r *clusterRepo) queryOne(ctx context.Context, query string, args ...any) (*Cluster, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	return r.dialect.ScanRow(rows)
}
