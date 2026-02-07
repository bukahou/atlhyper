// atlhyper_agent_v2/model/slo.go
// SLO 相关类型定义
// 为了保持 Agent 和 Master 的一致性，直接使用 model_v2 中的类型
package model

import "AtlHyper/model_v2"

// 类型别名，方便 Agent 内部使用
type (
	IngressMetrics         = model_v2.IngressMetrics
	IngressCounterMetric   = model_v2.IngressCounterMetric
	IngressHistogramMetric = model_v2.IngressHistogramMetric
	SLOPushRequest         = model_v2.SLOPushRequest
	IngressRouteInfo       = model_v2.IngressRouteInfo
	SLOSnapshot            = model_v2.SLOSnapshot
)
