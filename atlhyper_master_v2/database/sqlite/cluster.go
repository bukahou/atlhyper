// atlhyper_master_v2/database/sqlite/cluster.go
// SQLite ClusterDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type clusterDialect struct{}

func (d *clusterDialect) Insert(cluster *database.Cluster) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	query := `INSERT INTO clusters (cluster_uid, name, description, environment, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)`
	args := []any{cluster.ClusterUID, cluster.Name, cluster.Description, cluster.Environment, now, now}
	return query, args
}

func (d *clusterDialect) Update(cluster *database.Cluster) (string, []any) {
	return `UPDATE clusters SET name = ?, description = ?, environment = ?, updated_at = ? WHERE id = ?`,
		[]any{cluster.Name, cluster.Description, cluster.Environment, time.Now().Format(time.RFC3339), cluster.ID}
}

func (d *clusterDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM clusters WHERE id = ?", []any{id}
}

func (d *clusterDialect) SelectByID(id int64) (string, []any) {
	return "SELECT id, cluster_uid, name, description, environment, created_at, updated_at FROM clusters WHERE id = ?", []any{id}
}

func (d *clusterDialect) SelectByUID(uid string) (string, []any) {
	return "SELECT id, cluster_uid, name, description, environment, created_at, updated_at FROM clusters WHERE cluster_uid = ?", []any{uid}
}

func (d *clusterDialect) SelectAll() (string, []any) {
	return "SELECT id, cluster_uid, name, description, environment, created_at, updated_at FROM clusters", nil
}

func (d *clusterDialect) ScanRow(rows *sql.Rows) (*database.Cluster, error) {
	c := &database.Cluster{}
	var createdAt, updatedAt string
	err := rows.Scan(&c.ID, &c.ClusterUID, &c.Name, &c.Description, &c.Environment, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return c, nil
}

var _ database.ClusterDialect = (*clusterDialect)(nil)
