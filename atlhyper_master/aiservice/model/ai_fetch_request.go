// =============================================================
// 📦 文件路径: atlhyper_master/aiservice/model/ai_fetch_request.go
// =============================================================
// 🧠 模块说明:
//   该文件定义 AI Service 与 Master 之间的数据结构：
//   - AIFetchRequest：AI Service 提交的资源清单（请求）
//   - AIFetchResponse：Master 汇总后的上下文数据（响应）
// -------------------------------------------------------------
//   清单中可按需包含任意资源类型（Pod / Node / Service / ...）。
//   Master 会根据实际存在的字段动态调取各模块接口。
// =============================================================

package model

// =============================================================
// 🔹 ResourceRef —— 通用命名空间资源引用
// -------------------------------------------------------------
// 适用于 Pod / Deployment / Service / ConfigMap / Ingress 等
// =============================================================
type ResourceRef struct {
	Namespace string `json:"namespace"` // 资源所在命名空间
	Name      string `json:"name"`      // 资源名称
}

// =============================================================
// 🔸 AIFetchRequest —— AI Service 提交的清单结构
// -------------------------------------------------------------
// AI Service 会根据自身推理结果或事件分析生成此清单，
// Master 接收后按类型批量提取详细信息，返回结构化数据。
// =============================================================
type AIFetchRequest struct {
	ClusterID      string         `json:"clusterID"`                // 集群唯一标识
	Pods           []ResourceRef  `json:"pods,omitempty"`           // Pod 列表
	Deployments    []ResourceRef  `json:"deployments,omitempty"`    // Deployment 列表
	Services       []ResourceRef  `json:"services,omitempty"`       // Service 列表
	Nodes          []string       `json:"nodes,omitempty"`          // Node 名称列表
	ConfigMaps     []ResourceRef  `json:"configMaps,omitempty"`     // ConfigMap 列表
	Namespaces     []ResourceRef  `json:"namespaces,omitempty"`     // Namespace 列表
	Ingresses      []ResourceRef  `json:"ingresses,omitempty"`      // Ingress 列表
	EndpointSlices []ResourceRef  `json:"endpointSlices,omitempty"` // EndpointSlice 列表（预留）
}

// =============================================================
// 🔹 AIFetchResponse —— 汇总后返回的上下文数据（AI 二次分析输入）
// -------------------------------------------------------------
// 由 Master 聚合各资源详情生成，用于 AI 深度分析（诊断 / 报告）
// =============================================================
type AIFetchResponse struct {
	ClusterID      string      `json:"clusterID"`                // 集群标识
	Pods           []any       `json:"pods,omitempty"`           // Pod 详情数组
	Deployments    []any       `json:"deployments,omitempty"`    // Deployment 详情数组
	Services       []any       `json:"services,omitempty"`       // Service 详情数组
	Nodes          []any       `json:"nodes,omitempty"`          // Node 详情数组
	ConfigMaps     []any       `json:"configMaps,omitempty"`     // ConfigMap 列表（每命名空间）
	Namespaces     []any       `json:"namespaces,omitempty"`     // Namespace 详情
	Ingresses      []any       `json:"ingresses,omitempty"`      // Ingress 详情
	EndpointSlices []any       `json:"endpointSlices,omitempty"` // EndpointSlice（待实现）
	Metrics        []any       `json:"metrics,omitempty"`        // 节点指标（CPU/内存等）
}
