package model_v2

import "time"

// ============================================================
// NodeMetricsSnapshot 节点指标快照
// ============================================================

// NodeMetricsSnapshot 节点硬件指标快照
//
// 由 atlhyper_metrics_v2 采集器采集，通过 Agent 聚合到 ClusterSnapshot。
// 包含 CPU、内存、磁盘、网络、温度、进程等详细硬件指标。
type NodeMetricsSnapshot struct {
	// 标识
	NodeName  string    `json:"node_name"`  // 节点名称
	Timestamp time.Time `json:"timestamp"`  // 采集时间
	Hostname  string    `json:"hostname"`   // 主机名
	OS        string    `json:"os"`         // 操作系统
	Kernel    string    `json:"kernel"`     // 内核版本
	Uptime    int64     `json:"uptime"`     // 运行时长（秒）

	// 硬件指标
	CPU         CPUMetrics         `json:"cpu"`
	Memory      MemoryMetrics      `json:"memory"`
	Disks       []DiskMetrics      `json:"disks"`
	Networks    []NetworkMetrics   `json:"networks"`
	Temperature TemperatureMetrics `json:"temperature"`
	Processes   []ProcessMetrics   `json:"processes"` // Top N 进程
}

// ============================================================
// CPUMetrics CPU 指标
// ============================================================

// CPUMetrics CPU 使用指标
type CPUMetrics struct {
	// 整体使用率
	UsagePercent float64 `json:"usage_percent"` // 总使用率 (0-100)
	UserPercent  float64 `json:"user_percent"`  // 用户态使用率
	SystemPercent float64 `json:"system_percent"` // 内核态使用率
	IdlePercent  float64 `json:"idle_percent"`  // 空闲率
	IOWaitPercent float64 `json:"iowait_percent"` // I/O 等待率

	// 每核使用率
	PerCore []float64 `json:"per_core"` // 各核心使用率

	// 负载
	Load1  float64 `json:"load_1"`  // 1 分钟负载
	Load5  float64 `json:"load_5"`  // 5 分钟负载
	Load15 float64 `json:"load_15"` // 15 分钟负载

	// 硬件信息
	Model     string `json:"model"`      // CPU 型号
	Cores     int    `json:"cores"`      // 物理核心数
	Threads   int    `json:"threads"`    // 逻辑线程数
	Frequency float64 `json:"frequency"` // 主频 (MHz)
}

// ============================================================
// MemoryMetrics 内存指标
// ============================================================

// MemoryMetrics 内存使用指标
type MemoryMetrics struct {
	// 物理内存
	Total       int64   `json:"total"`         // 总内存 (bytes)
	Used        int64   `json:"used"`          // 已用内存 (bytes)
	Available   int64   `json:"available"`     // 可用内存 (bytes)
	Free        int64   `json:"free"`          // 空闲内存 (bytes)
	UsagePercent float64 `json:"usage_percent"` // 使用率 (0-100)

	// 缓存
	Cached  int64 `json:"cached"`  // 页面缓存 (bytes)
	Buffers int64 `json:"buffers"` // 缓冲区 (bytes)

	// Swap
	SwapTotal   int64   `json:"swap_total"`    // Swap 总量 (bytes)
	SwapUsed    int64   `json:"swap_used"`     // Swap 已用 (bytes)
	SwapFree    int64   `json:"swap_free"`     // Swap 空闲 (bytes)
	SwapPercent float64 `json:"swap_percent"`  // Swap 使用率 (0-100)
}

// ============================================================
// DiskMetrics 磁盘指标
// ============================================================

// DiskMetrics 磁盘使用指标
type DiskMetrics struct {
	// 设备信息
	Device     string `json:"device"`      // 设备名 (sda, nvme0n1p1)
	MountPoint string `json:"mount_point"` // 挂载点
	FSType     string `json:"fs_type"`     // 文件系统类型

	// 空间使用
	Total       int64   `json:"total"`         // 总空间 (bytes)
	Used        int64   `json:"used"`          // 已用空间 (bytes)
	Available   int64   `json:"available"`     // 可用空间 (bytes)
	UsagePercent float64 `json:"usage_percent"` // 使用率 (0-100)

	// I/O 统计
	ReadBytes   int64   `json:"read_bytes"`    // 累计读取 (bytes)
	WriteBytes  int64   `json:"write_bytes"`   // 累计写入 (bytes)
	ReadRate    float64 `json:"read_rate"`     // 读取速率 (bytes/s)
	WriteRate   float64 `json:"write_rate"`    // 写入速率 (bytes/s)
	ReadIOPS    float64 `json:"read_iops"`     // 读取 IOPS
	WriteIOPS   float64 `json:"write_iops"`    // 写入 IOPS
	IOUtil      float64 `json:"io_util"`       // I/O 利用率 (0-100)
}

