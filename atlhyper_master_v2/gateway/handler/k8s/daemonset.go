// atlhyper_master_v2/gateway/handler/daemonset.go
// DaemonSet 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model/convert"
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
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	daemonsets, err := h.svc.GetDaemonSets(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 DaemonSet 失败")
		return
	}

	items := convert.DaemonSetItems(daemonsets)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 DaemonSet 详情
// GET /api/v2/daemonsets/{name}?cluster_id=xxx&namespace=xxx
func (h *DaemonSetHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/daemonsets/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "daemonset name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	daemonsets, err := h.svc.GetDaemonSets(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 DaemonSet 失败")
		return
	}

	for _, d := range daemonsets {
		if d.GetName() == name {
			detail := convert.DaemonSetDetail(&d)
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "DaemonSet not found")
}
