package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	svc "AtlHyper/atlhyper_master/service/datahub/svc"

	"github.com/gin-gonic/gin"
)

// GetServiceOverviewHandler 获取 Service 概览
func GetServiceOverviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Service 概览失败: 参数错误")
		return
	}

	dto, err := svc.BuildServiceOverview(ctx, req.ClusterID)
	if err != nil {
		response.Error(c, "获取 Service 概览失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Service 概览成功", dto)
}

// GetServiceDetailHandler 获取 Service 详情
func GetServiceDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Service 详情失败: 参数错误")
		return
	}

	dto, err := svc.GetServiceDetail(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		response.Error(c, "获取 Service 详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Service 详情成功", dto)
}
