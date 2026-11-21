// atlhyper_master_v2/datahub/factory.go
// Store 工厂函数
package datahub

import "AtlHyper/atlhyper_master_v2/datahub/memory"

// NewStore 创建 Store 实例。
// 只有内存实现：主控的持久层是单个 PVC 上的 SQLite，结构上就只能单副本；要多副本先换数据库，届时再加共享实现
func NewStore(cfg Config) Store {
	return memory.NewMemoryStore(cfg.EventRetention, cfg.HeartbeatExpire, cfg.SnapshotRetention)
}
