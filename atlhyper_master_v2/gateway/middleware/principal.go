// atlhyper_master_v2/gateway/middleware/principal.go
// 请求主体解析与权限检查
//
// 所有请求统一经过 Authenticate：
//   - 无 token → 访客（0 级）
//   - token 有效 → 按 user_id 查库，取当前角色（提升 / 降级 / 禁用即时生效）
//   - 身份无效（格式错 / 签名错 / 已过期 / 用户不存在 / 已禁用 / 查库失败）→ 退化为访客，记录原因
//
// 不拒绝请求本身：0 级接口照常可用；受限接口由 RequirePermission 按主体返回 401 / 403。
package middleware

import (
	"context"
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var authLog = logger.Module("Auth")

// Principal 本次请求的主体
type Principal struct {
	UserID   int64 // 0 = 未识别出用户（访客）
	Username string
	Level    int    // 角色等级，访客为 RoleMember(0)
	Rejected string // 带了 token 但身份无效时的原因（供 401 提示）
}

// IsGuest 未识别出用户
func (p Principal) IsGuest() bool { return p.UserID == 0 }

// UserLoader 按 id 读取用户（由 service.Query 满足）
type UserLoader interface {
	GetUserByID(ctx context.Context, id int64) (*database.User, error)
}

type principalKey struct{}

// WithPrincipal 把主体放入 context
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// GetPrincipal 取主体；未经过 Authenticate 时视为访客
func GetPrincipal(ctx context.Context) Principal {
	if p, ok := ctx.Value(principalKey{}).(Principal); ok {
		return p
	}
	return Principal{Level: RoleMember}
}

// Authenticate 解析请求主体并注入 context
func Authenticate(loader UserLoader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := resolvePrincipal(r.Context(), loader, r.Header.Get("Authorization"))
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
		})
	}
}

func resolvePrincipal(ctx context.Context, loader UserLoader, authHeader string) Principal {
	guest := func(reason string) Principal { return Principal{Level: RoleMember, Rejected: reason} }

	if authHeader == "" {
		return guest("")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return guest("Token 格式错误，需以 Bearer 开头")
	}
	claims, err := ParseToken(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil {
		return guest("Token 无效或已过期，请重新登录")
	}
	idFloat, ok := claims["user_id"].(float64) // JWT 数字解析为 float64
	if !ok || idFloat <= 0 {
		return guest("Token 无效或已过期，请重新登录")
	}

	user, err := loader.GetUserByID(ctx, int64(idFloat))
	if err != nil {
		authLog.Error("查询用户失败，按访客处理", "user_id", int64(idFloat), "err", err)
		return guest("暂时无法验证身份，请稍后重试")
	}
	if user == nil {
		return guest("账号不存在，请重新登录")
	}
	if user.Status != 1 {
		return guest("账号已被禁用")
	}
	return Principal{UserID: user.ID, Username: user.Username, Level: user.Role}
}

// RequirePermission 检查当前主体是否拥有权限
// 访客（含身份无效者）→ 401（提示登录 / 重新登录）；已登录但不够 → 403
func RequirePermission(perm Permission, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := GetPrincipal(r.Context())
		if Allowed(p.Level, perm) {
			next(w, r)
			return
		}
		if p.IsGuest() {
			msg := p.Rejected
			if msg == "" {
				msg = "未登录，请先登录"
			}
			http.Error(w, `{"error": "`+msg+`"}`, http.StatusUnauthorized)
			return
		}
		http.Error(w, `{"error": "权限不足"}`, http.StatusForbidden)
	}
}

// GetUserID 从 context 获取用户 ID（访客返回 false）
func GetUserID(ctx context.Context) (int64, bool) {
	p := GetPrincipal(ctx)
	return p.UserID, !p.IsGuest()
}

// GetUsername 从 context 获取用户名（访客返回 false）
func GetUsername(ctx context.Context) (string, bool) {
	p := GetPrincipal(ctx)
	return p.Username, !p.IsGuest()
}

// GetRole 从 context 获取角色等级（访客返回 RoleMember, false）
func GetRole(ctx context.Context) (int, bool) {
	p := GetPrincipal(ctx)
	return p.Level, !p.IsGuest()
}
