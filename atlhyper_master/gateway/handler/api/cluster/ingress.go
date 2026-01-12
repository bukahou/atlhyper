package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/ingress"

	"github.com/gin-gonic/gin"
)

// GetIngressOverviewHandler 获取 Ingress 概览
func GetIngressOverviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Ingress 概览失败: 参数错误")
		return
	}

	dto, err := ingress.BuildIngressOverview(ctx, req.ClusterID)
	if err != nil {
		response.Error(c, "获取 Ingress 概览失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Ingress 概览成功", dto)
}

// GetIngressDetailHandler 获取 Ingress 详情
func GetIngressDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		Name      string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Ingress 详情失败: 参数错误")
		return
	}

	dto, err := ingress.BuildIngressDetail(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		response.Error(c, "获取 Ingress 详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Ingress 详情成功", dto)
}
