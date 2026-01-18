// atlhyper_master_v2/gateway/handler/overview.go
// 集群概览 Handler
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/query"
)

// OverviewHandler 集群概览处理器
type OverviewHandler struct {
	query query.Query
}

// NewOverviewHandler 创建 OverviewHandler
func NewOverviewHandler(q query.Query) *OverviewHandler {
	return &OverviewHandler{query: q}
}

// Get 获取集群概览
// GET /api/v2/overview?cluster_id=xxx
func (h *OverviewHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "cluster_id is required",
		})
		return
	}

	overview, err := h.query.GetOverview(r.Context(), clusterID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if overview == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "集群不存在或暂无数据",
		})
		return
	}

	writeJSON(w, http.StatusOK, overview)
}
