// NeuroController/external/server/api/podlist/podlist.go
package podlist

import (
	"context"
	"net/http"
	"time"

	ifacepod "NeuroController/external/interfaces/Pod"

	"github.com/gin-gonic/gin"
)

// Register 把路由挂到给定的 RouterGroup 上
// 最终路径示例：/api/pods/latest?cluster_id=cluster-001
// func Register(r *gin.RouterGroup) {
// 	r.GET("/pods/latest", handleGetLatestPodList)
// }

// handleGetLatestPodList
// GET /pods/latest?cluster_id=xxx
// 返回：{"clusterId": "...", "count": 123, "pods": [...]}
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
