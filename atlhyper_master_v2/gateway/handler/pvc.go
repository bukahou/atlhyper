// atlhyper_master_v2/gateway/handler/pvc.go
// PersistentVolumeClaim 查询 Handler
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// PVCHandler PersistentVolumeClaim Handler
type PVCHandler struct {
	svc service.Query
}

// NewPVCHandler 创建 PVCHandler
func NewPVCHandler(svc service.Query) *PVCHandler {
	return &PVCHandler{svc: svc}
}

// List 获取 PVC 列表
// GET /api/v2/pvcs?cluster_id=xxx&namespace=xxx
func (h *PVCHandler) List(w http.ResponseWriter, r *http.Request) {
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

	pvcs, err := h.svc.GetPersistentVolumeClaims(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 PVC 失败")
		return
	}

	items := convert.PVCItems(pvcs)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 PVC 详情
// GET /api/v2/pvcs/{name}?cluster_id=xxx&namespace=xxx
func (h *PVCHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/pvcs/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "pvc name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	pvcs, err := h.svc.GetPersistentVolumeClaims(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 PVC 失败")
		return
	}

	for i := range pvcs {
		if pvcs[i].Name == name {
			detail := convert.PVCDetail(&pvcs[i])
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "PVC not found")
}
