package metrics

// MemoryStat 表示节点的内存使用情况。
//
// 示例：
//   {
//     "totalMB": 16384,       // 总内存为 16 GB
//     "usedMB": 10240,        // 已使用 10 GB
//     "usageRate": 0.625      // 使用率为 62.5%
//   }
type MemoryStat struct {
	Total     uint64  `json:"total"`     // 总内存，单位：字节
	Used      uint64  `json:"used"`      // 已使用内存，单位：字节
	Available uint64  `json:"available"` // 可用内存，单位：字节
	Usage     float64 `json:"usage"`     // 使用率（0.0 ~ 1.0）

	TotalReadable     string `json:"totalReadable"`     // 总内存（可读）
	UsedReadable      string `json:"usedReadable"`      // 已使用内存（可读）
	AvailableReadable string `json:"availableReadable"` // 可用内存（可读）
	UsagePercent      string `json:"usagePercent"`      // 使用率百分比
}
