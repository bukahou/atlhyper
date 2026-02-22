package mock

import (
	"context"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// SnapshotService mock
type SnapshotService struct {
	CollectFn func(ctx context.Context) (*cluster.ClusterSnapshot, error)
}

func (m *SnapshotService) Collect(ctx context.Context) (*cluster.ClusterSnapshot, error) {
	if m.CollectFn != nil {
		return m.CollectFn(ctx)
	}
	return &cluster.ClusterSnapshot{}, nil
}

// CommandService mock
type CommandService struct {
	ExecuteFn func(ctx context.Context, cmd *command.Command) *command.Result
}

func (m *CommandService) Execute(ctx context.Context, cmd *command.Command) *command.Result {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, cmd)
	}
	return &command.Result{Success: true}
}
