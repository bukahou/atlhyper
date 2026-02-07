// atlhyper_agent_v2/model/slo.go
// SLO 相关类型定义
// 为了保持 Agent 和 Master 的一致性，直接使用 model_v2 中的类型
package model

import "AtlHyper/model_v2"

// 类型别名，方便 Agent 内部使用
type (
	SLOSnapshot        = model_v2.SLOSnapshot
	ServiceMetrics     = model_v2.ServiceMetrics
	RequestDelta       = model_v2.RequestDelta
	ServiceEdge        = model_v2.ServiceEdge
	IngressMetrics     = model_v2.IngressMetrics
	IngressRequestDelta = model_v2.IngressRequestDelta
	IngressRouteInfo   = model_v2.IngressRouteInfo
)
