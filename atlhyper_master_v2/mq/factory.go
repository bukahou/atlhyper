// atlhyper_master_v2/mq/factory.go
// CommandBus 工厂函数
package mq

import "AtlHyper/atlhyper_master_v2/mq/memory"

// NewCommandBus 创建 CommandBus 实例（只有内存实现，原因见 datahub.NewStore）
func NewCommandBus() CommandBus {
	return memory.NewMemoryBus()
}
