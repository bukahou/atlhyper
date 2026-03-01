// atlhyper_master_v2/gateway/handler/network_policy.go
// NetworkPolicy 查询 Handler
package k8s

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/service"
)

// NetworkPolicyHandler NetworkPolicy Handler
type NetworkPolicyHandler struct {
	svc service.Query
}

// NewNetworkPolicyHandler 创建 NetworkPolicyHandler
func NewNetworkPolicyHandler(svc service.Query) *NetworkPolicyHandler {
	return &NetworkPolicyHandler{svc: svc}
}

// List 获取 NetworkPolicy 列表
// GET /api/v2/network-policies?cluster_id=xxx&namespace=xxx
func (h *NetworkPolicyHandler) List(w http.ResponseWriter, r *http.Request) {
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

	policies, err := h.svc.GetNetworkPolicies(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 NetworkPolicy 失败")
		return
	}

	items := convert.NetworkPolicyItems(policies)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 NetworkPolicy 详情
// GET /api/v2/network-policies/{name}?cluster_id=xxx&namespace=xxx
func (h *NetworkPolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/network-policies/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		handler.WriteError(w, http.StatusBadRequest, "network policy name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	policies, err := h.svc.GetNetworkPolicies(r.Context(), clusterID, namespace)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "查询 NetworkPolicy 失败")
		return
	}

	for i := range policies {
		if policies[i].Name == name {
			detail := convert.NetworkPolicyDetail(&policies[i])
			handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	handler.WriteError(w, http.StatusNotFound, "NetworkPolicy not found")
}
