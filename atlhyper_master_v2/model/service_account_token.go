// atlhyper_master_v2/model/service_account_token.go
// AtlHyper 的服务账号与访问令牌（服务账号管理页）的请求、响应模型与业务参数。
// 不是 K8s 的 ServiceAccount（那个在 service_account.go）。
package model

// 业务参数：校验、列表里的「能否签发」与下发给前端的 ServiceAccountLimits 都用这一组
const (
	// ServiceAccountMaxRole 服务账号的等级上限（2 = Operator）：系统配置（3 级）只给人
	ServiceAccountMaxRole = 2
	// MaxActiveAPITokensPerAccount 每个账号未吊销的访问令牌上限
	MaxActiveAPITokensPerAccount = 10
	// MaxAPITokenExpiresInDays 访问令牌有效期的上限（天）；不给有效期 = 不过期
	MaxAPITokenExpiresInDays = 3650
)

// 访问令牌的状态（由后端按当前时间判定）
const (
	APITokenStatusActive  = "active"
	APITokenStatusExpired = "expired"
	APITokenStatusRevoked = "revoked" // 吊销优先于过期
)

// ServiceAccountView 服务账号管理页的一行：账号 + 它的令牌元数据。
//
// 令牌明文与哈希都不在其中 —— 明文只在签发的那一次响应里，哈希不出库。
type ServiceAccountView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"` // 账号名（与用户名同一命名空间）
	DisplayName string `json:"displayName"`
	Role        int    `json:"role"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"createdAt"` // RFC 3339（UTC）
	// LastUsedAt 各令牌里最近的一次使用，RFC 3339（UTC）；从未使用为空串
	LastUsedAt string `json:"lastUsedAt"`
	// CanIssueToken 未吊销的令牌是否还没到上限
	CanIssueToken bool           `json:"canIssueToken"`
	Tokens        []APITokenView `json:"tokens"` // 新签发的在前
}

// APITokenView 一枚访问令牌的元数据
type APITokenView struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	TokenPrefix string `json:"tokenPrefix"` // 明文的前几个字符，供辨认
	Status      string `json:"status"`      // active / expired / revoked
	CreatedBy   string `json:"createdBy"`   // 签发人的用户名
	CreatedAt   string `json:"createdAt"`   // RFC 3339（UTC）
	ExpiresAt   string `json:"expiresAt"`   // RFC 3339（UTC）；不过期为空串
	LastUsedAt  string `json:"lastUsedAt"`  // RFC 3339（UTC）；从未使用为空串
	RevokedAt   string `json:"revokedAt"`   // RFC 3339（UTC）；未吊销为空串
}

// ServiceAccountLimits 服务账号与令牌的业务参数（随列表下发，前端不另写一份）
type ServiceAccountLimits struct {
	AssignableRoles  []int `json:"assignableRoles"`  // 服务账号可选的等级，从低到高
	MaxActiveTokens  int   `json:"maxActiveTokens"`  // 每个账号未吊销的令牌上限
	MaxExpiresInDays int   `json:"maxExpiresInDays"` // 令牌有效期上限（天）
}

// CurrentServiceAccountLimits 当前的业务参数
func CurrentServiceAccountLimits() ServiceAccountLimits {
	roles := make([]int, 0, ServiceAccountMaxRole+1)
	for role := 0; role <= ServiceAccountMaxRole; role++ {
		roles = append(roles, role)
	}
	return ServiceAccountLimits{
		AssignableRoles:  roles,
		MaxActiveTokens:  MaxActiveAPITokensPerAccount,
		MaxExpiresInDays: MaxAPITokenExpiresInDays,
	}
}

// CreateServiceAccountRequest 新建服务账号
type CreateServiceAccountRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"` // 可不填
	Role        *int   `json:"role"`        // 必填；0 是有效值，⛔ 不当作没填
}

// UpdateServiceAccountRequest 修改服务账号（整体替换：三项都必须给出）
type UpdateServiceAccountRequest struct {
	DisplayName *string `json:"displayName"`
	Role        *int    `json:"role"`
	Enabled     *bool   `json:"enabled"`
}

// IssueAPITokenRequest 签发访问令牌
type IssueAPITokenRequest struct {
	Name          string `json:"name"`
	ExpiresInDays *int   `json:"expiresInDays"` // 不给 = 不过期
}

// APITokenIssued 一次签发的结果。
//
// Token 是明文，只在签发的这一次响应里出现；之后任何接口都取不回它。
type APITokenIssued struct {
	Token       string `json:"token"`
	TokenPrefix string `json:"tokenPrefix"`
	ExpiresAt   string `json:"expiresAt"` // RFC 3339（UTC）；不过期为空串
}
