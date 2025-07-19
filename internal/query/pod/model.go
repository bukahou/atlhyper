package pod

// PodInfo 是简化后的 Pod 结构体，用于 UI 展示列表页
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
}
