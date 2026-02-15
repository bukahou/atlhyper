// atlhyper_master_v2/database/sqlite/aiops_graph.go
// AIOps 依赖图快照 SQLite 方言实现
package sqlite

import (
	"database/sql"
)

// aIOpsGraphDialect AIOps 依赖图 SQLite 方言
type aIOpsGraphDialect struct{}

// Upsert 插入或更新图快照
func (d *aIOpsGraphDialect) Upsert(clusterID string, snapshot []byte) (string, []any) {
	return `INSERT INTO aiops_dependency_graph_snapshots (cluster_id, snapshot, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(cluster_id) DO UPDATE SET
			snapshot = excluded.snapshot,
			created_at = CURRENT_TIMESTAMP`,
		[]any{clusterID, snapshot}
}

// SelectByCluster 按集群查询图快照
func (d *aIOpsGraphDialect) SelectByCluster(clusterID string) (string, []any) {
	return `SELECT cluster_id, snapshot FROM aiops_dependency_graph_snapshots WHERE cluster_id = ?`,
		[]any{clusterID}
}

// SelectAllClusterIDs 查询所有集群 ID
func (d *aIOpsGraphDialect) SelectAllClusterIDs() (string, []any) {
	return `SELECT cluster_id FROM aiops_dependency_graph_snapshots`, nil
}

// ScanSnapshot 扫描图快照行
func (d *aIOpsGraphDialect) ScanSnapshot(rows *sql.Rows) (string, []byte, error) {
	var clusterID string
	var data []byte
	err := rows.Scan(&clusterID, &data)
	return clusterID, data, err
}
