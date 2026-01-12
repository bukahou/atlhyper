package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/deployment"

	"github.com/gin-gonic/gin"
)

// GetDeploymentOverviewHandler 获取 Deployment 概览
func GetDeploymentOverviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Deployment 概览失败: 参数错误")
		return
	}

	dto, err := deployment.BuildDeploymentOverview(ctx, req.ClusterID)
	if err != nil {
		response.Error(c, "获取 Deployment 概览失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Deployment 概览成功", dto)
}

// GetDeploymentDetailHandler 获取 Deployment 详情
func GetDeploymentDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Deployment 详情失败: 参数错误")
		return
	}

	dto, err := deployment.BuildDeploymentDetail(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		response.Error(c, "获取 Deployment 详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Deployment 详情成功", dto)
}
