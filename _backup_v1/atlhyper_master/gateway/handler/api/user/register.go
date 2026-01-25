// gateway/handler/api/user/register.go
// 用户管理路由注册
package user

import (
	"AtlHyper/atlhyper_master/gateway/middleware/auth"

	"github.com/gin-gonic/gin"
)

// Register 注册用户相关路由
// 权限说明：
//   - 公开：login
//   - Viewer+：list（查看用户列表）
//   - Admin：register, update-role, delete（用户管理）
func Register(router *gin.RouterGroup) {
	g := router.Group("/user")
	{
		// 公开接口
		g.POST("/login", HandleLogin)

		// 需要登录（Viewer 以上）
		authed := g.Group("")
		authed.Use(auth.RequireAuth(), auth.RequireMinRole(auth.RoleViewer))
		{
			authed.GET("/list", HandleListAllUsers)
		}

		// 管理员接口
		admin := g.Group("")
		admin.Use(auth.RequireAuth(), auth.RequireMinRole(auth.RoleAdmin))
		{
			admin.POST("/register", HandleRegisterUser)
			admin.POST("/update-role", HandleUpdateUserRole)
			admin.POST("/delete", HandleDeleteUser)
		}
	}
}
