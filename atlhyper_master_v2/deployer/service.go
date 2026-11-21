// atlhyper_master_v2/deployer/service.go
// Deployer 服务实现 — CD 轮询/渲染/部署
package deployer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/command"
)

type service struct {
	ghClient github.Client
	db       *database.DB
	bus      mq.Producer

	mu            sync.Mutex
	cancel        context.CancelFunc
	lastCommitSHA string          // 最近一次全部路径都部署成功的提交
	failedSHA     string          // 部署失败的提交：同一个提交不在每次轮询里重试，等新提交或手动同步
	deploying     map[string]bool // 正在部署的路径，防重复

	// build 把某个提交里的路径渲染成清单（默认 kustomizeBuild；测试替换）
	build func(ctx context.Context, repo, path, ref string) (string, error)
}

// NewService 创建 Deployer 服务
func NewService(ghClient github.Client, db *database.DB, bus mq.Producer) Deployer {
	s := &service{
		ghClient:  ghClient,
		db:        db,
		bus:       bus,
		deploying: make(map[string]bool),
	}
	s.build = s.kustomizeBuild
	return s
}

func (s *service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// 从 DB 恢复 lastCommitSHA，避免重启后首次轮询部署所有路径
	s.restoreLastCommitSHA(ctx)

	go s.pollLoop(ctx)
	logger.Info("[Deployer] started")
	return nil
}

// restoreLastCommitSHA 从部署历史恢复「已部署到哪个提交」：取每个路径最近一次成功的部署，
// 再取其中最旧的那个（从它往后的变更都会被重新比较、部署，重复应用是幂等的）。
// 有路径从没成功部署过时不恢复，首轮按全部路径部署。⛔ 失败或未完成的记录不算已部署。
func (s *service) restoreLastCommitSHA(ctx context.Context) {
	config := s.loadConfig(ctx)
	if config == nil {
		return
	}
	paths := parsePaths(config.Paths)
	if len(paths) == 0 {
		return
	}
	var oldest *database.DeployHistory
	for _, p := range paths {
		success, err := s.latestSuccess(ctx, config.ClusterID, p)
		if err != nil {
			logger.Warn("[Deployer] 读部署历史失败，首轮按全部路径部署", "path", p, "error", err)
			return
		}
		if success == nil {
			return
		}
		if oldest == nil || success.DeployedAt.Before(oldest.DeployedAt) {
			oldest = success
		}
	}
	s.mu.Lock()
	s.lastCommitSHA = oldest.CommitSHA
	s.mu.Unlock()
	logger.Info("[Deployer] restored lastCommitSHA from DB", "sha", oldest.CommitSHA[:min(8, len(oldest.CommitSHA))])
}

// latestSuccess 某路径最近一次成功的部署（在最近 50 条里找；找不到返回 nil）
func (s *service) latestSuccess(ctx context.Context, clusterID, path string) (*database.DeployHistory, error) {
	records, err := s.db.DeployHistory.List(ctx, database.DeployHistoryQueryOpts{ClusterID: clusterID, Path: path, Limit: 50})
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.Status == "success" {
			return r, nil
		}
	}
	return nil, nil
}

func (s *service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	logger.Info("[Deployer] stopped")
	return nil
}

func (s *service) pollLoop(ctx context.Context) {
	// 每轮重新读配置（支持热更新），间隔也按配置
	interval := defaultPollInterval
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
		config := s.loadConfig(ctx)
		if config == nil {
			interval = defaultPollInterval
			continue // 配置尚未就绪，等待下一轮
		}
		interval = pollInterval(config)
		if !config.AutoDeploy {
			continue
		}
		s.checkAndDeploy(ctx, config)
	}
}

const (
	defaultPollInterval = 30 * time.Second
	minPollInterval     = 10 * time.Second
)

// pollInterval 配置里的轮询间隔（IntervalSec）；没配用 30 秒，最短 10 秒
func pollInterval(config *database.DeployConfig) time.Duration {
	if config.IntervalSec <= 0 {
		return defaultPollInterval
	}
	if d := time.Duration(config.IntervalSec) * time.Second; d > minPollInterval {
		return d
	}
	return minPollInterval
}

