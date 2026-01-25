// executor/types.go
// 控制循环类型定义
package executor

import "AtlHyper/atlhyper_agent/model"

// 类型别名，保持向后兼容
type (
	Command    = model.Command
	CommandSet = model.CommandSet
	AckResult  = model.AckResult
	Result     = model.Result
)
