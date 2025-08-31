package service

import "time"

// ServiceOverviewDTO —— Service 概览（统计卡片 + 表格）
type ServiceOverviewDTO struct {
	Cards ServiceCards       `json:"cards"`
	Rows  []ServiceRowSimple `json:"rows"`
}

// 顶部卡片
type ServiceCards struct {
	TotalServices    int `json:"totalServices"`    // 服务总数
	ExternalServices int `json:"externalServices"` // 外部服务（NodePort/LoadBalancer）
	InternalServices int `json:"internalServices"` // 内部服务（ClusterIP 且非 Headless）
	HeadlessServices int `json:"headlessServices"` // Headless（clusterIP: None）
}

// 表格行（与 UI 列对齐）
type ServiceRowSimple struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	Type       string    `json:"type"`       // ClusterIP/NodePort/LoadBalancer/ExternalName
	ClusterIP  string    `json:"clusterIP"`  // 单值展示：优先 summary.clusterIP → network.clusterIPs[0] → "None"
	Ports      string    `json:"ports"`      // 形如 "80:8080(30080), 443:8443"
	Protocol   string    `json:"protocol"`   // 去重后 "TCP, UDP"
	Selector   string    `json:"selector"`   // "k1=v1, k2=v2"；无选择器显示 "-"
	CreatedAt  time.Time `json:"createdAt"`  // 创建时间（原样返回，前端格式化）
	// 可加：Badges []string `json:"badges,omitempty"`
}
