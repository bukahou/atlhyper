// atlhyper_master_v2/gateway/handler/settings/github.go
// GitHub 连接管理 + 仓库管理 Handler
package settings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/github"
)

// GitHubHandler GitHub 连接管理 Handler
type GitHubHandler struct {
	ghClient    github.Client
	installRepo database.GitHubInstallationRepository
	repoConfig  database.RepoConfigRepository
}

// NewGitHubHandler 创建 GitHubHandler
func NewGitHubHandler(
	ghClient github.Client,
	installRepo database.GitHubInstallationRepository,
	repoConfig database.RepoConfigRepository,
) *GitHubHandler {
	return &GitHubHandler{
		ghClient:    ghClient,
		installRepo: installRepo,
		repoConfig:  repoConfig,
	}
}

// Connection GET /api/github/connection — 获取连接状态
func (h *GitHubHandler) Connection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	inst, err := h.installRepo.Get(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取连接状态失败")
		return
	}

	status := github.ConnectionStatus{Connected: false}
	if inst != nil {
		status.Connected = true
		status.AccountLogin = inst.AccountLogin
		status.InstallationID = inst.InstallationID
		status.AvatarURL = "https://github.com/" + inst.AccountLogin + ".png"
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    status,
	})
}

// Connect POST /api/github/connect — 发起 OAuth 连接
func (h *GitHubHandler) Connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 生成随机 state 防 CSRF
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	oauthProvider, ok := h.ghClient.(interface {
		AuthURL(state string) string
	})
	if !ok {
		handler.WriteError(w, http.StatusInternalServerError, "GitHub OAuth 未配置")
		return
	}

	authURL := oauthProvider.AuthURL(state)
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{
			"authUrl": authURL,
		},
	})
}

// Callback POST /api/github/callback — OAuth 回调
func (h *GitHubHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少 code 参数")
		return
	}

	exchanger, ok := h.ghClient.(interface {
		ExchangeForConnection(ctx context.Context, code string) (*github.ConnectionStatus, *database.GitHubInstallation, error)
	})
	if !ok {
		handler.WriteError(w, http.StatusInternalServerError, "GitHub OAuth 未配置")
		return
	}

	status, inst, err := exchanger.ExchangeForConnection(r.Context(), req.Code)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "OAuth 交换失败: "+err.Error())
		return
	}

	// 保存安装记录
	if inst != nil {
		if err := h.installRepo.Upsert(r.Context(), inst); err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "保存安装记录失败")
			return
		}
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": status,
	})
}

// Disconnect DELETE /api/github/connection — 断开连接
func (h *GitHubHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := h.installRepo.Delete(r.Context()); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "断开连接失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "已断开 GitHub 连接",
	})
}

// Repos GET /api/github/repos — 获取已授权仓库列表
func (h *GitHubHandler) Repos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 获取安装记录
	inst, err := h.installRepo.Get(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取安装记录失败")
		return
	}
	if inst == nil {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": []interface{}{},
		})
		return
	}

	// 从 GitHub API 获取仓库列表
	repos, err := h.ghClient.ListRepos(r.Context(), inst.InstallationID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取仓库列表失败: "+err.Error())
		return
	}

	// 获取仓库映射配置
	configs, err := h.repoConfig.List(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取映射配置失败")
		return
	}
	configMap := make(map[string]bool)
	for _, c := range configs {
		configMap[c.Repo] = c.MappingEnabled
	}

	// 组合结果
	result := make([]github.AuthorizedRepo, len(repos))
	for i, repo := range repos {
		result[i] = github.AuthorizedRepo{
			Repository:     repo,
			MappingEnabled: configMap[repo.FullName],
		}
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": result,
	})
}

// RepoMapping PUT /api/github/repos/:repo/mapping — 切换映射开关
func (h *GitHubHandler) RepoMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从 URL 提取仓库名（/api/github/repos/{owner}/{repo}/mapping）
	repo := extractRepoFromPath(r.URL.Path, "/api/github/repos/", "/mapping")
	if repo == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少仓库名")
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	// 保存配置
	if err := h.repoConfig.Upsert(r.Context(), &database.RepoConfig{
		Repo:           repo,
		MappingEnabled: req.Enabled,
	}); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "保存配置失败")
		return
	}

	var response map[string]interface{}
	if req.Enabled {
		// 开启映射时，返回仓库顶层目录
		dirs, err := h.ghClient.ListTopDirs(r.Context(), repo, "")
		if err != nil {
			response = map[string]interface{}{
				"message": "映射已开启",
				"data":    map[string]interface{}{"repoDirs": []string{}},
			}
		} else {
			response = map[string]interface{}{
				"message": "映射已开启",
				"data":    map[string]interface{}{"repoDirs": dirs},
			}
		}
	} else {
		response = map[string]interface{}{
			"message": "映射已关闭",
		}
	}

	handler.WriteJSON(w, http.StatusOK, response)
}

// RepoSubRoute 路由分发 /api/github/repos/{owner}/{repo}/{action}
func (h *GitHubHandler) RepoSubRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/mapping") {
		h.RepoMapping(w, r)
	} else if strings.HasSuffix(path, "/dirs") {
		h.RepoDirs(w, r)
	} else {
		handler.WriteError(w, http.StatusNotFound, "未知的路由")
	}
}

// RepoDirs GET /api/github/repos/:repo/dirs — 获取仓库顶层目录
func (h *GitHubHandler) RepoDirs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	repo := extractRepoFromPath(r.URL.Path, "/api/github/repos/", "/dirs")
	if repo == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少仓库名")
		return
	}

	dirs, err := h.ghClient.ListTopDirs(r.Context(), repo, "")
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取目录失败: "+err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": dirs,
	})
}

// extractRepoFromPath 从 URL 路径中提取仓库名
// 例如：/api/github/repos/wuxiafeng/Config/mapping → "wuxiafeng/Config"
func extractRepoFromPath(path, prefix, suffix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if suffix != "" {
		idx := strings.LastIndex(rest, suffix)
		if idx >= 0 {
			rest = rest[:idx]
		}
	}
	// rest should be "owner/repo"
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}
