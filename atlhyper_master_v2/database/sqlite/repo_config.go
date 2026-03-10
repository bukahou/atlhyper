package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type repoConfigDialect struct{}

func (d *repoConfigDialect) Upsert(config *database.RepoConfig) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	enabled := 0
	if config.MappingEnabled {
		enabled = 1
	}
	return `INSERT INTO repo_config (repo, mapping_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(repo) DO UPDATE SET mapping_enabled = excluded.mapping_enabled, updated_at = excluded.updated_at`,
		[]any{config.Repo, enabled, now, now}
}

func (d *repoConfigDialect) SelectByRepo(repo string) (string, []any) {
	return "SELECT id, repo, mapping_enabled, created_at, updated_at FROM repo_config WHERE repo = ?", []any{repo}
}

func (d *repoConfigDialect) SelectAll() (string, []any) {
	return "SELECT id, repo, mapping_enabled, created_at, updated_at FROM repo_config ORDER BY repo", nil
}

func (d *repoConfigDialect) Delete(repo string) (string, []any) {
	return "DELETE FROM repo_config WHERE repo = ?", []any{repo}
}

func (d *repoConfigDialect) ScanRow(rows *sql.Rows) (*database.RepoConfig, error) {
	c := &database.RepoConfig{}
	var enabled int
	var createdAt, updatedAt string
	err := rows.Scan(&c.ID, &c.Repo, &enabled, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.MappingEnabled = enabled != 0
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return c, nil
}

var _ database.RepoConfigDialect = (*repoConfigDialect)(nil)
