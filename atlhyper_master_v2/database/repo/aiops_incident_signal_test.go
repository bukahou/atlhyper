package repo_test

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// 同一（事件, 成员, 指标）累计成一行：首次时刻不变、最近时刻更新、次数 +1、最高分取大、原文换成最新；不同成员 / 指标各成一行
func TestIncidentSignal_UpsertAccumulates(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	storeSubjectIncident(t, db, "inc-1", "c1", "shop/deployment/api", "high")

	t0 := time.Date(2026, 11, 15, 5, 52, 0, 0, time.UTC)
	upsert := func(entity, metric string, at time.Time, score float64, detail string) {
		t.Helper()
		err := db.AIOpsIncident.UpsertSignal(ctx, &database.AIOpsIncidentSignal{
			IncidentID: "inc-1", EntityKey: entity, Metric: metric, LastSeen: at, MaxScore: score, Detail: detail,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	upsert("shop/pod/api-a", "container_anomaly", t0, 0.8, "容器 app: CrashLoopBackOff")
	upsert("shop/pod/api-a", "container_anomaly", t0.Add(time.Minute), 0.9, "容器 app: OOMKilled")
	upsert("shop/pod/api-a", "container_anomaly", t0.Add(2*time.Minute), 0.7, "容器 app: CrashLoopBackOff")
	upsert("shop/pod/api-b", "container_anomaly", t0, 0.8, "容器 app: CrashLoopBackOff")
	upsert("shop/pod/api-a", "critical_event", t0, 0.85, "BackOff: x")

	signals, err := db.AIOpsIncident.GetSignals(ctx, "inc-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 3 {
		t.Fatalf("应有 3 行（a 的两个指标 + b），实际 %d", len(signals))
	}
	var a *database.AIOpsIncidentSignal
	for _, s := range signals {
		if s.EntityKey == "shop/pod/api-a" && s.Metric == "container_anomaly" {
			a = s
		}
	}
	if a == nil {
		t.Fatal("没有 api-a 的 container_anomaly")
	}
	if !a.FirstSeen.Equal(t0) || !a.LastSeen.Equal(t0.Add(2*time.Minute)) || a.Hits != 3 || a.MaxScore != 0.9 ||
		a.Detail != "容器 app: CrashLoopBackOff" {
		t.Errorf("累计不对: %+v", a)
	}
	if signals[0].MaxScore < signals[len(signals)-1].MaxScore {
		t.Error("应按最高分从高到低")
	}

	if empty, err := db.AIOpsIncident.GetSignals(ctx, "inc-none"); err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("没有记录时应是空切片，实际 (%v, %v)", empty, err)
	}
}
