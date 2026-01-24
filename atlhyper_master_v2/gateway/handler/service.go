// atlhyper_master_v2/gateway/handler/service.go
// Service 查询 Handler
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/service"
)

// ServiceHandler Service Handler
type ServiceHandler struct {
	svc service.Query
}

// NewServiceHandler 创建 ServiceHandler
func NewServiceHandler(svc service.Query) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

// List 获取 Service 列表
// GET /api/v2/services?cluster_id=xxx&namespace=xxx
func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	services, err := h.svc.GetServices(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Service 失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    services,
		"total":   len(services),
	})
}

// Get 获取单个 Service 详情
// GET /api/v2/services/{name}?cluster_id=xxx&namespace=xxx
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从路径提取 name: /api/v2/services/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/services/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "service name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	services, err := h.svc.GetServices(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Service 失败")
		return
	}

	for _, s := range services {
		if s.GetName() == name {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    s,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "Service not found")
}
