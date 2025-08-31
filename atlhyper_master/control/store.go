package control

import (
	"sync"
	"time"
)

var (
	// mu —— 全局互斥锁，用于保护 sets 与 waiters 的并发安全
	mu sync.Mutex

	// sets —— 保存每个集群的 CommandSet 副本
	// key: clusterID
	// val: *CommandSet（包含 RV 与命令列表）
	sets = map[string]*CommandSet{}

	// waiters —— 保存挂起的 Watcher 通道
	// key: clusterID
	// val: []chan struct{} —— 对该集群正在等待变更的所有客户端
	// 场景：Agent 发起 watch，若没更新则挂起，直到有命令/ACK 导致 RV 变化时被唤醒
	waiters = map[string][]chan struct{}{}
)

// ensureSet —— 确保某个 clusterID 一定有一个 CommandSet
// ------------------------------------------------------------
// - 如果 sets 里已有该集群副本，直接返回
// - 如果没有，就创建一个新的空 CommandSet（RV=1, Commands=[]）
//   并存入 sets
func ensureSet(clusterID string) *CommandSet {
	mu.Lock()
	defer mu.Unlock()
	if s, ok := sets[clusterID]; ok {
		return s
	}
	// 初始化新的副本
	s := &CommandSet{ClusterID: clusterID, RV: 1, Commands: []Command{}}
	sets[clusterID] = s
	return s
}

// getIfNewer —— 判断当前集群的副本是否比给定 rv 更新
// ------------------------------------------------------------
// - Agent 每次 watch 时会带上自己已知的 rv
// - Master 比较当前副本的 RV 与 Agent 的 RV
//   - 如果 RV > rv，说明有新版本 → 返回最新副本
//   - 否则返回 nil，表示没有新更新
func getIfNewer(clusterID string, rv uint64) (*CommandSet, bool) {
	mu.Lock()
	defer mu.Unlock()
	s := ensureSet(clusterID)
	if s.RV > rv {
		return s, true
	}
	return nil, false
}

// waitChange —— 挂起等待某个集群的副本更新
// ------------------------------------------------------------
// - 逻辑：Agent 请求 watch，若没有新版本，则会调用 waitChange
// - 实现：
//   1. 为该集群创建一个 chan struct{}，加入 waiters
//   2. Master 在有更新时（upsert/ack）会唤醒这些通道
//   3. select 等待：
//      - 如果通道被唤醒 → 返回最新副本
//      - 如果超时 → 返回 nil 表示无更新
func waitChange(clusterID string, timeout time.Duration) (*CommandSet, bool) {
	ch := make(chan struct{}, 1) // 用缓冲 1 避免阻塞
	mu.Lock()
	waiters[clusterID] = append(waiters[clusterID], ch)
	mu.Unlock()

	select {
	case <-ch: // 被唤醒
		mu.Lock()
		defer mu.Unlock()
		return sets[clusterID], true
	case <-time.After(timeout): // 超时无更新
		return nil, false
	}
}

// upsertCommand —— 向某集群副本里追加一条命令
// ------------------------------------------------------------
// - 用于 Master 入队命令
// - 操作：
//   1. 确保集群副本存在
//   2. RV 自增（表示有新版本）
//   3. Commands 追加新的命令
//   4. 唤醒所有等待该集群的 Watcher
func upsertCommand(clusterID string, cmd Command) *CommandSet {
	mu.Lock()
	defer mu.Unlock()
	s := ensureSet(clusterID)
	s.RV++
	s.Commands = append(s.Commands, cmd)
	// 唤醒所有 watcher
	for _, ch := range waiters[clusterID] {
		ch <- struct{}{}
	}
	waiters[clusterID] = nil
	return s
}

// applyAck —— 处理 Agent 回执（AckResult）
// ------------------------------------------------------------
// - 用于 Agent 执行命令后反馈结果
// - 简化实现：收到 ACK 后直接清空该集群的 Commands
//   （真实逻辑可按需要：只移除成功的、标记失败的等）
// - 操作：
//   1. 确保集群副本存在
//   2. 清空命令列表（或按规则更新）
//   3. RV 自增（表示有新版本）
//   4. 唤醒所有 watcher
func applyAck(clusterID string, results []AckResult) *CommandSet {
    mu.Lock(); defer mu.Unlock()
    _ = results // 避免未使用警告

    s := ensureSet(clusterID)
    s.Commands = []Command{} // 先简化成清空
    s.RV++
    for _, ch := range waiters[clusterID] {
        ch <- struct{}{}
    }
    waiters[clusterID] = nil
    return s
}
