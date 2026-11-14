package baseline

import (
	"context"
	"sort"
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// fakeBaselineRepo 记录写入与删除，用内存 map 模拟表
type fakeBaselineRepo struct {
	rows    map[string]*database.AIOpsBaselineState // cluster|entity|metric
	calls   []string                                // 调用顺序："delete" / "upsert"
	deleted []string                                // cluster|entity
}

func newFakeBaselineRepo() *fakeBaselineRepo {
	return &fakeBaselineRepo{rows: map[string]*database.AIOpsBaselineState{}}
}

func rowKey(clusterID, entityKey, metricName string) string {
	return clusterID + "|" + entityKey + "|" + metricName
}

func (r *fakeBaselineRepo) BatchUpsert(_ context.Context, states []*database.AIOpsBaselineState) error {
	r.calls = append(r.calls, "upsert")
	for _, s := range states {
		r.rows[rowKey(s.ClusterID, s.EntityKey, s.MetricName)] = s
	}
	return nil
}

func (r *fakeBaselineRepo) ListAll(context.Context) ([]*database.AIOpsBaselineState, error) {
	all := make([]*database.AIOpsBaselineState, 0, len(r.rows))
	for _, s := range r.rows {
		all = append(all, s)
	}
	return all, nil
}

func (r *fakeBaselineRepo) DeleteEntities(_ context.Context, clusterID string, entityKeys []string) error {
	r.calls = append(r.calls, "delete")
	for _, entityKey := range entityKeys {
		r.deleted = append(r.deleted, clusterID+"|"+entityKey)
		for key, s := range r.rows {
			if s.ClusterID == clusterID && s.EntityKey == entityKey {
				delete(r.rows, key)
			}
		}
	}
	return nil
}

const (
	clusterA = "cluster-a"
	clusterB = "cluster-b"
	podKey   = "default/pod/nginx"
)

func cpuPoint(entityKey string, value float64) []aiops.MetricDataPoint {
	return []aiops.MetricDataPoint{{EntityKey: entityKey, MetricName: "cpu_usage", Value: value}}
}

func singleState(t *testing.T, m *StateManager, clusterID, entityKey string) *aiops.BaselineState {
	t.Helper()
	states := m.GetStates(clusterID, entityKey).States
	if len(states) != 1 {
		t.Fatalf("%s 的 %s 期望 1 条基线，实际 %d 条", clusterID, entityKey, len(states))
	}
	return states[0]
}

// 两个集群里的同名实体各有各的基线，互不影响。
func TestUpdate_SameEntityKeyInTwoClusters_SeparateBaselines(t *testing.T) {
	m := NewStateManager(newFakeBaselineRepo())

	for i := 0; i < 5; i++ {
		m.Update(clusterA, cpuPoint(podKey, 10))
	}
	m.Update(clusterB, cpuPoint(podKey, 90))

	stateA := singleState(t, m, clusterA, podKey)
	stateB := singleState(t, m, clusterB, podKey)
	if stateA.Count != 5 || stateB.Count != 1 {
		t.Errorf("样本数混在了一起: A=%d（期望 5） B=%d（期望 1）", stateA.Count, stateB.Count)
	}
	if stateA.EMA == stateB.EMA {
		t.Errorf("两个集群的基线均值相同（%v），说明共用了同一条", stateA.EMA)
	}
	if stateA.ClusterID != clusterA || stateB.ClusterID != clusterB {
		t.Errorf("基线没有带上所属集群: A=%q B=%q", stateA.ClusterID, stateB.ClusterID)
	}
}

// 一个集群的快照只清本集群里已消失的实体，不碰其他集群。
func TestCleanupStaleEntities_OnlyTouchesOwnCluster(t *testing.T) {
	m := NewStateManager(newFakeBaselineRepo())
	m.Update(clusterA, cpuPoint(podKey, 10))
	m.Update(clusterA, cpuPoint("default/pod/gone", 10))
	m.Update(clusterB, cpuPoint("default/pod/only-in-b", 10))

	removed := m.CleanupStaleEntities(clusterA, map[string]bool{podKey: true})

	if removed != 1 {
		t.Errorf("期望清掉 1 条（集群 A 里已消失的），实际 %d 条", removed)
	}
	if got := len(m.GetStates(clusterA, "default/pod/gone").States); got != 0 {
		t.Errorf("集群 A 里已消失的实体仍有 %d 条基线", got)
	}
	singleState(t, m, clusterA, podKey)
	singleState(t, m, clusterB, "default/pod/only-in-b")
}

// 实体消失后，库里的行在下一次 flush 时删除；先删后写。
func TestFlushToDB_DeletesRowsOfRemovedEntities(t *testing.T) {
	repo := newFakeBaselineRepo()
	m := NewStateManager(repo)
	ctx := context.Background()

	m.Update(clusterA, cpuPoint(podKey, 10))
	m.Update(clusterA, cpuPoint("default/pod/gone", 10))
	if err := m.FlushToDB(ctx); err != nil {
		t.Fatal(err)
	}
	if len(repo.rows) != 2 {
		t.Fatalf("首次 flush 后期望 2 行，实际 %d 行", len(repo.rows))
	}

	m.CleanupStaleEntities(clusterA, map[string]bool{podKey: true})
	m.Update(clusterA, cpuPoint(podKey, 11))
	repo.calls = nil
	if err := m.FlushToDB(ctx); err != nil {
		t.Fatal(err)
	}

	if _, ok := repo.rows[rowKey(clusterA, "default/pod/gone", "cpu_usage")]; ok {
		t.Error("已消失实体的行仍在库里")
	}
	if _, ok := repo.rows[rowKey(clusterA, podKey, "cpu_usage")]; !ok {
		t.Error("仍存在的实体的行被误删")
	}
	if len(repo.calls) != 2 || repo.calls[0] != "delete" || repo.calls[1] != "upsert" {
		t.Errorf("期望先删后写，实际调用顺序 %v", repo.calls)
	}

	// 已经删过的不再重复删
	repo.deleted = nil
	if err := m.FlushToDB(ctx); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 0 {
		t.Errorf("没有新消失的实体，却又删了 %v", repo.deleted)
	}
}

// 实体在两次 flush 之间消失又出现（StatefulSet 的 Pod 重建）：不能把新写入的行删掉。
func TestFlushToDB_ReappearedEntityIsNotDeleted(t *testing.T) {
	repo := newFakeBaselineRepo()
	m := NewStateManager(repo)
	ctx := context.Background()

	m.Update(clusterA, cpuPoint(podKey, 10))
	m.CleanupStaleEntities(clusterA, map[string]bool{})
	m.Update(clusterA, cpuPoint(podKey, 10))
	if err := m.FlushToDB(ctx); err != nil {
		t.Fatal(err)
	}

	if _, ok := repo.rows[rowKey(clusterA, podKey, "cpu_usage")]; !ok {
		t.Error("重新出现的实体，其基线行没有写进库")
	}
	for _, deleted := range repo.deleted {
		if deleted == clusterA+"|"+podKey {
			t.Error("重新出现的实体仍被列入删除")
		}
	}
}

// 启动恢复后，基线回到各自的集群下。
func TestLoadFromDB_RestoresPerCluster(t *testing.T) {
	repo := newFakeBaselineRepo()
	writer := NewStateManager(repo)
	writer.Update(clusterA, cpuPoint(podKey, 10))
	writer.Update(clusterB, cpuPoint(podKey, 90))
	if err := writer.FlushToDB(context.Background()); err != nil {
		t.Fatal(err)
	}

	restored := NewStateManager(repo)
	if err := restored.LoadFromDB(context.Background()); err != nil {
		t.Fatal(err)
	}

	emas := []float64{
		singleState(t, restored, clusterA, podKey).EMA,
		singleState(t, restored, clusterB, podKey).EMA,
	}
	sort.Float64s(emas)
	if emas[0] == emas[1] {
		t.Errorf("恢复后两个集群的基线相同（%v）", emas[0])
	}
}
