// atlhyper_master_v2/gateway/handler/network_policy.go
// NetworkPolicy 查询 Handler
package handler

import (
	"net/http"
	"strings"

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
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	policies, err := h.svc.GetNetworkPolicies(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 NetworkPolicy 失败")
		return
	}

	items := convert.NetworkPolicyItems(policies)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
		"total":   len(items),
	})
}

// Get 获取单个 NetworkPolicy 详情
// GET /api/v2/network-policies/{name}?cluster_id=xxx&namespace=xxx
func (h *NetworkPolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v2/network-policies/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "network policy name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	policies, err := h.svc.GetNetworkPolicies(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 NetworkPolicy 失败")
		return
	}

	for i := range policies {
		if policies[i].Name == name {
			detail := convert.NetworkPolicyDetail(&policies[i])
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    detail,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "NetworkPolicy not found")
}
