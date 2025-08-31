package metrics

import (
	mod "AtlHyper/model/metrics"
	"time"
)

// 详情页：单节点时间序列 + 最新快照
// metrics/dto.go (或你的 DTO 定义处)
type NodeMetricsDetailDTO struct {
    Node      string         `json:"node"`
    Latest    NodeMetricsRow `json:"latest"`
    Series    NodeSeries     `json:"series"`
    Processes []mod.TopCPUProcess `json:"processes"` // ✅ 新增：完整进程列表
    TimeRange struct {
        Since time.Time `json:"since"`
        Until time.Time `json:"until"`
    } `json:"timeRange"`
}


type NodeSeries struct {
	At       []time.Time `json:"at"`
	CPUPct   []float64   `json:"cpuPct"`
	MemPct   []float64   `json:"memPct"`
	TempC    []float64   `json:"tempC"`
	DiskPct  []float64   `json:"diskPct"`
	Eth0Tx   []float64   `json:"eth0TxKBps"`
	Eth0Rx   []float64   `json:"eth0RxKBps"`
}
