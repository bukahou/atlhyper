package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// fakeGraphRepo 记录保存次数；block 非空时保存会停在那里直到被放行
type fakeGraphRepo struct {
	database.AIOpsGraphRepository
	mu    sync.Mutex
	saves int
	err   error
	block chan struct{}
}

func (f *fakeGraphRepo) Save(context.Context, string, []byte) error {
	if f.block != nil {
		<-f.block
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saves++
	return f.err
}

func (f *fakeGraphRepo) saveCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.saves
}

func twoNodeGraph(extra ...string) *aiops.DependencyGraph {
	g := aiops.NewDependencyGraph("c1")
	g.UpdatedAt = time.Now()
	for _, key := range append([]string{"a", "b"}, extra...) {
		g.Nodes[key] = &aiops.GraphNode{Key: key, Type: "pod", Name: key}
	}
	g.Edges = []*aiops.GraphEdge{{From: "a", To: "b", Type: "calls", Weight: 1}}
	return g
}

// 拓扑没变就不写库：依赖图每 10 秒重建一次，而拓扑很少变。
func TestGraphSaver_UnchangedGraphIsSavedOnce(t *testing.T) {
	repo := &fakeGraphRepo{}
	saver := newGraphSnapshotSaver(context.Background(), repo)

	for i := 0; i < 5; i++ {
		saver.SaveIfChanged("c1", twoNodeGraph())
		saver.wait()
	}
	if got := repo.saveCount(); got != 1 {
		t.Errorf("同一拓扑保存 %d 次，want 1", got)
	}
}

func TestGraphSaver_ChangedGraphIsSaved(t *testing.T) {
	repo := &fakeGraphRepo{}
	saver := newGraphSnapshotSaver(context.Background(), repo)

	saver.SaveIfChanged("c1", twoNodeGraph())
	saver.wait()
	saver.SaveIfChanged("c1", twoNodeGraph("c"))
	saver.wait()
	if got := repo.saveCount(); got != 2 {
		t.Errorf("拓扑变化后应再保存一次，实际 %d 次", got)
	}
}

// 保存失败不算数：下一份快照到来时重试，直到成功。
func TestGraphSaver_FailedSaveIsRetried(t *testing.T) {
	repo := &fakeGraphRepo{err: errors.New("database is locked")}
	saver := newGraphSnapshotSaver(context.Background(), repo)

	saver.SaveIfChanged("c1", twoNodeGraph())
	saver.wait()
	repo.mu.Lock()
	repo.err = nil
	repo.mu.Unlock()
	saver.SaveIfChanged("c1", twoNodeGraph())
	saver.wait()
	saver.SaveIfChanged("c1", twoNodeGraph())
	saver.wait()

	if got := repo.saveCount(); got != 2 {
		t.Errorf("失败 1 次 + 成功 1 次后不应再写，实际调用 %d 次", got)
	}
}

// 写库卡住时不堆积：同一集群同时只有一个保存在进行，其间到来的快照直接跳过。
func TestGraphSaver_OneSaveInFlightPerCluster(t *testing.T) {
	repo := &fakeGraphRepo{block: make(chan struct{})}
	saver := newGraphSnapshotSaver(context.Background(), repo)

	saver.SaveIfChanged("c1", twoNodeGraph())
	for i := 0; i < 5; i++ {
		saver.SaveIfChanged("c1", twoNodeGraph("extra")) // 拓扑不同，但前一个还没写完
	}
	close(repo.block)
	saver.wait()

	if got := repo.saveCount(); got != 1 {
		t.Errorf("写库卡住期间发起了 %d 次保存，want 1", got)
	}
}
