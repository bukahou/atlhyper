// atlhyper_master_v2/database/sqlite/cluster.go
// SQLite ClusterDialect 实现（集群注册表）
package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type clusterDialect struct{}

const clusterColumns = "cluster_id, name, description, token_hash, token_prefix, token_issued_at, created_at, updated_at"

func (d *clusterDialect) Insert(cluster *database.Cluster, now time.Time) (string, []any) {
	stamp := now.UTC().Format(time.RFC3339)
	return `INSERT INTO clusters (cluster_id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		[]any{cluster.ClusterID, cluster.Name, cluster.Description, stamp, stamp}
}

func (d *clusterDialect) UpdateProfile(clusterID, name, description string, now time.Time) (string, []any) {
	return `UPDATE clusters SET name = ?, description = ?, updated_at = ? WHERE cluster_id = ?`,
		[]any{name, description, now.UTC().Format(time.RFC3339), clusterID}
}

func (d *clusterDialect) UpdateToken(clusterID, tokenHash, tokenPrefix string, issuedAt, now time.Time) (string, []any) {
	return `UPDATE clusters SET token_hash = ?, token_prefix = ?, token_issued_at = ?, updated_at = ? WHERE cluster_id = ?`,
		[]any{tokenHash, tokenPrefix, issuedAt.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339), clusterID}
}

func (d *clusterDialect) Delete(clusterID string) (string, []any) {
	return "DELETE FROM clusters WHERE cluster_id = ?", []any{clusterID}
}

func (d *clusterDialect) SelectByID(clusterID string) (string, []any) {
	return "SELECT " + clusterColumns + " FROM clusters WHERE cluster_id = ?", []any{clusterID}
}

func (d *clusterDialect) SelectByTokenHash(tokenHash string) (string, []any) {
	return "SELECT " + clusterColumns + " FROM clusters WHERE token_hash = ?", []any{tokenHash}
}

func (d *clusterDialect) SelectAll() (string, []any) {
	return "SELECT " + clusterColumns + " FROM clusters ORDER BY cluster_id", nil
}

func (d *clusterDialect) ScanRow(rows *sql.Rows) (*database.Cluster, error) {
	cluster := &database.Cluster{}
	var tokenIssuedAt sql.NullString
	var createdAt, updatedAt string
	err := rows.Scan(&cluster.ClusterID, &cluster.Name, &cluster.Description,
		&cluster.TokenHash, &cluster.TokenPrefix, &tokenIssuedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if tokenIssuedAt.Valid {
		if issuedAt, err := time.Parse(time.RFC3339, tokenIssuedAt.String); err == nil {
			cluster.TokenIssuedAt = &issuedAt
		}
	}
	cluster.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	cluster.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return cluster, nil
}

// IsDuplicateKey SQLite 的主键冲突文案形如 "UNIQUE constraint failed: clusters.cluster_id"
func (d *clusterDialect) IsDuplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: clusters.cluster_id")
}

var _ database.ClusterDialect = (*clusterDialect)(nil)
