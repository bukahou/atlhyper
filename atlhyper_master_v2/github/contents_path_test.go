package github

import "testing"

func TestContentsURLPath(t *testing.T) {
	got, err := contentsURLPath("owner/repo", "dir/a b.yaml", "main")
	if err != nil || got != "/repos/owner/repo/contents/dir/a%20b.yaml?ref=main" {
		t.Errorf("got %q, %v", got, err)
	}
	if got, err := contentsURLPath("owner/repo", "", "main"); err != nil || got != "/repos/owner/repo/contents/?ref=main" {
		t.Errorf("仓库根目录: got %q, %v", got, err)
	}
	// 经 .. 走出仓库、或用 ?/# 改写查询：一律拒绝或转义
	for _, bad := range []string{"../other/repo/contents/x", "a/../../b", "./x", "a//b"} {
		if _, err := contentsURLPath("owner/repo", bad, "main"); err == nil {
			t.Errorf("路径 %q 应拒绝", bad)
		}
	}
	if got, _ := contentsURLPath("owner/repo", "a?ref=evil#x", "main"); got != "/repos/owner/repo/contents/a%3Fref=evil%23x?ref=main" {
		t.Errorf("? 与 # 应被转义，got %q", got)
	}
	if _, err := contentsURLPath("owner/repo/x", "a", "main"); err == nil {
		t.Error("仓库名格式不对应拒绝")
	}
}
