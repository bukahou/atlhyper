package mock

import (
	"context"
	"sync"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// MasterGateway mock
type MasterGateway struct {
	mu sync.Mutex

	PushSnapshotFn func(ctx context.Context, snapshot *cluster.ClusterSnapshot) error
	PollCommandsFn func(ctx context.Context, topic string) ([]command.Command, error)
	ReportResultFn func(ctx context.Context, result *command.Result) error
	HeartbeatFn    func(ctx context.Context) error

	// Tracking fields for assertions
	PushSnapshotCalls int
	ReportResultCalls int
	LastSnapshot      *cluster.ClusterSnapshot
	LastResult        *command.Result
}

func (m *MasterGateway) PushSnapshot(ctx context.Context, snapshot *cluster.ClusterSnapshot) error {
	m.mu.Lock()
	m.PushSnapshotCalls++
	m.LastSnapshot = snapshot
	m.mu.Unlock()
	if m.PushSnapshotFn != nil {
		return m.PushSnapshotFn(ctx, snapshot)
	}
	return nil
}

func (m *MasterGateway) PollCommands(ctx context.Context, topic string) ([]command.Command, error) {
	if m.PollCommandsFn != nil {
		return m.PollCommandsFn(ctx, topic)
	}
	return nil, nil
}

func (m *MasterGateway) ReportResult(ctx context.Context, result *command.Result) error {
	m.mu.Lock()
	m.ReportResultCalls++
	m.LastResult = result
	m.mu.Unlock()
	if m.ReportResultFn != nil {
		return m.ReportResultFn(ctx, result)
	}
	return nil
}

func (m *MasterGateway) Heartbeat(ctx context.Context) error {
	if m.HeartbeatFn != nil {
		return m.HeartbeatFn(ctx)
	}
	return nil
}