// loadConfig 从数据库加载部署配置（不依赖 clusters 表）
func (s *service) loadConfig(ctx context.Context) *database.DeployConfig {
	configs, err := s.db.DeployConfig.List(ctx)
	if err != nil || len(configs) == 0 {
		return nil
	}
	return configs[0]
}

func (s *service) checkAndDeploy(ctx context.Context, config *database.DeployConfig) {
	repo := config.RepoURL
	if repo == "" {
		return
	}

	// 获取最新 commit SHA
	sha, err := s.ghClient.GetLatestCommitSHA(ctx, repo, "main")
	if err != nil {
		logger.Error("[Deployer] failed to get latest commit", "error", err)
		return
	}

	s.mu.Lock()
	lastSHA, failedSHA := s.lastCommitSHA, s.failedSHA
	s.mu.Unlock()

	if sha == lastSHA || sha == failedSHA {
		return
	}

	shortLast := lastSHA
	if len(shortLast) > 8 {
		shortLast = shortLast[:8]
	}
	logger.Info("[Deployer] new commit detected", "sha", sha[:min(8, len(sha))], "previous", shortLast)

	// 确定受影响的路径
	var affectedPaths []string
	var compareResult *github.CompareResult
	if lastSHA == "" {
		// 首次运行，部署所有配置的路径
		affectedPaths = parsePaths(config.Paths)
	} else {
		// 比较 commit 找出变更文件（同时获取 compare URL 和行数统计）
		cr, err := s.ghClient.CompareCommitsDetail(ctx, repo, lastSHA, sha)
		if err != nil {
			logger.Error("[Deployer] compare failed", "error", err)
			affectedPaths = parsePaths(config.Paths)
		} else {
			compareResult = cr
			affectedPaths = matchPaths(parsePaths(config.Paths), cr.Files)
		}
	}

	if len(affectedPaths) == 0 {
		s.mu.Lock()
		s.lastCommitSHA = sha
		s.mu.Unlock()
		return
	}

	// 部署每个受影响的路径（跳过正在部署的）。全部成功才推进提交号
	failed, skipped := false, false
	for _, path := range affectedPaths {
		s.mu.Lock()
		if s.deploying[path] {
			s.mu.Unlock()
			logger.Info("[Deployer] skipping path (already deploying)", "path", path)
			skipped = true
			continue
		}
		s.deploying[path] = true
		s.mu.Unlock()

		if err := s.deployPath(ctx, config, path, sha, "auto", compareResult); err != nil {
			failed = true
		}
		s.clearDeploying(path)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case failed:
		s.failedSHA = sha // 不推进：这个提交的变更等新提交或手动同步时再部署
	case skipped:
		// 有路径正被别的同步占着：不推进，下一轮再比较
	default:
		s.lastCommitSHA = sha
		s.failedSHA = ""
	}
}

// deployPath 部署一个路径并写部署历史；返回 nil 表示部署成功且历史已记下
func (s *service) deployPath(ctx context.Context, config *database.DeployConfig, path, commitSHA, trigger string, compareResult *github.CompareResult) error {
	clusterID := config.ClusterID
	start := time.Now()
	record := &database.DeployHistory{
		ClusterID:    clusterID,
		Path:         path,
		CommitSHA:    commitSHA,
		ChangedFiles: "[]",
		DeployedAt:   start,
		Trigger:      trigger,
		Status:       "failed",
	}

	deployErr := s.applyPath(ctx, config, record, path, commitSHA, compareResult)
	if deployErr == nil {
		record.Status = "success"
	} else {
		record.ErrorMessage = deployErr.Error()
	}
	record.DurationMs = int(time.Since(start).Milliseconds())

	if err := s.db.DeployHistory.Create(ctx, record); err != nil {
		logger.Error("[Deployer] 写部署历史失败", "path", path, "error", err)
		return fmt.Errorf("写部署历史: %w", err)
	}
	if deployErr != nil {
		logger.Error("[Deployer] deploy failed", "path", path, "error", deployErr)
		return deployErr
	}
	logger.Info("[Deployer] deployed", "path", path, "status", record.Status, "durationMs", record.DurationMs)
	return nil
}

