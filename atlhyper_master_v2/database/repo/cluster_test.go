package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

func registerCluster(t *testing.T, db *database.DB, clusterID, name string) {
	t.Helper()
	if err := db.Cluster.Create(context.Background(), &database.Cluster{ClusterID: clusterID, Name: name}); err != nil {
		t.Fatal(err)
	}
}

func TestClusterCreate_ThenGet(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	registerCluster(t, db, "c1", "生产")

	got, err := db.Cluster.GetByID(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ClusterID != "c1" || got.Name != "生产" {
		t.Fatalf("取回的集群不对: %+v", got)
	}
	if got.TokenHash != "" || got.TokenPrefix != "" || got.TokenIssuedAt != nil {
		t.Errorf("新登记的集群不应有令牌: %+v", got)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Errorf("时间戳没有写入: %+v", got)
	}
}

func TestClusterCreate_DuplicateID(t *testing.T) {
	db := openTestDB(t)
	registerCluster(t, db, "c1", "生产")

	err := db.Cluster.Create(context.Background(), &database.Cluster{ClusterID: "c1", Name: "另一个"})
	if !errors.Is(err, database.ErrClusterExists) {
		t.Errorf("重复登记应返回 ErrClusterExists，实际 %v", err)
	}
}

func TestClusterGetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	got, err := db.Cluster.GetByID(context.Background(), "missing")
	if err != nil || got != nil {
		t.Errorf("不存在的集群应返回 (nil, nil)，实际 (%+v, %v)", got, err)
	}
}

func TestClusterSetToken_LookupByHash(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	registerCluster(t, db, "c1", "生产")
	registerCluster(t, db, "c2", "测试")
	issuedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	if err := db.Cluster.SetToken(ctx, "c1", "hash-1", "atlk_Ab3", issuedAt); err != nil {
		t.Fatal(err)
	}

	got, err := db.Cluster.GetByTokenHash(ctx, "hash-1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ClusterID != "c1" || got.TokenPrefix != "atlk_Ab3" {
		t.Fatalf("按哈希应找到 c1，实际 %+v", got)
	}
	if got.TokenIssuedAt == nil || !got.TokenIssuedAt.Equal(issuedAt) {
		t.Errorf("签发时间不对: %v", got.TokenIssuedAt)
	}

	// 轮换：旧哈希立即查不到
	if err := db.Cluster.SetToken(ctx, "c1", "hash-2", "atlk_Cd4", issuedAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if old, _ := db.Cluster.GetByTokenHash(ctx, "hash-1"); old != nil {
		t.Errorf("轮换后旧哈希仍能查到: %+v", old)
	}
}

// 没签发令牌的集群，哈希列是空串；空哈希不得匹配到任何集群。
func TestClusterGetByTokenHash_EmptyHashMatchesNothing(t *testing.T) {
	db := openTestDB(t)
	registerCluster(t, db, "c1", "生产")

	got, err := db.Cluster.GetByTokenHash(context.Background(), "")
	if err != nil || got != nil {
		t.Errorf("空哈希应返回 (nil, nil)，实际 (%+v, %v)", got, err)
	}
}

func TestClusterSetToken_UnknownCluster(t *testing.T) {
	db := openTestDB(t)
	err := db.Cluster.SetToken(context.Background(), "missing", "hash", "atlk_Ab3", time.Now())
	if !errors.Is(err, database.ErrClusterNotFound) {
		t.Errorf("给不存在的集群签发应返回 ErrClusterNotFound，实际 %v", err)
	}
}

func TestClusterUpdateProfile(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	registerCluster(t, db, "c1", "生产")
	if err := db.Cluster.SetToken(ctx, "c1", "hash-1", "atlk_Ab3", time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := db.Cluster.UpdateProfile(ctx, "c1", "生产（东京）", "主集群"); err != nil {
		t.Fatal(err)
	}
	got, _ := db.Cluster.GetByID(ctx, "c1")
	if got.Name != "生产（东京）" || got.Description != "主集群" {
		t.Errorf("显示名或描述没有更新: %+v", got)
	}
	if got.TokenHash != "hash-1" {
		t.Errorf("改显示名不应动令牌: %q", got.TokenHash)
	}

	if err := db.Cluster.UpdateProfile(ctx, "missing", "x", ""); !errors.Is(err, database.ErrClusterNotFound) {
		t.Errorf("更新不存在的集群应返回 ErrClusterNotFound，实际 %v", err)
	}
}

func TestClusterDelete_InvalidatesToken(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	registerCluster(t, db, "c1", "生产")
	if err := db.Cluster.SetToken(ctx, "c1", "hash-1", "atlk_Ab3", time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := db.Cluster.Delete(ctx, "c1"); err != nil {
		t.Fatal(err)
	}
	if got, _ := db.Cluster.GetByTokenHash(ctx, "hash-1"); got != nil {
		t.Errorf("删除登记后令牌仍然有效: %+v", got)
	}
	if err := db.Cluster.Delete(ctx, "c1"); !errors.Is(err, database.ErrClusterNotFound) {
		t.Errorf("删除不存在的集群应返回 ErrClusterNotFound，实际 %v", err)
	}
}

func TestClusterList_OrderedByID(t *testing.T) {
	db := openTestDB(t)
	registerCluster(t, db, "c2", "二")
	registerCluster(t, db, "c1", "一")

	clusters, err := db.Cluster.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 2 || clusters[0].ClusterID != "c1" || clusters[1].ClusterID != "c2" {
		t.Errorf("期望按集群 ID 排序的两行，实际 %+v", clusters)
	}
}
