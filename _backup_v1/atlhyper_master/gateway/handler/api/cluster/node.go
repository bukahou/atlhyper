package cluster

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/node"

	"github.com/gin-gonic/gin"
)

// GetNodeOverviewHandler 获取节点概览
func GetNodeOverviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取节点概览失败: 参数错误")
		return
	}

	dto, err := node.BuildNodeOverview(ctx, req.ClusterID)
	if err != nil {
		response.Error(c, "获取节点概览失败: "+err.Error())
		return
	}

	response.Success(c, "获取节点概览成功", dto)
}

// GetNodeDetailHandler 获取节点详情
func GetNodeDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		NodeName  string `json:"nodeName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取节点详情失败: 参数错误")
		return
	}

	dto, err := node.GetNodeDetail(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		response.Error(c, "获取节点详情失败: "+err.Error())
		return
	}

	response.Success(c, "获取节点详情成功", dto)
}
