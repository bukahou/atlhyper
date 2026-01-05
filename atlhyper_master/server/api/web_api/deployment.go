package web_api

import (
	"AtlHyper/atlhyper_master/service/deployment"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

func GetDeploymentOverviewHandler(c *gin.Context) {
	// 获取请求上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	// 获取 Deployment 概览
	dto, err := deployment.BuildDeploymentOverview(ctx, req.ClusterID)
	if err != nil {
		response.Error(c, "获取 Deployment 概览失败")
		return
	}

	// 返回成功响应
	response.Success(c, "获取 Deployment 概览成功", dto)
}

func GetDeploymentDetailHandler(c *gin.Context) {
	// 获取请求上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID  string `json:"clusterID" binding:"required"`
		Namespace  string `json:"namespace" binding:"required"`
		Name       string `json:"name" binding:"required"`
	}

	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	// 提取参数
	clusterID := req.ClusterID
	namespace := req.Namespace
	name := req.Name

	// 获取 Deployment 详情
	dto, err := deployment.BuildDeploymentDetail(ctx, clusterID, namespace, name)
	if err != nil {
		response.Error(c, "获取 Deployment 详情失败")
		return
	}

	// 返回成功响应
	response.Success(c, "获取 Deployment 详情成功", dto)
}