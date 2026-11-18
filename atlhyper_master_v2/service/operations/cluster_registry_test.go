package operations

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/agentsdk/clustertoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// fakeClusterRegistryWriter 注册表写接口的假实现，行为与真实仓库一致：
// 重复登记返回 ErrClusterExists，操作未登记的集群返回 ErrClusterNotFound
type fakeClusterRegistryWriter struct {
	clusters map[string]*database.Cluster
	writes   int
	failWith error
}

func newFakeClusterRegistryWriter(registeredIDs ...string) *fakeClusterRegistryWriter {
	fake := &fakeClusterRegistryWriter{clusters: map[string]*database.Cluster{}}
	for _, clusterID := range registeredIDs {
		fake.clusters[clusterID] = &database.Cluster{ClusterID: clusterID, Name: clusterID}
	}
	return fake
}

func (f *fakeClusterRegistryWriter) Create(_ context.Context, cluster *database.Cluster) error {
	f.writes++
	if f.failWith != nil {
		return f.failWith
	}
	if _, exists := f.clusters[cluster.ClusterID]; exists {
		return database.ErrClusterExists
	}
	stored := *cluster
	f.clusters[cluster.ClusterID] = &stored
	return nil
}

func (f *fakeClusterRegistryWriter) UpdateProfile(_ context.Context, clusterID, name, description string) error {
	f.writes++
	registered, exists := f.clusters[clusterID]
	if !exists {
		return database.ErrClusterNotFound
	}
	registered.Name, registered.Description = name, description
	return nil
}

func (f *fakeClusterRegistryWriter) SetToken(_ context.Context, clusterID, tokenHash, tokenPrefix string, issuedAt time.Time) error {
	f.writes++
	registered, exists := f.clusters[clusterID]
	if !exists {
		return database.ErrClusterNotFound
	}
	registered.TokenHash, registered.TokenPrefix, registered.TokenIssuedAt = tokenHash, tokenPrefix, &issuedAt
	return nil
}

func (f *fakeClusterRegistryWriter) Delete(_ context.Context, clusterID string) error {
	f.writes++
	if _, exists := f.clusters[clusterID]; !exists {
		return database.ErrClusterNotFound
	}
	delete(f.clusters, clusterID)
	return nil
}

func TestRegisterCluster(t *testing.T) {
	cases := []struct {
		name       string
		req        model.RegisterClusterRequest
		wantErr    error
		wantStored *database.Cluster
	}{
		{
			name:       "正常登记",
			req:        model.RegisterClusterRequest{ClusterID: "prod", Name: "生产", Description: "东京"},
			wantStored: &database.Cluster{ClusterID: "prod", Name: "生产", Description: "东京"},
		},
		{
			name:       "显示名与描述去掉首尾空白",
			req:        model.RegisterClusterRequest{ClusterID: "prod", Name: "  生产 ", Description: " 东京\t"},
			wantStored: &database.Cluster{ClusterID: "prod", Name: "生产", Description: "东京"},
		},
		{name: "集群 ID 不合规", req: model.RegisterClusterRequest{ClusterID: "Prod_1", Name: "生产"}, wantErr: model.ErrInvalidClusterInput},
		{name: "集群 ID 首尾空白不被自动修正", req: model.RegisterClusterRequest{ClusterID: " prod", Name: "生产"}, wantErr: model.ErrInvalidClusterInput},
		{name: "显示名只有空白", req: model.RegisterClusterRequest{ClusterID: "prod", Name: "   "}, wantErr: model.ErrInvalidClusterInput},
		{name: "集群 ID 已登记", req: model.RegisterClusterRequest{ClusterID: "existing", Name: "生产"}, wantErr: database.ErrClusterExists},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := newFakeClusterRegistryWriter("existing")
			err := NewClusterRegistryService(repo).RegisterCluster(context.Background(), &c.req)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if errors.Is(c.wantErr, model.ErrInvalidClusterInput) && repo.writes != 0 {
				t.Errorf("输入不合规时不应触碰注册表，实际写了 %d 次", repo.writes)
			}
			if c.wantStored != nil {
				if got := repo.clusters[c.wantStored.ClusterID]; got == nil || *got != *c.wantStored {
					t.Errorf("入库 %+v, want %+v", got, c.wantStored)
				}
			}
		})
	}
}

// 新登记的集群没有令牌：令牌只能经签发得到
func TestRegisterCluster_StartsWithoutToken(t *testing.T) {
	repo := newFakeClusterRegistryWriter()
	req := &model.RegisterClusterRequest{ClusterID: "prod", Name: "生产"}
	if err := NewClusterRegistryService(repo).RegisterCluster(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	stored := repo.clusters["prod"]
	if stored.TokenHash != "" || stored.TokenPrefix != "" || stored.TokenIssuedAt != nil {
		t.Errorf("新登记的集群不应带令牌: %+v", stored)
	}
}

func TestUpdateClusterProfile(t *testing.T) {
	cases := []struct {
		name      string
		clusterID string
		req       model.UpdateClusterProfileRequest
		wantErr   error
	}{
		{"正常修改", "prod", model.UpdateClusterProfileRequest{Name: " 生产二 ", Description: "大阪"}, nil},
		{"显示名为空", "prod", model.UpdateClusterProfileRequest{Name: ""}, model.ErrInvalidClusterInput},
		{"描述超长", "prod", model.UpdateClusterProfileRequest{Name: "生产", Description: strings.Repeat("述", 300)}, model.ErrInvalidClusterInput},
		{"集群未登记", "missing", model.UpdateClusterProfileRequest{Name: "生产"}, database.ErrClusterNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := newFakeClusterRegistryWriter("prod")
			err := NewClusterRegistryService(repo).UpdateClusterProfile(context.Background(), c.clusterID, &c.req)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if c.wantErr == nil {
				if got := repo.clusters["prod"]; got.Name != "生产二" || got.Description != "大阪" {
					t.Errorf("入库 %+v", got)
				}
			}
		})
	}
}

