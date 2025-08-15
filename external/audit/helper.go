// external/audit/helper.go
package audit

import "github.com/gin-gonic/gin"

// 从 JWT 中间件注入的上下文获取用户信息；取不到就给安全默认
func GetUserFromCtxSafe(c *gin.Context) (userID int, username string, role int) {
	userID = 0
	username = "anonymous"
	role = 1

	if c == nil {
		return
	}
	if v, ok := c.Get("user_id"); ok {
		switch t := v.(type) {
		case int:
			userID = t
		case int64:
			userID = int(t)
		case float64:
			userID = int(t)
		case string:
			// 忽略转换错误，保持 0
		}
	}
	if v, ok := c.Get("username"); ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			username = s
		}
	}
	if v, ok := c.Get("role"); ok {
		switch t := v.(type) {
		case int:
			role = t
		case int64:
			role = int(t)
		case float64:
			role = int(t)
		}
		if role < 1 || role > 3 {
			role = 1
		}
	}
	return
}

func safeUsername(s string) string {
	if s == "" {
		return "anonymous"
	}
	return s
}
