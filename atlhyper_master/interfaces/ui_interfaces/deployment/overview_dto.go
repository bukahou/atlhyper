// ui_interfaces/deployment/dto_overview.go
package deployment

import "time"

// DeploymentOverviewDTO —— 概览页：卡片 + 表格
type DeploymentOverviewDTO struct {
	Cards OverviewCards        `json:"cards"`
	Rows  []DeploymentRowSimple`json:"rows"`
}

// 顶部卡片（对齐你截图语义）
type OverviewCards struct {
	TotalDeployments int `json:"totalDeployments"` // Deployment 总数
	Namespaces       int `json:"namespaces"`       // 命名空间数（去重）
	TotalReplicas    int `json:"totalReplicas"`    // 期望副本总数（sum(spec.replicas)）
	ReadyReplicas    int `json:"readyReplicas"`    // Ready 副本总数（sum(status.readyReplicas)）
}

// 表格行（与截图列对齐）
type DeploymentRowSimple struct {
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Image       string    `json:"image"`          // 取模板第一个容器镜像；多容器时可拼接
	Replicas    string    `json:"replicas"`       // "ready/desired" 形式（如 1/1）
	LabelCount  int       `json:"labelCount"`     // 顶层 labels 数
	AnnoCount   int       `json:"annoCount"`      // 顶层 annotations 数
	CreatedAt   time.Time `json:"createdAt"`
}
