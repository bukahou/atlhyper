package atlhyper_master_v2

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/agentsdk"
	"AtlHyper/atlhyper_master_v2/agentsdk/clustertoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
)

// 集群注册表的完整流程，接真实的 SQLite 与内存 Store（与 master.go 的装配方式一致）：
// 登记 → 签发 → 轮换 → 上报 → 删除登记。
// 关注的是各层接起来之后的行为：管理接口签发的令牌，Agent 入口按哈希查得到；轮换与删除登记后查不到。
func TestClusterRegistryFlow(t *testing.T) {
	ctx := context.Background()

	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "master.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())

	store := datahub.NewStore(datahub.Config{Type: "memory", EventRetention: time.Minute, HeartbeatExpire: time.Minute, SnapshotRetention: time.Minute})
	registryOps := operations.NewClusterRegistryService(db.Cluster)
	registryQuery := query.NewQueryService(query.QueryServiceDeps{
		Store:             store,
		ClusterRepo:       db.Cluster,
		ClusterAuthStatus: agentsdk.NewAuthStatusTracker(),
	})

	// lookupByToken 与 Agent 入口的认证中间件用同一种方式找集群：令牌 → 哈希 → 注册表
	lookupByToken := func(token string) *database.Cluster {
		t.Helper()
		found, err := db.Cluster.GetByTokenHash(ctx, clustertoken.Hash(token))
		if err != nil {
			t.Fatal(err)
		}
		return found
	}
	rowOf := func(clusterID string) *model.ClusterRegistryItem {
		t.Helper()
		items, err := registryQuery.ListClusterRegistry(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for i := range items {
			if items[i].ClusterID == clusterID {
				return &items[i]
			}
		}
		return nil
	}

	// 1. 登记：出现在列表里，从未上报、没有令牌
	if err := registryOps.RegisterCluster(ctx, &model.RegisterClusterRequest{ClusterID: "prod", Name: "生产"}); err != nil {
		t.Fatal(err)
	}
	if row := rowOf("prod"); row == nil || !row.Registered || row.Status != model.ClusterStatusNever || row.TokenIssued {
		t.Fatalf("登记后的行不对: %+v", row)
	}
	if err := registryOps.RegisterCluster(ctx, &model.RegisterClusterRequest{ClusterID: "prod", Name: "另一个"}); !errors.Is(err, database.ErrClusterExists) {
		t.Errorf("重复登记应返回 ErrClusterExists，实际 %v", err)
	}

	// 2. 签发：按令牌查得到这个集群；列表显示已签发与前缀
	first, err := registryOps.IssueClusterToken(ctx, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if found := lookupByToken(first.Token); found == nil || found.ClusterID != "prod" || !clustertoken.Matches(first.Token, found.TokenHash) {
		t.Fatalf("签发的令牌应当对应到 prod，实际 %+v", found)
	}
	if row := rowOf("prod"); !row.TokenIssued || row.TokenPrefix != first.TokenPrefix || row.TokenIssuedAt != first.TokenIssuedAt {
		t.Errorf("签发后的行不对: %+v（签发结果前缀 %q 时间 %q）", row, first.TokenPrefix, first.TokenIssuedAt)
	}

	// 3. 轮换：旧令牌查不到，新令牌查得到
	second, err := registryOps.IssueClusterToken(ctx, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if lookupByToken(first.Token) != nil {
		t.Error("轮换后旧令牌应当失效")
	}
	if found := lookupByToken(second.Token); found == nil || found.ClusterID != "prod" {
		t.Errorf("轮换后的新令牌应当对应到 prod，实际 %+v", found)
	}

	// 4. 上报：已登记的变为在线；没登记的也出现在列表里，标为未登记
	for _, reporting := range []string{"prod", "stray"} {
		if err := store.UpdateHeartbeat(reporting); err != nil {
			t.Fatal(err)
		}
	}
	if row := rowOf("prod"); row.Status != model.ClusterStatusOnline || row.LastSeen == "" {
		t.Errorf("上报后 prod 应为在线: %+v", row)
	}
	if row := rowOf("stray"); row == nil || row.Registered || row.Status != model.ClusterStatusOnline {
		t.Errorf("正在上报但未登记的集群应出现在列表里: %+v", row)
	}

	// 5. 删除登记：令牌失效；集群还在上报，所以仍在列表里，只是变回未登记
	if err := registryOps.UnregisterCluster(ctx, "prod"); err != nil {
		t.Fatal(err)
	}
	if lookupByToken(second.Token) != nil {
		t.Error("删除登记后令牌应当失效")
	}
	if row := rowOf("prod"); row == nil || row.Registered || row.TokenIssued || row.Status != model.ClusterStatusOnline {
		t.Errorf("删除登记后 prod 应变回未登记、仍在线: %+v", row)
	}
	if _, err := registryOps.IssueClusterToken(ctx, "prod"); !errors.Is(err, database.ErrClusterNotFound) {
		t.Errorf("未登记的集群不能签发令牌，实际 %v", err)
	}
}
