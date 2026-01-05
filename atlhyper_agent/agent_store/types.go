package agent_store

import (
	"sync"

	"AtlHyper/model/collect"
)

// Store 仅维护"每个节点的最新一条快照"。
// 不持有 TTL；过期策略在 cleanup/janitor 里按需处理。
type Store struct {
	mu   sync.RWMutex
	data map[string]collect.NodeMetricsSnapshot
}
