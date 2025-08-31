package testapi

import (
	"context"
	"net/http"
	"time"

	ifacepod "AtlHyper/atlhyper_master/interfaces/test_interfaces"

	"github.com/gin-gonic/gin"
)

func HandleGetLatestPodList(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少必填参数：cluster_id"})
		return
	}

	// 设一个超时，避免接口被长时间占用
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	pods, err := ifacepod.GetLatestPodListByCluster(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取最新 Pod 列表失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusterId": clusterID,
		"count":     len(pods),
		"pods":      pods,
	})
}
