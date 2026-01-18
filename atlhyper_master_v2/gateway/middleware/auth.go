// atlhyper_master_v2/gateway/middleware/auth.go
// JWT 认证中间件
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/config"

	"github.com/golang-jwt/jwt/v5"
)

// 角色常量（数值越大权限越高）
const (
	RoleViewer   = 1
	RoleOperator = 2
	RoleAdmin    = 3
)

// 上下文 key 类型（避免与其他包冲突）
type contextKey string

const (
	CtxUserID   contextKey = "user_id"
	CtxUsername contextKey = "username"
	CtxRole     contextKey = "role"
)

// GenerateToken 生成 JWT Token
func GenerateToken(userID int64, username string, role int) (string, error) {
	expiry := config.GlobalConfig.JWT.TokenExpiry
	if expiry == 0 {
		expiry = 24 * time.Hour
	}

	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GlobalConfig.JWT.SecretKey))
}

// ParseToken 解析 JWT Token
func ParseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GlobalConfig.JWT.SecretKey), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrInvalidKeyType
	}
	return claims, nil
}

// AuthRequired 认证中间件（默认要求认证）
// 验证 JWT Token 并将用户信息注入到 context
// 如果 Token 无效或缺失，返回 401
func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			http.Error(w, `{"error": "未登录，请先登录获取 Token"}`, http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error": "Token 格式错误，需以 Bearer 开头"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ParseToken(tokenStr)
		if err != nil {
			http.Error(w, `{"error": "Token 无效或已过期，请重新登录"}`, http.StatusUnauthorized)
			return
		}

		// 将用户信息注入 context
		ctx := r.Context()
		ctx = context.WithValue(ctx, CtxUserID, claims["user_id"])
		ctx = context.WithValue(ctx, CtxUsername, claims["username"])
		ctx = context.WithValue(ctx, CtxRole, claims["role"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Auth 是 AuthRequired 的别名（保持向后兼容）
var Auth = AuthRequired

// RequireMinRole 检查最低角色权限
// 必须在 AuthRequired 之后使用
func RequireMinRole(minRole int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleValue := r.Context().Value(CtxRole)
		if roleValue == nil {
			http.Error(w, `{"error": "权限验证失败"}`, http.StatusForbidden)
			return
		}

		// JWT 中的数字会被解析为 float64
		roleFloat, ok := roleValue.(float64)
		if !ok || int(roleFloat) < minRole {
			http.Error(w, `{"error": "权限不足"}`, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// GetUserID 从 context 获取用户 ID
func GetUserID(ctx context.Context) (int64, bool) {
	val := ctx.Value(CtxUserID)
	if val == nil {
		return 0, false
	}
	if id, ok := val.(float64); ok {
		return int64(id), true
	}
	return 0, false
}

// GetUsername 从 context 获取用户名
func GetUsername(ctx context.Context) (string, bool) {
	val := ctx.Value(CtxUsername)
	if val == nil {
		return "", false
	}
	if name, ok := val.(string); ok {
		return name, true
	}
	return "", false
}

// GetRole 从 context 获取角色
func GetRole(ctx context.Context) (int, bool) {
	val := ctx.Value(CtxRole)
	if val == nil {
		return 0, false
	}
	if role, ok := val.(float64); ok {
		return int(role), true
	}
	return 0, false
}
