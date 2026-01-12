// atlhyper_master/repository/interfaces.go
// 仓库层接口定义
package repository

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master/model/entity"
	"AtlHyper/model/transport"
)

// ============================================================
// SQL 仓库接口（持久化数据）
// ============================================================

// UserRepository 用户仓库接口
type UserRepository interface {
	GetByID(ctx context.Context, id int) (*entity.User, error)
	GetByUsername(ctx context.Context, username string) (*entity.User, error)
	GetAll(ctx context.Context) ([]entity.User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByID(ctx context.Context, id int) (bool, error)
	Count(ctx context.Context) (int, error)
	Insert(ctx context.Context, u *entity.User) (int64, error)
	UpdateRole(ctx context.Context, id int, role int) error
	UpdateLastLogin(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

// AuditRepository 审计仓库接口
type AuditRepository interface {
	Insert(ctx context.Context, log *AuditLog) error
	GetAll(ctx context.Context) ([]AuditLog, error)
	GetByUserID(ctx context.Context, userID int, limit int) ([]AuditLog, error)
}

// EventRepository 事件仓库接口（SQL 持久化）
type EventRepository interface {
	Insert(ctx context.Context, e *transport.EventLog) error
	InsertBatch(ctx context.Context, events []transport.EventLog) error
	GetSince(ctx context.Context, clusterID string, since string) ([]transport.EventLog, error)
}

// ConfigRepository 配置仓库接口
type ConfigRepository interface {
	GetSlackConfig(ctx context.Context) (*SlackConfig, error)
	UpdateSlackConfig(ctx context.Context, cfg *SlackConfig) error
	GetMailConfig(ctx context.Context) (*MailConfig, error)
	UpdateMailConfig(ctx context.Context, cfg *MailConfig) error
}

// MetricsRepository 指标仓库接口（SQL 持久化）
type MetricsRepository interface {
	UpsertNodeMetrics(ctx context.Context, metrics *NodeMetricsFlat) error
	UpsertTopProcesses(ctx context.Context, nodeName, ts string, procs []TopProcess) error
	CleanupBefore(ctx context.Context, cutoff string) (metricsDeleted, procsDeleted int64, err error)
	GetLatestByNode(ctx context.Context, nodeName string) (*NodeMetricsFlat, error)
}

// ============================================================
// 内存仓库接口（运行时数据）
// ============================================================

// MemReader 内存数据读取接口
type MemReader interface {
	// 事件
	GetK8sEventsRecent(ctx context.Context, clusterID string, limit int) ([]LogEvent, error)

	// 指标
	GetClusterMetricsLatest(ctx context.Context, clusterID string) ([]NodeMetricsSnapshot, error)
	GetClusterMetricsRange(ctx context.Context, clusterID string, since, until time.Time) ([]NodeMetricsSnapshot, error)

	// 资源列表
	GetPodListLatest(ctx context.Context, clusterID string) ([]Pod, error)
	GetNodeListLatest(ctx context.Context, clusterID string) ([]Node, error)
	GetServiceListLatest(ctx context.Context, clusterID string) ([]Service, error)
	GetNamespaceListLatest(ctx context.Context, clusterID string) ([]Namespace, error)
	GetIngressListLatest(ctx context.Context, clusterID string) ([]Ingress, error)
	GetDeploymentListLatest(ctx context.Context, clusterID string) ([]Deployment, error)
	GetConfigMapListLatest(ctx context.Context, clusterID string) ([]ConfigMap, error)

	// 集群列表
	ListClusterIDs(ctx context.Context) ([]string, error)
}

// MemWriter 内存数据写入接口
type MemWriter interface {
	AppendEnvelope(ctx context.Context, env transport.Envelope) error
	AppendEnvelopeBatch(ctx context.Context, envs []transport.Envelope) error
	ReplaceLatest(ctx context.Context, env transport.Envelope) error
}
