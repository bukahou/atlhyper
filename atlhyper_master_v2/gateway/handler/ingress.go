// atlhyper_master_v2/gateway/handler/ingress.go
// Ingress 查询 Handler
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// IngressHandler Ingress Handler
type IngressHandler struct {
	svc service.Query
}

// NewIngressHandler 创建 IngressHandler
func NewIngressHandler(svc service.Query) *IngressHandler {
	return &IngressHandler{svc: svc}
}

// List 获取 Ingress 列表
// GET /api/v2/ingresses?cluster_id=xxx&namespace=xxx
func (h *IngressHandler) List(w http.ResponseWriter, r *http.Request) {
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

	ingresses, err := h.svc.GetIngresses(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Ingress 失败")
		return
	}

	items := convert.IngressItems(ingresses)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 Ingress 详情
// GET /api/v2/ingresses/{name}?cluster_id=xxx&namespace=xxx
func (h *IngressHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/ingresses/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "ingress name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	ingresses, err := h.svc.GetIngresses(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Ingress 失败")
		return
	}

	for _, i := range ingresses {
		if i.GetName() == name {
			detail := convert.IngressDetail(&i)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "Ingress not found")
}
