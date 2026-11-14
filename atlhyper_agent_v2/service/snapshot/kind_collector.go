package snapshot

import (
	"sort"
	"sync"

	"AtlHyper/model_v3/cluster"
)

// kindCollector 一类 K8s 资源的采集。
//
// 20 类资源并发采集、互不依赖。某一类失败不应让整份快照作废，所以每类除了「拉取」
// 还带一个「沿用」：用上一轮快照里的同类数据顶上。
type kindCollector struct {
	kind      string
	essential bool                                    // 没有它其余数据就没有意义（Pod / Node）
	collect   func() error                            // 拉取并写入本轮快照
	reuse     func(previous *cluster.ClusterSnapshot) // 用上一轮快照的同类数据写入本轮快照
}

// collectorFor 为快照里的一个切片字段构造采集器。
// field 返回该字段在某份快照里的地址，采集结果与沿用的数据都通过它写入。
func collectorFor[T any](
	kind string,
	essential bool,
	snapshot *cluster.ClusterSnapshot,
	field func(*cluster.ClusterSnapshot) *[]T,
	list func() ([]T, error),
) kindCollector {
	return kindCollector{
		kind:      kind,
		essential: essential,
		collect: func() error {
			items, err := list()
			if err == nil {
				*field(snapshot) = items
			}
			return err
		},
		reuse: func(previous *cluster.ClusterSnapshot) {
			*field(snapshot) = *field(previous)
		},
	}
}

// runCollectors 并发执行全部采集器，返回失败的那些及其错误。
// 各采集器写的是快照里不同的字段，互不冲突，不需要加锁。
func runCollectors(collectors []kindCollector) map[string]error {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed = make(map[string]error)
	)
	wg.Add(len(collectors))
	for _, collector := range collectors {
		go func() {
			defer wg.Done()
			if err := collector.collect(); err != nil {
				mu.Lock()
				failed[collector.kind] = err
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return failed
}

func sortedKinds(kinds []string) []string {
	sort.Strings(kinds)
	return kinds
}
