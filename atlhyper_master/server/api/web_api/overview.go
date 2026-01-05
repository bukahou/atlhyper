package web_api

import (
	"github.com/gin-gonic/gin"

	"AtlHyper/atlhyper_master/service/overview"
	"AtlHyper/atlhyper_master/server/api/response"
)

// 参数: clusterID
// GET /uiapi/overview?clusterID=atlhyper
func GetOverviewHandler(c *gin.Context) {

	// 获取上下文
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	// 绑定请求参数
	err := c.ShouldBindJSON(&req)

	// 处理请求参数错误
	if err != nil {
		response.Error(c, "获取集群概览失败: 参数错误")
		return
	}

	// 校验参数
	clusterID := req.ClusterID

	// 获取集群概览
	dto, err := overview.BuildOverview(ctx, clusterID)

	// 处理响应
	if err != nil {
		response.ErrorCode(c, 50000, "构建 Overview 失败: "+err.Error())
		return
	}

	// 统一成功返回
	response.Success(c, "获取集群概览成功", dto)
}
