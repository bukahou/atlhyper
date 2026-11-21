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

// IsPrivateRepo 查询仓库是否私有（GET /repos/{owner}/{repo}）
func (c *clientImpl) IsPrivateRepo(ctx context.Context, repo string) (bool, error) {
	data, err := c.apiGet(ctx, "/repos/"+repo)
	if err != nil {
		return false, err
	}
	var result struct {
		Private *bool `json:"private"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return false, err
	}
	if result.Private == nil {
		return false, fmt.Errorf("仓库信息里没有 private 字段: %s", repo)
	}
	return *result.Private, nil
}

// ReadFile 读取仓库文件内容
func (c *clientImpl) ReadFile(ctx context.Context, repo, filePath, ref string) (string, error) {
	path, err := contentsURLPath(repo, filePath, ref)
	if err != nil {
		return "", err
	}
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
		Author struct {
			AvatarURL string `json:"avatar_url"`
		} `json:"author"`
	}
	if err := json.Unmarshal(data, &ghCommits); err != nil {
		return nil, err
	}

	commits := make([]Commit, len(ghCommits))
	for i, gc := range ghCommits {
		sha := gc.SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		commits[i] = Commit{
			SHA:       sha,
			Message:   gc.Commit.Message,
			Author:    gc.Commit.Author.Name,
			AvatarURL: gc.Author.AvatarURL,
			Date:      gc.Commit.Author.Date,
		}
	}
	return commits, nil
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

// contentsURLPath 仓库内容接口的路径。仓库内路径先拆成段再逐段转义：
// ⛔ 不允许 . / .. / 空段（否则可以经 .. 走出这个仓库，绕过仓库范围的检查）
func contentsURLPath(repo, repoPath, ref string) (string, error) {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("仓库名格式不对，应为 owner/repo: %q", repo)
	}
	var segments []string
	if trimmed := strings.Trim(repoPath, "/"); trimmed != "" {
		for _, seg := range strings.Split(trimmed, "/") {
			if seg == "" || seg == "." || seg == ".." {
				return "", fmt.Errorf("路径不合法: %q", repoPath)
			}
			segments = append(segments, url.PathEscape(seg))
		}
	}
	return fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s",
		url.PathEscape(owner), url.PathEscape(name), strings.Join(segments, "/"), url.QueryEscape(ref)), nil
}
