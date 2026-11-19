package deployer

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/model_v3/command"
)

// ---- 测试替身 ----

type fakeGitHub struct {
	github.Client
	headSHA string
}

func (f *fakeGitHub) GetLatestCommitSHA(context.Context, string, string) (string, error) {
	return f.headSHA, nil
}

func (f *fakeGitHub) CompareCommitsDetail(_ context.Context, _, _, _ string) (*github.CompareResult, error) {
	return &github.CompareResult{Files: []github.ChangedFile{{Filename: "apps/web/deployment.yaml"}}}, nil
}

// 来源信息补全用到的调用：本测试不关心，一律返回取不到
func (f *fakeGitHub) ReadFile(context.Context, string, string, string) (string, error) {
	return "", errors.New("not found")
}

func (f *fakeGitHub) GetCommitByRef(context.Context, string, string) (*github.Commit, error) {
	return nil, errors.New("not found")
}

type fakeHistory struct {
	database.DeployHistoryRepository
	mu        sync.Mutex
	records   []*database.DeployHistory
	createErr error
}

func (f *fakeHistory) Create(_ context.Context, r *database.DeployHistory) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	copied := *r
	f.records = append(f.records, &copied)
	return nil
}

// List 按部署时间倒序返回某路径的记录
func (f *fakeHistory) List(_ context.Context, opts database.DeployHistoryQueryOpts) ([]*database.DeployHistory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*database.DeployHistory
	for _, r := range f.records {
		if r.ClusterID == opts.ClusterID && r.Path == opts.Path {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DeployedAt.After(out[j].DeployedAt) })
	return out, nil
}

