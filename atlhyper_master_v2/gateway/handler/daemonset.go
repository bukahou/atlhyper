// atlhyper_master_v2/gateway/handler/daemonset.go
// DaemonSet 查询 Handler
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/service"
)

// DaemonSetHandler DaemonSet Handler
type DaemonSetHandler struct {
	svc service.Query
}

// NewDaemonSetHandler 创建 DaemonSetHandler
func NewDaemonSetHandler(svc service.Query) *DaemonSetHandler {
	return &DaemonSetHandler{svc: svc}
}

// List 获取 DaemonSet 列表
// GET /api/v2/daemonsets?cluster_id=xxx&namespace=xxx
func (h *DaemonSetHandler) List(w http.ResponseWriter, r *http.Request) {
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

	daemonsets, err := h.svc.GetDaemonSets(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 DaemonSet 失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    daemonsets,
		"total":   len(daemonsets),
	})
}

// Get 获取单个 DaemonSet 详情
// GET /api/v2/daemonsets/{name}?cluster_id=xxx&namespace=xxx
func (h *DaemonSetHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/daemonsets/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "daemonset name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	daemonsets, err := h.svc.GetDaemonSets(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 DaemonSet 失败")
		return
	}

	for _, d := range daemonsets {
		if d.GetName() == name {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    d,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "DaemonSet not found")
}
