// atlhyper_master_v2/gateway/handler/cluster.go
// 集群相关 API Handler
// 通过 Query 层访问数据，禁止直接访问 DataHub
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/service"
)

// ClusterHandler 集群 Handler
type ClusterHandler struct {
	svc service.Query
}

// NewClusterHandler 创建 ClusterHandler
func NewClusterHandler(svc service.Query) *ClusterHandler {
	return &ClusterHandler{svc: svc}
}

// List 列出所有集群
func (h *ClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusters, err := h.svc.ListClusters(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list clusters")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"clusters": clusters,
		"total":    len(clusters),
	})
}
