package settings

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/github"
)

// fakeGitHub 只实现安装管理需要的方法；仓库读取能力用嵌入的 nil 接口占位（被调用即 panic）
type fakeGitHub struct {
	github.Client
	owner         string
	ownerErr      error
	accounts      map[int64]string // installationID → 账号；不在表里 = GitHub 返回 404
	installations []github.Installation
	setID         int64
}

func (f *fakeGitHub) AuthURL(state string) string {
	return "https://github.example/install?state=" + state
}
func (f *fakeGitHub) SetInstallationID(id int64) { f.setID = id }
func (f *fakeGitHub) GetAppOwner(context.Context) (string, error) {
	return f.owner, f.ownerErr
}
func (f *fakeGitHub) GetInstallationAccount(_ context.Context, id int64) (string, error) {
	if a, ok := f.accounts[id]; ok {
		return a, nil
	}
	return "", errors.New("404 Not Found")
}
func (f *fakeGitHub) ListInstallations(context.Context) ([]github.Installation, error) {
	return f.installations, nil
}

type fakeGitHubStore struct {
	saved *database.GitHubInstallation
}

func (s *fakeGitHubStore) GetGitHubInstallation(context.Context) (*database.GitHubInstallation, error) {
	return s.saved, nil
}
func (s *fakeGitHubStore) ListRepoConfigs(context.Context) ([]*database.RepoConfig, error) {
	return nil, nil
}
func (s *fakeGitHubStore) UpsertGitHubInstallation(_ context.Context, inst *database.GitHubInstallation) error {
	s.saved = inst
	return nil
}
func (s *fakeGitHubStore) DeleteGitHubInstallation(context.Context) error { s.saved = nil; return nil }

func postCallback(h *GitHubHandler, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.Callback(rec, httptest.NewRequest(http.MethodPost, "/api/github/callback", strings.NewReader(body)))
	return rec
}

// 回调里的 installation_id 必须属于 App 所有者账号；校验通过之前不切换客户端、不落库。
func TestCallback_RejectsForeignInstallation(t *testing.T) {
	gh := &fakeGitHub{owner: "owner-acct", accounts: map[int64]string{42: "someone-else"}}
	store := &fakeGitHubStore{}
	rec := postCallback(NewGitHubHandler(gh, store, store), `{"installation_id":42}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d, want 403", rec.Code)
	}
	if store.saved != nil || gh.setID != 0 {
		t.Errorf("被拒的安装不得写入：saved=%v setID=%d", store.saved, gh.setID)
	}
}

func TestCallback_AcceptsOwnerInstallation(t *testing.T) {
	gh := &fakeGitHub{owner: "Owner-Acct", accounts: map[int64]string{7: "owner-acct"}} // 账号名大小写不敏感
	store := &fakeGitHubStore{}
	rec := postCallback(NewGitHubHandler(gh, store, store), `{"installation_id":7}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if store.saved == nil || store.saved.InstallationID != 7 || store.saved.AccountLogin != "owner-acct" {
		t.Errorf("saved = %+v", store.saved)
	}
	if gh.setID != 7 {
		t.Errorf("setID = %d, want 7", gh.setID)
	}
}

func TestCallback_LookupFailureWritesNothing(t *testing.T) {
	cases := []struct {
		name string
		gh   *fakeGitHub
		want int
	}{
		{"安装不存在", &fakeGitHub{owner: "owner-acct", accounts: map[int64]string{}}, http.StatusBadRequest},
		{"查不到 App 所有者", &fakeGitHub{ownerErr: errors.New("timeout"), accounts: map[int64]string{7: "owner-acct"}}, http.StatusBadGateway},
	}
	for _, c := range cases {
		store := &fakeGitHubStore{}
		rec := postCallback(NewGitHubHandler(c.gh, store, store), `{"installation_id":7}`)
		if rec.Code != c.want {
			t.Errorf("%s: code = %d, want %d", c.name, rec.Code, c.want)
		}
		if store.saved != nil || c.gh.setID != 0 {
			t.Errorf("%s: 不得写入", c.name)
		}
	}
}

// 自动检测已有安装时只认 App 所有者账号的安装；没有就走安装跳转。
func TestConnect_PicksOwnerInstallationOnly(t *testing.T) {
	gh := &fakeGitHub{owner: "owner-acct", installations: []github.Installation{
		{InstallationID: 1, AccountLogin: "someone-else"},
		{InstallationID: 2, AccountLogin: "owner-acct"},
	}}
	store := &fakeGitHubStore{}
	rec := httptest.NewRecorder()
	NewGitHubHandler(gh, store, store).Connect(rec, httptest.NewRequest(http.MethodPost, "/api/github/connect", nil))
	if rec.Code != http.StatusOK || store.saved == nil || store.saved.InstallationID != 2 || gh.setID != 2 {
		t.Fatalf("code=%d saved=%+v setID=%d", rec.Code, store.saved, gh.setID)
	}

	gh = &fakeGitHub{owner: "owner-acct", installations: []github.Installation{{InstallationID: 1, AccountLogin: "someone-else"}}}
	store = &fakeGitHubStore{}
	rec = httptest.NewRecorder()
	NewGitHubHandler(gh, store, store).Connect(rec, httptest.NewRequest(http.MethodPost, "/api/github/connect", nil))
	if store.saved != nil || gh.setID != 0 || !strings.Contains(rec.Body.String(), "authUrl") {
		t.Errorf("只有别人的安装时应走安装跳转：code=%d saved=%+v body=%s", rec.Code, store.saved, rec.Body.String())
	}
}
