// atlhyper_master_v2/gateway/handler/pv.go
// PersistentVolume 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
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
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	pvs, err := h.svc.GetPersistentVolumes(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 PV 失败")
		return
	}

	items := convert.PVItems(pvs)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 PV 详情
// GET /api/v2/pvs/{name}?cluster_id=xxx
func (h *PVHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/pvs/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "pv name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	pvs, err := h.svc.GetPersistentVolumes(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 PV 失败")
		return
	}

	for i := range pvs {
		if pvs[i].Name == name {
			detail := convert.PVDetail(&pvs[i])
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "PV not found")
}
