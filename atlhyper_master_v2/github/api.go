// atlhyper_master_v2/github/api.go
// GitHub REST API 调用
package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// apiGet 发起 GitHub API GET 请求（使用 Installation Token）
func (c *clientImpl) apiGet(ctx context.Context, path string) ([]byte, error) {
	token, err := c.GetInstallationToken(ctx)
	if err != nil {
		return nil, err
	}

	url := "https://api.github.com" + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API %s: %d %s", path, resp.StatusCode, string(body))
	}

	return body, nil
}

// ListRepos 获取安装授权的仓库列表
func (c *clientImpl) ListRepos(ctx context.Context, installationID int64) ([]Repository, error) {
	data, err := c.apiGet(ctx, "/installation/repositories?per_page=100")
	if err != nil {
		return nil, err
	}

	var result struct {
		Repositories []struct {
			FullName      string `json:"full_name"`
			DefaultBranch string `json:"default_branch"`
			Private       bool   `json:"private"`
		} `json:"repositories"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	repos := make([]Repository, len(result.Repositories))
	for i, r := range result.Repositories {
		repos[i] = Repository{
			FullName:      r.FullName,
			DefaultBranch: r.DefaultBranch,
			Private:       r.Private,
		}
	}
	return repos, nil
}

// ListTopDirs 获取仓库顶层目录
func (c *clientImpl) ListTopDirs(ctx context.Context, repo, branch string) ([]string, error) {
	path := fmt.Sprintf("/repos/%s/contents/?ref=%s", repo, url.QueryEscape(branch))
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.Type == "dir" {
			dirs = append(dirs, e.Name+"/")
		}
	}
	return dirs, nil
}

// ScanKustomizePaths 扫描仓库中所有 kustomization.yaml 路径
func (c *clientImpl) ScanKustomizePaths(ctx context.Context, repo, branch string) ([]string, error) {
	// 使用 GitHub Search API 查找 kustomization.yaml
	query := url.QueryEscape(fmt.Sprintf("filename:kustomization.yaml repo:%s", repo))
	path := fmt.Sprintf("/search/code?q=%s&per_page=100", query)

	data, err := c.apiGet(ctx, path)
	if err != nil {
		// Search API 可能受限，降级为递归扫描
		log.Warn("Search API 失败，尝试递归扫描", "err", err)
		return c.scanKustomizePathsRecursive(ctx, repo, branch, "")
	}

	var result struct {
		Items []struct {
			Path string `json:"path"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	var paths []string
	for _, item := range result.Items {
		// path 是 "a/b/kustomization.yaml"，取目录部分
		dir := strings.TrimSuffix(item.Path, "/kustomization.yaml")
		if dir != item.Path {
			paths = append(paths, dir)
		}
	}
	return paths, nil
}

// scanKustomizePathsRecursive 递归扫描 kustomize 路径（降级方案）
func (c *clientImpl) scanKustomizePathsRecursive(ctx context.Context, repo, branch, dir string) ([]string, error) {
	path := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, dir, url.QueryEscape(branch))
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	var paths []string
	hasKustomization := false

	for _, e := range entries {
		if e.Type == "file" && e.Name == "kustomization.yaml" {
			hasKustomization = true
		}
	}

	if hasKustomization {
		paths = append(paths, dir)
	}

	// 递归子目录
	for _, e := range entries {
		if e.Type == "dir" {
			subPaths, err := c.scanKustomizePathsRecursive(ctx, repo, branch, e.Path)
			if err != nil {
				continue // 跳过错误的子目录
			}
			paths = append(paths, subPaths...)
		}
	}

	return paths, nil
}

// GetLatestCommitSHA 获取最新 commit SHA
func (c *clientImpl) GetLatestCommitSHA(ctx context.Context, repo, branch string) (string, error) {
	path := fmt.Sprintf("/repos/%s/commits?sha=%s&per_page=1", repo, url.QueryEscape(branch))
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return "", err
	}

	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(data, &commits); err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", fmt.Errorf("no commits found")
	}
	return commits[0].SHA, nil
}

// CompareCommits 比较两个 commit 的差异
func (c *clientImpl) CompareCommits(ctx context.Context, repo, base, head string) ([]ChangedFile, error) {
	path := fmt.Sprintf("/repos/%s/compare/%s...%s", repo, base, head)
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var result struct {
		Files []struct {
			Filename string `json:"filename"`
			Status   string `json:"status"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	files := make([]ChangedFile, len(result.Files))
	for i, f := range result.Files {
		files[i] = ChangedFile{Filename: f.Filename, Status: f.Status}
	}
	return files, nil
}

// ReadFile 读取仓库文件内容
func (c *clientImpl) ReadFile(ctx context.Context, repo, filePath, ref string) (string, error) {
	path := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, filePath, url.QueryEscape(ref))
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return "", err
	}

	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if result.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	return result.Content, nil
}

// ReadDirectory 读取目录内容
func (c *clientImpl) ReadDirectory(ctx context.Context, repo, dirPath, ref string) ([]FileEntry, error) {
	path := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, dirPath, url.QueryEscape(ref))
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	result := make([]FileEntry, len(entries))
	for i, e := range entries {
		result[i] = FileEntry{Name: e.Name, Type: e.Type, Path: e.Path}
	}
	return result, nil
}

// ListCommits 获取 commit 列表
func (c *clientImpl) ListCommits(ctx context.Context, repo, branch string, limit int) ([]Commit, error) {
	if limit <= 0 {
		limit = 10
	}
	path := fmt.Sprintf("/repos/%s/commits?sha=%s&per_page=%d", repo, url.QueryEscape(branch), limit)
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var ghCommits []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name string    `json:"name"`
				Date time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}
	if err := json.Unmarshal(data, &ghCommits); err != nil {
		return nil, err
	}

	commits := make([]Commit, len(ghCommits))
	for i, gc := range ghCommits {
		commits[i] = Commit{
			SHA:     gc.SHA[:7],
			Message: gc.Commit.Message,
			Author:  gc.Commit.Author.Name,
			Date:    gc.Commit.Author.Date,
		}
	}
	return commits, nil
}

// GetPRByCommit 获取 commit 关联的 PR
func (c *clientImpl) GetPRByCommit(ctx context.Context, repo, sha string) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/commits/%s/pulls", repo, sha)
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var prs []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		URL    string `json:"html_url"`
	}
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, err
	}

	if len(prs) == 0 {
		return nil, nil
	}

	return &PullRequest{
		Number: prs[0].Number,
		Title:  prs[0].Title,
		State:  prs[0].State,
		URL:    prs[0].URL,
	}, nil
}

// SearchCode 搜索代码
func (c *clientImpl) SearchCode(ctx context.Context, repo, query string) ([]CodeSearchResult, error) {
	q := url.QueryEscape(fmt.Sprintf("%s repo:%s", query, repo))
	path := fmt.Sprintf("/search/code?q=%s&per_page=10", q)
	data, err := c.apiGet(ctx, path)
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []struct {
			Path string `json:"path"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	items := make([]CodeSearchResult, len(result.Items))
	for i, item := range result.Items {
		items[i] = CodeSearchResult{Path: item.Path}
	}
	return items, nil
}
