package tools

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/common/truncate"
)

const (
	defaultGitHubBranch = "main"
	maxFileRunes        = 10000
	maxRecentCommits    = 30
)

type githubReadFileParams struct {
	Repo   string `json:"repo" jsonschema:"仓库全名 owner/repo"`
	Path   string `json:"path" jsonschema:"文件路径，相对于仓库根目录"`
	Branch string `json:"branch,omitempty" jsonschema:"分支名（不填为 main）"`
}

type githubSearchParams struct {
	Repo  string `json:"repo" jsonschema:"仓库全名 owner/repo"`
	Query string `json:"query" jsonschema:"搜索关键字（GitHub 代码搜索语法）"`
}

type githubCommitsParams struct {
	Repo  string `json:"repo" jsonschema:"仓库全名 owner/repo"`
	Limit int    `json:"limit,omitempty" jsonschema:"返回最近 N 条（不填为 10）"`
}

// githubTools 只读、只在仓库范围内（公开仓库，或列入白名单的私有仓库）
func githubTools(client github.Client, scope *GitHubRepoScope) []*toolkit.Tool {
	check := func(ctx context.Context, repo string) error {
		if scope == nil {
			return fmt.Errorf("GitHub 工具未配置仓库范围，拒绝读取")
		}
		return scope.Check(ctx, repo)
	}
	return []*toolkit.Tool{
		toolkit.MustDefine(toolkit.Spec[githubReadFileParams]{
			Name:        "github_read_file",
			Description: fmt.Sprintf("读取关联 GitHub 仓库里的文件（超过 %d 个字符截断；Kubernetes Secret 清单里的值会被脱敏）。分析代码或配置时用。", maxFileRunes),
			Permission:  toolkit.PermNone, // 能读哪些仓库由仓库范围限定
			Handler: func(ctx context.Context, _ string, p githubReadFileParams) (string, error) {
				if err := check(ctx, p.Repo); err != nil {
					return "", err
				}
				content, err := client.ReadFile(ctx, p.Repo, p.Path, orDefaultString(p.Branch, defaultGitHubBranch))
				if err != nil {
					return "", fmt.Errorf("读取文件失败: %w", err)
				}
				content = redactSecretManifests(content)
				if clipped := truncate.Runes(content, maxFileRunes, ""); len(clipped) < len(content) {
					content = clipped + fmt.Sprintf("\n... (truncated, total %d bytes)", len(content))
				}
				return content, nil
			},
		}),
		toolkit.MustDefine(toolkit.Spec[githubSearchParams]{
			Name:        "github_search_code",
			Description: "在关联 GitHub 仓库里搜索代码（关键字、类名、函数名）。找某段逻辑在哪里时用。",
			Permission:  toolkit.PermNone,
			Handler: func(ctx context.Context, _ string, p githubSearchParams) (string, error) {
				if err := check(ctx, p.Repo); err != nil {
					return "", err
				}
				results, err := client.SearchCode(ctx, p.Repo, p.Query)
				if err != nil {
					return "", fmt.Errorf("搜索代码失败: %w", err)
				}
				if len(results) == 0 {
					return fmt.Sprintf("仓库 '%s' 里没有匹配 '%s' 的代码", p.Repo, p.Query), nil
				}
				return marshal(map[string]any{"repo": p.Repo, "query": p.Query, "results": results, "count": len(results)})
			},
		}),
		toolkit.MustDefine(toolkit.Spec[githubCommitsParams]{
			Name:        "github_recent_commits",
			Description: fmt.Sprintf("查看仓库 %s 分支最近的提交。了解最近的代码变更时用。", defaultGitHubBranch),
			Permission:  toolkit.PermNone,
			Refine: func(s *jsonschema.Schema) {
				s.Properties["limit"].Minimum = ptr(1.0)
				s.Properties["limit"].Maximum = ptr(float64(maxRecentCommits))
			},
			Handler: func(ctx context.Context, _ string, p githubCommitsParams) (string, error) {
				if err := check(ctx, p.Repo); err != nil {
					return "", err
				}
				commits, err := client.ListCommits(ctx, p.Repo, defaultGitHubBranch, orDefault(p.Limit, 10))
				if err != nil {
					return "", fmt.Errorf("获取提交记录失败: %w", err)
				}
				if len(commits) == 0 {
					return fmt.Sprintf("仓库 '%s' 没有提交记录", p.Repo), nil
				}
				return marshal(map[string]any{"repo": p.Repo, "commits": commits, "count": len(commits)})
			},
		}),
	}
}
