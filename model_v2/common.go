// Package model_v2 定义 AtlHyper V2 统一数据模型
//
// 本包由 Agent V2 和 Master V2 共用，确保数据结构一致。
// 所有 JSON 字段使用 snake_case 命名规范。
//
// 设计原则：
//   - 只包含 Web 展示需要的字段，避免冗余
//   - 所有资源嵌入 CommonMeta，支持统一查询
//   - 敏感信息（Secret/ConfigMap 值）不存储
package model_v2

import (
	"strconv"
	"strings"
	"time"
)

// ============================================================
// 公共元数据
// ============================================================

// CommonMeta 资源公共元数据
//
// 所有 K8s 资源都嵌入此结构，提供：
//   - 基础标识：UID, Name, Namespace, Kind
//   - 关联字段：NodeName, OwnerKind, OwnerName（用于快速关联查询）
//   - 时间信息：CreatedAt
//
// 使用示例：
//
//	type Pod struct {
//	    CommonMeta
//	    Phase string `json:"phase"`
//	}
type CommonMeta struct {
	// 基础标识
	UID       string `json:"uid"`                 // K8s 资源唯一 ID
	Name      string `json:"name"`                // 资源名称
	Namespace string `json:"namespace,omitempty"` // 命名空间（集群级资源为空）
	Kind      string `json:"kind"`                // 资源类型

	// 关联字段（用于快速关联查询）
	NodeName  string `json:"node_name,omitempty"`  // 所在 Node（Pod 填充）
	OwnerKind string `json:"owner_kind,omitempty"` // 所有者类型（如 Deployment）
	OwnerName string `json:"owner_name,omitempty"` // 所有者名称

	// 标签（用于筛选和角色判断）
	Labels map[string]string `json:"labels,omitempty"`

	// 时间
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// ============================================================
// 资源量定义
// ============================================================

// ResourceList 资源量表
//
// 用于表示 Node 的 Capacity/Allocatable 或 Container 的资源需求。
// 字符串格式与 K8s 一致，如 "4" (4核), "500m" (0.5核), "8Gi" (8GB)。
type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`    // CPU，如 "4" 或 "500m"
	Memory string `json:"memory,omitempty"` // 内存，如 "8Gi" 或 "512Mi"
}

// ResourceRequirements 资源请求和限制
//
// 表示 Container 的 resources.requests 和 resources.limits。
type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"` // 请求量（调度依据）
	Limits   ResourceList `json:"limits,omitempty"`   // 限制量（运行上限）
}

// ============================================================
// 资源引用
// ============================================================

// ResourceRef 资源引用
//
// 用于表示资源之间的引用关系，如 Event 关联的资源对象。
type ResourceRef struct {
	Kind      string `json:"kind"`                // 资源类型
	Namespace string `json:"namespace,omitempty"` // 命名空间
	Name      string `json:"name"`                // 资源名称
	UID       string `json:"uid,omitempty"`       // 资源 UID
}

// ============================================================
// 资源解析函数
// ============================================================

// ParseCPU 解析 CPU 字符串为毫核数（millicores）
//
// 支持格式：
//   - "4" 或 "4000m" -> 4000 (4核 = 4000毫核)
//   - "500m" -> 500 (0.5核)
//   - "100m" -> 100
//   - "123456789n" -> 123 (纳核转毫核，metrics-server 返回格式)
func ParseCPU(s string) int64 {
	if s == "" {
		return 0
	}

	// 移除空格
	s = strings.TrimSpace(s)

	// 检查是否以 "n" 结尾（纳核，metrics-server 返回格式）
	if strings.HasSuffix(s, "n") {
		s = strings.TrimSuffix(s, "n")
		val, _ := strconv.ParseInt(s, 10, 64)
		return val / 1000000 // 纳核 -> 毫核
	}

	// 检查是否以 "m" 结尾（毫核）
	if strings.HasSuffix(s, "m") {
		s = strings.TrimSuffix(s, "m")
		val, _ := strconv.ParseInt(s, 10, 64)
		return val
	}

	// 否则是核数，转换为毫核
	val, _ := strconv.ParseFloat(s, 64)
	return int64(val * 1000)
}

// ParseMemory 解析 Memory 字符串为字节数
//
// 支持格式：
//   - "8Gi" -> 8 * 1024^3
//   - "512Mi" -> 512 * 1024^2
//   - "1024Ki" -> 1024 * 1024
//   - "1000000000" -> 1000000000 (纯数字，单位为字节)
func ParseMemory(s string) int64 {
	if s == "" {
		return 0
	}

	s = strings.TrimSpace(s)

	// 二进制单位（IEC）
	units := map[string]int64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
		// 十进制单位（SI）
		"K": 1000,
		"M": 1000 * 1000,
		"G": 1000 * 1000 * 1000,
		"T": 1000 * 1000 * 1000 * 1000,
	}

	for suffix, multiplier := range units {
		if strings.HasSuffix(s, suffix) {
			s = strings.TrimSuffix(s, suffix)
			val, _ := strconv.ParseFloat(s, 64)
			return int64(val * float64(multiplier))
		}
	}

	// 纯数字，单位为字节
	val, _ := strconv.ParseInt(s, 10, 64)
	return val
}

// IsSidecarContainer 判断是否为服务网格注入的 sidecar 容器
func IsSidecarContainer(name string) bool {
	switch name {
	case "linkerd-proxy", "linkerd-init":
		return true
	}
	return false
}
