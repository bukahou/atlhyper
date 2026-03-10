// atlhyper_master_v2/github/interfaces.go
// GitHub 集成模块 — 接口定义
package github

import "context"

// Client GitHub API 客户端接口
type Client interface {
	// App 认证
	GetInstallationToken(ctx context.Context) (string, error)

	// 仓库信息
	ListRepos(ctx context.Context, installationID int64) ([]Repository, error)
	ListTopDirs(ctx context.Context, repo, branch string) ([]string, error)
	ScanKustomizePaths(ctx context.Context, repo, branch string) ([]string, error)

	// CD 轮询
	GetLatestCommitSHA(ctx context.Context, repo, branch string) (string, error)
	CompareCommits(ctx context.Context, repo, base, head string) ([]ChangedFile, error)
	ReadFile(ctx context.Context, repo, path, ref string) (string, error)
	ReadDirectory(ctx context.Context, repo, path, ref string) ([]FileEntry, error)

	// Code Intel
	ListCommits(ctx context.Context, repo, branch string, limit int) ([]Commit, error)
	GetPRByCommit(ctx context.Context, repo, sha string) (*PullRequest, error)
	SearchCode(ctx context.Context, repo, query string) ([]CodeSearchResult, error)
}
