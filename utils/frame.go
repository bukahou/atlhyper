// utils/frame.go
package utils

// FrameHeader 是所有“快照帧”的公共部分
type Frame[T any] struct {
	Seq   int `json:"seq"`   // 到达顺序（从 1 开始）
	Items []T `json:"items"` // 该帧的完整列表
}