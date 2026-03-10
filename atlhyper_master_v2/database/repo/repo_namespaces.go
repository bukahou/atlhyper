package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type repoNamespaceRepo struct {
	db      *sql.DB
	dialect database.RepoNamespaceDialect
}

func newRepoNamespaceRepo(db *sql.DB, dialect database.RepoNamespaceDialect) *repoNamespaceRepo {
	return &repoNamespaceRepo{db: db, dialect: dialect}
}

func (r *repoNamespaceRepo) Add(ctx context.Context, repo, namespace string) error {
	query, args := r.dialect.Insert(repo, namespace)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repoNamespaceRepo) Remove(ctx context.Context, repo, namespace string) error {
	query, args := r.dialect.Delete(repo, namespace)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repoNamespaceRepo) ListByRepo(ctx context.Context, repo string) ([]string, error) {
	query, args := r.dialect.SelectByRepo(repo)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		ns, err := r.dialect.ScanNamespace(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, ns)
	}
	return result, rows.Err()
}
