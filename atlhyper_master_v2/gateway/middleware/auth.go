// atlhyper_master_v2/gateway/middleware/auth.go
// JWT 签发与解析（请求主体解析见 principal.go）
package middleware

import (
	"time"

	"AtlHyper/atlhyper_master_v2/config"

	"github.com/golang-jwt/jwt/v5"
)

// 角色常量（数值越大权限越高；权限集合见 permission.go）
const (
	RoleMember   = 0 // 访客与新注册成员：权限完全相同
	RoleViewer   = 1
	RoleOperator = 2
	RoleAdmin    = 3
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
