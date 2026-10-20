// atlhyper_master_v2/gateway/handler/settings/github.go
// GitHub 连接管理 + 仓库管理 Handler
package settings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/common/logger"
)

var githubLog = logger.Module("GitHub-Handler")

// GitHubHandler GitHub 连接管理 Handler
// GitHubQuery / GitHubOps 本 handler 所需的最小 service 能力。
// 此前直接持有 database repository —— 绕过 service 层，
// 违反 CLAUDE.md「Gateway 直接访问 Database」禁令。
type GitHubQuery interface {
	GetGitHubInstallation(ctx context.Context) (*database.GitHubInstallation, error)
	ListRepoConfigs(ctx context.Context) ([]*database.RepoConfig, error)
}

type GitHubOps interface {
	UpsertGitHubInstallation(ctx context.Context, inst *database.GitHubInstallation) error
	DeleteGitHubInstallation(ctx context.Context) error
}

type GitHubHandler struct {
	ghClient github.Client
	installs github.InstallationManager // nil = 客户端不支持安装管理
	querySvc GitHubQuery
	opsSvc   GitHubOps
}

// NewGitHubHandler 创建 GitHubHandler
func NewGitHubHandler(
	ghClient github.Client,
	querySvc GitHubQuery,
	opsSvc GitHubOps,
) *GitHubHandler {
	installs, _ := ghClient.(github.InstallationManager)
	return &GitHubHandler{
		ghClient: ghClient,
		installs: installs,
		querySvc: querySvc,
		opsSvc:   opsSvc,
	}
}

var (
	errAppOwnerUnavailable = errors.New("无法从 GitHub 确认 App 所有者")
	errInstallationUnknown = errors.New("GitHub 上找不到这个安装")
	errInstallationForeign = errors.New("该安装不属于本 GitHub App 的所有者账号")
)

// verifyOwnerInstallation 确认安装属于 App 所有者账号，返回账号名。
// 部署（GitOps）与 AI 的 GitHub 工具都以这个安装的身份读仓库，
// 所以只接入所有者名下的安装：别人账号的安装、随手编的 installation_id 一律拒绝。
func (h *GitHubHandler) verifyOwnerInstallation(ctx context.Context, installationID int64) (string, error) {
	owner, err := h.installs.GetAppOwner(ctx)
	if err != nil {
		githubLog.Warn("查询 App 所有者失败", "err", err)
		return "", errAppOwnerUnavailable
	}
	account, err := h.installs.GetInstallationAccount(ctx, installationID)
	if err != nil {
		githubLog.Warn("查询安装失败", "installation", installationID, "err", err)
		return "", errInstallationUnknown
	}
	if !strings.EqualFold(account, owner) {
		githubLog.Warn("拒绝接入非所有者账号的安装", "installation", installationID, "account", account)
		return "", errInstallationForeign
	}
	return account, nil
}

// Connection GET /api/github/connection — 获取连接状态
func (h *GitHubHandler) Connection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	inst, err := h.querySvc.GetGitHubInstallation(r.Context())
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

// Connect POST /api/github/connect — 发起 GitHub App 安装
// 优先自动检测已有安装，无需用户手动操作；无安装时才跳转 GitHub
func (h *GitHubHandler) Connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.installs == nil {
		handler.WriteError(w, http.StatusInternalServerError, "GitHub App 未配置")
		return
	}

	// 尝试自动检测已有安装：只认 App 所有者名下的
	if inst, ok := h.findOwnerInstallation(r.Context()); ok {
		if err := h.opsSvc.UpsertGitHubInstallation(r.Context(), &database.GitHubInstallation{
			InstallationID: inst.InstallationID,
			AccountLogin:   inst.AccountLogin,
		}); err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "保存安装记录失败")
			return
		}
		// 设置 Installation ID 供后续 API 调用使用
		h.installs.SetInstallationID(inst.InstallationID)

		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": github.ConnectionStatus{
				Connected:      true,
				AccountLogin:   inst.AccountLogin,
				AvatarURL:      "https://github.com/" + inst.AccountLogin + ".png",
				InstallationID: inst.InstallationID,
			},
		})
		return
	}

	// 无已有安装，跳转 GitHub 安装页面
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	installURL := h.installs.AuthURL(hex.EncodeToString(stateBytes))
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]string{
			"authUrl": installURL,
		},
	})
}

// Callback POST /api/github/callback — GitHub App 安装回调
func (h *GitHubHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		InstallationID int64  `json:"installation_id"`
		SetupAction    string `json:"setup_action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InstallationID == 0 {
		handler.WriteError(w, http.StatusBadRequest, "缺少 installation_id 参数")
		return
	}
	if h.installs == nil {
		handler.WriteError(w, http.StatusInternalServerError, "GitHub App 未配置")
		return
	}

	// 先校验，校验通过之前不落库、不切换客户端身份
	accountLogin, err := h.verifyOwnerInstallation(r.Context(), req.InstallationID)
	switch {
	case errors.Is(err, errInstallationForeign):
		handler.WriteError(w, http.StatusForbidden, err.Error())
		return
	case errors.Is(err, errInstallationUnknown):
		handler.WriteError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		handler.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}

	inst := &database.GitHubInstallation{
		InstallationID: req.InstallationID,
		AccountLogin:   accountLogin,
	}
	if err := h.opsSvc.UpsertGitHubInstallation(r.Context(), inst); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "保存安装记录失败")
		return
	}
	h.installs.SetInstallationID(req.InstallationID)

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": github.ConnectionStatus{
			Connected:      true,
			AccountLogin:   accountLogin,
			AvatarURL:      "https://github.com/" + accountLogin + ".png",
			InstallationID: req.InstallationID,
		},
	})
}

// findOwnerInstallation 在 App 的全部安装里找 App 所有者名下的那个
func (h *GitHubHandler) findOwnerInstallation(ctx context.Context) (github.Installation, bool) {
	owner, err := h.installs.GetAppOwner(ctx)
	if err != nil {
		githubLog.Warn("查询 App 所有者失败", "err", err)
		return github.Installation{}, false
	}
	installations, err := h.installs.ListInstallations(ctx)
	if err != nil {
		githubLog.Warn("列出安装失败", "err", err)
		return github.Installation{}, false
	}
	for _, inst := range installations {
		if strings.EqualFold(inst.AccountLogin, owner) {
			return inst, true
		}
	}
	return github.Installation{}, false
}

// Disconnect DELETE /api/github/disconnect — 断开连接
func (h *GitHubHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := h.opsSvc.DeleteGitHubInstallation(r.Context()); err != nil {
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
	inst, err := h.querySvc.GetGitHubInstallation(r.Context())
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
	configs, err := h.querySvc.ListRepoConfigs(r.Context())
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

// RepoSubRoute 路由分发 /api/github/repos/{owner}/{repo}/{action}
func (h *GitHubHandler) RepoSubRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/dirs") {
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