// applyPath 渲染清单、经 Agent 应用并等待结果；成功时把资源数写进 record
func (s *service) applyPath(ctx context.Context, config *database.DeployConfig, record *database.DeployHistory, path, commitSHA string, compareResult *github.CompareResult) error {
	manifests, err := s.build(ctx, config.RepoURL, path, commitSHA)
	if err != nil {
		return fmt.Errorf("kustomize build failed: %w", err)
	}

	record.Namespace = extractNamespace(manifests)
	record.ResourceTotal = strings.Count(manifests, "\nkind:")
	if record.ResourceTotal == 0 && strings.HasPrefix(manifests, "kind:") {
		record.ResourceTotal = 1
	}

	// 从 manifests 解析源码仓库信息（image tag → source SHA → GitHub API）
	s.enrichSourceInfo(ctx, record, manifests, config.RepoURL, compareResult)

	cmd := &command.Command{
		ID:        fmt.Sprintf("deploy-%s-%d", path, time.Now().UnixMilli()),
		ClusterID: config.ClusterID,
		Action:    command.ActionApplyManifests,
		Params: map[string]any{
			"manifests": manifests,
			"path":      path,
		},
		Source:    command.SourceDeployer,
		CreatedAt: time.Now(),
	}
	if err := s.bus.EnqueueCommand(config.ClusterID, mq.TopicOps, cmd); err != nil {
		return fmt.Errorf("enqueue command failed: %w", err)
	}

	result, err := s.bus.WaitCommandResult(ctx, cmd.ID, 120*time.Second)
	if err != nil {
		if errors.Is(err, mq.ErrWaitTimeout) {
			return fmt.Errorf("等待 Agent 执行结果超时（120 秒）: %w", err)
		}
		return fmt.Errorf("等待 Agent 执行结果: %w", err)
	}
	if !result.Success {
		return errors.New(result.Error)
	}

	// 从 Agent 返回的 Output 中解析实际资源变更数（任何一项失败时 Agent 整条指令报失败，不会走到这里）
	var applyResult struct {
		ResourceTotal   int `json:"resourceTotal"`
		ResourceChanged int `json:"resourceChanged"`
	}
	if err := json.Unmarshal([]byte(result.Output), &applyResult); err == nil {
		record.ResourceTotal = applyResult.ResourceTotal
		record.ResourceChanged = applyResult.ResourceChanged
	}
	return nil
}

func (s *service) SyncNow(ctx context.Context, path string) error {
	// 防重复：检查该路径是否正在部署
	s.mu.Lock()
	if s.deploying[path] {
		s.mu.Unlock()
		return fmt.Errorf("path %s is already deploying", path)
	}
	s.deploying[path] = true
	s.mu.Unlock()

	config := s.loadConfig(ctx)
	if config == nil {
		s.clearDeploying(path)
		return fmt.Errorf("no deploy config found")
	}

	sha, err := s.ghClient.GetLatestCommitSHA(ctx, config.RepoURL, "main")
	if err != nil {
		s.clearDeploying(path)
		return fmt.Errorf("get latest commit: %w", err)
	}

	// 手动同步时尝试获取 compare 信息
	var compareResult *github.CompareResult
	s.mu.Lock()
	lastSHA := s.lastCommitSHA
	s.mu.Unlock()
	if lastSHA != "" && lastSHA != sha {
		cr, err := s.ghClient.CompareCommitsDetail(ctx, config.RepoURL, lastSHA, sha)
		if err == nil {
			compareResult = cr
		}
	}

	// 使用独立 context，避免 HTTP 请求完成后 context 被 cancel
	go func() {
		defer s.clearDeploying(path)
		_ = s.deployPath(context.Background(), config, path, sha, "manual", compareResult) // 结果已写进部署历史与日志
	}()
	return nil
}

func (s *service) clearDeploying(path string) {
	s.mu.Lock()
	delete(s.deploying, path)
	s.mu.Unlock()
}

