// atlhyper_master_v2/gateway/handler/cluster.go
// 集群相关 API Handler
// 通过 Query 层访问数据，禁止直接访问 DataHub
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/query"
)

// ClusterHandler 集群 Handler
type ClusterHandler struct {
	query query.Query
}

// NewClusterHandler 创建 ClusterHandler
func NewClusterHandler(q query.Query) *ClusterHandler {
	return &ClusterHandler{
		query: q,
	}
}

// List 列出所有集群
func (h *ClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusters, err := h.query.ListClusters(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list clusters")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"clusters": clusters,
		"total":    len(clusters),
	})
}

// Get 获取单个集群详情
func (h *ClusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 解析 clusterID: /api/v2/clusters/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/clusters/")
	clusterID := strings.TrimSuffix(path, "/")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id required")
		return
	}

	detail, err := h.query.GetCluster(r.Context(), clusterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get cluster")
		return
	}
	if detail == nil || detail.Snapshot == nil {
		writeError(w, http.StatusNotFound, "cluster not found")
		return
	}

	writeJSON(w, http.StatusOK, detail)
}
