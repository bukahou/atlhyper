// atlhyper_master_v2/gateway/handler/settings/github.go
// GitHub 连接管理 + 仓库管理 Handler
package settings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
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
	db          *database.DB
}

// NewGitHubHandler 创建 GitHubHandler
func NewGitHubHandler(
	ghClient github.Client,
	installRepo database.GitHubInstallationRepository,
	repoConfig database.RepoConfigRepository,
	db *database.DB,
) *GitHubHandler {
	return &GitHubHandler{
		ghClient:    ghClient,
		installRepo: installRepo,
		repoConfig:  repoConfig,
		db:          db,
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

// Connect POST /api/github/connect — 发起 GitHub App 安装
func (h *GitHubHandler) Connect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 生成随机 state 防 CSRF
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	urlBuilder, ok := h.ghClient.(interface {
		AuthURL(state string) string
	})
	if !ok {
		handler.WriteError(w, http.StatusInternalServerError, "GitHub App 未配置")
		return
	}

	installURL := urlBuilder.AuthURL(state)
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

	// 设置 Installation ID 到 GitHub Client
	if setter, ok := h.ghClient.(interface{ SetInstallationID(int64) }); ok {
		setter.SetInstallationID(req.InstallationID)
	}

	// 通过 Installation API 获取账号信息
	accountLogin := ""
	if getter, ok := h.ghClient.(interface {
		GetInstallationAccount(ctx context.Context, installationID int64) (string, error)
	}); ok {
		login, err := getter.GetInstallationAccount(r.Context(), req.InstallationID)
		if err == nil {
			accountLogin = login
		}
	}

	// 保存安装记录
	inst := &database.GitHubInstallation{
		InstallationID: req.InstallationID,
		AccountLogin:   accountLogin,
	}
	if err := h.installRepo.Upsert(r.Context(), inst); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "保存安装记录失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": github.ConnectionStatus{
			Connected:      true,
			AccountLogin:   accountLogin,
			AvatarURL:      "https://github.com/" + accountLogin + ".png",
			InstallationID: req.InstallationID,
		},
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
	} else if strings.Contains(path, "/namespaces/") {
		h.NamespaceDelete(w, r)
	} else if strings.HasSuffix(path, "/namespaces") {
		h.Namespaces(w, r)
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

// Namespaces GET/POST /api/github/repos/{owner}/{repo}/namespaces — 查询/添加命名空间
func (h *GitHubHandler) Namespaces(w http.ResponseWriter, r *http.Request) {
	repo := extractRepoFromPath(r.URL.Path, "/api/github/repos/", "/namespaces")
	if repo == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少仓库名")
		return
	}

	switch r.Method {
	case http.MethodGet:
		namespaces, err := h.db.RepoNamespace.ListByRepo(r.Context(), repo)
		if err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "获取命名空间列表失败")
			return
		}
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"data": namespaces,
		})

	case http.MethodPost:
		var req struct {
			Namespace string `json:"namespace"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Namespace == "" {
			handler.WriteError(w, http.StatusBadRequest, "缺少 namespace 参数")
			return
		}

		if err := h.db.RepoNamespace.Add(r.Context(), repo, req.Namespace); err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "添加命名空间失败")
			return
		}

		namespaces, err := h.db.RepoNamespace.ListByRepo(r.Context(), repo)
		if err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "获取命名空间列表失败")
			return
		}
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"message": "命名空间已添加",
			"data":    namespaces,
		})

	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// NamespaceDelete DELETE /api/github/repos/{owner}/{repo}/namespaces/{ns} — 删除命名空间
func (h *GitHubHandler) NamespaceDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 提取 repo 和 namespace: /api/github/repos/{owner}/{repo}/namespaces/{ns}
	path := r.URL.Path
	prefix := "/api/github/repos/"
	rest := strings.TrimPrefix(path, prefix)

	// rest = "owner/repo/namespaces/ns"
	nsIdx := strings.Index(rest, "/namespaces/")
	if nsIdx < 0 {
		handler.WriteError(w, http.StatusBadRequest, "无效的路径")
		return
	}

	repoPath := rest[:nsIdx]
	ns := rest[nsIdx+len("/namespaces/"):]

	parts := strings.SplitN(repoPath, "/", 3)
	if len(parts) < 2 || ns == "" {
		handler.WriteError(w, http.StatusBadRequest, "缺少仓库名或命名空间")
		return
	}
	repo := parts[0] + "/" + parts[1]

	// 删除命名空间
	if err := h.db.RepoNamespace.Remove(r.Context(), repo, ns); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "删除命名空间失败")
		return
	}

	// 删除该命名空间下未确认的映射
	if err := h.db.RepoMapping.DeleteByRepoAndNamespace(r.Context(), repo, ns); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "清理映射失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "命名空间已删除",
	})
}

// Mappings GET/POST /api/github/mappings — 映射列表/创建
func (h *GitHubHandler) Mappings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listMappings(w, r)
	case http.MethodPost:
		h.createMapping(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *GitHubHandler) listMappings(w http.ResponseWriter, r *http.Request) {
	mappings, err := h.db.RepoMapping.List(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "获取映射列表失败")
		return
	}
	if mappings == nil {
		mappings = []*database.RepoDeployMapping{}
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":  mappings,
		"total": len(mappings),
	})
}

func (h *GitHubHandler) createMapping(w http.ResponseWriter, r *http.Request) {
	var mapping database.RepoDeployMapping
	if err := json.NewDecoder(r.Body).Decode(&mapping); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "无效的请求体")
		return
	}

	if err := h.db.RepoMapping.Create(r.Context(), &mapping); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "创建映射失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "映射已创建",
		"data":    mapping,
	})
}

// MappingByID PUT/DELETE /api/github/mappings/{id} — 更新/确认/删除映射
func (h *GitHubHandler) MappingByID(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// 检查是否是确认操作: /api/github/mappings/{id}/confirm
	if strings.HasSuffix(path, "/confirm") {
		h.confirmMapping(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateMapping(w, r)
	case http.MethodDelete:
		h.deleteMapping(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (h *GitHubHandler) confirmMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idStr := extractMappingID(r.URL.Path, "/confirm")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handler.WriteError(w, http.StatusBadRequest, "无效的映射 ID")
		return
	}

	if err := h.db.RepoMapping.Confirm(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "确认映射失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "映射已确认",
	})
}

func (h *GitHubHandler) updateMapping(w http.ResponseWriter, r *http.Request) {
	idStr := extractMappingID(r.URL.Path, "")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handler.WriteError(w, http.StatusBadRequest, "无效的映射 ID")
		return
	}

	var mapping database.RepoDeployMapping
	if err := json.NewDecoder(r.Body).Decode(&mapping); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "无效的请求体")
		return
	}
	mapping.ID = id

	if err := h.db.RepoMapping.Update(r.Context(), &mapping); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "更新映射失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "映射已更新",
	})
}

func (h *GitHubHandler) deleteMapping(w http.ResponseWriter, r *http.Request) {
	idStr := extractMappingID(r.URL.Path, "")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		handler.WriteError(w, http.StatusBadRequest, "无效的映射 ID")
		return
	}

	if err := h.db.RepoMapping.Delete(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "删除映射失败")
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "映射已删除",
	})
}

// extractMappingID 从 /api/github/mappings/{id}[/suffix] 提取 ID
func extractMappingID(path, suffix string) string {
	if suffix != "" {
		path = strings.TrimSuffix(path, suffix)
	}
	prefix := "/api/github/mappings/"
	rest := strings.TrimPrefix(path, prefix)
	// rest could be "123" or "123/"
	rest = strings.TrimSuffix(rest, "/")
	return rest
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
