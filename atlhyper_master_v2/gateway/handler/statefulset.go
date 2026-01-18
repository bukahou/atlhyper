// atlhyper_master_v2/gateway/handler/statefulset.go
// StatefulSet 查询 Handler
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/query"
)

// StatefulSetHandler StatefulSet Handler
type StatefulSetHandler struct {
	query query.Query
}

// NewStatefulSetHandler 创建 StatefulSetHandler
func NewStatefulSetHandler(q query.Query) *StatefulSetHandler {
	return &StatefulSetHandler{query: q}
}

// List 获取 StatefulSet 列表
// GET /api/v2/statefulsets?cluster_id=xxx&namespace=xxx
func (h *StatefulSetHandler) List(w http.ResponseWriter, r *http.Request) {
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

	statefulsets, err := h.query.GetStatefulSets(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 StatefulSet 失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    statefulsets,
		"total":   len(statefulsets),
	})
}

// Get 获取单个 StatefulSet 详情
// GET /api/v2/statefulsets/{name}?cluster_id=xxx&namespace=xxx
func (h *StatefulSetHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/statefulsets/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "statefulset name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	statefulsets, err := h.query.GetStatefulSets(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 StatefulSet 失败")
		return
	}

	for _, s := range statefulsets {
		if s.GetName() == name {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    s,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "StatefulSet not found")
}
