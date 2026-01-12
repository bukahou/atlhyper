package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/configmap"

	"github.com/gin-gonic/gin"
)

// GetConfigMapDetailHandler 获取 ConfigMap 详情
func GetConfigMapDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		Namespace string `json:"namespace" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取 ConfigMap 详情失败: 参数错误")
		return
	}

	dto, err := configmap.BuildConfigMapListFullByNamespace(ctx, req.ClusterID, req.Namespace)
	if err != nil {
		response.Error(c, "获取 ConfigMap 详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取 ConfigMap 详情成功", dto)
}
