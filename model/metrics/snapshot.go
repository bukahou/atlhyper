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
