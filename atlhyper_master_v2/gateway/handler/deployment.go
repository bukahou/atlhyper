// atlhyper_master_v2/gateway/handler/deployment.go
// Deployment 查询 Handler
package handler

import (
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/service"
)

// DeploymentHandler Deployment Handler
type DeploymentHandler struct {
	svc service.Query
}

// NewDeploymentHandler 创建 DeploymentHandler
func NewDeploymentHandler(svc service.Query) *DeploymentHandler {
	return &DeploymentHandler{svc: svc}
}

// List 获取 Deployment 列表
// GET /api/v2/deployments?cluster_id=xxx&namespace=xxx
func (h *DeploymentHandler) List(w http.ResponseWriter, r *http.Request) {
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

	deployments, err := h.svc.GetDeployments(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Deployment 失败")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    deployments,
		"total":   len(deployments),
	})
}

// Get 获取单个 Deployment 详情
// GET /api/v2/deployments/{name}?cluster_id=xxx&namespace=xxx
func (h *DeploymentHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从路径提取 name: /api/v2/deployments/{name}
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/deployments/")
	name := strings.TrimSuffix(path, "/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "deployment name is required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	namespace := r.URL.Query().Get("namespace")

	deployments, err := h.svc.GetDeployments(r.Context(), clusterID, namespace)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询 Deployment 失败")
		return
	}

	for _, d := range deployments {
		if d.GetName() == name {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data":    d,
			})
			return
		}
	}

	writeError(w, http.StatusNotFound, "Deployment not found")
}
