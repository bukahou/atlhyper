package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/model_v3/command"
)

// stubCommandSvc 只实现指令查询用到的三个方法。
type stubCommandSvc struct {
	service.Service
	status  *command.Status
	history []*database.CommandHistory
}

func (s *stubCommandSvc) GetCommandStatus(_ context.Context, _ string) (*command.Status, error) {
	return s.status, nil
}

func (s *stubCommandSvc) ListCommandHistory(_ context.Context, _ database.CommandQueryOpts) ([]*database.CommandHistory, error) {
	return s.history, nil
}

func (s *stubCommandSvc) CountCommandHistory(_ context.Context, _ database.CommandQueryOpts) (int64, error) {
	return int64(len(s.history)), nil
}

const secretOutput = `{"password":"hunter2"}`

// bus 内存里的结果仍带 Output（同步等待方要用），状态接口返回前必须剥掉，且不得改动 bus 里的原值。
func TestGetStatus_StripsOutput(t *testing.T) {
	inBus := &command.Result{CommandID: "c1", Success: true, Output: secretOutput}
	h := NewCommandHandler(&stubCommandSvc{status: &command.Status{CommandID: "c1", Status: "success", Result: inBus}})

	rec := httptest.NewRecorder()
	h.GetStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v2/commands/c1", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("响应不得含 output，得到 %s", rec.Body.String())
	}
	if inBus.Output != secretOutput {
		t.Error("不得修改 bus 内存里的原始结果")
	}
}

// 早期版本写入的行可能含 output：列表接口返回前统一剥离。
func TestListHistory_RedactsLegacyOutput(t *testing.T) {
	legacy, _ := json.Marshal(command.Result{CommandID: "c1", Success: true, Output: secretOutput})
	h := NewCommandHandler(&stubCommandSvc{history: []*database.CommandHistory{
		{CommandID: "c1", Action: command.ActionGetSecret, Result: string(legacy)},
	}})

	rec := httptest.NewRecorder()
	h.ListHistory(rec, httptest.NewRequest(http.MethodGet, "/api/v2/commands/history", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("响应不得含旧行的 output，得到 %s", rec.Body.String())
	}
}
