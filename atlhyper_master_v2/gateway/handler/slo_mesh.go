// atlhyper_master_v2/gateway/handler/slo_mesh.go
// 服务网格 SLO API Handler
//
// 依赖 service.Query 接口，不直接访问 Database。
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/service"
)

// SLOMeshHandler 服务网格 API Handler
type SLOMeshHandler struct {
	query service.Query
}

// NewSLOMeshHandler 创建 SLOMeshHandler
func NewSLOMeshHandler(query service.Query) *SLOMeshHandler {
	return &SLOMeshHandler{query: query}
}

// MeshTopology GET /api/v2/slo/mesh/topology
// 返回服务拓扑（节点 + 边）
func (h *SLOMeshHandler) MeshTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = "default"
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}

	resp, err := h.query.GetMeshTopology(r.Context(), clusterID, timeRange)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ServiceDetail GET /api/v2/slo/mesh/service/detail
// 返回单个服务详情 + 历史 + 上下游
func (h *SLOMeshHandler) ServiceDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = "default"
	}

	namespace := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	if namespace == "" || name == "" {
		writeError(w, http.StatusBadRequest, "namespace and name required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}

	resp, err := h.query.GetServiceDetail(r.Context(), clusterID, namespace, name, timeRange)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if resp == nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
