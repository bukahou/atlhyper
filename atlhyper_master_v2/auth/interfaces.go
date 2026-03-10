// atlhyper_master_v2/auth/interfaces.go
// 通用认证框架 — 接口定义
package auth

import "context"

// OAuthProvider OAuth 认证提供商接口
// auth 包不知道 GitHub 的存在，只定义通用接口
type OAuthProvider interface {
	// AuthURL 生成 OAuth 授权 URL
	AuthURL(state string) string
	// Exchange 用授权码换取用户信息
	Exchange(ctx context.Context, code string) (*OAuthUser, error)
}