// ============================================================
// NetworkMetrics 网络指标
// ============================================================

// NetworkMetrics 网络接口指标
type NetworkMetrics struct {
	// 接口信息
	Interface string `json:"interface"` // 接口名 (eth0, ens192)
	IPAddress string `json:"ip_address"` // IP 地址
	MACAddress string `json:"mac_address"` // MAC 地址
	Status    string `json:"status"`     // 状态 (up/down)
	Speed     int64  `json:"speed"`      // 链路速度 (Mbps)
	MTU       int    `json:"mtu"`        // MTU

	// 流量统计 (累计)
	RxBytes   int64 `json:"rx_bytes"`   // 接收字节
	TxBytes   int64 `json:"tx_bytes"`   // 发送字节
	RxPackets int64 `json:"rx_packets"` // 接收包数
	TxPackets int64 `json:"tx_packets"` // 发送包数

	// 速率 (bytes/s)
	RxRate float64 `json:"rx_rate"` // 接收速率
	TxRate float64 `json:"tx_rate"` // 发送速率

	// 错误统计
	RxErrors  int64 `json:"rx_errors"`  // 接收错误
	TxErrors  int64 `json:"tx_errors"`  // 发送错误
	RxDropped int64 `json:"rx_dropped"` // 接收丢包
	TxDropped int64 `json:"tx_dropped"` // 发送丢包
}

// ============================================================
// TemperatureMetrics 温度指标
// ============================================================

// TemperatureMetrics 温度传感器指标
type TemperatureMetrics struct {
	CPUTemp    float64         `json:"cpu_temp"`    // CPU 温度 (°C)
	CPUTempMax float64         `json:"cpu_temp_max"` // CPU 最高温度限制
	Sensors    []SensorReading `json:"sensors"`     // 所有传感器读数
}

// SensorReading 传感器读数
type SensorReading struct {
	Name     string  `json:"name"`     // 传感器名称
	Label    string  `json:"label"`    // 标签 (Core 0, Package id 0)
	Current  float64 `json:"current"`  // 当前温度 (°C)
	Max      float64 `json:"max"`      // 最高阈值
	Critical float64 `json:"critical"` // 临界阈值
}

// ============================================================
// ProcessMetrics 进程指标
// ============================================================

// ProcessMetrics 进程资源使用指标
type ProcessMetrics struct {
	PID        int     `json:"pid"`         // 进程 ID
	Name       string  `json:"name"`        // 进程名
	Cmdline    string  `json:"cmdline"`     // 完整命令行
	User       string  `json:"user"`        // 用户名
	Status     string  `json:"status"`      // 状态 (R/S/D/Z/T)
	CPUPercent float64 `json:"cpu_percent"` // CPU 使用率 (0-100)
	MemPercent float64 `json:"mem_percent"` // 内存使用率 (0-100)
	MemRSS     int64   `json:"mem_rss"`     // 常驻内存 (bytes)
	Threads    int     `json:"threads"`     // 线程数
	StartTime  int64   `json:"start_time"`  // 启动时间 (Unix timestamp)
}

// ============================================================
// MetricsDataPoint 历史数据点
// ============================================================

// MetricsDataPoint 指标历史数据点
//
// 用于趋势图展示，存储在 node_metrics_history 表。
// 每 5 分钟采样一次，保留 30 天。
type MetricsDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	NodeName     string    `json:"node_name"`
	CPUUsage     float64   `json:"cpu_usage"`      // CPU 使用率 (0-100)
	MemoryUsage  float64   `json:"memory_usage"`   // 内存使用率 (0-100)
	DiskUsage    float64   `json:"disk_usage"`     // 主磁盘使用率 (0-100)
	DiskIORead   float64   `json:"disk_io_read"`   // 磁盘读速率 (bytes/s)
	DiskIOWrite  float64   `json:"disk_io_write"`  // 磁盘写速率 (bytes/s)
	NetworkRx    float64   `json:"network_rx"`     // 网络接收速率 (bytes/s)
	NetworkTx    float64   `json:"network_tx"`     // 网络发送速率 (bytes/s)
	CPUTemp      float64   `json:"cpu_temp"`       // CPU 温度 (°C)
	Load1        float64   `json:"load_1"`         // 1 分钟负载
}

