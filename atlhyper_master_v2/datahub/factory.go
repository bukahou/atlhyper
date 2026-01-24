// atlhyper_master_v2/datahub/factory.go
// Store 工厂函数
package datahub

import (
	"AtlHyper/atlhyper_master_v2/datahub/memory"
)

// New 创建 Store 实例
func New(cfg Config) Store {
	switch cfg.Type {
	default:
		return memory.New(cfg.EventRetention, cfg.HeartbeatExpire)
	}
}
