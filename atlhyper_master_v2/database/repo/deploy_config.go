package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type deployConfigRepo struct {
	db      *sql.DB
	dialect database.DeployConfigDialect
}

func newDeployConfigRepo(db *sql.DB, dialect database.DeployConfigDialect) *deployConfigRepo {
	return &deployConfigRepo{db: db, dialect: dialect}
}

func (r *deployConfigRepo) Upsert(ctx context.Context, config *database.DeployConfig) error {
	query, args := r.dialect.Upsert(config)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *deployConfigRepo) GetByCluster(ctx context.Context, clusterID string) (*database.DeployConfig, error) {
	query, args := r.dialect.SelectByCluster(clusterID)
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

func (r *deployConfigRepo) Delete(ctx context.Context, clusterID string) error {
	query, args := r.dialect.Delete(clusterID)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}
