// atlhyper_master_v2/gateway/handler/resource_quota.go
// ResourceQuota 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// ResourceQuotaHandler ResourceQuota Handler
type ResourceQuotaHandler struct {
	svc service.Query
}

// NewResourceQuotaHandler 创建 ResourceQuotaHandler
func NewResourceQuotaHandler(svc service.Query) *ResourceQuotaHandler {
	return &ResourceQuotaHandler{svc: svc}
}

// List 获取 ResourceQuota 列表
// GET /api/v2/resource-quotas?cluster_id=xxx&namespace=xxx
func (h *ResourceQuotaHandler) List(w http.ResponseWriter, r *http.Request) {
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

	quotas, err := h.svc.GetResourceQuotas(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 ResourceQuota 失败")
		return
	}

	items := convert.ResourceQuotaItems(quotas)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 ResourceQuota 详情
// GET /api/v2/resource-quotas/{name}?cluster_id=xxx&namespace=xxx
func (h *ResourceQuotaHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/resource-quotas/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "resource quota name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	quotas, err := h.svc.GetResourceQuotas(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 ResourceQuota 失败")
		return
	}

	for i := range quotas {
		if quotas[i].Name == name {
			detail := convert.ResourceQuotaDetail(&quotas[i])
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "ResourceQuota not found")
}