func (s *service) GetPathStatus(ctx context.Context) ([]PathStatus, error) {
	config := s.loadConfig(ctx)
	if config == nil {
		return nil, nil
	}

	paths := parsePaths(config.Paths)
	var statuses []PathStatus
	for _, p := range paths {
		latest, err := s.db.DeployHistory.GetLatestByPath(ctx, config.ClusterID, p)
		if err != nil {
			return nil, fmt.Errorf("读取路径 %s 的部署历史: %w", p, err)
		}
		status := PathStatus{
			Path: p,
		}
		if latest != nil {
			status.Namespace = latest.Namespace
			status.LastSyncAt = latest.DeployedAt
			status.InSync = latest.Status == "success"
			status.ResourceCount = latest.ResourceTotal
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// kustomizeBuild 从 GitHub 下载文件并执行 kustomize build
func (s *service) kustomizeBuild(ctx context.Context, repo, path, ref string) (string, error) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "kustomize-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	// 从 GitHub 下载路径下的所有文件
	entries, err := s.ghClient.ReadDirectory(ctx, repo, path, ref)
	if err != nil {
		return "", fmt.Errorf("read directory %s: %w", path, err)
	}

	for _, entry := range entries {
		if entry.Type != "file" {
			continue
		}
		content, err := s.ghClient.ReadFile(ctx, repo, filepath.Join(path, entry.Name), ref)
		if err != nil {
			return "", fmt.Errorf("read file %s: %w", entry.Name, err)
		}
		filePath := filepath.Join(tmpDir, entry.Name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", err
		}
	}

	// 执行 kustomize build
	cmd := exec.CommandContext(ctx, "kubectl", "kustomize", tmpDir)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("kustomize build: %s", string(exitErr.Stderr))
		}
		return "", err
	}

	return string(output), nil
}

