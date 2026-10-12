package agentsdk

import (
	"context"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/command"
)

// fakeCmdRepo 只实现 persistResult 用到的两个方法，其余方法未实现（调用即 panic）。
type fakeCmdRepo struct {
	database.CommandHistoryRepository
	row *database.CommandHistory
}

func (f *fakeCmdRepo) GetByCommandID(_ context.Context, _ string) (*database.CommandHistory, error) {
	return f.row, nil
}

func (f *fakeCmdRepo) Update(_ context.Context, cmd *database.CommandHistory) error {
	f.row = cmd
	return nil
}

// 指令历史只保存执行结论，Output（读类指令的原始内容）不落库。
func TestPersistResult_NeverStoresOutput(t *testing.T) {
	repo := &fakeCmdRepo{row: &database.CommandHistory{
		CommandID: "c1", Action: command.ActionGetSecret, CreatedAt: time.Now(),
	}}
	s := &Server{cmdRepo: repo}

	s.persistResult("c1", &command.Result{CommandID: "c1", Success: true, Output: `{"password":"hunter2"}`})

	if strings.Contains(repo.row.Result, "hunter2") || strings.Contains(repo.row.Result, `"output"`) {
		t.Errorf("落库结果不得含 output，得到 %s", repo.row.Result)
	}
	if repo.row.Status != command.StatusSuccess {
		t.Errorf("status = %q，期望 success", repo.row.Status)
	}
}
