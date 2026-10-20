// atlhyper_master_v2/ai/github_scope.go
// AI 的 GitHub 工具能读哪些仓库
//
// GitHub App 的安装范围可能包含私有仓库（例如放着部署清单与凭据的配置仓库），
// 而 AI 对话只要 1 级就能用。所以：
//   - 公开仓库直接放行：内容本来就公开，AI 读到不增加泄露面
//   - 私有仓库必须显式列入白名单（MASTER_AI_GITHUB_PRIVATE_REPOS），默认一个都不放行
//   - 确认不了可见性时拒绝（失败即关）
package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// RepoVisibilityChecker 查询仓库是否私有（由 github.Client 提供）
type RepoVisibilityChecker interface {
	IsPrivateRepo(ctx context.Context, repo string) (bool, error)
}

// repoVisibilityTTL 可见性缓存时长：仓库改可见性很少见，但不能永久缓存
const repoVisibilityTTL = 10 * time.Minute

// repoNamePattern owner/repo；GitHub 账号名只含字母数字与连字符，仓库名另可含 . 和 _
var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}$`)

type repoVisibility struct {
	private bool
	expires time.Time
}

// GitHubRepoScope 判断 AI 的 GitHub 工具能否读某个仓库
type GitHubRepoScope struct {
	checker        RepoVisibilityChecker
	privateAllowed map[string]bool // 小写 owner/repo

	mu    sync.Mutex
	cache map[string]repoVisibility
	now   func() time.Time
}

// NewGitHubRepoScope 创建仓库范围；privateAllowed 为允许 AI 读取的私有仓库（owner/repo，大小写不敏感）
func NewGitHubRepoScope(checker RepoVisibilityChecker, privateAllowed []string) *GitHubRepoScope {
	allowed := make(map[string]bool, len(privateAllowed))
	for _, r := range privateAllowed {
		allowed[strings.ToLower(strings.TrimSpace(r))] = true
	}
	return &GitHubRepoScope{
		checker:        checker,
		privateAllowed: allowed,
		cache:          make(map[string]repoVisibility),
		now:            time.Now,
	}
}

// Check 返回 nil 表示可以读；否则返回给模型看的拒绝原因
func (s *GitHubRepoScope) Check(ctx context.Context, repo string) error {
	if !validRepoName(repo) {
		return fmt.Errorf("仓库名格式不对，应为 owner/repo: %q", repo)
	}
	key := strings.ToLower(repo)
	if s.privateAllowed[key] {
		return nil
	}
	private, err := s.isPrivate(ctx, key)
	if err != nil {
		return fmt.Errorf("无法确认仓库 %s 是否公开，拒绝读取: %w", repo, err)
	}
	if private {
		return fmt.Errorf("仓库 %s 是私有仓库，未列入 AI 可读白名单（MASTER_AI_GITHUB_PRIVATE_REPOS）", repo)
	}
	return nil
}

func validRepoName(repo string) bool {
	if !repoNamePattern.MatchString(repo) {
		return false
	}
	name := repo[strings.IndexByte(repo, '/')+1:]
	return name != "." && name != ".."
}

func (s *GitHubRepoScope) isPrivate(ctx context.Context, key string) (bool, error) {
	s.mu.Lock()
	if v, ok := s.cache[key]; ok && s.now().Before(v.expires) {
		s.mu.Unlock()
		return v.private, nil
	}
	s.mu.Unlock()

	private, err := s.checker.IsPrivateRepo(ctx, key)
	if err != nil {
		return false, err // 失败不缓存
	}
	s.mu.Lock()
	s.cache[key] = repoVisibility{private: private, expires: s.now().Add(repoVisibilityTTL)}
	s.mu.Unlock()
	return private, nil
}