// parsePaths 解析 JSON 编码的路径字符串
func parsePaths(pathsStr string) []string {
	pathsStr = strings.TrimSpace(pathsStr)
	if pathsStr == "" || pathsStr == "[]" {
		return nil
	}
	// 简单解析: 去除方括号并按逗号分割
	pathsStr = strings.Trim(pathsStr, "[]")
	var result []string
	for _, p := range strings.Split(pathsStr, ",") {
		p = strings.Trim(strings.TrimSpace(p), `"`)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// matchPaths 找出受变更文件影响的配置路径
func matchPaths(configuredPaths []string, changedFiles []github.ChangedFile) []string {
	affected := make(map[string]bool)
	for _, f := range changedFiles {
		for _, p := range configuredPaths {
			if strings.HasPrefix(f.Filename, p+"/") || strings.HasPrefix(f.Filename, p) {
				affected[p] = true
			}
		}
	}
	var result []string
	for p := range affected {
		result = append(result, p)
	}
	return result
}

// extractNamespace 从 YAML manifests 中提取第一个 namespace
func extractNamespace(manifests string) string {
	for _, line := range strings.Split(manifests, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "namespace:") {
			ns := strings.TrimSpace(strings.TrimPrefix(line, "namespace:"))
			ns = strings.Trim(ns, `"'`)
			if ns != "" {
				return ns
			}
		}
	}
	return "default"
}

// === 源码仓库信息解析（source.json 方案） ===

// sourceJSON 是 CI 写入 Config 仓库的源码元数据文件
// 支持两种命名：
//   - 单服务: {path}/source.json（传统模式，如 AtlHyper）
//   - 多服务: {path}/source-{service}.json（微服务模式，如 Geass V2）
type sourceJSON struct {
	Repo   string `json:"repo"`   // 源码仓库 (e.g. "example-org/shop-auth")
	SHA    string `json:"sha"`    // 源码 commit SHA (full 40-char)
	Branch string `json:"branch"` // 源码分支 (e.g. "main")
}

// enrichSourceInfo 读取 source.json 解析源码仓库信息，填充到 DeployHistory
// configCompare 是 Config 仓库的 commit compare 结果，用于定位 source-*.json
func (s *service) enrichSourceInfo(ctx context.Context, record *database.DeployHistory, manifests, configRepo string, configCompare *github.CompareResult) {
	src := s.readSourceJSON(ctx, configRepo, record.Path, record.CommitSHA, configCompare)
	if src == nil {
		// source.json 不存在（CI 尚未适配），回退显示 Config 仓库 commit 信息
		s.enrichConfigInfo(ctx, record, configRepo)
		return
	}

	record.SourceRepo = src.Repo
	record.SourceCommitSHA = src.SHA

	// 获取源码 commit 详情
	srcCommit, err := s.ghClient.GetCommitByRef(ctx, src.Repo, src.SHA)
	if err == nil && srcCommit != nil {
		record.CommitMessage = srcCommit.Message
		record.CommitAuthor = srcCommit.Author
		record.CommitAvatarURL = srcCommit.AvatarURL
	}

	// 获取源码关联 PR
	srcPR, err := s.ghClient.GetPRByCommit(ctx, src.Repo, src.SHA)
	if err == nil && srcPR != nil {
		record.PRNumber = srcPR.Number
		record.PRTitle = srcPR.Title
		record.PRURL = srcPR.URL
	}

	// 源码变更文件：与上一次部署的源码 SHA 对比
	prevSHA := s.getPreviousSourceSHA(ctx, record.ClusterID, record.Path)
	if prevSHA != "" && prevSHA != src.SHA {
		srcCompare, err := s.ghClient.CompareCommitsDetail(ctx, src.Repo, prevSHA, src.SHA)
		if err == nil && srcCompare != nil {
			record.CompareURL = srcCompare.HTMLURL
			if filesData, err := json.Marshal(srcCompare.Files); err == nil {
				record.ChangedFiles = string(filesData)
			}
		}
	}

	logger.Info("[Deployer] resolved source info", "sourceRepo", src.Repo, "sourceSHA", src.SHA[:min(8, len(src.SHA))])
}

// enrichConfigInfo 回退方案：使用 Config 仓库的 commit 信息（source.json 不存在时）
func (s *service) enrichConfigInfo(ctx context.Context, record *database.DeployHistory, configRepo string) {
	commit, err := s.ghClient.GetCommitByRef(ctx, configRepo, record.CommitSHA)
	if err == nil && commit != nil {
		record.CommitMessage = commit.Message
		record.CommitAuthor = commit.Author
		record.CommitAvatarURL = commit.AvatarURL
	}
	logger.Info("[Deployer] using config repo commit info (no source.json)", "path", record.Path)
}

// readSourceJSON 从 Config 仓库读取 source 元数据
// 查找优先级：
//  1. 从 Config commit changed files 中找本次变更的 source-*.json（微服务模式）
//  2. 回退读取 {path}/source.json（传统模式）
func (s *service) readSourceJSON(ctx context.Context, configRepo, path, ref string, configCompare *github.CompareResult) *sourceJSON {
	// 优先：从 Config commit 的 changed files 中找 source-*.json
	if configCompare != nil {
		prefix := path + "/source-"
		for _, f := range configCompare.Files {
			if strings.HasPrefix(f.Filename, prefix) && strings.HasSuffix(f.Filename, ".json") {
				src := s.readSourceFile(ctx, configRepo, f.Filename, ref)
				if src != nil {
					return src
				}
			}
		}
	}

	// 回退：传统单文件 source.json
	return s.readSourceFile(ctx, configRepo, path+"/source.json", ref)
}

// readSourceFile 读取并解析单个 source JSON 文件
func (s *service) readSourceFile(ctx context.Context, configRepo, filePath, ref string) *sourceJSON {
	content, err := s.ghClient.ReadFile(ctx, configRepo, filePath, ref)
	if err != nil {
		return nil
	}
	var src sourceJSON
	if err := json.Unmarshal([]byte(content), &src); err != nil {
		logger.Error("[Deployer] failed to parse source json", "path", filePath, "error", err)
		return nil
	}
	if src.Repo == "" || src.SHA == "" {
		return nil
	}
	return &src
}

// getPreviousSourceSHA 从上一次部署记录获取源码 commit SHA
func (s *service) getPreviousSourceSHA(ctx context.Context, clusterID, path string) string {
	latest, err := s.db.DeployHistory.GetLatestByPath(ctx, clusterID, path)
	if err != nil || latest == nil {
		return ""
	}
	return latest.SourceCommitSHA
}
