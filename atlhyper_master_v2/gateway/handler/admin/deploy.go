// atlhyper_master_v2/gateway/handler/admin/deploy.go
// 部署管理 Handler — 配置/kustomize 路径/历史
package admin

import (
	"encoding/json"
	"net/http"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/github"
)

// DeployHandler 部署管理 Handler
type DeployHandler struct {
	ghClient      github.Client
	deployConfig  database.DeployConfigRepository
	deployHistory database.DeployHistoryRepository
	installRepo   database.GitHubInstallationRepository
}

// NewDeployHandler 创建 DeployHandler
func NewDeployHandler(
	ghClient github.Client,
	deployConfig database.DeployConfigRepository,
	deployHistory database.DeployHistoryRepository,
	installRepo database.GitHubInstallationRepository,
) *DeployHandler {
	return &DeployHandler{
		ghClient:      ghClient,
		deployConfig:  deployConfig,
		deployHistory: deployHistory,
		installRepo:   installRepo,
	}
}

// Config GET/PUT /api/deploy/config — 部署配置
func (h *DeployHandler) Config(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r)
	case http.MethodPut:
		h.saveConfig(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *DeployHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("clusterId")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少 clusterId 参数")
		return
	}

	config, err := h.deployConfig.GetByCluster(r.Context(), clusterID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取配置失败")
		return
	}

	if config == nil {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": nil,
		})
		return
	}

	// 解析 paths JSON
	var paths []string
	json.Unmarshal([]byte(config.Paths), &paths)

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"repoUrl":     config.RepoURL,
			"paths":       paths,
			"intervalSec": config.IntervalSec,
			"autoDeploy":  config.AutoDeploy,
			"clusterId":   config.ClusterID,
		},
	})
}

func (h *DeployHandler) saveConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClusterID   string   `json:"clusterId"`
		RepoURL     string   `json:"repoUrl"`
		Paths       []string `json:"paths"`
		IntervalSec int      `json:"intervalSec"`
		AutoDeploy  bool     `json:"autoDeploy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	if req.ClusterID == "" || req.RepoURL == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少必填参数")
		return
	}

	pathsJSON, _ := json.Marshal(req.Paths)
	if err := h.deployConfig.Upsert(r.Context(), &database.DeployConfig{
		ClusterID:   req.ClusterID,
		RepoURL:     req.RepoURL,
		Paths:       string(pathsJSON),
		IntervalSec: req.IntervalSec,
		AutoDeploy:  req.AutoDeploy,
	}); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "保存配置失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "配置已保存",
	})
}

// KustomizePaths GET /api/deploy/kustomize-paths — 扫描仓库中的 kustomize 路径
func (h *DeployHandler) KustomizePaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	repo := r.URL.Query().Get("repo")
	if repo == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少 repo 参数")
		return
	}

	paths, err := h.ghClient.ScanKustomizePaths(r.Context(), repo, "")
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "扫描 kustomize 路径失败: "+err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": paths,
	})
}

// TestConnection POST /api/deploy/test-connection — 测试 GitHub 连接
func (h *DeployHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	inst, err := h.installRepo.Get(r.Context())
	if err != nil || inst == nil {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": map[string]bool{"success": false},
		})
		return
	}

	// 测试能否获取 Installation Token
	_, err = h.ghClient.GetInstallationToken(r.Context())
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]bool{"success": err == nil},
	})
}

// History GET /api/deploy/history — 部署历史
func (h *DeployHandler) History(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("clusterId")
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少 clusterId 参数")
		return
	}

	opts := database.DeployHistoryQueryOpts{
		ClusterID: clusterID,
		Path:      r.URL.Query().Get("path"),
		Limit:     50,
	}

	records, err := h.deployHistory.List(r.Context(), opts)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取部署历史失败")
		return
	}

	total, _ := h.deployHistory.Count(r.Context(), opts)

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  records,
		"total": total,
	})
}
