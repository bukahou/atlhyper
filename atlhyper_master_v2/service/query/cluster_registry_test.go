package query

import (
	"context"
	"errors"
	"testing"
	"time"

	agentmodel "AtlHyper/model_v3/agent"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// fakeClusterRegistry 注册表的假实现：只支持读（查询层不写注册表）
type fakeClusterRegistry struct {
	clusters []*database.Cluster
	err      error
}

func (f *fakeClusterRegistry) List(context.Context) ([]*database.Cluster, error) {
	return f.clusters, f.err
}

func (f *fakeClusterRegistry) GetByID(_ context.Context, clusterID string) (*database.Cluster, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, registered := range f.clusters {
		if registered.ClusterID == clusterID {
			return registered, nil
		}
	}
	return nil, nil
}

// fakeAuthStatus 认证状态的假实现：键存在 = 该集群上报过，值 = 最近一次是否带了有效令牌
type fakeAuthStatus map[string]bool

func (f fakeAuthStatus) IsAuthenticated(clusterID string) (authenticated, known bool) {
	authenticated, known = f[clusterID]
	return authenticated, known
}

func newRegistryQueryService(registry *fakeClusterRegistry, agents []agentmodel.AgentInfo, auth ClusterAuthStatus) *QueryService {
	return NewQueryService(QueryServiceDeps{
		Store:             &mockStoreForOverview{agents: agents},
		ClusterRepo:       registry,
		ClusterAuthStatus: auth,
	})
}

func TestListClusterRegistry_MergesRegistryPresenceAndAuth(t *testing.T) {
	issuedAt := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	seenAt := time.Date(2026, 11, 14, 10, 5, 0, 0, time.UTC)
	registry := &fakeClusterRegistry{clusters: []*database.Cluster{
		{ClusterID: "prod", Name: "生产", TokenHash: "not-a-real-hash", TokenPrefix: "atlk_Ab3", TokenIssuedAt: &issuedAt},
		{ClusterID: "idle", Name: "备用"},
		{ClusterID: "legacy", Name: "旧集群"},
	}}
	agents := []agentmodel.AgentInfo{
		// 故意乱序：结果必须按集群 ID 排序
		{ClusterID: "unregistered", Status: agentmodel.StatusOnline, LastHeartbeat: seenAt},
		{ClusterID: "prod", Status: agentmodel.StatusOnline, LastHeartbeat: seenAt},
		{ClusterID: "legacy", Status: agentmodel.StatusOffline, LastHeartbeat: seenAt},
	}
	auth := fakeAuthStatus{"prod": true, "legacy": false, "unregistered": false}

	got, err := newRegistryQueryService(registry, agents, auth).ListClusterRegistry(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	want := []model.ClusterRegistryItem{
		{ClusterID: "idle", Name: "备用", Registered: true, Status: model.ClusterStatusNever},
		{ClusterID: "legacy", Name: "旧集群", Registered: true, Status: model.ClusterStatusOffline, LastSeen: "2026-11-14T10:05:00Z"},
		{
			ClusterID: "prod", Name: "生产", Registered: true,
			TokenIssued: true, TokenPrefix: "atlk_Ab3", TokenIssuedAt: "2026-11-14T10:00:00Z",
			Status: model.ClusterStatusOnline, LastSeen: "2026-11-14T10:05:00Z", Authenticated: true,
		},
		{ClusterID: "unregistered", Registered: false, Status: model.ClusterStatusOnline, LastSeen: "2026-11-14T10:05:00Z"},
	}
	if len(got) != len(want) {
		t.Fatalf("行数 = %d, want %d：%+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 行\n got %+v\nwant %+v", i, got[i], want[i])
		}
	}
}

func TestListClusterRegistry_Edges(t *testing.T) {
	seenAt := time.Date(2026, 11, 14, 10, 5, 0, 0, time.UTC)
	online := []agentmodel.AgentInfo{{ClusterID: "c1", Status: agentmodel.StatusOnline, LastHeartbeat: seenAt}}

	t.Run("两侧都为空时返回空切片而非 nil", func(t *testing.T) {
		got, err := newRegistryQueryService(&fakeClusterRegistry{}, nil, fakeAuthStatus{}).ListClusterRegistry(context.Background())
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("got (%v, %v), want 空切片", got, err)
		}
	})

	t.Run("没上报过的集群认证状态为假", func(t *testing.T) {
		registry := &fakeClusterRegistry{clusters: []*database.Cluster{{ClusterID: "c1", Name: "生产"}}}
		got, err := newRegistryQueryService(registry, nil, fakeAuthStatus{}).ListClusterRegistry(context.Background())
		if err != nil || len(got) != 1 || got[0].Authenticated {
			t.Errorf("got (%+v, %v)", got, err)
		}
	})

	t.Run("没有接入认证状态时一律为假，不崩溃", func(t *testing.T) {
		got, err := newRegistryQueryService(&fakeClusterRegistry{}, online, nil).ListClusterRegistry(context.Background())
		if err != nil || len(got) != 1 || got[0].Authenticated {
			t.Errorf("got (%+v, %v)", got, err)
		}
	})

	t.Run("注册表读取失败时报错", func(t *testing.T) {
		registry := &fakeClusterRegistry{err: errors.New("db down")}
		if _, err := newRegistryQueryService(registry, online, fakeAuthStatus{}).ListClusterRegistry(context.Background()); err == nil {
			t.Error("期望报错")
		}
	})
}

func TestGetClusterRegistryItem(t *testing.T) {
	seenAt := time.Date(2026, 11, 14, 10, 5, 0, 0, time.UTC)
	registry := &fakeClusterRegistry{clusters: []*database.Cluster{
		{ClusterID: "prod", Name: "生产"},
		{ClusterID: "idle", Name: "备用"},
	}}
	agents := []agentmodel.AgentInfo{
		{ClusterID: "prod", Status: agentmodel.StatusOnline, LastHeartbeat: seenAt},
		{ClusterID: "unregistered", Status: agentmodel.StatusOnline, LastHeartbeat: seenAt},
	}
	svc := newRegistryQueryService(registry, agents, fakeAuthStatus{"prod": true})

	cases := []struct {
		name      string
		clusterID string
		want      *model.ClusterRegistryItem
	}{
		{"已登记且在线", "prod", &model.ClusterRegistryItem{
			ClusterID: "prod", Name: "生产", Registered: true,
			Status: model.ClusterStatusOnline, LastSeen: "2026-11-14T10:05:00Z", Authenticated: true,
		}},
		{"已登记未上报", "idle", &model.ClusterRegistryItem{
			ClusterID: "idle", Name: "备用", Registered: true, Status: model.ClusterStatusNever,
		}},
		{"只在上报", "unregistered", &model.ClusterRegistryItem{
			ClusterID: "unregistered", Status: model.ClusterStatusOnline, LastSeen: "2026-11-14T10:05:00Z",
		}},
		{"两侧都没有", "missing", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := svc.GetClusterRegistryItem(context.Background(), c.clusterID)
			if err != nil {
				t.Fatal(err)
			}
			if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
				t.Errorf("\n got %+v\nwant %+v", got, c.want)
			}
		})
	}
}
