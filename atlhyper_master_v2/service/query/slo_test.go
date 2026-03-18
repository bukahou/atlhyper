package query

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	agentmodel "AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
	slomodel "AtlHyper/model_v3/slo"
)

// ==================== Mock: database.SLORepository ====================

type mockSLORepo struct {
	targets  []*database.SLOTarget
	mappings []*database.SLORouteMapping
	domains  []string
	err      error
}

func (m *mockSLORepo) GetTargets(ctx context.Context, clusterID string) ([]*database.SLOTarget, error) {
	return m.targets, m.err
}
func (m *mockSLORepo) GetTargetsByHost(ctx context.Context, clusterID, host string) ([]*database.SLOTarget, error) {
	return m.targets, m.err
}
func (m *mockSLORepo) UpsertTarget(ctx context.Context, t *database.SLOTarget) error {
	return m.err
}
func (m *mockSLORepo) DeleteTarget(ctx context.Context, clusterID, host, timeRange string) error {
	return m.err
}
func (m *mockSLORepo) UpsertRouteMapping(ctx context.Context, rm *database.SLORouteMapping) error {
	return m.err
}
func (m *mockSLORepo) GetRouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*database.SLORouteMapping, error) {
	if len(m.mappings) > 0 {
		return m.mappings[0], m.err
	}
	return nil, m.err
}
func (m *mockSLORepo) GetRouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*database.SLORouteMapping, error) {
	return m.mappings, m.err
}
func (m *mockSLORepo) GetAllRouteMappings(ctx context.Context, clusterID string) ([]*database.SLORouteMapping, error) {
	return m.mappings, m.err
}
func (m *mockSLORepo) GetAllDomains(ctx context.Context, clusterID string) ([]string, error) {
	return m.domains, m.err
}
func (m *mockSLORepo) DeleteRouteMapping(ctx context.Context, clusterID, serviceKey string) error {
	return m.err
}

// ==================== 测试用例 ====================

func TestGetSLOTargets_Success(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	repo := &mockSLORepo{
		targets: []*database.SLOTarget{
			{
				ID: 1, ClusterID: "cluster-1", Host: "example.com",
				TimeRange: "1d", AvailabilityTarget: 99.9, P95LatencyTarget: 200,
				CreatedAt: now, UpdatedAt: now,
			},
			{
				ID: 2, ClusterID: "cluster-1", Host: "api.example.com",
				TimeRange: "7d", AvailabilityTarget: 99.5, P95LatencyTarget: 500,
				CreatedAt: now, UpdatedAt: now,
			},
		},
	}

	// QueryService 在 package query 内部，可以直接构造
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLOTargets(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(results))
	}

	// 验证 database.SLOTarget → model.SLOTargetResponse 转换
	r := results[0]
	if r.ID != 1 {
		t.Errorf("expected ID=1, got %d", r.ID)
	}
	if r.Host != "example.com" {
		t.Errorf("expected Host=example.com, got %s", r.Host)
	}
	if r.AvailabilityTarget != 99.9 {
		t.Errorf("expected AvailabilityTarget=99.9, got %f", r.AvailabilityTarget)
	}
	// 验证时间格式化（ISO 8601）
	expected := "2025-06-01T12:00:00Z"
	if r.CreatedAt != expected {
		t.Errorf("expected CreatedAt=%s, got %s", expected, r.CreatedAt)
	}
}

func TestGetSLOTargets_Empty(t *testing.T) {
	repo := &mockSLORepo{targets: []*database.SLOTarget{}}
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLOTargets(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(results))
	}
}

