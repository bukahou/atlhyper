package metrics

// CPUStat 表示节点 CPU 的总体使用情况。
// 包含当前使用率、核心数以及系统 1/5/15 分钟的平均负载。
//
// 示例：
//   {
//     "usage": 0.82,          // CPU 总使用率为 82%
//     "cores": 4,             // 4 核心 CPU
//     "load1": 3.21,          // 最近 1 分钟平均负载
//     "load5": 2.75,          // 最近 5 分钟平均负载
//     "load15": 1.90          // 最近 15 分钟平均负载
//   }
type CPUStat struct {
	Usage        float64 `json:"usage"`         // 原始比例，如 0.67
	UsagePercent string  `json:"usagePercent"`  // 可读字符串，如 "67.23%"
	Cores        int     `json:"cores"`
	Load1        float64 `json:"load1"`
	Load5        float64 `json:"load5"`
	Load15       float64 `json:"load15"`
}




// TopCPUProcess 表示占用 CPU 较高的单个进程信息。
// 如果启用了该功能，每个节点会上传若干个此类结构体。
// 
// 示例：
//   {
//     "pid": 1234,
//     "user": "root",
//     "command": "/usr/bin/qemu",
//     "cpuPercent": 73.2,
//     "memoryMB": 521.4
//   }
type TopCPUProcess struct {
	PID         int     `json:"pid"`         // 进程 ID
	User        string  `json:"user"`        // 拥有者
	Command     string  `json:"command"`     // 命令名
	CPUPercent  float64 `json:"cpuPercent"`  // CPU 使用率（单位 %）
	CPUUsage    string  `json:"cpuUsage"`    // CPU 使用率字符串（如 25.45%）
	MemoryMB    float64 `json:"memoryMB"`    // 内存使用量（MB）
	MemoryUsage string  `json:"memoryUsage"` // 内存使用量字符串（如 256.23 MB）
}