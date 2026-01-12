package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/pod"

	"github.com/gin-gonic/gin"
)

// GetPodOverviewHandler 获取 Pod 概览
func GetPodOverviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Pod 概览失败: 参数错误")
		return
	}

	dto, err := pod.BuildPodOverview(ctx, req.ClusterID)
	if err != nil {
		response.ErrorCode(c, 50000, "构建 Pod Overview 失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Pod 概览成功", dto)
}

// GetPodDetailHandler 获取 Pod 详情
func GetPodDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
		PodName   string `json:"podName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 Pod 详情失败: 参数错误")
		return
	}

	dto, err := pod.GetPodDetail(ctx, req.ClusterID, req.Namespace, req.PodName)
	if err != nil {
		response.ErrorCode(c, 50000, "获取 Pod 详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取 Pod 详情成功", dto)
}
