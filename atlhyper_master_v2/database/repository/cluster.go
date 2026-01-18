// atlhyper_master_v2/database/repository/cluster.go
// ClusterRepository 接口定义
package repository

import (
	"context"
	"time"
)

// Cluster 集群信息
type Cluster struct {
	ID          int64
	ClusterUID  string // 集群 UID（来自 kube-system）
	Name        string // 显示名称
	Description string
	Environment string // prod / staging / dev
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ClusterRepository 集群接口
type ClusterRepository interface {
	Create(ctx context.Context, cluster *Cluster) error
	Update(ctx context.Context, cluster *Cluster) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*Cluster, error)
	GetByUID(ctx context.Context, uid string) (*Cluster, error)
	List(ctx context.Context) ([]*Cluster, error)
}
