package model

type PodInfo struct {
	Namespace    string `json:"namespace"`
	Deployment   string `json:"deployment"`
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	Phase        string `json:"phase"`
	RestartCount int32  `json:"restartCount"`
	StartTime    string `json:"startTime"`
	PodIP        string `json:"podIP"`
	NodeName     string `json:"nodeName"`

	// 指标字段
	CPUUsage        string `json:"cpuUsage"`        // 例如：23m
	CPUUsagePercent string `json:"cpuUsagePercent"` // 例如：2.3%
	MemoryUsage     string `json:"memoryUsage"`     // 例如：112Mi
	MemoryPercent   string `json:"memoryPercent"`   // 例如：6.4%
}
