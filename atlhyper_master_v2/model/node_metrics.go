// atlhyper_master_v2/model/node_metrics.go
// 节点指标 Web API 响应类型（camelCase JSON tag）
package model

import "time"

// ==================== API 响应封装 ====================

// ClusterNodeMetricsResponse 集群节点指标列表响应
type ClusterNodeMetricsResponse struct {
	Summary ClusterMetricsSummary `json:"summary"`
	Nodes   []NodeMetricsSnapshot `json:"nodes"`
}

// NodeMetricsHistoryResponse 节点历史数据响应
type NodeMetricsHistoryResponse struct {
	NodeName string             `json:"nodeName"`
	Start    time.Time          `json:"start"`
	End      time.Time          `json:"end"`
	Data     []MetricsDataPoint `json:"data"`
}

// ==================== 节点指标快照 ====================

// NodeMetricsSnapshot 节点指标快照
type NodeMetricsSnapshot struct {
	NodeName     string             `json:"nodeName"`
	Timestamp    time.Time          `json:"timestamp"`
	Uptime       int64              `json:"uptime"`
	OS           string             `json:"os"`
	Kernel       string             `json:"kernel"`
	CPU          CPUMetrics         `json:"cpu"`
	Memory       MemoryMetrics      `json:"memory"`
	Disks        []DiskMetrics      `json:"disks"`
	Networks     []NetworkMetrics   `json:"networks"`
	Temperature  TemperatureMetrics `json:"temperature"`
	TopProcesses []ProcessMetrics   `json:"topProcesses"`
	PSI          PSIMetrics         `json:"psi"`
	TCP          TCPMetrics         `json:"tcp"`
	System       SystemMetrics      `json:"system"`
	VMStat       VMStatMetrics      `json:"vmstat"`
	NTP          NTPMetrics         `json:"ntp"`
	Softnet      SoftnetMetrics     `json:"softnet"`
}

// ==================== 子指标类型 ====================

// CPUMetrics CPU 指标
type CPUMetrics struct {
	UsagePercent float64   `json:"usagePercent"`
	CoreCount    int       `json:"coreCount"`
	ThreadCount  int       `json:"threadCount"`
	CoreUsages   []float64 `json:"coreUsages"`
	LoadAvg1     float64   `json:"loadAvg1"`
	LoadAvg5     float64   `json:"loadAvg5"`
	LoadAvg15    float64   `json:"loadAvg15"`
	Model        string    `json:"model"`
	Frequency    float64   `json:"frequency"`
}

// MemoryMetrics 内存指标
type MemoryMetrics struct {
	TotalBytes       int64   `json:"totalBytes"`
	UsedBytes        int64   `json:"usedBytes"`
	AvailableBytes   int64   `json:"availableBytes"`
	UsagePercent     float64 `json:"usagePercent"`
	SwapTotalBytes   int64   `json:"swapTotalBytes"`
	SwapUsedBytes    int64   `json:"swapUsedBytes"`
	SwapUsagePercent float64 `json:"swapUsagePercent"`
	Cached           int64   `json:"cached"`
	Buffers          int64   `json:"buffers"`
}

// DiskMetrics 磁盘指标
type DiskMetrics struct {
	Device         string  `json:"device"`
	MountPoint     string  `json:"mountPoint"`
	FSType         string  `json:"fsType"`
	TotalBytes     int64   `json:"totalBytes"`
	UsedBytes      int64   `json:"usedBytes"`
	AvailableBytes int64   `json:"availableBytes"`
	UsagePercent   float64 `json:"usagePercent"`
	ReadBytesPS    float64 `json:"readBytesPS"`
	WriteBytesPS   float64 `json:"writeBytesPS"`
	IOPS           float64 `json:"iops"`
	IOUtil         float64 `json:"ioUtil"`
}

// NetworkMetrics 网络指标
type NetworkMetrics struct {
	Interface   string  `json:"interface"`
	IPAddress   string  `json:"ipAddress"`
	MACAddress  string  `json:"macAddress"`
	Status      string  `json:"status"`
	Speed       int64   `json:"speed"`
	RxBytesPS   float64 `json:"rxBytesPS"`
	TxBytesPS   float64 `json:"txBytesPS"`
	RxPacketsPS float64 `json:"rxPacketsPS"`
	TxPacketsPS float64 `json:"txPacketsPS"`
	RxErrors    int64   `json:"rxErrors"`
	TxErrors    int64   `json:"txErrors"`
	RxDropped   int64   `json:"rxDropped"`
	TxDropped   int64   `json:"txDropped"`
}

