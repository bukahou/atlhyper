package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type repoConfigRepo struct {
	db      *sql.DB
	dialect database.RepoConfigDialect
}

func newRepoConfigRepo(db *sql.DB, dialect database.RepoConfigDialect) *repoConfigRepo {
	return &repoConfigRepo{db: db, dialect: dialect}
}

func (r *repoConfigRepo) Upsert(ctx context.Context, config *database.RepoConfig) error {
	query, args := r.dialect.Upsert(config)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repoConfigRepo) GetByRepo(ctx context.Context, repo string) (*database.RepoConfig, error) {
	query, args := r.dialect.SelectByRepo(repo)
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

func (r *repoConfigRepo) List(ctx context.Context) ([]*database.RepoConfig, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var configs []*database.RepoConfig
	for rows.Next() {
		c, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

func (r *repoConfigRepo) Delete(ctx context.Context, repo string) error {
	query, args := r.dialect.Delete(repo)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}
