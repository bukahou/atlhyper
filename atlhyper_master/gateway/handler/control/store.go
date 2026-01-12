package control

import (
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	sets    = map[string]*CommandSet{}     // key: clusterID
	waiters = map[string][]chan struct{}{} // key: clusterID

	// ★ 新增：按 clusterID + commandID 等待某条命令的 ACK
	ackWaiters = map[string]map[string][]chan AckResult{} // clusterID -> commandID -> chans
)

func ensureSetLocked(clusterID string) *CommandSet {
	if s, ok := sets[clusterID]; ok {
		return s
	}
	s := &CommandSet{ClusterID: clusterID, RV: 1, Commands: []Command{}}
	sets[clusterID] = s
	return s
}

func getIfNewer(clusterID string, rv uint64) (*CommandSet, bool) {
	mu.Lock()
	defer mu.Unlock()
	s := ensureSetLocked(clusterID)
	if s.RV > rv { return s, true }
	return nil, false
}

func waitChange(clusterID string, timeout time.Duration) (*CommandSet, bool) {
	ch := make(chan struct{}, 1)
	mu.Lock()
	waiters[clusterID] = append(waiters[clusterID], ch)
	mu.Unlock()

	select {
	case <-ch:
		mu.Lock(); defer mu.Unlock()
		return ensureSetLocked(clusterID), true
	case <-time.After(timeout):
		return nil, false
	}
}

// WaitAck 等待某个 commandID 的 ACK（带超时）- 导出供其他包调用
func WaitAck(clusterID, commandID string, timeout time.Duration) (AckResult, bool) {
	return waitAck(clusterID, commandID, timeout)
}

// waitAck 等待某个 commandID 的 ACK（带超时）
func waitAck(clusterID, commandID string, timeout time.Duration) (AckResult, bool) {
	ch := make(chan AckResult, 1)

	mu.Lock()
	if _, ok := ackWaiters[clusterID]; !ok {
		ackWaiters[clusterID] = make(map[string][]chan AckResult)
	}
	ackWaiters[clusterID][commandID] = append(ackWaiters[clusterID][commandID], ch)
	mu.Unlock()

	select {
	case res := <-ch:
		return res, true
	case <-time.After(timeout):
		return AckResult{}, false
	}
}

// UpsertCommand 添加命令到队列 - 导出供其他包调用
func UpsertCommand(clusterID string, cmd Command) *CommandSet {
	return upsertCommand(clusterID, cmd)
}

func upsertCommand(clusterID string, cmd Command) *CommandSet {
	mu.Lock(); defer mu.Unlock()
	s := ensureSetLocked(clusterID)
	s.Commands = append(s.Commands, cmd)
	s.RV++
	for _, ch := range waiters[clusterID] { select { case ch<-struct{}{}: default: } }
	waiters[clusterID] = nil
	return s
}

func applyAck(clusterID string, results []AckResult) *CommandSet {
	mu.Lock(); defer mu.Unlock()

	// 这里仍然采用“清空命令 + RV++”的简化策略
	s := ensureSetLocked(clusterID)
	s.Commands = nil
	s.RV++

	// 唤醒 watch 等待者
	for _, ch := range waiters[clusterID] { select { case ch<-struct{}{}: default: } }
	waiters[clusterID] = nil

	// ★ 关键：把 ACK 投递给 waitAck 的调用方
	if len(results) > 0 {
		if m := ackWaiters[clusterID]; m != nil {
			for _, r := range results {
				if lst := m[r.CommandID]; len(lst) > 0 {
					for _, ch := range lst { select { case ch <- r: default: } }
					delete(m, r.CommandID) // 清理该 commandID 的等待者
				}
			}
		}
	}
	return s
}
