// atlhyper_master_v2/gateway/handler/service.go
// Service 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model/convert"
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
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	services, err := h.svc.GetServices(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 Service 失败")
		return
	}

	items := convert.ServiceItems(services)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 Service 详情
// GET /api/v2/services/{name}?cluster_id=xxx&namespace=xxx
func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从路径提取 name: /api/v2/services/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/services/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "service name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	services, err := h.svc.GetServices(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 Service 失败")
		return
	}

	for _, s := range services {
		if s.GetName() == name {
			detail := convert.ServiceDetail(&s)
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "Service not found")
}
