// atlhyper_master_v2/database/sqlite/impl/cluster.go
// ClusterRepository SQLite 实现
package impl

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

type ClusterRepository struct {
	db *sql.DB
}

func NewClusterRepository(db *sql.DB) *ClusterRepository {
	return &ClusterRepository{db: db}
}

func (r *ClusterRepository) Create(ctx context.Context, cluster *repository.Cluster) error {
	now := time.Now().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO clusters (cluster_uid, name, description, environment, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		cluster.ClusterUID, cluster.Name, cluster.Description, cluster.Environment, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	cluster.ID = id
	return nil
}

func (r *ClusterRepository) Update(ctx context.Context, cluster *repository.Cluster) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE clusters SET name = ?, description = ?, environment = ?, updated_at = ? WHERE id = ?`,
		cluster.Name, cluster.Description, cluster.Environment, time.Now().Format(time.RFC3339), cluster.ID,
	)
	return err
}

func (r *ClusterRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM clusters WHERE id = ?", id)
	return err
}

func (r *ClusterRepository) GetByID(ctx context.Context, id int64) (*repository.Cluster, error) {
	return r.scanOne(ctx, "SELECT * FROM clusters WHERE id = ?", id)
}

func (r *ClusterRepository) GetByUID(ctx context.Context, uid string) (*repository.Cluster, error) {
	return r.scanOne(ctx, "SELECT * FROM clusters WHERE cluster_uid = ?", uid)
}

func (r *ClusterRepository) List(ctx context.Context) ([]*repository.Cluster, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT * FROM clusters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []*repository.Cluster
	for rows.Next() {
		c, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, c)
	}
	return clusters, rows.Err()
}

func (r *ClusterRepository) scanOne(ctx context.Context, query string, args ...interface{}) (*repository.Cluster, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	c := &repository.Cluster{}
	var createdAt, updatedAt string
	err := row.Scan(&c.ID, &c.ClusterUID, &c.Name, &c.Description, &c.Environment, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return c, nil
}

func (r *ClusterRepository) scanRow(rows *sql.Rows) (*repository.Cluster, error) {
	c := &repository.Cluster{}
	var createdAt, updatedAt string
	err := rows.Scan(&c.ID, &c.ClusterUID, &c.Name, &c.Description, &c.Environment, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return c, nil
}

var _ repository.ClusterRepository = (*ClusterRepository)(nil)