func TestGetSLORouteMappingsByDomain_Success(t *testing.T) {
	repo := &mockSLORepo{
		mappings: []*database.SLORouteMapping{
			{
				Domain: "example.com", PathPrefix: "/api",
				IngressName: "ing-1", Namespace: "default", TLS: true,
				ServiceKey: "default-svc-80@kubernetes",
				ServiceName: "svc", ServicePort: 80,
			},
		},
	}
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLORouteMappingsByDomain(context.Background(), "cluster-1", "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(results))
	}

	// 验证 database.SLORouteMapping → model.SLORouteMapping 转换
	m := results[0]
	if m.Domain != "example.com" {
		t.Errorf("expected Domain=example.com, got %s", m.Domain)
	}
	if m.ServiceKey != "default-svc-80@kubernetes" {
		t.Errorf("expected ServiceKey=default-svc-80@kubernetes, got %s", m.ServiceKey)
	}
	if m.ServicePort != 80 {
		t.Errorf("expected ServicePort=80, got %d", m.ServicePort)
	}
}

func TestGetSLOAllDomains_Success(t *testing.T) {
	repo := &mockSLORepo{
		domains: []string{"example.com", "api.example.com"},
	}
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLOAllDomains(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(results))
	}
	if results[0] != "example.com" {
		t.Errorf("expected first domain=example.com, got %s", results[0])
	}
}

func TestGetSLORouteMappingByServiceKey_NotFound(t *testing.T) {
	repo := &mockSLORepo{mappings: nil}
	svc := &QueryService{sloRepo: repo}

	result, err := svc.GetSLORouteMappingByServiceKey(context.Background(), "cluster-1", "nonexistent@kubernetes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for not-found, got %+v", result)
	}
}

// ==================== Phase 1: 纯辅助函数测试 ====================

func TestDetermineMeshStatus_Healthy(t *testing.T) {
	// errRate ≤ 1 且 p99 ≤ 500 → healthy
	tests := []struct {
		errRate, p99 float64
	}{
		{0, 100},
		{0.5, 500},
		{1.0, 500},
	}
	for _, tt := range tests {
		got := determineMeshStatus(tt.errRate, tt.p99)
		if got != "healthy" {
			t.Errorf("determineMeshStatus(%v, %v) = %q, want healthy", tt.errRate, tt.p99, got)
		}
	}
}

func TestDetermineMeshStatus_Warning(t *testing.T) {
	// errRate > 1 (但 ≤ 5) 或 p99 > 500 → warning
	tests := []struct {
		errRate, p99 float64
	}{
		{1.1, 200},   // errRate > 1
		{0.5, 501},   // p99 > 500
		{5.0, 100},   // errRate == 5（≤ 5 但 > 1）
	}
	for _, tt := range tests {
		got := determineMeshStatus(tt.errRate, tt.p99)
		if got != "warning" {
			t.Errorf("determineMeshStatus(%v, %v) = %q, want warning", tt.errRate, tt.p99, got)
		}
	}
}

func TestDetermineMeshStatus_Critical(t *testing.T) {
	// errRate > 5 → critical（优先于 warning）
	tests := []struct {
		errRate, p99 float64
	}{
		{5.1, 100},
		{10, 1000},
		{100, 0},
	}
	for _, tt := range tests {
		got := determineMeshStatus(tt.errRate, tt.p99)
		if got != "critical" {
			t.Errorf("determineMeshStatus(%v, %v) = %q, want critical", tt.errRate, tt.p99, got)
		}
	}
}

