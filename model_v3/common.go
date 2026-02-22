// Package model_v3 定义 AtlHyper V3 跨域共用类型
//
// 子包按领域划分：
//   - cluster/  — K8s 集群资源快照
//   - command/  — 指令模型
//   - agent/    — Agent 状态
//   - apm/      — APM (Traces)
//   - log/      — 日志
//   - metrics/  — 基础设施指标
//   - slo/      — SLO
package model_v3

import (
	"strconv"
	"strings"
	"time"
)

// ============================================================
// 公共元数据（K8s 资源通用）
// ============================================================

// CommonMeta 资源公共元数据
type CommonMeta struct {
	UID       string            `json:"uid"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace,omitempty"`
	Kind      string            `json:"kind"`
	NodeName  string            `json:"nodeName,omitempty"`
	OwnerKind string            `json:"ownerKind,omitempty"`
	OwnerName string            `json:"ownerName,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

// ============================================================
// 资源量定义
// ============================================================

// ResourceList 资源量表（CPU/Memory）
type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// ResourceRequirements 资源请求和限制
type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

// ============================================================
// 资源引用
// ============================================================

// ResourceRef 资源引用（如 Event 关联对象）
type ResourceRef struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
	UID       string `json:"uid,omitempty"`
}

// K8sRef K8s 对象引用
type K8sRef struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	UID       string `json:"uid,omitempty"`
}

// ============================================================
// 通用枚举
// ============================================================

// HealthStatus 健康状态（跨域通用）
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// ============================================================
// 时间范围（查询参数用）
// ============================================================

// TimeRange 预定义时间范围
type TimeRange string

const (
	TimeRange15Min TimeRange = "15min"
	TimeRange1H    TimeRange = "1h"
	TimeRange6H    TimeRange = "6h"
	TimeRange24H   TimeRange = "24h"
	TimeRange7D    TimeRange = "7d"
	TimeRange15D   TimeRange = "15d"
	TimeRange30D   TimeRange = "30d"
)

// ============================================================
// 分页
// ============================================================

// PageRequest 分页请求参数
type PageRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// PageResponse 分页响应元数据
type PageResponse struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// ============================================================
// 资源解析函数
// ============================================================

// ParseCPU 解析 CPU 字符串为毫核数
func ParseCPU(s string) int64 {
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "n") {
		val, _ := strconv.ParseInt(strings.TrimSuffix(s, "n"), 10, 64)
		return val / 1000000
	}
	if strings.HasSuffix(s, "m") {
		val, _ := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		return val
	}
	val, _ := strconv.ParseFloat(s, 64)
	return int64(val * 1000)
}

// ParseMemory 解析 Memory 字符串为字节数
func ParseMemory(s string) int64 {
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	units := map[string]int64{
		"Ti": 1024 * 1024 * 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Mi": 1024 * 1024,
		"Ki": 1024,
		"T":  1000 * 1000 * 1000 * 1000,
		"G":  1000 * 1000 * 1000,
		"M":  1000 * 1000,
		"K":  1000,
	}
	for suffix, multiplier := range units {
		if strings.HasSuffix(s, suffix) {
			val, _ := strconv.ParseFloat(strings.TrimSuffix(s, suffix), 64)
			return int64(val * float64(multiplier))
		}
	}
	val, _ := strconv.ParseInt(s, 10, 64)
	return val
}

// IsSidecarContainer 判断是否为服务网格 sidecar 容器
func IsSidecarContainer(name string) bool {
	switch name {
	case "linkerd-proxy", "linkerd-init":
		return true
	}
	return false
}