// ============================================================
// ClusterMetricsSummary 集群指标汇总
// ============================================================

// ClusterMetricsSummary 集群指标汇总统计
//
// 聚合所有节点的指标，用于集群总览展示。
type ClusterMetricsSummary struct {
	// 节点统计
	TotalNodes   int `json:"total_nodes"`    // 总节点数
	OnlineNodes  int `json:"online_nodes"`   // 有指标的节点数
	OfflineNodes int `json:"offline_nodes"`  // 无指标的节点数

	// 平均使用率
	AvgCPUUsage    float64 `json:"avg_cpu_usage"`    // 平均 CPU 使用率
	AvgMemoryUsage float64 `json:"avg_memory_usage"` // 平均内存使用率
	AvgDiskUsage   float64 `json:"avg_disk_usage"`   // 平均磁盘使用率

	// 最高使用率
	MaxCPUUsage    float64 `json:"max_cpu_usage"`
	MaxMemoryUsage float64 `json:"max_memory_usage"`
	MaxDiskUsage   float64 `json:"max_disk_usage"`

	// 温度
	AvgCPUTemp float64 `json:"avg_cpu_temp"` // 平均 CPU 温度
	MaxCPUTemp float64 `json:"max_cpu_temp"` // 最高 CPU 温度

	// 聚合资源
	TotalMemory     int64 `json:"total_memory"`     // 总内存 (bytes)
	UsedMemory      int64 `json:"used_memory"`      // 已用内存 (bytes)
	TotalDisk       int64 `json:"total_disk"`       // 总磁盘空间 (bytes)
	UsedDisk        int64 `json:"used_disk"`        // 已用磁盘空间 (bytes)
	TotalNetworkRx  int64 `json:"total_network_rx"` // 总网络接收速率 (bytes/s)
	TotalNetworkTx  int64 `json:"total_network_tx"` // 总网络发送速率 (bytes/s)
}

// ============================================================
// 辅助方法
// ============================================================

// IsHealthy 检查节点指标是否健康
func (m *NodeMetricsSnapshot) IsHealthy() bool {
	// CPU 使用率 > 90% 或 内存使用率 > 90% 或 磁盘使用率 > 90% 视为不健康
	if m.CPU.UsagePercent > 90 || m.Memory.UsagePercent > 90 {
		return false
	}
	for _, disk := range m.Disks {
		if disk.UsagePercent > 90 {
			return false
		}
	}
	return true
}

// GetPrimaryDisk 获取主磁盘 (根分区或最大分区)
func (m *NodeMetricsSnapshot) GetPrimaryDisk() *DiskMetrics {
	if len(m.Disks) == 0 {
		return nil
	}
	// 优先返回根分区
	for i := range m.Disks {
		if m.Disks[i].MountPoint == "/" {
			return &m.Disks[i]
		}
	}
	// 否则返回第一个
	return &m.Disks[0]
}

// GetPrimaryNetwork 获取主网络接口 (第一个 up 状态的非 loopback)
func (m *NodeMetricsSnapshot) GetPrimaryNetwork() *NetworkMetrics {
	if len(m.Networks) == 0 {
		return nil
	}
	for i := range m.Networks {
		if m.Networks[i].Status == "up" && m.Networks[i].Interface != "lo" {
			return &m.Networks[i]
		}
	}
	return &m.Networks[0]
}

// ToDataPoint 转换为历史数据点
func (m *NodeMetricsSnapshot) ToDataPoint() MetricsDataPoint {
	dp := MetricsDataPoint{
		Timestamp:   m.Timestamp,
		NodeName:    m.NodeName,
		CPUUsage:    m.CPU.UsagePercent,
		MemoryUsage: m.Memory.UsagePercent,
		CPUTemp:     m.Temperature.CPUTemp,
		Load1:       m.CPU.Load1,
	}

	// 主磁盘
	if disk := m.GetPrimaryDisk(); disk != nil {
		dp.DiskUsage = disk.UsagePercent
		dp.DiskIORead = disk.ReadRate
		dp.DiskIOWrite = disk.WriteRate
	}

	// 主网络
	if net := m.GetPrimaryNetwork(); net != nil {
		dp.NetworkRx = net.RxRate
		dp.NetworkTx = net.TxRate
	}

	return dp
}
