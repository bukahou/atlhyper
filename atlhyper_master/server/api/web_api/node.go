package web_api

import (
	"AtlHyper/atlhyper_master/interfaces/ui_interfaces/node"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

//参数: clusterID
//url: /node/overview?clusterID=cluster1
func GetNodeOverviewHandler(c *gin.Context){

	// 获取上下文
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "获取节点概览失败: 参数错误")
		return
	}

	clusterID := req.ClusterID

	if clusterID == "" {
		response.Error(c, "clusterID is required")
		return
	}

	dto, err := node.BuildNodeOverview(ctx, clusterID)

	if err != nil {
		response.Error(c, "failed to get node overview")
		return
	}

	response.Success(c, "node overview retrieved successfully", dto)
}

//参数: clusterID, nodeName
//url: /node/detail?clusterID=cluster1&nodeName=node1
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

	clusterID := req.ClusterID
	nodeName := req.NodeName

	if clusterID == "" || nodeName == "" {
		response.Error(c, "clusterID and nodeName are required")
		return
	}

	dto, err := node.GetNodeDetail(ctx, clusterID, nodeName)

	if err != nil {
		response.Error(c, "failed to get node detail")
		return
	}

	response.Success(c, "node detail retrieved successfully", dto)
}
