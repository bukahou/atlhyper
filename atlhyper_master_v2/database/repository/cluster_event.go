// atlhyper_master_v2/database/repository/cluster_event.go
// ClusterEventRepository 接口定义
// Event 持久化是平台核心功能
package repository

import (
	"context"
	"time"
)

// ClusterEvent 集群事件（只存 Warning）
// DedupKey = MD5(ClusterID + InvolvedKind + InvolvedNamespace + InvolvedName + Reason)
type ClusterEvent struct {
	ID                int64
	DedupKey          string // 业务去重键
	ClusterID         string
	Namespace         string
	Name              string
	Type              string // Warning（只存 Warning）
	Reason            string
	Message           string
	SourceComponent   string
	SourceHost        string
	InvolvedKind      string
	InvolvedName      string
	InvolvedNamespace string
	FirstTimestamp    time.Time
	LastTimestamp     time.Time
	Count             int32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EventQueryOpts 查询选项
type EventQueryOpts struct {
	Type      string    // 过滤类型: Normal / Warning
	Reason    string    // 过滤原因
	Since     time.Time // 起始时间
	Until     time.Time // 结束时间
	Limit     int       // 限制条数
	Offset    int       // 偏移量
}

// ClusterEventRepository Event 持久化接口
type ClusterEventRepository interface {
	// ==================== 写入 ====================

	// Upsert 插入或更新事件（基于 dedup_key 去重）
	// 存在 → 更新 count, last_timestamp, message, updated_at
	// 不存在 → 插入新记录
	Upsert(ctx context.Context, event *ClusterEvent) error

	// UpsertBatch 批量插入或更新
	UpsertBatch(ctx context.Context, events []*ClusterEvent) error

	// ==================== 查询 ====================

	// ListByCluster 按集群查询
	ListByCluster(ctx context.Context, clusterID string, opts EventQueryOpts) ([]*ClusterEvent, error)

	// ListByInvolvedResource 按关联资源查询（故障排查核心）
	ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*ClusterEvent, error)

	// ListByType 按类型查询
	ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*ClusterEvent, error)

	// ==================== 清理 ====================

	// DeleteBefore 删除指定时间之前的事件
	DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error)

	// DeleteOldest 删除最旧的事件，保留最新 keepCount 条
	DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error)

	// ==================== 统计 ====================

	// CountByCluster 统计集群事件数
	CountByCluster(ctx context.Context, clusterID string) (int64, error)

	// CountByHour 按小时统计事件数（用于趋势图）
	// 返回最近 hours 小时内每小时的 Warning/Normal 事件数
	CountByHour(ctx context.Context, clusterID string, hours int) ([]HourlyEventCount, error)

	// CountByHourAndKind 按小时和资源类型统计（用于趋势图）
	// 返回最近 hours 小时内每小时每种资源类型的事件数
	CountByHourAndKind(ctx context.Context, clusterID string, hours int) ([]HourlyKindCount, error)
}

// HourlyEventCount 每小时事件统计
type HourlyEventCount struct {
	Hour         string // 格式: 2006-01-02T15
	WarningCount int
	NormalCount  int
}

// HourlyKindCount 每小时按资源类型统计
type HourlyKindCount struct {
	Hour  string // 格式: 2006-01-02T15
	Kind  string // 资源类型: Pod, Node, Deployment 等
	Count int
}
