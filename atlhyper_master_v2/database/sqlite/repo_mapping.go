package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type repoMappingDialect struct{}

func (d *repoMappingDialect) Insert(m *database.RepoDeployMapping) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `INSERT INTO repo_deploy_mapping (cluster_id, repo, namespace, deployment, container, image_prefix, source_path, confirmed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{m.ClusterID, m.Repo, m.Namespace, m.Deployment, m.Container, m.ImagePrefix, m.SourcePath, m.Confirmed, now, now}
}

func (d *repoMappingDialect) Update(m *database.RepoDeployMapping) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `UPDATE repo_deploy_mapping SET namespace=?, deployment=?, container=?, image_prefix=?, source_path=?, confirmed=?, updated_at=? WHERE id=?`,
		[]any{m.Namespace, m.Deployment, m.Container, m.ImagePrefix, m.SourcePath, m.Confirmed, now, m.ID}
}

func (d *repoMappingDialect) Confirm(id int64) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	return `UPDATE repo_deploy_mapping SET confirmed=1, updated_at=? WHERE id=?`, []any{now, id}
}

func (d *repoMappingDialect) Delete(id int64) (string, []any) {
	return `DELETE FROM repo_deploy_mapping WHERE id=?`, []any{id}
}

func (d *repoMappingDialect) SelectByID(id int64) (string, []any) {
	return `SELECT id, cluster_id, repo, namespace, deployment, container, image_prefix, source_path, confirmed, created_at, updated_at FROM repo_deploy_mapping WHERE id=?`, []any{id}
}

func (d *repoMappingDialect) SelectAll() (string, []any) {
	return `SELECT id, cluster_id, repo, namespace, deployment, container, image_prefix, source_path, confirmed, created_at, updated_at FROM repo_deploy_mapping ORDER BY repo, namespace`, nil
}

func (d *repoMappingDialect) SelectByRepo(repo string) (string, []any) {
	return `SELECT id, cluster_id, repo, namespace, deployment, container, image_prefix, source_path, confirmed, created_at, updated_at FROM repo_deploy_mapping WHERE repo=? ORDER BY namespace`, []any{repo}
}

func (d *repoMappingDialect) DeleteByRepoAndNamespace(repo, namespace string) (string, []any) {
	return `DELETE FROM repo_deploy_mapping WHERE repo=? AND namespace=?`, []any{repo, namespace}
}

func (d *repoMappingDialect) ScanRow(rows *sql.Rows) (*database.RepoDeployMapping, error) {
	m := &database.RepoDeployMapping{}
	var createdAt, updatedAt string
	err := rows.Scan(&m.ID, &m.ClusterID, &m.Repo, &m.Namespace, &m.Deployment, &m.Container, &m.ImagePrefix, &m.SourcePath, &m.Confirmed, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return m, nil
}

var _ database.RepoMappingDialect = (*repoMappingDialect)(nil)
