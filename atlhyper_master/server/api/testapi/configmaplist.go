package testapi

import (
	"context"
	"net/http"
	"time"

	ifacecm "AtlHyper/atlhyper_master/interfaces/test_interfaces"

	"github.com/gin-gonic/gin"
)

// HandleGetLatestConfigMapList
// GET /api/configmaplist/latest?cluster_id=xxx
// - 从 master_store 中读取指定集群最新一帧 ConfigMap 列表（接收端已用 ReplaceLatest 只保留一帧）
// - 返回 JSON：{ clusterId, count, configmaps }
func HandleGetLatestConfigMapList(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少必填参数：cluster_id"})
		return
	}

	// 设置超时，避免接口被长时间占用
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	cms, err := ifacecm.GetLatestConfigMapListByCluster(ctx, clusterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "获取最新 ConfigMap 列表失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusterId":  clusterID,
		"count":      len(cms),
		"configmaps": cms,
	})
}
