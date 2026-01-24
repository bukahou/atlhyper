// atlhyper_master_v2/mq/factory.go
// CommandBus 工厂函数
package mq

import (
	"AtlHyper/atlhyper_master_v2/mq/memory"
	redisBus "AtlHyper/atlhyper_master_v2/mq/redis"
)

// Config CommandBus 配置
type Config struct {
	Type string // 类型: memory / redis

	// Redis 配置（Type=redis 时使用）
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// New 创建 CommandBus 实例
func New(cfg Config) CommandBus {
	switch cfg.Type {
	case "redis":
		return redisBus.New(redisBus.Config{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
	default:
		return memory.New()
	}
}
