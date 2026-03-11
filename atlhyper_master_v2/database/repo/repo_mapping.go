package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type repoMappingRepo struct {
	db      *sql.DB
	dialect database.RepoMappingDialect
}

func newRepoMappingRepo(db *sql.DB, dialect database.RepoMappingDialect) *repoMappingRepo {
	return &repoMappingRepo{db: db, dialect: dialect}
}

func (r *repoMappingRepo) Create(ctx context.Context, m *database.RepoDeployMapping) error {
	query, args := r.dialect.Insert(m)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err == nil {
		m.ID = id
	}
	return nil
}

func (r *repoMappingRepo) Update(ctx context.Context, m *database.RepoDeployMapping) error {
	query, args := r.dialect.Update(m)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// PartialUpdate 部分更新：只更新 fields 中存在的字段
func (r *repoMappingRepo) PartialUpdate(ctx context.Context, id int64, fields map[string]interface{}) error {
	// 允许更新的字段（JSON key → DB column）
	allowed := map[string]string{
		"namespace":   "namespace",
		"deployment":  "deployment",
		"container":   "container",
		"imagePrefix": "image_prefix",
		"sourcePath":  "source_path",
		"confirmed":   "confirmed",
	}

	var setClauses []string
	var args []any
	for jsonKey, col := range allowed {
		if val, ok := fields[jsonKey]; ok {
			setClauses = append(setClauses, col+"=?")
			args = append(args, val)
		}
	}
	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at=?")
	args = append(args, time.Now().Format(time.RFC3339))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE repo_deploy_mapping SET %s WHERE id=?", strings.Join(setClauses, ", "))
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repoMappingRepo) Confirm(ctx context.Context, id int64) error {
	query, args := r.dialect.Confirm(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repoMappingRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *repoMappingRepo) GetByID(ctx context.Context, id int64) (*database.RepoDeployMapping, error) {
	query, args := r.dialect.SelectByID(id)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, sql.ErrNoRows
	}
	return r.dialect.ScanRow(rows)
}

func (r *repoMappingRepo) List(ctx context.Context) ([]*database.RepoDeployMapping, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*database.RepoDeployMapping
	for rows.Next() {
		m, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *repoMappingRepo) ListByRepo(ctx context.Context, repo string) ([]*database.RepoDeployMapping, error) {
	query, args := r.dialect.SelectByRepo(repo)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*database.RepoDeployMapping
	for rows.Next() {
		m, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *repoMappingRepo) DeleteByRepoAndNamespace(ctx context.Context, repo, namespace string) error {
	query, args := r.dialect.DeleteByRepoAndNamespace(repo, namespace)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}
