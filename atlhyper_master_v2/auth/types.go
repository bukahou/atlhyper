// atlhyper_master_v2/auth/types.go
// 通用认证类型
package auth

// OAuthUser OAuth 认证返回的用户信息
type OAuthUser struct {
	Provider   string // 提供商标识 (如 "github")
	ExternalID string // 外部系统用户 ID
	Login      string // 用户名
	Email      string // 邮箱
	AvatarURL  string // 头像 URL
}
