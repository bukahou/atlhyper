package web_api

import (
	"AtlHyper/atlhyper_master/service/metrics"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

func GetMetricsOverviewHandler(c *gin.Context) {
	// 解析请求参数
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

	// 获取集群ID
	clusterID := req.ClusterID

	// 调用业务逻辑层
	dto, err := metrics.BuildNodeMetricsOverview(ctx, clusterID)
	if err != nil {
		response.Error(c, "获取集群指标概览失败")
		return
	}

	// 返回响应
	response.Success(c, "获取集群指标概览成功", dto)
}


func GetMetricsNodeDetailHandler(c *gin.Context) {
	// 解析请求参数
	ctx := c.Request.Context()

	// 解析请求参数
	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		NodeID    string `json:"nodeID" binding:"required"`
	}

	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	// 获取集群ID和节点ID
	clusterID := req.ClusterID
	nodeID := req.NodeID

	// 调用业务逻辑层
	dto, err := metrics.BuildNodeMetricsDetail(ctx, clusterID, nodeID)
	if err != nil {
		response.Error(c, "获取节点指标详情失败")
		return
	}

	// 返回响应
	response.Success(c, "获取节点指标详情成功", dto)
}