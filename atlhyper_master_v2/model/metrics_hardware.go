// metrics_hardware.go 硬件健康 API 响应模型（GET /api/v2/observe/metrics/hardware）
//
// 与 model_v3/metrics 的上报模型分开：这里是「判定完的结果」，每格自带 status，
// 前端只渲染 chip 颜色，不再做任何阈值比较（大后端小前端）。
// 指针字段为 nil → JSON null → 前端显示「无数据」；绝不用零值冒充正常。
package model

import "AtlHyper/model_v3/metrics"

// HardwareStatus 单格 / 节点的健康状态
type HardwareStatus string

const (
	HardwareGood HardwareStatus = "good"
	HardwareWarn HardwareStatus = "warn"
	HardwareCrit HardwareStatus = "crit"
	// HardwareUnknown 没有这个读数（节点没装传感器、或这一轮没采到）：⛔ 不能当成正常
	HardwareUnknown HardwareStatus = "unknown"
)

// HardwareTempCell 温度格（CPU / 磁盘 / 其他）
type HardwareTempCell struct {
	Value  float64        `json:"value"`
	Max    float64        `json:"max"`  // warn 阈值（传感器自带或画像表）
	Crit   float64        `json:"crit"` // crit 阈值
	Label  string         `json:"label,omitempty"`
	Status HardwareStatus `json:"status"`
}

// HardwareUndervoltCell 欠压格（树莓派）
type HardwareUndervoltCell struct {
	Alarm  bool           `json:"alarm"`
	Status HardwareStatus `json:"status"`
}

// HardwareFanCell 风扇格。RPM 为 nil = 没有转速传感器（只有 cooling state）。
type HardwareFanCell struct {
	RPM      *float64       `json:"rpm"`
	State    int            `json:"state"`
	MaxState int            `json:"maxState"`
	Status   HardwareStatus `json:"status"`
}

// HardwareFreqCell CPU 频率格（热降频判定）
type HardwareFreqCell struct {
	CurrentGHz float64        `json:"currentGHz"`
	MaxGHz     float64        `json:"maxGHz"`
	RatioPct   float64        `json:"ratioPct"`
	Status     HardwareStatus `json:"status"`
}

// HardwareAwaitCell 磁盘延迟格（该节点最差的块设备）
type HardwareAwaitCell struct {
	ValueMs float64        `json:"valueMs"`
	Device  string         `json:"device"`
	Status  HardwareStatus `json:"status"`
}

// HardwareUsageCell 资源使用率格（CPU / 内存 / 磁盘）。
// 硬件矩阵里排在温度之前 —— 四大件先看，硬件传感器随后。
type HardwareUsageCell struct {
	Value  float64        `json:"value"`
	Status HardwareStatus `json:"status"`
}

// HardwareSensorCell 单个温度传感器的判定结果（节点详情的温度卡用）。
// 与矩阵里的三个温度格不同：那三格是每类取最热的一个，这里是逐个列出。
type HardwareSensorCell struct {
	Label  string         `json:"label"`  // chip 可读名（coretemp / nvme0 / rp1_adc）
	Sensor string         `json:"sensor"` // temp1 / temp2 ...
	Class  string         `json:"class"`  // cpu / disk / other
	Value  float64        `json:"value"`
	Max    float64        `json:"max"`
	Crit   float64        `json:"crit"`
	Status HardwareStatus `json:"status"`
}

// HardwareRow 矩阵一行 = 一个节点
type HardwareRow struct {
	NodeName     string `json:"nodeName"`
	NodeIP       string `json:"nodeIP,omitempty"`
	Uptime       int64  `json:"uptime,omitempty"` // 秒
	Kernel       string `json:"kernel,omitempty"`
	Profile      string `json:"profile"`
	ProfileLabel string `json:"profileLabel"`
	// 四大件：CPU / 内存 / 磁盘 在前，温度及其余硬件传感器在后
	CPUUsage  *HardwareUsageCell     `json:"cpuUsage"`
	MemUsage  *HardwareUsageCell     `json:"memUsage"`
	DiskUsage *HardwareUsageCell     `json:"diskUsage"`
	CPUTemp   *HardwareTempCell      `json:"cpuTemp"`
	DiskTemp  *HardwareTempCell      `json:"diskTemp"`
	OtherTemp *HardwareTempCell      `json:"otherTemp"`
	Undervolt *HardwareUndervoltCell `json:"undervolt"`
	Fan       *HardwareFanCell       `json:"fan"`
	CPUFreq   *HardwareFreqCell      `json:"cpuFreq"`
	DiskAwait *HardwareAwaitCell     `json:"diskAwait"`
	// Sensors 全部温度传感器逐个判定，供节点详情的温度卡渲染；无传感器时为空数组
	Sensors []HardwareSensorCell `json:"sensors"`
	Overall HardwareStatus       `json:"overall"`
}

// HardwareMaxTemp 速览：集群最高温（含来源）
type HardwareMaxTemp struct {
	Value    float64        `json:"value"`
	NodeName string         `json:"nodeName"`
	Sensor   string         `json:"sensor"`
	Status   HardwareStatus `json:"status"`
}

// HardwareMaxAwait 速览：集群最差磁盘延迟
type HardwareMaxAwait struct {
	ValueMs  float64        `json:"valueMs"`
	NodeName string         `json:"nodeName"`
	Device   string         `json:"device"`
	Status   HardwareStatus `json:"status"`
}

