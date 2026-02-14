// atlhyper_master_v2/gateway/handler/pod.go
// Pod 查询 Handler
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// PodHandler Pod Handler
type PodHandler struct {
	svc service.Query
}

// NewPodHandler 创建 PodHandler
func NewPodHandler(svc service.Query) *PodHandler {
	return &PodHandler{svc: svc}
}

// List 获取 Pod 列表
// GET /api/v2/pods?cluster_id=xxx&namespace=xxx&node=xxx&phase=xxx&limit=50&offset=0
func (h *PodHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 构建查询选项
	opts := model.PodQueryOpts{
		Namespace: r.URL.Query().Get("namespace"),
		NodeName:  r.URL.Query().Get("node"),
		Phase:     r.URL.Query().Get("phase"),
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			opts.Limit = v
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if v, err := strconv.Atoi(offset); err == nil {
			opts.Offset = v
		}
	}

	pods, err := h.svc.GetPods(r.Context(), clusterID, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Pod 失败")
		return
	}

	items := convert.PodItems(pods)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 Pod 详情
// GET /api/v2/pods/{name}?cluster_id=xxx&namespace=xxx
func (h *PodHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从路径提取 name: /api/v2/pods/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/pods/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "pod name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	// 获取 Pod 列表并查找
	pods, err := h.svc.GetPods(r.Context(), clusterID, model.PodQueryOpts{
		Namespace: namespace,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Pod 失败")
		return
	}

	for _, pod := range pods {
		if pod.GetName() == name {
			detail := convert.PodDetail(&pod)
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "Pod not found")
}
