// atlhyper_master_v2/database/repository/command_history.go
// CommandHistoryRepository 接口定义
package repository

import (
	"context"
	"time"
)

// CommandHistory 指令历史
type CommandHistory struct {
	ID              int64
	CommandID       string // 指令唯一 ID
	ClusterID       string
	Source          string // web / ai
	UserID          int64
	Action          string // scale / restart / delete_pod
	TargetKind      string
	TargetNamespace string
	TargetName      string
	Params          string // JSON
	Status          string // pending / running / success / failed / timeout
	Result          string // JSON
	ErrorMessage    string
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	DurationMs      int64
}

// CommandHistoryRepository 指令历史接口
type CommandHistoryRepository interface {
	Create(ctx context.Context, cmd *CommandHistory) error
	Update(ctx context.Context, cmd *CommandHistory) error
	GetByCommandID(ctx context.Context, cmdID string) (*CommandHistory, error)
	ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*CommandHistory, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*CommandHistory, error)
}
