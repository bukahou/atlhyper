package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/namespace"

	"github.com/gin-gonic/gin"
)

// GetNamespaceOverviewHandler 获取命名空间概览
func GetNamespaceOverviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取命名空间概览失败: 参数错误")
		return
	}

	dto, err := namespace.BuildNamespaceOverview(ctx, req.ClusterID)
	if err != nil {
		response.Error(c, "获取命名空间概览失败: "+err.Error())
		return
	}

	response.Success(c, "获取命名空间概览成功", dto)
}

// GetNamespaceDetailHandler 获取命名空间详情
func GetNamespaceDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取命名空间详情失败: 参数错误")
		return
	}

	dto, err := namespace.BuildNamespaceDetail(ctx, req.ClusterID, req.Namespace)
	if err != nil {
		response.Error(c, "获取命名空间详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取命名空间详情成功", dto)
}
