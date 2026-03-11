// atlhyper_master_v2/deployer/service.go
// Deployer 服务实现 — CD 轮询/渲染/部署
package deployer

import (
	"context"
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
	lastCommitSHA string
}

// NewService 创建 Deployer 服务
func NewService(ghClient github.Client, db *database.DB, bus mq.Producer) Deployer {
	return &service{
		ghClient: ghClient,
		db:       db,
		bus:      bus,
	}
}

func (s *service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go s.pollLoop(ctx)
	logger.Info("[Deployer] started")
	return nil
}

func (s *service) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	logger.Info("[Deployer] stopped")
	return nil
}

func (s *service) pollLoop(ctx context.Context) {
	// 使用固定间隔轮询，每次动态读取配置（支持热更新）
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			config := s.loadConfig(ctx)
			if config == nil {
				continue // 配置尚未就绪，等待下一轮
			}
			if !config.AutoDeploy {
				continue
			}
			s.checkAndDeploy(ctx, config)
		}
	}
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
	lastSHA := s.lastCommitSHA
	s.mu.Unlock()

	if sha == lastSHA {
		return
	}

	shortLast := lastSHA
	if len(shortLast) > 8 {
		shortLast = shortLast[:8]
	}
	logger.Info("[Deployer] new commit detected", "sha", sha[:min(8, len(sha))], "previous", shortLast)

	// 确定受影响的路径
	var affectedPaths []string
	if lastSHA == "" {
		// 首次运行，部署所有配置的路径
		affectedPaths = parsePaths(config.Paths)
	} else {
		// 比较 commit 找出变更文件
		changed, err := s.ghClient.CompareCommits(ctx, repo, lastSHA, sha)
		if err != nil {
			logger.Error("[Deployer] compare failed", "error", err)
			affectedPaths = parsePaths(config.Paths)
		} else {
			affectedPaths = matchPaths(parsePaths(config.Paths), changed)
		}
	}

	if len(affectedPaths) == 0 {
		s.mu.Lock()
		s.lastCommitSHA = sha
		s.mu.Unlock()
		return
	}

	// 部署每个受影响的路径
	for _, path := range affectedPaths {
		s.deployPath(ctx, config, path, sha, "auto")
	}

	s.mu.Lock()
	s.lastCommitSHA = sha
	s.mu.Unlock()
}

func (s *service) deployPath(ctx context.Context, config *database.DeployConfig, path, commitSHA, trigger string) {
	repo := config.RepoURL
	clusterID := config.ClusterID

	// 获取 commit message
	commitMsg := ""
	commits, err := s.ghClient.ListCommits(ctx, repo, "main", 1)
	if err == nil && len(commits) > 0 {
		commitMsg = commits[0].Message
	}

	// 创建 pending 历史记录
	record := &database.DeployHistory{
		ClusterID:     clusterID,
		Path:          path,
		CommitSHA:     commitSHA,
		CommitMessage: commitMsg,
		DeployedAt:    time.Now(),
		Trigger:       trigger,
		Status:        "pending",
	}
	_ = s.db.DeployHistory.Create(ctx, record)

	start := time.Now()

	// 使用 kustomize 构建 manifests
	manifests, err := s.kustomizeBuild(ctx, repo, path, commitSHA)
	if err != nil {
		record.Status = "failed"
		record.ErrorMessage = fmt.Sprintf("kustomize build failed: %v", err)
		record.DurationMs = int(time.Since(start).Milliseconds())
		_ = s.db.DeployHistory.Create(ctx, record)
		logger.Error("[Deployer] kustomize build failed", "path", path, "error", err)
		return
	}

	// 从 manifests 提取 namespace
	record.Namespace = extractNamespace(manifests)

	// 统计资源数量
	record.ResourceTotal = strings.Count(manifests, "\nkind:")
	if record.ResourceTotal == 0 && strings.HasPrefix(manifests, "kind:") {
		record.ResourceTotal = 1
	}

	// 通过 MQ 发送 apply_manifests 指令
	cmd := &command.Command{
		ID:        fmt.Sprintf("deploy-%s-%d", path, time.Now().UnixMilli()),
		ClusterID: clusterID,
		Action:    command.ActionApplyManifests,
		Params: map[string]any{
			"manifests": manifests,
			"path":      path,
		},
		Source:    "deployer",
		CreatedAt: time.Now(),
	}

	if err := s.bus.EnqueueCommand(clusterID, mq.TopicOps, cmd); err != nil {
		record.Status = "failed"
		record.ErrorMessage = fmt.Sprintf("enqueue command failed: %v", err)
		record.DurationMs = int(time.Since(start).Milliseconds())
		_ = s.db.DeployHistory.Create(ctx, record)
		logger.Error("[Deployer] enqueue failed", "path", path, "error", err)
		return
	}

	// 等待结果（超时 120 秒）
	result, err := s.bus.WaitCommandResult(ctx, cmd.ID, 120*time.Second)
	if err != nil {
		record.Status = "failed"
		record.ErrorMessage = fmt.Sprintf("command timeout: %v", err)
	} else if result != nil && result.Success {
		record.Status = "success"
	} else if result != nil {
		record.Status = "failed"
		record.ErrorMessage = result.Error
	}

	record.DurationMs = int(time.Since(start).Milliseconds())
	_ = s.db.DeployHistory.Create(ctx, record)
	logger.Info("[Deployer] deployed", "path", path, "status", record.Status, "durationMs", record.DurationMs)
}

func (s *service) SyncNow(ctx context.Context, path string) error {
	config := s.loadConfig(ctx)
	if config == nil {
		return fmt.Errorf("no deploy config found")
	}

	sha, err := s.ghClient.GetLatestCommitSHA(ctx, config.RepoURL, "main")
	if err != nil {
		return fmt.Errorf("get latest commit: %w", err)
	}

	go s.deployPath(ctx, config, path, sha, "manual")
	return nil
}

func (s *service) GetPathStatus(ctx context.Context) ([]PathStatus, error) {
	config := s.loadConfig(ctx)
	if config == nil {
		return nil, nil
	}

	paths := parsePaths(config.Paths)
	var statuses []PathStatus
	for _, p := range paths {
		latest, _ := s.db.DeployHistory.GetLatestByPath(ctx, config.ClusterID, p)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
