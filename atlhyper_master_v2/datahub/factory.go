// atlhyper_master_v2/datahub/factory.go
// Store 工厂函数
package datahub

import (
	"AtlHyper/atlhyper_master_v2/datahub/memory"
	redisStore "AtlHyper/atlhyper_master_v2/datahub/redis"
)

// NewStore 创建 Store 实例
func NewStore(cfg Config) Store {
	switch cfg.Type {
	case "redis":
		return redisStore.NewRedisStore(redisStore.Config{
			Addr:            cfg.RedisAddr,
			Password:        cfg.RedisPassword,
			DB:              cfg.RedisDB,
			EventRetention:  cfg.EventRetention,
			HeartbeatExpire: cfg.HeartbeatExpire,
		})
	default:
		return memory.NewMemoryStore(cfg.EventRetention, cfg.HeartbeatExpire, cfg.SnapshotRetention)
	}
}
