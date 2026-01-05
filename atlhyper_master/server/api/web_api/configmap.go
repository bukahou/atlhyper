package web_api

import (
	"AtlHyper/atlhyper_master/service/configmap"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

func GetConfigMapDetailHandler(c *gin.Context) {
	// 获取请求上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
	}

	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	// 提取参数
	clusterID := req.ClusterID
	namespace := req.Namespace

	// 获取 ConfigMap 详情
	dto, err := configmap.BuildConfigMapListFullByNamespace(ctx, clusterID, namespace)
	if err != nil {
		response.Error(c, "获取 ConfigMap 详情失败")
		return
	}

	// 返回成功响应
	response.Success(c, "获取 ConfigMap 详情成功", dto)
}