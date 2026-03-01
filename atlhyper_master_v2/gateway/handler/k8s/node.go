// atlhyper_master_v2/gateway/handler/node.go
// Node 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// NodeHandler Node Handler
type NodeHandler struct {
	svc service.Query
}

// NewNodeHandler 创建 NodeHandler
func NewNodeHandler(svc service.Query) *NodeHandler {
	return &NodeHandler{svc: svc}
}

// List 获取 Node 列表
// GET /api/v2/nodes?cluster_id=xxx
func (h *NodeHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	nodes, err := h.svc.GetNodes(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 Node 失败")
		return
	}

	items := convert.NodeItems(nodes)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 Node 详情
// GET /api/v2/nodes/{name}?cluster_id=xxx
func (h *NodeHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从路径提取 name: /api/v2/nodes/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/nodes/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "node name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	nodes, err := h.svc.GetNodes(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 Node 失败")
		return
	}

	for _, node := range nodes {
		if node.GetName() == name {
			detail := convert.NodeDetail(&node)
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "Node not found")
}
