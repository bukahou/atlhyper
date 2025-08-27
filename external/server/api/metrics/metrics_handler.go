package metrics

import (
	Snapshot "NeuroController/external/interfaces/metrics"
	"NeuroController/external/server/api/response"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/metrics/latest?cluster_id=xxx
func GetInMemoryLatestHandler(c *gin.Context) {
	clusterID := c.Query("cluster_id")
	if clusterID == "" {
		response.Error(c, "cluster_id 必填")
		return
	}

	latest, err := Snapshot.GetLatestNodeMetricsByCluster(c.Request.Context(), clusterID)
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	// latest: map[string]metrics.NodeMetricsSnapshot（每个节点一条“最新”）
	response.Success(c, "OK", latest)
}


// // 获取全部节点的快照（可选 since、limit）
// func GetInMemoryAllHandler(c *gin.Context) {
// 	since, _ := parseSince(c.Query("since")) 
// 	limit := parseLimit(c.Query("limit"))

// 	// 从 metrics_store 直接取全部数据
// 	all := metrics_store.SnapshotInMemoryMetrics()

// 	// 如果 since 或 limit 需要生效，就逐节点调用 GetNodeSnapshotsFiltered
// 	if !since.IsZero() || limit > 0 {
// 		filtered := make(map[string][]*model.NodeMetricsSnapshot, len(all))
// 		for node := range all {
// 			arr := metrics_store.GetNodeSnapshotsFiltered(node, since, limit)
// 			if len(arr) > 0 {
// 				filtered[node] = arr
// 			}
// 		}
// 		all = filtered
// 	}

// 	response.Success(c, "OK", all)
// }

// // 获取指定节点的快照（可选 since、limit）
// func GetInMemoryByNodeHandler(c *gin.Context) {
// 	node := c.Param("node")
// 	since, _ := parseSince(c.Query("since"))
// 	limit := parseLimit(c.Query("limit"))

// 	arr := metrics_store.GetNodeSnapshotsFiltered(node, since, limit)
// 	out := map[string][]*model.NodeMetricsSnapshot{}
// 	if len(arr) > 0 {
// 		out[node] = arr
// 	}

// 	response.Success(c, "OK", out)
// }

