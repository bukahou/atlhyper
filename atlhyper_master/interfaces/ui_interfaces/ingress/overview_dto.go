// ui_interfaces/ingress/dto_overview.go
package ingress

import "time"

// IngressOverviewDTO —— 概览页：卡片 + 表格
type IngressOverviewDTO struct {
	Cards OverviewCards      `json:"cards"`
	Rows  []IngressRowSimple `json:"rows"` // 将 Rule.Path 展平成多行
}

// 顶部卡片（与截图一致的语义）
type OverviewCards struct {
	TotalIngresses int `json:"totalIngresses"` // Ingress 总数
	UsedHosts      int `json:"usedHosts"`      // 使用的域名数量（去重）
	TLSCerts       int `json:"tlsCerts"`       // TLS 证书条目数（sum(len(spec.tls)))
	TotalPaths     int `json:"totalPaths"`     // 路由路径总数（sum(len(rule.paths)))
}

// 表格行（与截图列对齐；每条 path 一行）
type IngressRowSimple struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Host        string    `json:"host"`        // rule.host；为空表示“*”或所有 host
	Path        string    `json:"path"`        // httpPath.path（空时用 "/"）
	ServiceName string    `json:"serviceName"` // backend.service.name
	ServicePort string    `json:"servicePort"` // 端口名或数字，统一成字符串
	TLS         string    `json:"tls"`         // 逗号拼接的 TLS hosts；无则空
	CreatedAt   time.Time `json:"createdAt"`
}
