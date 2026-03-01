// atlhyper_master_v2/gateway/handler/configmap.go
// ConfigMap 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// ConfigMapHandler ConfigMap Handler
type ConfigMapHandler struct {
	svc service.Query
}

// NewConfigMapHandler 创建 ConfigMapHandler
func NewConfigMapHandler(svc service.Query) *ConfigMapHandler {
	return &ConfigMapHandler{svc: svc}
}

// List 获取 ConfigMap 列表
// GET /api/v2/configmaps?cluster_id=xxx&namespace=xxx
func (h *ConfigMapHandler) List(w http.ResponseWriter, r *http.Request) {
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

	configmaps, err := h.svc.GetConfigMaps(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 ConfigMap 失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    configmaps,
		"total":   len(configmaps),
	})
}

// Get 获取单个 ConfigMap 详情
// GET /api/v2/configmaps/{uid}?cluster_id=xxx
func (h *ConfigMapHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/configmaps/")
	uid := strings.TrimSuffix(path, "/")
	if uid == "" {
		handler.WriteError(w, http.StatusBadRequest, "configmap uid is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	configmaps, err := h.svc.GetConfigMaps(r.Context(), clusterID, "")
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 ConfigMap 失败")
		return
	}

	for _, c := range configmaps {
		if c.UID == uid {
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    c,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "ConfigMap not found")
}
