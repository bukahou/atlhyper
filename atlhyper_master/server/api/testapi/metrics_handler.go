package testapi

import (
	Snapshot "AtlHyper/atlhyper_master/interfaces/test_interfaces"
	response "AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/metrics/latest?cluster_id=xxx
func GetInMemoryLatestHandler(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		response.Error(c, "cluster_id 必填")
		return
	}

	latest, err := Snapshot.GetAllNodeMetricsByCluster(c.Request.Context(), clusterID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	// latest: map[string]metrics.NodeMetricsSnapshot（每个节点一条“最新”）
	response.Success(c, "OK", latest)
}
