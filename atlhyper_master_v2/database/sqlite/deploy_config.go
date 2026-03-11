package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type deployConfigDialect struct{}

func (d *deployConfigDialect) Upsert(config *database.DeployConfig) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	autoDeploy := 0
	if config.AutoDeploy {
		autoDeploy = 1
	}
	return `INSERT INTO deploy_config (cluster_id, repo_url, paths, interval_sec, auto_deploy, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id) DO UPDATE SET repo_url = excluded.repo_url, paths = excluded.paths,
		interval_sec = excluded.interval_sec, auto_deploy = excluded.auto_deploy, updated_at = excluded.updated_at`,
		[]any{config.ClusterID, config.RepoURL, config.Paths, config.IntervalSec, autoDeploy, now, now}
}

func (d *deployConfigDialect) SelectByCluster(clusterID string) (string, []any) {
	return "SELECT id, cluster_id, repo_url, paths, interval_sec, auto_deploy, created_at, updated_at FROM deploy_config WHERE cluster_id = ?", []any{clusterID}
}

func (d *deployConfigDialect) SelectAll() (string, []any) {
	return "SELECT id, cluster_id, repo_url, paths, interval_sec, auto_deploy, created_at, updated_at FROM deploy_config", nil
}

func (d *deployConfigDialect) Delete(clusterID string) (string, []any) {
	return "DELETE FROM deploy_config WHERE cluster_id = ?", []any{clusterID}
}

func (d *deployConfigDialect) ScanRow(rows *sql.Rows) (*database.DeployConfig, error) {
	c := &database.DeployConfig{}
	var autoDeploy int
	var createdAt, updatedAt string
	err := rows.Scan(&c.ID, &c.ClusterID, &c.RepoURL, &c.Paths, &c.IntervalSec, &autoDeploy, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.AutoDeploy = autoDeploy != 0
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return c, nil
}

var _ database.DeployConfigDialect = (*deployConfigDialect)(nil)
