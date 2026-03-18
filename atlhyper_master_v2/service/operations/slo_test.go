package operations

import (
	"context"
	"fmt"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// ==================== Mock: database.SLORepository ====================

type mockSLORepo struct {
	upsertedTarget *database.SLOTarget
	err            error
}

func (m *mockSLORepo) GetTargets(ctx context.Context, clusterID string) ([]*database.SLOTarget, error) {
	return nil, nil
}
func (m *mockSLORepo) GetTargetsByHost(ctx context.Context, clusterID, host string) ([]*database.SLOTarget, error) {
	return nil, nil
}
func (m *mockSLORepo) UpsertTarget(ctx context.Context, t *database.SLOTarget) error {
	m.upsertedTarget = t
	return m.err
}
func (m *mockSLORepo) DeleteTarget(ctx context.Context, clusterID, host, timeRange string) error {
	return nil
}
func (m *mockSLORepo) UpsertRouteMapping(ctx context.Context, rm *database.SLORouteMapping) error {
	return nil
}
func (m *mockSLORepo) GetRouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*database.SLORouteMapping, error) {
	return nil, nil
}
func (m *mockSLORepo) GetRouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*database.SLORouteMapping, error) {
	return nil, nil
}
func (m *mockSLORepo) GetAllRouteMappings(ctx context.Context, clusterID string) ([]*database.SLORouteMapping, error) {
	return nil, nil
}
func (m *mockSLORepo) GetAllDomains(ctx context.Context, clusterID string) ([]string, error) {
	return nil, nil
}
func (m *mockSLORepo) DeleteRouteMapping(ctx context.Context, clusterID, serviceKey string) error {
	return nil
}

// ==================== 测试用例 ====================

func TestUpsertSLOTarget_Success(t *testing.T) {
	repo := &mockSLORepo{}
	svc := NewSLOService(repo)

	req := &model.UpdateSLOTargetRequest{
		ClusterID:          "cluster-1",
		Host:               "example.com",
		TimeRange:          "1d",
		AvailabilityTarget: 99.9,
		P95LatencyTarget:   200,
	}

	err := svc.UpsertSLOTarget(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 验证 model.UpdateSLOTargetRequest → database.SLOTarget 转换
	if repo.upsertedTarget == nil {
		t.Fatal("expected UpsertTarget to be called")
	}
	if repo.upsertedTarget.ClusterID != "cluster-1" {
		t.Errorf("expected ClusterID=cluster-1, got %s", repo.upsertedTarget.ClusterID)
	}
	if repo.upsertedTarget.Host != "example.com" {
		t.Errorf("expected Host=example.com, got %s", repo.upsertedTarget.Host)
	}
	if repo.upsertedTarget.AvailabilityTarget != 99.9 {
		t.Errorf("expected AvailabilityTarget=99.9, got %f", repo.upsertedTarget.AvailabilityTarget)
	}
}

func TestUpsertSLOTarget_Error(t *testing.T) {
	repo := &mockSLORepo{err: fmt.Errorf("db write failed")}
	svc := NewSLOService(repo)

	req := &model.UpdateSLOTargetRequest{
		ClusterID:          "cluster-1",
		Host:               "example.com",
		TimeRange:          "1d",
		AvailabilityTarget: 99.9,
		P95LatencyTarget:   200,
	}

	err := svc.UpsertSLOTarget(context.Background(), req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