func TestIssueClusterToken_StoresOnlyHashAndPrefix(t *testing.T) {
	repo := newFakeClusterRegistryWriter("prod")
	svc := NewClusterRegistryService(repo)
	issuedAt := time.Date(2026, 11, 14, 19, 0, 0, 0, time.FixedZone("JST", 9*3600))
	svc.now = func() time.Time { return issuedAt }

	issued, err := svc.IssueClusterToken(context.Background(), "prod")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(issued.Token, "atlk_") || !strings.HasPrefix(issued.Token, issued.TokenPrefix) {
		t.Errorf("令牌形态不对: 前缀 %q", issued.TokenPrefix)
	}
	if issued.TokenIssuedAt != "2026-11-14T10:00:00Z" {
		t.Errorf("签发时间 = %q, want UTC 的 RFC 3339", issued.TokenIssuedAt)
	}

	stored := repo.clusters["prod"]
	if !clustertoken.Matches(issued.Token, stored.TokenHash) {
		t.Error("库里的哈希应当与返回的明文相符")
	}
	if stored.TokenPrefix != issued.TokenPrefix {
		t.Errorf("库里的前缀 %q 与返回的 %q 不一致", stored.TokenPrefix, issued.TokenPrefix)
	}
	if stored.TokenIssuedAt == nil || !stored.TokenIssuedAt.Equal(issuedAt) {
		t.Errorf("库里的签发时间 = %v", stored.TokenIssuedAt)
	}
	// 明文不得以任何形式入库
	for _, column := range []string{stored.TokenHash, stored.TokenPrefix, stored.Name, stored.Description} {
		if strings.Contains(column, issued.Token) {
			t.Error("令牌明文出现在了入库字段里")
		}
	}
}

// 再次签发 = 轮换：新令牌与旧的不同，旧令牌不再与库里的哈希相符
func TestIssueClusterToken_ReissueReplacesOldToken(t *testing.T) {
	repo := newFakeClusterRegistryWriter("prod")
	svc := NewClusterRegistryService(repo)

	first, err := svc.IssueClusterToken(context.Background(), "prod")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.IssueClusterToken(context.Background(), "prod")
	if err != nil {
		t.Fatal(err)
	}

	if first.Token == second.Token {
		t.Fatal("两次签发得到了同一枚令牌")
	}
	storedHash := repo.clusters["prod"].TokenHash
	if clustertoken.Matches(first.Token, storedHash) {
		t.Error("重新签发后旧令牌应当失效")
	}
	if !clustertoken.Matches(second.Token, storedHash) {
		t.Error("新令牌应当与库里的哈希相符")
	}
}

func TestIssueClusterToken_Failures(t *testing.T) {
	t.Run("集群未登记时不返回令牌", func(t *testing.T) {
		issued, err := NewClusterRegistryService(newFakeClusterRegistryWriter()).IssueClusterToken(context.Background(), "missing")
		if !errors.Is(err, database.ErrClusterNotFound) || issued != nil {
			t.Errorf("got (%v, %v), want (nil, ErrClusterNotFound)", issued != nil, err)
		}
	})

	t.Run("令牌生成失败时不写注册表", func(t *testing.T) {
		repo := newFakeClusterRegistryWriter("prod")
		svc := NewClusterRegistryService(repo)
		svc.generateToken = func() (clustertoken.Issued, error) {
			return clustertoken.Issued{}, errors.New("entropy unavailable")
		}
		issued, err := svc.IssueClusterToken(context.Background(), "prod")
		if err == nil || issued != nil || repo.writes != 0 {
			t.Errorf("got (issued=%v, err=%v, writes=%d)", issued != nil, err, repo.writes)
		}
	})
}

func TestUnregisterCluster(t *testing.T) {
	repo := newFakeClusterRegistryWriter("prod")
	svc := NewClusterRegistryService(repo)

	if err := svc.UnregisterCluster(context.Background(), "prod"); err != nil {
		t.Fatal(err)
	}
	if _, exists := repo.clusters["prod"]; exists {
		t.Error("删除登记后注册表里不应再有这个集群")
	}
	if err := svc.UnregisterCluster(context.Background(), "prod"); !errors.Is(err, database.ErrClusterNotFound) {
		t.Errorf("重复删除应返回 ErrClusterNotFound，实际 %v", err)
	}
}
