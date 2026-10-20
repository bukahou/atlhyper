package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/github"
)

// fakeRepoGitHub 记录可见性查询次数与文件读取
type fakeRepoGitHub struct {
	github.Client
	private  map[string]bool // 小写 owner/repo → 是否私有；不在表里 = 查询失败
	queries  int
	reads    int
	contents string
}

func (f *fakeRepoGitHub) IsPrivateRepo(_ context.Context, repo string) (bool, error) {
	f.queries++
	p, ok := f.private[strings.ToLower(repo)]
	if !ok {
		return false, errors.New("404 Not Found")
	}
	return p, nil
}

func (f *fakeRepoGitHub) ReadFile(_ context.Context, _, _, _ string) (string, error) {
	f.reads++
	return f.contents, nil
}

func newScopeFake() *fakeRepoGitHub {
	return &fakeRepoGitHub{private: map[string]bool{
		"owner/public-repo":  false,
		"owner/private-repo": true,
		"owner/allowed-repo": true,
	}}
}

func TestGitHubRepoScope_Check(t *testing.T) {
	gh := newScopeFake()
	scope := NewGitHubRepoScope(gh, []string{"Owner/Allowed-Repo"})
	ctx := context.Background()

	if err := scope.Check(ctx, "owner/public-repo"); err != nil {
		t.Errorf("公开仓库应放行：%v", err)
	}
	if err := scope.Check(ctx, "owner/private-repo"); err == nil || !strings.Contains(err.Error(), "私有") {
		t.Errorf("不在白名单的私有仓库应拒绝：%v", err)
	}
	if err := scope.Check(ctx, "OWNER/allowed-repo"); err != nil {
		t.Errorf("白名单里的私有仓库应放行（大小写不敏感）：%v", err)
	}
	if err := scope.Check(ctx, "owner/unknown"); err == nil {
		t.Error("确认不了可见性时应拒绝")
	}
}

func TestGitHubRepoScope_RejectsMalformedNamesWithoutQuery(t *testing.T) {
	gh := newScopeFake()
	scope := NewGitHubRepoScope(gh, nil)
	for _, repo := range []string{"", "owner", "owner/repo/extra", "../x", "owner/..", "owner/.", "own er/repo", "owner/repo?x=1"} {
		if err := scope.Check(context.Background(), repo); err == nil {
			t.Errorf("仓库名 %q 应被拒绝", repo)
		}
	}
	if gh.queries != 0 {
		t.Errorf("格式不对的仓库名不应发起查询，got %d 次", gh.queries)
	}
}

func TestGitHubRepoScope_CachesVisibility(t *testing.T) {
	gh := newScopeFake()
	scope := NewGitHubRepoScope(gh, nil)
	for i := 0; i < 3; i++ {
		_ = scope.Check(context.Background(), "owner/public-repo")
	}
	if gh.queries != 1 {
		t.Errorf("可见性应缓存，got %d 次查询", gh.queries)
	}
}

// 读文件工具：范围外不读；读到的 Secret 清单被脱敏
func TestGitHubReadFileHandler_ScopeAndRedaction(t *testing.T) {
	gh := newScopeFake()
	gh.contents = "kind: Secret\ndata:\n  PASSWORD: aHVudGVyMg==\n"
	h := NewGitHubReadFileHandler(gh, NewGitHubRepoScope(gh, nil))
	ctx := context.Background()

	out, _ := h(ctx, "c1", map[string]interface{}{"repo": "owner/private-repo", "path": "secret.yaml"})
	if gh.reads != 0 || !strings.Contains(out, "私有") {
		t.Errorf("私有仓库不应读取：reads=%d out=%q", gh.reads, out)
	}

	out, _ = h(ctx, "c1", map[string]interface{}{"repo": "owner/public-repo", "path": "secret.yaml"})
	if gh.reads != 1 || strings.Contains(out, "aHVudGVyMg==") || !strings.Contains(out, "<redacted>") {
		t.Errorf("公开仓库可读但 Secret 值要脱敏：reads=%d out=%q", gh.reads, out)
	}
}
