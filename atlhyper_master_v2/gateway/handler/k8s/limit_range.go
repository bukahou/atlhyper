// atlhyper_master_v2/gateway/handler/limit_range.go
// LimitRange 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// LimitRangeHandler LimitRange Handler
type LimitRangeHandler struct {
	svc service.Query
}

// NewLimitRangeHandler 创建 LimitRangeHandler
func NewLimitRangeHandler(svc service.Query) *LimitRangeHandler {
	return &LimitRangeHandler{svc: svc}
}

// List 获取 LimitRange 列表
// GET /api/v2/limit-ranges?cluster_id=xxx&namespace=xxx
func (h *LimitRangeHandler) List(w http.ResponseWriter, r *http.Request) {
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

	ranges, err := h.svc.GetLimitRanges(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 LimitRange 失败")
		return
	}

	items := convert.LimitRangeItems(ranges)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 LimitRange 详情
// GET /api/v2/limit-ranges/{name}?cluster_id=xxx&namespace=xxx
func (h *LimitRangeHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/limit-ranges/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "limit range name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	ranges, err := h.svc.GetLimitRanges(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 LimitRange 失败")
		return
	}

	for i := range ranges {
		if ranges[i].Name == name {
			detail := convert.LimitRangeDetail(&ranges[i])
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "LimitRange not found")
}
