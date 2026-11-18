package repo_test

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

func baselineRow(clusterID, entityKey string, ema float64) *database.AIOpsBaselineState {
	return &database.AIOpsBaselineState{
		ClusterID: clusterID, EntityKey: entityKey, MetricName: "cpu_usage",
		EMA: ema, Variance: 1, Count: 3, UpdatedAt: 1700000000,
	}
}

// 两个集群的同名实体、同名指标是两行，各自保留自己的值。
func TestBaselineUpsert_SameEntityInTwoClusters_TwoRows(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	err := db.AIOpsBaseline.BatchUpsert(ctx, []*database.AIOpsBaselineState{
		baselineRow("c1", "default/pod/nginx", 10),
		baselineRow("c2", "default/pod/nginx", 90),
	})
	if err != nil {
		t.Fatal(err)
	}
	// 再写一次 c1：只更新 c1 那一行
	if err := db.AIOpsBaseline.BatchUpsert(ctx, []*database.AIOpsBaselineState{baselineRow("c1", "default/pod/nginx", 11)}); err != nil {
		t.Fatal(err)
	}

	rows, err := db.AIOpsBaseline.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	emaByCluster := map[string]float64{}
	for _, row := range rows {
		emaByCluster[row.ClusterID] = row.EMA
	}
	if len(rows) != 2 || emaByCluster["c1"] != 11 || emaByCluster["c2"] != 90 {
		t.Errorf("期望两行 c1=11 c2=90，实际 %d 行 %v", len(rows), emaByCluster)
	}
}

// 删除只作用于指定集群里的指定实体。
func TestBaselineDeleteEntities_ScopedToCluster(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	err := db.AIOpsBaseline.BatchUpsert(ctx, []*database.AIOpsBaselineState{
		baselineRow("c1", "default/pod/gone", 10),
		baselineRow("c1", "default/pod/kept", 10),
		baselineRow("c2", "default/pod/gone", 10),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AIOpsBaseline.DeleteEntities(ctx, "c1", []string{"default/pod/gone"}); err != nil {
		t.Fatal(err)
	}

	rows, err := db.AIOpsBaseline.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	remaining := map[string]bool{}
	for _, row := range rows {
		remaining[row.ClusterID+"|"+row.EntityKey] = true
	}
	if len(remaining) != 2 || !remaining["c1|default/pod/kept"] || !remaining["c2|default/pod/gone"] {
		t.Errorf("期望剩下 c1 的 kept 与 c2 的 gone，实际 %v", remaining)
	}

	// 空列表是无操作
	if err := db.AIOpsBaseline.DeleteEntities(ctx, "c1", nil); err != nil {
		t.Errorf("空列表不应报错: %v", err)
	}
}

func storeIncident(t *testing.T, db *database.DB, id, clusterID, entityKey string, startedAt time.Time) {
	t.Helper()
	ctx := context.Background()
	err := db.AIOpsIncident.CreateIncident(ctx, &database.AIOpsIncident{
		ID: id, ClusterID: clusterID, State: "warning", Severity: "low",
		RootCause: entityKey, StartedAt: startedAt, CreatedAt: startedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.AIOpsIncident.AddEntity(ctx, &database.AIOpsIncidentEntity{
		IncidentID: id, EntityKey: entityKey, EntityType: "pod", Role: "root_cause",
	})
	if err != nil {
		t.Fatal(err)
	}
}

// 查某个实体的历史事件，只返回它所在集群的。
func TestIncidentListByEntity_ScopedToCluster(t *testing.T) {
	db := openTestDB(t)
	startedAt := time.Now().Add(-time.Hour)
	storeIncident(t, db, "inc-c1", "c1", "default/pod/nginx", startedAt)
	storeIncident(t, db, "inc-c2", "c2", "default/pod/nginx", startedAt)

	incidents, err := db.AIOpsIncident.ListByEntity(context.Background(), "c1", "default/pod/nginx", startedAt.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(incidents) != 1 || incidents[0].ID != "inc-c1" {
		ids := make([]string, len(incidents))
		for i, inc := range incidents {
			ids[i] = inc.ID
		}
		t.Errorf("期望只有 inc-c1，实际 %v", ids)
	}
}
