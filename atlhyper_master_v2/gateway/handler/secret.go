// atlhyper_master_v2/gateway/handler/secret.go
// Secret 查询 Handler
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/query"
)

// SecretHandler Secret Handler
type SecretHandler struct {
	query query.Query
}

// NewSecretHandler 创建 SecretHandler
func NewSecretHandler(q query.Query) *SecretHandler {
	return &SecretHandler{query: q}
}

// List 获取 Secret 列表
// GET /api/v2/secrets?cluster_id=xxx&namespace=xxx
func (h *SecretHandler) List(w http.ResponseWriter, r *http.Request) {
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

	secrets, err := h.query.GetSecrets(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Secret 失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    secrets,
		"total":   len(secrets),
	})
}
