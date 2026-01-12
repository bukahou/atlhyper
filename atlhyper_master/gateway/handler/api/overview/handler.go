// gateway/handler/api/overview/handler.go
// 总览页处理器
package overview

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/overview"

	"github.com/gin-gonic/gin"
)

// GetClusterListHandler 获取可用集群列表
// GET /uiapi/overview/cluster/list
func GetClusterListHandler(c *gin.Context) {
	ctx := c.Request.Context()

	clusterIDs, err := overview.ListClusterIDs(ctx)
	if err != nil {
		response.ErrorCode(c, 50000, "获取集群列表失败: "+err.Error())
		return
	}

	response.Success(c, "获取集群列表成功", gin.H{
		"clusters": clusterIDs,
	})
}

// GetClusterDetailHandler 获取集群概览详情
// POST /uiapi/overview/cluster/detail
func GetClusterDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取集群概览失败: 参数错误")
		return
	}

	dto, err := overview.BuildOverview(ctx, req.ClusterID)
	if err != nil {
		response.ErrorCode(c, 50000, "构建 Overview 失败: "+err.Error())
		return
	}

	response.Success(c, "获取集群概览成功", dto)
}
