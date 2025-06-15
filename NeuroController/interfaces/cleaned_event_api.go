// =======================================================================================
// 📄 interface/cleaned_event_api.go
//
// 📦 Description:
//     Diagnosis 模块的接口桥接层，向 external 层提供清理后事件池的访问能力。
//     封装了对 diagnosis.GetCleanedEvents 的调用，隐藏具体实现细节。
//
// 🔌 Responsibilities:
//     - 提供统一的清理事件数据访问接口
//     - 避免 external 层直接依赖 internal.diagnosis 包
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package interfaces

import (
	"NeuroController/internal/diagnosis"
	"NeuroController/internal/types"
)

func GetCleanedEventLogs() []types.LogEvent {
	return diagnosis.GetCleanedEvents()
}
