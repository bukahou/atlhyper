// atlhyper_master_v2/gateway/handler/pv.go
// PersistentVolume 查询 Handler
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// PVHandler PersistentVolume Handler
type PVHandler struct {
	svc service.Query
}

// NewPVHandler 创建 PVHandler
func NewPVHandler(svc service.Query) *PVHandler {
	return &PVHandler{svc: svc}
}

// List 获取 PV 列表（集群级，无 namespace）
// GET /api/v2/pvs?cluster_id=xxx
func (h *PVHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	pvs, err := h.svc.GetPersistentVolumes(r.Context(), clusterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 PV 失败")
		return
	}

	items := convert.PVItems(pvs)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}
