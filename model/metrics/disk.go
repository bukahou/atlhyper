package metrics

// DiskStat 表示某一挂载点的磁盘使用情况
type DiskStat struct {
	MountPoint string  `json:"mountPoint"` // 挂载点标签（如 host_root）

	Total uint64  `json:"total"`  // 总大小（字节）
	Used  uint64  `json:"used"`   // 已用（字节）
	Free  uint64  `json:"free"`   // 可用（字节）
	Usage float64 `json:"usage"`  // 使用率（0.0 ~ 1.0）

	// ✅ 可读字段
	TotalReadable string `json:"totalReadable"` // 总大小（GB/MB）
	UsedReadable  string `json:"usedReadable"`  // 已用
	FreeReadable  string `json:"freeReadable"`  // 可用
	UsagePercent  string `json:"usagePercent"`  // 使用率百分比字符串（如 "21.32%"）
}
