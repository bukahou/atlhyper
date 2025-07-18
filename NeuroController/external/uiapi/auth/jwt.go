package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 🔐 JWT 签名密钥（建议后续使用环境变量传入，而非硬编码）
var jwtSecret = []byte("neuro_secret")

// =====================================================
// ✅ GenerateToken：根据用户信息生成 JWT Token
// =====================================================
// 输入参数：用户 ID、用户名、角色（可用于权限控制）
// 输出：字符串形式的 JWT + 错误
func GenerateToken(userID int, username string, role int) (string, error) {
	// 创建 Claims（载荷），包含自定义字段和过期时间（exp）
	claims := jwt.MapClaims{
		"user_id":  userID,                            // 自定义字段：用户 ID
		"username": username,                          // 自定义字段：用户名
		"role":     role,                              // 自定义字段：用户角色（如管理员、普通用户）
		"exp":      time.Now().Add(24 * time.Hour).Unix(), // 过期时间：24 小时后
	}

	// 创建签名对象，使用 HMAC SHA256 签名算法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 对 Token 进行签名并返回字符串
	return token.SignedString(jwtSecret)
}

// =====================================================
// ✅ ParseToken：解析 JWT 字符串，返回其中的 Claims（载荷）
// =====================================================
// 输入参数：JWT 字符串
// 返回：MapClaims（包含用户信息）+ 错误
func ParseToken(tokenStr string) (jwt.MapClaims, error) {
	// 尝试解析 Token，并校验签名
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	// 如果无效或签名失败，直接返回错误
	if err != nil || !token.Valid {
		return nil, err
	}

	// 类型断言：提取 Claims（必须是 MapClaims 类型）
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrInvalidKeyType
	}
	return claims, nil
}
