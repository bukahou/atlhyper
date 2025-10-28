// atlhyper_aiservice/router/router.go
package router

import (
	"AtlHyper/atlhyper_aiservice/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 aiservice 所有路由
func RegisterRoutes(r *gin.Engine) {
    ai := r.Group("/ai")
    {
        ai.POST("/diagnose", handler.DiagnoseEventHandler)
    }
}

