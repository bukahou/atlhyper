// atlhyper_master_v2/auth/oidc/interfaces.go
// OIDC 登录模块：对外 Facade 与依赖契约
//
// 对外只暴露 Module 的三个 HTTP handler（Providers / Start / Callback）。
// 协议细节（state / nonce / PKCE / 验签）交给 oidcrp；本模块只负责
// 「上游身份 → 本地用户」与「签发本系统 token」。
package oidc

import (
	"context"
	"net/http"

	"AtlHyper/atlhyper_master_v2/database"

	"github.com/bukahou/akasha/pkg/oidcrp"
)

// Accounts 本模块所需的账号存取能力（由 service.Service 满足）
type Accounts interface {
	FindUserIDByIdentity(ctx context.Context, issuer, subject string) (int64, bool, error)
	GetUserByID(ctx context.Context, id int64) (*database.User, error)
	CreateFederatedUser(ctx context.Context, user *database.User, issuer, subject string) error
	UpdateUser(ctx context.Context, user *database.User) error
	UpdateUserLastLogin(ctx context.Context, id int64, ip string) error
}

// AdmissionPolicy 新身份是否允许自动建号（已有账号不受影响）
type AdmissionPolicy interface {
	Admit(identity oidcrp.Identity) bool
}

// AdmitAll 全放行：新账号一律为成员（0 级），与访客同权，无额外能力
type AdmitAll struct{}

// Admit 总是允许
func (AdmitAll) Admit(oidcrp.Identity) bool { return true }

// TokenIssuer 签发本系统的登录 token（由网关注入，避免本模块反向依赖网关）
type TokenIssuer func(userID int64, username string, role int) (string, error)

// flowHandler oidcrp.Flow 的最小能力（测试替身用）
type flowHandler interface {
	Start(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
}
