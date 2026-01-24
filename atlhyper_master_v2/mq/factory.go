// atlhyper_master_v2/mq/factory.go
// CommandBus 工厂函数
package mq

import (
	"AtlHyper/atlhyper_master_v2/mq/memory"
)

// Config CommandBus 配置
type Config struct {
	Type string // 类型: memory / redis
}

// New 创建 CommandBus 实例
func New(cfg Config) CommandBus {
	switch cfg.Type {
	default:
		return memory.New()
	}
}
