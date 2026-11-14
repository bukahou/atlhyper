// atlhyper_master_v2/model/cluster_registry.go
// 集群注册表（集群管理页）的请求与响应模型
package model

// 集群的在线状态
const (
	ClusterStatusOnline  = "online"  // Agent 正在上报
	ClusterStatusOffline = "offline" // 上报过，现在已离线
	ClusterStatusNever   = "never"   // 登记了但从未上报
)

// ClusterRegistryItem 集群管理页的一行：登记信息 + 在线状态 + 认证状态。
//
// 令牌哈希不在其中 —— 它不出库。
type ClusterRegistryItem struct {
	ClusterID   string `json:"clusterId"`
	Name        string `json:"name"`        // 显示名；未登记的集群为空
	Description string `json:"description"` // 未登记的集群为空

	// Registered 为假表示这一行来自正在上报但尚未登记的集群，可以直接登记收编
	Registered bool `json:"registered"`

	TokenIssued   bool   `json:"tokenIssued"`
	TokenPrefix   string `json:"tokenPrefix"`   // 令牌明文的前几个字符，供辨认；未签发时为空
	TokenIssuedAt string `json:"tokenIssuedAt"` // RFC 3339（UTC）；未签发时为空

	Status   string `json:"status"`   // online / offline / never
	LastSeen string `json:"lastSeen"` // RFC 3339（UTC）；从未上报时为空

	// Authenticated 该集群最近一次上报是否带了有效令牌
	Authenticated bool `json:"authenticated"`
}

// RegisterClusterRequest 登记集群
type RegisterClusterRequest struct {
	ClusterID   string `json:"clusterId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateClusterProfileRequest 修改集群的显示名与描述
type UpdateClusterProfileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ClusterTokenIssued 一次令牌签发的结果。
//
// Token 是明文，只在签发的这一次响应里出现；之后任何接口都取不回它。
type ClusterTokenIssued struct {
	Token         string `json:"token"`
	TokenPrefix   string `json:"tokenPrefix"`
	TokenIssuedAt string `json:"tokenIssuedAt"` // RFC 3339（UTC）
}
