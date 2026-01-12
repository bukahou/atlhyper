package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ==========================================================
// ✅ AuthMiddleware：Gin 中间件，用于校验 JWT 并提取用户信息
// ==========================================================
// - 拦截所有请求，检查 Authorization Header 是否存在合法 Bearer Token
// - 校验 Token 合法性与有效期
// - 若合法，将用户信息注入到 Gin 的上下文中（Context）
// - 后续处理函数可通过 c.Get("user_id") 等方式获取用户身份信息
// AuthMiddleware 拦截请求并验证 JWT Token，有效则提取用户信息

const (
	RoleViewer   = 1
	RoleOperator = 2
	RoleAdmin    = 3
)



func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未登录，请先登录获取 Token",
			})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token 格式错误，需以 Bearer 开头",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token 无效或已过期，请重新登录",
			})
			return
		}

		// ✅ 注入上下文
		c.Set("user_id", claims["user_id"])
		c.Set("username", claims["username"])
		c.Set("role", claims["role"])

		c.Next()
	}
}


// OptionalAuth 可选认证中间件
// - 有 Token 且有效：解析并注入用户信息
// - 无 Token 或无效：继续执行（不阻止），但不设置用户信息
// 后续可通过 c.Get("user_id") 判断用户是否已登录
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		// 无 Token，继续执行
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ParseToken(tokenStr)
		if err != nil {
			// Token 无效，继续执行但不设置用户信息
			c.Next()
			return
		}

		// Token 有效，注入用户信息
		c.Set("user_id", claims["user_id"])
		c.Set("username", claims["username"])
		c.Set("role", claims["role"])

		c.Next()
	}
}

// RequireAuth 要求必须登录（配合 OptionalAuth 使用）
// 检查 OptionalAuth 是否已解析出用户信息
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("user_id"); !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未登录，请先登录",
			})
			return
		}
		c.Next()
	}
}

// RequireMinRole 只允许角色 ≥ minRole 的用户访问
func RequireMinRole(minRole int) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限验证失败"})
			return
		}
		roleInt, ok := roleValue.(float64) // jwt 的 int 会变成 float64
		if !ok || int(roleInt) < minRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			return
		}
		c.Next()
	}
}
