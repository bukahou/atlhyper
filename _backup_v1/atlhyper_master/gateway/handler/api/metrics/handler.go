// gateway/handler/api/metrics/handler.go
package metrics

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	"AtlHyper/atlhyper_master/service/datahub/metrics"

	"github.com/gin-gonic/gin"
)

// GetOverviewHandler 获取集群指标概览
// POST /uiapi/metrics/overview
func GetOverviewHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	dto, err := metrics.BuildNodeMetricsOverview(ctx, req.ClusterID)
	if err != nil {
		response.ErrorCode(c, 50000, "获取集群指标概览失败")
		return
	}

	response.Success(c, "获取集群指标概览成功", dto)
}

// GetNodeDetailHandler 获取节点指标详情
// POST /uiapi/metrics/node/detail
func GetNodeDetailHandler(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		ClusterID string `json:"clusterID" binding:"required"`
		NodeID    string `json:"nodeID" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "请求参数错误")
		return
	}

	dto, err := metrics.BuildNodeMetricsDetail(ctx, req.ClusterID, req.NodeID)
	if err != nil {
		response.ErrorCode(c, 50000, "获取节点指标详情失败")
		return
	}

	response.Success(c, "获取节点指标详情成功", dto)
}