func TestServiceToNode(t *testing.T) {
	svc := slomodel.ServiceSLO{
		Name:        "api-server",
		Namespace:   "production",
		RPS:         150.5,
		P50Ms:       10,
		P90Ms:       50,
		P99Ms:       200,
		SuccessRate: 99.5,
		MTLSEnabled: true,
		StatusCodes: []slomodel.StatusCodeCount{
			{Code: "200", Count: 900},
			{Code: "500", Count: 100},
		},
	}

	node := serviceToNode(svc)

	if node.ID != "production/api-server" {
		t.Errorf("ID = %q, want production/api-server", node.ID)
	}
	if node.Name != "api-server" {
		t.Errorf("Name = %q", node.Name)
	}
	if node.Namespace != "production" {
		t.Errorf("Namespace = %q", node.Namespace)
	}
	if node.RPS != 150.5 {
		t.Errorf("RPS = %v, want 150.5", node.RPS)
	}
	if node.AvgLatencyMs != 10 {
		t.Errorf("AvgLatencyMs = %v, want 10 (P50)", node.AvgLatencyMs)
	}
	if node.P50LatencyMs != 10 {
		t.Errorf("P50LatencyMs = %v", node.P50LatencyMs)
	}
	if node.P95LatencyMs != 50 {
		t.Errorf("P95LatencyMs = %v, want 50 (P90)", node.P95LatencyMs)
	}
	if node.P99LatencyMs != 200 {
		t.Errorf("P99LatencyMs = %v", node.P99LatencyMs)
	}
	// ErrorRate = 100 - SuccessRate
	if node.ErrorRate != 0.5 {
		t.Errorf("ErrorRate = %v, want 0.5 (100-99.5)", node.ErrorRate)
	}
	if node.Availability != 99.5 {
		t.Errorf("Availability = %v", node.Availability)
	}
	if !node.MtlsEnabled {
		t.Error("MtlsEnabled = false, want true")
	}
	if node.TotalRequests != 1000 {
		t.Errorf("TotalRequests = %d, want 1000", node.TotalRequests)
	}
	if node.Status != "healthy" {
		t.Errorf("Status = %q, want healthy", node.Status)
	}
}

func TestConvertEdge(t *testing.T) {
	e := slomodel.ServiceEdge{
		SrcNamespace: "ns-a",
		SrcName:      "svc-a",
		DstNamespace: "ns-b",
		DstName:      "svc-b",
		RPS:          100,
		AvgMs:        25.5,
		SuccessRate:  98.0,
	}

	result := convertEdge(e)

	if result.Source != "ns-a/svc-a" {
		t.Errorf("Source = %q, want ns-a/svc-a", result.Source)
	}
	if result.Target != "ns-b/svc-b" {
		t.Errorf("Target = %q, want ns-b/svc-b", result.Target)
	}
	if result.RPS != 100 {
		t.Errorf("RPS = %v", result.RPS)
	}
	if result.AvgLatencyMs != 25.5 {
		t.Errorf("AvgLatencyMs = %v", result.AvgLatencyMs)
	}
	// ErrorRate = 100 - SuccessRate
	if result.ErrorRate != 2.0 {
		t.Errorf("ErrorRate = %v, want 2.0 (100-98)", result.ErrorRate)
	}
}

func TestConvertEdges_Empty(t *testing.T) {
	result := convertEdges(nil)
	if result == nil {
		t.Fatal("convertEdges(nil) returned nil, want empty slice")
	}
	if len(result) != 0 {
		t.Fatalf("convertEdges(nil) returned %d items, want 0", len(result))
	}
}

func TestTotalFromStatusCodes(t *testing.T) {
	codes := []slomodel.StatusCodeCount{
		{Code: "200", Count: 500},
		{Code: "404", Count: 30},
		{Code: "500", Count: 10},
	}
	got := totalFromStatusCodes(codes)
	if got != 540 {
		t.Fatalf("totalFromStatusCodes = %d, want 540", got)
	}
}

func TestTotalFromStatusCodes_Empty(t *testing.T) {
	if got := totalFromStatusCodes(nil); got != 0 {
		t.Fatalf("totalFromStatusCodes(nil) = %d, want 0", got)
	}
	if got := totalFromStatusCodes([]slomodel.StatusCodeCount{}); got != 0 {
		t.Fatalf("totalFromStatusCodes([]) = %d, want 0", got)
	}
}

func TestGetTimeStart(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		timeRange string
		want      time.Duration // 期望的 now - result 差值
	}{
		{"1h", "1h", time.Hour},
		{"6h", "6h", 6 * time.Hour},
		{"24h", "24h", 24 * time.Hour},
		{"1d alias", "1d", 24 * time.Hour},
		{"7d", "7d", 7 * 24 * time.Hour},
		{"30d", "30d", 30 * 24 * time.Hour},
		{"unknown defaults to 24h", "unknown", 24 * time.Hour},
		{"empty defaults to 24h", "", 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTimeStart(now, tt.timeRange)
			diff := now.Sub(result)
			if diff != tt.want {
				t.Errorf("getTimeStart(now, %q): diff = %v, want %v", tt.timeRange, diff, tt.want)
			}
		})
	}
}

