package testapi

import (
	"context"
	"net/http"
	"time"

	ifacenode "AtlHyper/atlhyper_master/interfaces/test_interfaces"

	"github.com/gin-gonic/gin"
)

// HandleGetLatestNodeList
// GET /api/nodes/latest?cluster_id=xxx
func HandleGetLatestNodeList(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少必填参数：cluster_id"})
		return
	}

	// 设置超时，避免阻塞
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	nodes, err := ifacenode.GetLatestNodeListByCluster(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取最新 Node 列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusterId": clusterID,
		"count":     len(nodes),
		"nodes":     nodes,
	})
}