// TemperatureMetrics 温度指标
type TemperatureMetrics struct {
	CPUTemp    float64         `json:"cpuTemp"`
	CPUTempMax float64         `json:"cpuTempMax"`
	Sensors    []SensorReading `json:"sensors"`
}

// SensorReading 传感器读数
type SensorReading struct {
	Name     string  `json:"name"`
	Label    string  `json:"label"`
	Temp     float64 `json:"temp"`
	High     float64 `json:"high,omitempty"`
	Critical float64 `json:"critical,omitempty"`
}

// ProcessMetrics 进程指标
type ProcessMetrics struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	User       string  `json:"user"`
	State      string  `json:"state"`
	CPUPercent float64 `json:"cpuPercent"`
	MemPercent float64 `json:"memPercent"`
	MemBytes   int64   `json:"memBytes"`
	Threads    int     `json:"threads"`
	StartTime  string  `json:"startTime"` // ISO 8601
	Command    string  `json:"command"`
}

// PSIMetrics 压力信息
type PSIMetrics struct {
	CPUSomePercent    float64 `json:"cpuSomePercent"`
	MemorySomePercent float64 `json:"memorySomePercent"`
	MemoryFullPercent float64 `json:"memoryFullPercent"`
	IOSomePercent     float64 `json:"ioSomePercent"`
	IOFullPercent     float64 `json:"ioFullPercent"`
}

// TCPMetrics TCP 连接状态
type TCPMetrics struct {
	CurrEstab   int64 `json:"currEstab"`
	TimeWait    int64 `json:"timeWait"`
	Orphan      int64 `json:"orphan"`
	Alloc       int64 `json:"alloc"`
	InUse       int64 `json:"inUse"`
	SocketsUsed int64 `json:"socketsUsed"`
}

// SystemMetrics 系统资源指标
type SystemMetrics struct {
	ConntrackEntries int64 `json:"conntrackEntries"`
	ConntrackLimit   int64 `json:"conntrackLimit"`
	FilefdAllocated  int64 `json:"filefdAllocated"`
	FilefdMaximum    int64 `json:"filefdMaximum"`
	EntropyAvailable int64 `json:"entropyAvailable"`
}

// VMStatMetrics 虚拟内存统计
type VMStatMetrics struct {
	PgFaultPS    float64 `json:"pgfaultPS"`
	PgMajFaultPS float64 `json:"pgmajfaultPS"`
	PswpInPS     float64 `json:"pswpinPS"`
	PswpOutPS    float64 `json:"pswpoutPS"`
}

// NTPMetrics 时间同步
type NTPMetrics struct {
	OffsetSeconds float64 `json:"offsetSeconds"`
	Synced        bool    `json:"synced"`
}

// SoftnetMetrics 软中断统计
type SoftnetMetrics struct {
	Dropped  int64 `json:"dropped"`
	Squeezed int64 `json:"squeezed"`
}

// ==================== 历史数据 ====================

// MetricsDataPoint 历史数据点
type MetricsDataPoint struct {
	Timestamp   int64   `json:"timestamp"` // Unix 毫秒
	CPUUsage    float64 `json:"cpuUsage"`
	MemUsage    float64 `json:"memUsage"`
	DiskUsage   float64 `json:"diskUsage"`
	Temperature float64 `json:"temperature"`
}

// ==================== 集群汇总 ====================

// ClusterMetricsSummary 集群指标汇总
type ClusterMetricsSummary struct {
	TotalNodes     int     `json:"totalNodes"`
	OnlineNodes    int     `json:"onlineNodes"`
	OfflineNodes   int     `json:"offlineNodes"`
	AvgCPUUsage    float64 `json:"avgCPUUsage"`
	AvgMemoryUsage float64 `json:"avgMemoryUsage"`
	AvgDiskUsage   float64 `json:"avgDiskUsage"`
	MaxCPUUsage    float64 `json:"maxCPUUsage"`
	MaxMemoryUsage float64 `json:"maxMemoryUsage"`
	MaxDiskUsage   float64 `json:"maxDiskUsage"`
	AvgCPUTemp     float64 `json:"avgCPUTemp"`
	MaxCPUTemp     float64 `json:"maxCPUTemp"`
	TotalMemory    int64   `json:"totalMemory"`
	UsedMemory     int64   `json:"usedMemory"`
	TotalDisk      int64   `json:"totalDisk"`
	UsedDisk       int64   `json:"usedDisk"`
	TotalNetworkRx int64   `json:"totalNetworkRx"`
	TotalNetworkTx int64   `json:"totalNetworkTx"`
}
