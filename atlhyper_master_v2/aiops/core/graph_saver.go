// atlhyper_master_v2/aiops/core/graph_saver.go
// 依赖图快照的持久化：只在拓扑变化时写库
package core

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/correlator"
	"AtlHyper/atlhyper_master_v2/database"
)

const graphSaveTimeout = 10 * time.Second

// graphSnapshotSaver 把依赖图异步写入数据库。
//
// 依赖图每份快照（10 秒）重建一次，但拓扑很少变，所以：
//   - 拓扑指纹与上次成功保存的相同时不写
//   - 同一集群同时只有一个保存在进行；写库慢的时候后来的快照直接跳过，不堆积
//   - 保存失败不更新指纹，下一份快照到来时自然重试
type graphSnapshotSaver struct {
	ctx  context.Context
	repo database.AIOpsGraphRepository

	mu       sync.Mutex
	saved    map[string]string // clusterID → 上次成功保存的拓扑指纹
	inFlight map[string]bool   // clusterID → 是否有保存正在进行
	wg       sync.WaitGroup
}

func newGraphSnapshotSaver(ctx context.Context, repo database.AIOpsGraphRepository) *graphSnapshotSaver {
	return &graphSnapshotSaver{
		ctx:      ctx,
		repo:     repo,
		saved:    make(map[string]string),
		inFlight: make(map[string]bool),
	}
}

// SaveIfChanged 拓扑相对上次保存有变化时异步写库；立即返回，不阻塞调用方
func (s *graphSnapshotSaver) SaveIfChanged(clusterID string, graph *aiops.DependencyGraph) {
	fingerprint := correlator.Fingerprint(graph)

	s.mu.Lock()
	if s.saved[clusterID] == fingerprint || s.inFlight[clusterID] {
		s.mu.Unlock()
		return
	}
	s.inFlight[clusterID] = true
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := s.save(clusterID, graph)

		s.mu.Lock()
		delete(s.inFlight, clusterID)
		if err == nil {
			s.saved[clusterID] = fingerprint
		}
		s.mu.Unlock()

		if err != nil {
			log.Error("持久化依赖图失败", "cluster", clusterID, "err", err)
		}
	}()
}

func (s *graphSnapshotSaver) save(clusterID string, graph *aiops.DependencyGraph) error {
	data, err := correlator.Serialize(graph)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(s.ctx, graphSaveTimeout)
	defer cancel()
	return s.repo.Save(ctx, clusterID, data)
}

// wait 等待进行中的保存结束（退出与测试用）
func (s *graphSnapshotSaver) wait() {
	s.wg.Wait()
}
