package cluster

import "github.com/gin-gonic/gin"

// RegisterClusterRoutes 注册 cluster 模块的所有 HTTP 接口
func RegisterClusterRoutes(rg *gin.RouterGroup) {
	rg.GET("/overview", ClusterOverviewHandler)
}