// HardwareSummary 速览 tile 数据
type HardwareSummary struct {
	MaxTemp        *HardwareMaxTemp  `json:"maxTemp"`
	MaxDiskTemp    *HardwareMaxTemp  `json:"maxDiskTemp"`
	UndervoltNodes int               `json:"undervoltNodes"`
	ThrottledNodes int               `json:"throttledNodes"`
	MaxDiskAwait   *HardwareMaxAwait `json:"maxDiskAwait"`
	// WarningNodes 整行结论（overall）为 warn / crit 的节点数
	WarningNodes int `json:"warningNodes"`
}

// HardwareHealthResponse 完整响应 data
type HardwareHealthResponse struct {
	Rows    []HardwareRow   `json:"rows"`
	Summary HardwareSummary `json:"summary"`
}

// ──────────────────────────────────────────────────────────────
// 节点对比表（GET /api/v2/observe/metrics/compare）
// ──────────────────────────────────────────────────────────────
//
// 与硬件矩阵的区别：矩阵只看硬件传感器，对比表把硬件与四大资源放在一张表里，
// 用来横向找出「哪台不一样」。列顺序由后端固定（硬件优先），前端不重排。

// CompareCell 对比表的一格。Value 为 nil 表示该节点取不到这个信号。
type CompareCell struct {
	Value *float64 `json:"value"`
	// Text 带单位的展示串（"124 ms" / "62.5%"）。只放数字与单位，
	// 任何需要翻译的文案都由前端按 i18n 出，后端不返回自然语言。
	Text   string         `json:"text,omitempty"`
	Status HardwareStatus `json:"status"`
}

// CompareRow 对比表一行 = 一个节点
type CompareRow struct {
	NodeName string                 `json:"nodeName"`
	Profile  string                 `json:"profile"`
	Cells    map[string]CompareCell `json:"cells"`
	Overall  HardwareStatus         `json:"overall"`
}

// NodeComparisonResponse 完整响应 data
type NodeComparisonResponse struct {
	Columns []string     `json:"columns"`
	Rows    []CompareRow `json:"rows"`
}

// NodeDiskLevels 单块盘各读数的等级
type NodeDiskLevels struct {
	Usage  HardwareStatus `json:"usage"` // 容量（inode 单独一项；矩阵里的磁盘格取两者更满的一个）
	Inode  HardwareStatus `json:"inode"`
	IOUtil HardwareStatus `json:"ioUtil"`
}

// NodePSILevels 压力失速（some / full）各项的等级
type NodePSILevels struct {
	CPUSome HardwareStatus `json:"cpuSome"`
	MemSome HardwareStatus `json:"memSome"`
	MemFull HardwareStatus `json:"memFull"`
	IOSome  HardwareStatus `json:"ioSome"`
	IOFull  HardwareStatus `json:"ioFull"`
}

// NodeVMStatLevels 换页与缺页的等级
type NodeVMStatLevels struct {
	MajorFaults HardwareStatus `json:"majorFaults"`
	SwapIn      HardwareStatus `json:"swapIn"`
	SwapOut     HardwareStatus `json:"swapOut"`
}

// NodeLevels 节点详情各读数的等级（后端判定，前端只按它上色）。
// 读数所在的采集项没采到时为 unknown。整台节点的结论看硬件矩阵那一行的 overall。
type NodeLevels struct {
	CPU          HardwareStatus   `json:"cpu"`
	CPULoad      HardwareStatus   `json:"cpuLoad"` // 每核负载（排队等 CPU）
	IOWait       HardwareStatus   `json:"iowait"`
	Memory       HardwareStatus   `json:"memory"`
	Swap         HardwareStatus   `json:"swap"`
	OOMKill      HardwareStatus   `json:"oomKill"`
	Disks        []NodeDiskLevels `json:"disks"` // 与节点的 disks 一一对应
	PSI          NodePSILevels    `json:"psi"`
	VMStat       NodeVMStatLevels `json:"vmstat"`
	FileFD       HardwareStatus   `json:"fileFd"`
	Conntrack    HardwareStatus   `json:"conntrack"`
	ProcsBlocked HardwareStatus   `json:"procsBlocked"`
	TimeSync     HardwareStatus   `json:"timeSync"`
	Entropy      HardwareStatus   `json:"entropy"`
}

// NodeMetricsView 节点指标 + 派生值 + 等级（GET /api/v2/observe/metrics/nodes 的一项）。
// 内嵌上报的原始结构，原有字段在 JSON 里保持不变。
type NodeMetricsView struct {
	metrics.NodeMetrics
	// PrimaryDiskIndex 主磁盘在 disks 里的下标（规则在 model_v3/metrics），没有为 null
	PrimaryDiskIndex *int `json:"primaryDiskIndex"`
	// CPULoadPerCore 1 分钟负载 ÷ 核数（跨机型可比）；取不到核数为 null
	CPULoadPerCore *float64 `json:"cpuLoadPerCore"`
	// FileFDUsagePct 文件句柄占用率；内核不设上限或取不到上限时为 null
	FileFDUsagePct *float64 `json:"fileFdUsagePct"`
	// ConntrackUsagePct 连接跟踪表占用率；取不到上限时为 null
	ConntrackUsagePct *float64   `json:"conntrackUsagePct"`
	Levels            NodeLevels `json:"levels"`
}
