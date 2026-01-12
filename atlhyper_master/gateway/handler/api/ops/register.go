// gateway/handler/api/ops/register.go
// 操作命令路由注册
package ops

import (
	"AtlHyper/atlhyper_master/gateway/middleware/auth"

	"github.com/gin-gonic/gin"
)

// Register 注册操作相关路由
// 权限说明：全部需要 Operator 权限（包括敏感的 Pod 日志查看）
func Register(router *gin.RouterGroup) {
	g := router.Group("/ops")
	g.Use(auth.RequireAuth(), auth.RequireMinRole(auth.RoleOperator))
	{
		// Pod 操作
		g.POST("/pod/logs", HandleGetPodLogs)
		g.POST("/pod/restart", HandleRestartPod)

		// Node 操作
		g.POST("/node/cordon", HandleCordonNode)
		g.POST("/node/uncordon", HandleUncordonNode)

		// Workload 操作
		g.POST("/workload/scale", HandleScaleWorkload)
		g.POST("/workload/updateImage", HandleUpdateImage)
	}
}
