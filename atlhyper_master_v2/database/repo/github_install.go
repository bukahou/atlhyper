package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type gitHubInstallRepo struct {
	db      *sql.DB
	dialect database.GitHubInstallDialect
}

func newGitHubInstallRepo(db *sql.DB, dialect database.GitHubInstallDialect) *gitHubInstallRepo {
	return &gitHubInstallRepo{db: db, dialect: dialect}
}

func (r *gitHubInstallRepo) Upsert(ctx context.Context, inst *database.GitHubInstallation) error {
	query, args := r.dialect.Upsert(inst)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *gitHubInstallRepo) Get(ctx context.Context) (*database.GitHubInstallation, error) {
	query, args := r.dialect.Select()
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

func (r *gitHubInstallRepo) Delete(ctx context.Context) error {
	query, args := r.dialect.Delete()
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}