// ==================== Phase 2: Mock Store + GetMeshTopology / GetServiceDetail ====================

// mockStoreForSLO 最小 mock，只实现 GetSnapshot，其余空实现
type mockStoreForSLO struct {
	snapshot *cluster.ClusterSnapshot
}

func (m *mockStoreForSLO) SetSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
	return nil
}
func (m *mockStoreForSLO) GetSnapshot(clusterID string) (*cluster.ClusterSnapshot, error) {
	return m.snapshot, nil
}
func (m *mockStoreForSLO) UpdateHeartbeat(clusterID string) error { return nil }
func (m *mockStoreForSLO) GetAgentStatus(clusterID string) (*agentmodel.AgentStatus, error) {
	return nil, nil
}
func (m *mockStoreForSLO) ListAgents() ([]agentmodel.AgentInfo, error) { return nil, nil }
func (m *mockStoreForSLO) GetEvents(clusterID string) ([]cluster.Event, error) { return nil, nil }
func (m *mockStoreForSLO) GetOTelTimeline(clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	return nil, nil
}
func (m *mockStoreForSLO) Start() error { return nil }
func (m *mockStoreForSLO) Stop() error  { return nil }

// ==================== GetMeshTopology 测试 ====================

func TestGetMeshTopology_FromSLOWindows(t *testing.T) {
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{
			OTel: &cluster.OTelSnapshot{
				SLOWindows: map[string]*slomodel.SLOWindowData{
					"1d": {
						MeshServices: []slomodel.ServiceSLO{
							{Name: "svc-a", Namespace: "ns-a", SuccessRate: 99, P99Ms: 100},
							{Name: "svc-b", Namespace: "ns-b", SuccessRate: 95, P99Ms: 200},
						},
						MeshEdges: []slomodel.ServiceEdge{
							{SrcNamespace: "ns-a", SrcName: "svc-a", DstNamespace: "ns-b", DstName: "svc-b", RPS: 50, SuccessRate: 98},
						},
					},
				},
				// 回退数据（不应使用）
				SLOServices: []slomodel.ServiceSLO{
					{Name: "fallback-svc", Namespace: "fallback-ns"},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetMeshTopology(context.Background(), "cluster-1", "1d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(resp.Nodes) != 2 {
		t.Fatalf("Nodes = %d, want 2", len(resp.Nodes))
	}
	if resp.Nodes[0].Name != "svc-a" {
		t.Errorf("Nodes[0].Name = %q, want svc-a", resp.Nodes[0].Name)
	}
	if len(resp.Edges) != 1 {
		t.Fatalf("Edges = %d, want 1", len(resp.Edges))
	}
	if resp.Edges[0].Source != "ns-a/svc-a" {
		t.Errorf("Edge Source = %q", resp.Edges[0].Source)
	}
}

func TestGetMeshTopology_FallbackToSnapshot(t *testing.T) {
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{
			OTel: &cluster.OTelSnapshot{
				// SLOWindows 无 "7d" 匹配
				SLOWindows: map[string]*slomodel.SLOWindowData{
					"1d": {MeshServices: []slomodel.ServiceSLO{{Name: "wrong"}}},
				},
				// 回退到这里
				SLOServices: []slomodel.ServiceSLO{
					{Name: "fallback", Namespace: "default", SuccessRate: 100},
				},
				SLOEdges: []slomodel.ServiceEdge{
					{SrcNamespace: "default", SrcName: "fallback", DstNamespace: "default", DstName: "db", SuccessRate: 99},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetMeshTopology(context.Background(), "cluster-1", "7d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(resp.Nodes) != 1 {
		t.Fatalf("Nodes = %d, want 1 (fallback)", len(resp.Nodes))
	}
	if resp.Nodes[0].Name != "fallback" {
		t.Errorf("Nodes[0].Name = %q, want fallback", resp.Nodes[0].Name)
	}
	if len(resp.Edges) != 1 {
		t.Fatalf("Edges = %d, want 1", len(resp.Edges))
	}
}

func TestGetMeshTopology_NoOTel(t *testing.T) {
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{OTel: nil},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetMeshTopology(context.Background(), "cluster-1", "1d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil, want empty response")
	}
	if len(resp.Nodes) != 0 {
		t.Errorf("Nodes = %d, want 0", len(resp.Nodes))
	}
}

func TestGetMeshTopology_NoSnapshot(t *testing.T) {
	store := &mockStoreForSLO{snapshot: nil}
	svc := &QueryService{store: store}

	resp, err := svc.GetMeshTopology(context.Background(), "nonexistent", "1d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil, want empty response")
	}
	if len(resp.Nodes) != 0 {
		t.Errorf("Nodes = %d, want 0", len(resp.Nodes))
	}
}

// ==================== GetServiceDetail 测试 ====================

func TestGetServiceDetail_FullData(t *testing.T) {
	ts1 := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{
			OTel: &cluster.OTelSnapshot{
				SLOWindows: map[string]*slomodel.SLOWindowData{
					"1d": {
						MeshServices: []slomodel.ServiceSLO{
							{
								Name: "api", Namespace: "prod",
								RPS: 100, P50Ms: 10, P90Ms: 50, P99Ms: 200, SuccessRate: 99.5,
								StatusCodes: []slomodel.StatusCodeCount{
									{Code: "200", Count: 900},
									{Code: "500", Count: 10},
								},
								LatencyBuckets: []slomodel.LatencyBucket{
									{LE: 50, Count: 800},
									{LE: 100, Count: 950},
								},
							},
						},
						MeshEdges: []slomodel.ServiceEdge{
							{SrcNamespace: "prod", SrcName: "web", DstNamespace: "prod", DstName: "api", RPS: 80, SuccessRate: 99},
							{SrcNamespace: "prod", SrcName: "api", DstNamespace: "prod", DstName: "db", RPS: 60, SuccessRate: 100},
						},
					},
				},
				SLOTimeSeries: []cluster.SLOServiceTimeSeries{
					{
						ServiceName: "prod/api",
						Points: []cluster.SLOTimePoint{
							{Timestamp: ts1, RPS: 95, P99Ms: 180, ErrorRate: 0.5, SuccessRate: 99.5},
						},
					},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetServiceDetail(context.Background(), "cluster-1", "prod", "api", "1d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}

	// 1. 节点基础信息
	if resp.Name != "api" || resp.Namespace != "prod" {
		t.Errorf("Name=%q Namespace=%q", resp.Name, resp.Namespace)
	}
	if resp.RPS != 100 {
		t.Errorf("RPS = %v, want 100", resp.RPS)
	}

	// 2. 历史时序
	if len(resp.History) != 1 {
		t.Fatalf("History = %d, want 1", len(resp.History))
	}
	if resp.History[0].RPS != 95 {
		t.Errorf("History[0].RPS = %v, want 95", resp.History[0].RPS)
	}

	// 3. 状态码
	if len(resp.StatusCodes) != 2 {
		t.Fatalf("StatusCodes = %d, want 2", len(resp.StatusCodes))
	}

	// 4. 延迟分布桶
	if len(resp.LatencyBuckets) != 2 {
		t.Fatalf("LatencyBuckets = %d, want 2", len(resp.LatencyBuckets))
	}
	if resp.LatencyBuckets[0].LE != 50 {
		t.Errorf("LatencyBuckets[0].LE = %v, want 50", resp.LatencyBuckets[0].LE)
	}

	// 5. 上下游边
	if len(resp.Upstreams) != 1 {
		t.Fatalf("Upstreams = %d, want 1", len(resp.Upstreams))
	}
	if resp.Upstreams[0].Source != "prod/web" {
		t.Errorf("Upstream source = %q, want prod/web", resp.Upstreams[0].Source)
	}
	if len(resp.Downstreams) != 1 {
		t.Fatalf("Downstreams = %d, want 1", len(resp.Downstreams))
	}
	if resp.Downstreams[0].Target != "prod/db" {
		t.Errorf("Downstream target = %q, want prod/db", resp.Downstreams[0].Target)
	}
}

func TestGetServiceDetail_ServiceNotFound(t *testing.T) {
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{
			OTel: &cluster.OTelSnapshot{
				SLOServices: []slomodel.ServiceSLO{
					{Name: "other-svc", Namespace: "default"},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetServiceDetail(context.Background(), "cluster-1", "default", "nonexistent", "1d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil for not-found service, got %+v", resp)
	}
}

func TestGetServiceDetail_NoOTel(t *testing.T) {
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{OTel: nil},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetServiceDetail(context.Background(), "cluster-1", "ns", "svc", "1d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil when no OTel, got %+v", resp)
	}
}

func TestGetServiceDetail_UpstreamDownstream(t *testing.T) {
	// 中间节点 B: A→B（upstream）, B→C（downstream）
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{
			OTel: &cluster.OTelSnapshot{
				SLOServices: []slomodel.ServiceSLO{
					{Name: "svc-b", Namespace: "ns", SuccessRate: 100},
				},
				SLOEdges: []slomodel.ServiceEdge{
					{SrcNamespace: "ns", SrcName: "svc-a", DstNamespace: "ns", DstName: "svc-b", RPS: 10, SuccessRate: 99},
					{SrcNamespace: "ns", SrcName: "svc-b", DstNamespace: "ns", DstName: "svc-c", RPS: 8, SuccessRate: 100},
					{SrcNamespace: "ns", SrcName: "svc-x", DstNamespace: "ns", DstName: "svc-y", RPS: 5, SuccessRate: 100}, // 无关边
				},
			},
		},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetServiceDetail(context.Background(), "cluster-1", "ns", "svc-b", "1d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil")
	}

	// svc-a → svc-b: upstream of svc-b
	if len(resp.Upstreams) != 1 {
		t.Fatalf("Upstreams = %d, want 1", len(resp.Upstreams))
	}
	if resp.Upstreams[0].Source != "ns/svc-a" {
		t.Errorf("Upstream source = %q, want ns/svc-a", resp.Upstreams[0].Source)
	}

	// svc-b → svc-c: downstream of svc-b
	if len(resp.Downstreams) != 1 {
		t.Fatalf("Downstreams = %d, want 1", len(resp.Downstreams))
	}
	if resp.Downstreams[0].Target != "ns/svc-c" {
		t.Errorf("Downstream target = %q, want ns/svc-c", resp.Downstreams[0].Target)
	}
}

func TestGetServiceDetail_FallbackToSnapshot(t *testing.T) {
	store := &mockStoreForSLO{
		snapshot: &cluster.ClusterSnapshot{
			OTel: &cluster.OTelSnapshot{
				// SLOWindows 无 "7d" 匹配
				SLOWindows: map[string]*slomodel.SLOWindowData{
					"1d": {MeshServices: []slomodel.ServiceSLO{{Name: "wrong"}}},
				},
				// 回退到这里
				SLOServices: []slomodel.ServiceSLO{
					{Name: "fallback-svc", Namespace: "default", SuccessRate: 98, P99Ms: 300},
				},
			},
		},
	}
	svc := &QueryService{store: store}

	resp, err := svc.GetServiceDetail(context.Background(), "cluster-1", "default", "fallback-svc", "7d")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if resp == nil {
		t.Fatal("response is nil, expected fallback service")
	}
	if resp.Name != "fallback-svc" {
		t.Errorf("Name = %q, want fallback-svc", resp.Name)
	}
}
