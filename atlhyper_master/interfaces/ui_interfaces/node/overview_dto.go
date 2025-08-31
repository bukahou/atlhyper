package node

// NodeOverviewDTO —— 节点概览（统计卡片 + 表格）
type NodeOverviewDTO struct {
	Cards NodeCards       `json:"cards"`
	Rows  []NodeRowSimple `json:"rows"`
}

// 顶部卡片
type NodeCards struct {
	TotalNodes     int     `json:"totalNodes"`     // 节点总数
	ReadyNodes     int     `json:"readyNodes"`     // 就绪节点数
	TotalCPU       int     `json:"totalCPU"`       // 总 CPU（核）
	TotalMemoryGiB float64 `json:"totalMemoryGiB"` // 总内存（GiB，1 位小数）
}

// 表格行（与 UI 列对齐）
type NodeRowSimple struct {
	Name         string  `json:"name"`
	Ready        bool    `json:"ready"`        // true → Ready
	InternalIP   string  `json:"internalIP"`
	OSImage      string  `json:"osImage"`
	Architecture string  `json:"architecture"` // amd64/arm64
	CPUCores     int     `json:"cpuCores"`     // 例如 8
	MemoryGiB    float64 `json:"memoryGiB"`    // 例如 32.7
	Schedulable  bool    `json:"schedulable"`  // 可调度
}
