// atlhyper_master_v2/gateway/handler/service_account.go
// ServiceAccount 查询 Handler
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// ServiceAccountHandler ServiceAccount Handler
type ServiceAccountHandler struct {
	svc service.Query
}

// NewServiceAccountHandler 创建 ServiceAccountHandler
func NewServiceAccountHandler(svc service.Query) *ServiceAccountHandler {
	return &ServiceAccountHandler{svc: svc}
}

// List 获取 ServiceAccount 列表
// GET /api/v2/service-accounts?cluster_id=xxx&namespace=xxx
func (h *ServiceAccountHandler) List(w http.ResponseWriter, r *http.Request) {
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

	accounts, err := h.svc.GetServiceAccounts(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 ServiceAccount 失败")
		return
	}

	items := convert.ServiceAccountItems(accounts)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}