func (f *fakeHistory) GetLatestByPath(ctx context.Context, clusterID, path string) (*database.DeployHistory, error) {
	list, _ := f.List(ctx, database.DeployHistoryQueryOpts{ClusterID: clusterID, Path: path})
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

type fakeConfigs struct {
	database.DeployConfigRepository
	config *database.DeployConfig
}

func (f *fakeConfigs) List(context.Context) ([]*database.DeployConfig, error) {
	return []*database.DeployConfig{f.config}, nil
}

type fakeBus struct {
	mq.Producer
	result *command.Result
	err    error
}

func (f *fakeBus) EnqueueCommand(string, string, *command.Command) error { return nil }

func (f *fakeBus) WaitCommandResult(context.Context, string, time.Duration) (*command.Result, error) {
	return f.result, f.err
}

func newTestService(head string, bus *fakeBus, history *fakeHistory) (*service, *int) {
	builds := 0
	cfg := &database.DeployConfig{ClusterID: "c1", RepoURL: "org/config", Paths: `["apps/web"]`, AutoDeploy: true}
	s := NewService(&fakeGitHub{headSHA: head}, &database.DB{DeployHistory: history, DeployConfig: &fakeConfigs{config: cfg}}, bus).(*service)
	s.build = func(context.Context, string, string, string) (string, error) {
		builds++
		return "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n  namespace: shop\n", nil
	}
	return s, &builds
}

func latestRecord(t *testing.T, h *fakeHistory) *database.DeployHistory {
	t.Helper()
	if len(h.records) == 0 {
		t.Fatal("没有写部署历史")
	}
	return h.records[len(h.records)-1]
}

// ---- 用例 ----

func TestCheckAndDeploy_SuccessAdvancesCommit(t *testing.T) {
	history := &fakeHistory{}
	s, _ := newTestService("sha-2", &fakeBus{result: &command.Result{Success: true, Output: `{"resourceTotal":1,"resourceChanged":1}`}}, history)
	s.lastCommitSHA = "sha-1"

	s.checkAndDeploy(context.Background(), s.loadConfig(context.Background()))

	if s.lastCommitSHA != "sha-2" {
		t.Errorf("成功后应推进到 sha-2，实际 %q", s.lastCommitSHA)
	}
	if r := latestRecord(t, history); r.Status != "success" || r.ResourceChanged != 1 {
		t.Errorf("历史 = %+v", r)
	}
}

// 部署失败：提交号不推进（重启或下一个提交时这次的变更仍会被部署）；同一个失败的提交不在每次轮询里重试
func TestCheckAndDeploy_FailureKeepsCommitAndDoesNotLoop(t *testing.T) {
	history := &fakeHistory{}
	s, builds := newTestService("sha-2", &fakeBus{result: &command.Result{Success: false, Error: "应用中断（已应用 0/1）"}}, history)
	s.lastCommitSHA = "sha-1"
	cfg := s.loadConfig(context.Background())

	s.checkAndDeploy(context.Background(), cfg)
	s.checkAndDeploy(context.Background(), cfg)

	if s.lastCommitSHA != "sha-1" {
		t.Errorf("失败后不应推进提交号，实际 %q", s.lastCommitSHA)
	}
	if *builds != 1 {
		t.Errorf("同一个失败的提交被部署了 %d 次，want 1", *builds)
	}
	if r := latestRecord(t, history); r.Status != "failed" || !strings.Contains(r.ErrorMessage, "应用中断") {
		t.Errorf("历史 = %+v", r)
	}
}

// 等 Agent 结果超时：记为失败（此前停在 pending）
func TestDeployPath_WaitTimeoutRecordedAsFailed(t *testing.T) {
	history := &fakeHistory{}
	s, _ := newTestService("sha-2", &fakeBus{err: mq.ErrWaitTimeout}, history)

	err := s.deployPath(context.Background(), s.loadConfig(context.Background()), "apps/web", "sha-2", "manual", nil)

	if err == nil {
		t.Fatal("超时应返回错误")
	}
	if r := latestRecord(t, history); r.Status != "failed" || !strings.Contains(r.ErrorMessage, "超时") {
		t.Errorf("历史 = %+v", r)
	}
}

// 部署历史写不进库：本次算失败（重启后要靠历史恢复提交号）
func TestDeployPath_HistoryWriteFailureIsFailure(t *testing.T) {
	history := &fakeHistory{createErr: errors.New("database is locked")}
	s, _ := newTestService("sha-2", &fakeBus{result: &command.Result{Success: true, Output: `{}`}}, history)
	if err := s.deployPath(context.Background(), s.loadConfig(context.Background()), "apps/web", "sha-2", "auto", nil); err == nil {
		t.Fatal("写历史失败应返回错误")
	}
}

// 重启恢复：取每个路径最近一次「成功」的部署，再取其中最旧的；有路径从没成功过则不恢复（首轮全量部署）
func TestRestoreLastCommitSHA(t *testing.T) {
	t0 := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	rec := func(path, sha, status string, at time.Time) *database.DeployHistory {
		return &database.DeployHistory{ClusterID: "c1", Path: path, CommitSHA: sha, Status: status, DeployedAt: at}
	}
	cases := []struct {
		name    string
		paths   string
		records []*database.DeployHistory
		want    string
	}{
		{"最近一次失败时取再前一次成功", `["apps/web"]`, []*database.DeployHistory{
			rec("apps/web", "sha-2", "success", t0),
			rec("apps/web", "sha-3", "failed", t0.Add(time.Hour)),
		}, "sha-2"},
		{"多个路径取最旧的成功", `["apps/web","apps/api"]`, []*database.DeployHistory{
			rec("apps/web", "sha-3", "success", t0.Add(2 * time.Hour)),
			rec("apps/api", "sha-2", "success", t0.Add(time.Hour)),
		}, "sha-2"},
		{"有路径从没成功过", `["apps/web","apps/api"]`, []*database.DeployHistory{
			rec("apps/web", "sha-3", "success", t0),
			rec("apps/api", "sha-3", "failed", t0),
		}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			history := &fakeHistory{records: c.records}
			s, _ := newTestService("sha-9", &fakeBus{}, history)
			s.db.DeployConfig.(*fakeConfigs).config.Paths = c.paths
			s.restoreLastCommitSHA(context.Background())
			if s.lastCommitSHA != c.want {
				t.Errorf("恢复的提交号 = %q, want %q", s.lastCommitSHA, c.want)
			}
		})
	}
}

func TestPollInterval(t *testing.T) {
	cases := []struct {
		sec  int
		want time.Duration
	}{
		{0, 30 * time.Second},
		{5, 10 * time.Second},
		{60, time.Minute},
	}
	for _, c := range cases {
		if got := pollInterval(&database.DeployConfig{IntervalSec: c.sec}); got != c.want {
			t.Errorf("IntervalSec=%d → %v, want %v", c.sec, got, c.want)
		}
	}
}
