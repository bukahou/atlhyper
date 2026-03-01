// atlhyper_master_v2/gateway/handler/cronjob.go
// CronJob 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// CronJobHandler CronJob Handler
type CronJobHandler struct {
	svc service.Query
}

// NewCronJobHandler 创建 CronJobHandler
func NewCronJobHandler(svc service.Query) *CronJobHandler {
	return &CronJobHandler{svc: svc}
}

// List 获取 CronJob 列表
// GET /api/v2/cronjobs?cluster_id=xxx&namespace=xxx
func (h *CronJobHandler) List(w http.ResponseWriter, r *http.Request) {
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

	cronjobs, err := h.svc.GetCronJobs(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 CronJob 失败")
		return
	}

	items := convert.CronJobItems(cronjobs)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 CronJob 详情
// GET /api/v2/cronjobs/{name}?cluster_id=xxx&namespace=xxx
func (h *CronJobHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/cronjobs/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "cronjob name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	cronjobs, err := h.svc.GetCronJobs(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 CronJob 失败")
		return
	}

	for i := range cronjobs {
		if cronjobs[i].Name == name {
			detail := convert.CronJobDetail(&cronjobs[i])
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "CronJob not found")
}
