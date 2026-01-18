// atlhyper_master_v2/gateway/handler/namespace.go
// Namespace 查询 Handler
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/query"
)

// NamespaceHandler Namespace Handler
type NamespaceHandler struct {
	query query.Query
}

// NewNamespaceHandler 创建 NamespaceHandler
func NewNamespaceHandler(q query.Query) *NamespaceHandler {
	return &NamespaceHandler{query: q}
}

// List 获取 Namespace 列表
// GET /api/v2/namespaces?cluster_id=xxx
func (h *NamespaceHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespaces, err := h.query.GetNamespaces(r.Context(), clusterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Namespace 失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    namespaces,
		"total":   len(namespaces),
	})
}

// Get 获取单个 Namespace 详情
// GET /api/v2/namespaces/{name}?cluster_id=xxx
func (h *NamespaceHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/namespaces/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "namespace name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespaces, err := h.query.GetNamespaces(r.Context(), clusterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Namespace 失败")
		return
	}

	for _, ns := range namespaces {
		if ns.GetName() == name {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    ns,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "Namespace not found")
}
