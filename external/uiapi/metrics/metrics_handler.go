package metrics

import (
	"NeuroController/external/metrics_store"
	"NeuroController/external/uiapi/response"
	model "NeuroController/model/metrics"

	"github.com/gin-gonic/gin"
)

// 获取每个节点最新的一条快照
func GetInMemoryLatestHandler(c *gin.Context) {
	all := metrics_store.SnapshotInMemoryMetrics()

	out := make(map[string]*model.NodeMetricsSnapshot, len(all))
	for node, arr := range all {
		if n := len(arr); n > 0 {
			// memBuf 为时间升序，末尾即最新
			out[node] = arr[n-1]
		}
	}

	response.Success(c, "OK", out)
}

// 获取全部节点的快照（可选 since、limit）
func GetInMemoryAllHandler(c *gin.Context) {
	since, _ := parseSince(c.Query("since")) // parseSince 返回 time.Time.Zero 表示无过滤
	limit := parseLimit(c.Query("limit"))

	// 从 metrics_store 直接取全部数据
	all := metrics_store.SnapshotInMemoryMetrics()

	// 如果 since 或 limit 需要生效，就逐节点调用 GetNodeSnapshotsFiltered
	if !since.IsZero() || limit > 0 {
		filtered := make(map[string][]*model.NodeMetricsSnapshot, len(all))
		for node := range all {
			arr := metrics_store.GetNodeSnapshotsFiltered(node, since, limit)
			if len(arr) > 0 {
				filtered[node] = arr
			}
		}
		all = filtered
	}

	response.Success(c, "OK", all)
}

// 获取指定节点的快照（可选 since、limit）
func GetInMemoryByNodeHandler(c *gin.Context) {
	node := c.Param("node")
	since, _ := parseSince(c.Query("since"))
	limit := parseLimit(c.Query("limit"))

	arr := metrics_store.GetNodeSnapshotsFiltered(node, since, limit)
	out := map[string][]*model.NodeMetricsSnapshot{}
	if len(arr) > 0 {
		out[node] = arr
	}

	response.Success(c, "OK", out)
}

