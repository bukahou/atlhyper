package metrics

import "time"

type NodeMetricsSnapshot struct {
	NodeName         string           `json:"nodeName"`                    // 节点名，由宿主机 hostname 获取
	Timestamp        time.Time        `json:"timestamp"`                   // 指标采集时间
	CPU              CPUStat          `json:"cpu"`                         // CPU 使用情况
	Memory           MemoryStat       `json:"memory"`                      // 内存使用情况
	Temperature      TemperatureStat  `json:"temperature"`                 // 温度指标（CPU/GPU/NVMe）
	Disk             []DiskStat       `json:"disk"`                        // 多个挂载点的磁盘使用情况
	Network          []NetworkStat    `json:"network"`                     // 多个网卡的网络流量
	TopCPUProcesses  []TopCPUProcess  `json:"topCPUProcesses,omitempty"`   // 可选：占用 CPU 较高的进程列表
}


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

type NetworkStat struct {
	Interface string  `json:"interface"`      // 网络接口名称，如 eth0, wlan0
	RxKBps    float64 `json:"rxKBps"`         // 接收速率（KB/s）
	TxKBps    float64 `json:"txKBps"`         // 发送速率（KB/s）
	RxSpeed   string  `json:"rxSpeed"`        // 接收速率（如 1.25 MB/s）
	TxSpeed   string  `json:"txSpeed"`        // 发送速率（如 982 KB/s）
}



type TemperatureStat struct {
	CPUDegrees  float64 `json:"cpuDegrees"`  // CPU 温度（℃）
	GPUDegrees  float64 `json:"gpuDegrees"`  // GPU 温度（可选）
	NVMEDegrees float64 `json:"nvmeDegrees"` // NVMe 温度（可选）
}

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


type CPUStat struct {
	Usage        float64 `json:"usage"`         // 原始比例，如 0.67
	UsagePercent string  `json:"usagePercent"`  // 可读字符串，如 "67.23%"
	Cores        int     `json:"cores"`
	Load1        float64 `json:"load1"`
	Load5        float64 `json:"load5"`
	Load15       float64 `json:"load15"`
}


type TopCPUProcess struct {
	PID         int     `json:"pid"`         // 进程 ID
	User        string  `json:"user"`        // 拥有者
	Command     string  `json:"command"`     // 命令名
	CPUPercent  float64 `json:"cpuPercent"`  // CPU 使用率（单位 %）
	CPUUsage    string  `json:"cpuUsage"`    // CPU 使用率字符串（如 25.45%）
	MemoryMB    float64 `json:"memoryMB"`    // 内存使用量（MB）
	MemoryUsage string  `json:"memoryUsage"` // 内存使用量字符串（如 256.23 MB）
}