// atlhyper_master_v2/gateway/handler/job.go
// Job 查询 Handler
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// JobHandler Job Handler
type JobHandler struct {
	svc service.Query
}

// NewJobHandler 创建 JobHandler
func NewJobHandler(svc service.Query) *JobHandler {
	return &JobHandler{svc: svc}
}

// List 获取 Job 列表
// GET /api/v2/jobs?cluster_id=xxx&namespace=xxx
func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
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

	jobs, err := h.svc.GetJobs(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Job 失败")
		return
	}

	items := convert.JobItems(jobs)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}
