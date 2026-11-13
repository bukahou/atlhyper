package agentsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/command"
)

// fakeBus 按顺序吐出预置的指令，吐完后返回 nil（相当于长轮询超时）；记录收到的确认
type fakeBus struct {
	queue []*command.Command
	acked map[string]*command.Result
}

func (b *fakeBus) WaitCommand(context.Context, string, string, time.Duration) (*command.Command, error) {
	if len(b.queue) == 0 {
		return nil, nil
	}
	cmd := b.queue[0]
	b.queue = b.queue[1:]
	return cmd, nil
}

func (b *fakeBus) AckCommand(cmdID string, result *command.Result) error {
	if b.acked == nil {
		b.acked = map[string]*command.Result{}
	}
	b.acked[cmdID] = result
	return nil
}

func pollCommands(t *testing.T, s *Server) CommandResponse {
	t.Helper()
	rec := httptest.NewRecorder()
	s.handleCommands(rec, httptest.NewRequest(http.MethodGet, "/agent/commands?cluster_id=c1&topic=ops", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp CommandResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

// Agent 离线期间积压的指令，恢复连接后不得再下发执行：过期的直接判失败，下发后面仍有效的。
func TestHandleCommands_ExpiredCommandIsNotDelivered(t *testing.T) {
	now := time.Now()
	stale := &command.Command{ID: "stale", Action: command.ActionRestart, CreatedAt: now.Add(-3 * time.Hour)}
	fresh := &command.Command{ID: "fresh", Action: command.ActionRestart, CreatedAt: now.Add(-2 * time.Second)}
	bus := &fakeBus{queue: []*command.Command{stale, fresh}}
	repo := &fakeCmdRepo{row: &database.CommandHistory{CommandID: "stale", Action: command.ActionRestart, CreatedAt: stale.CreatedAt}}
	s := &Server{bus: bus, cmdRepo: repo, timeout: time.Second}

	resp := pollCommands(t, s)

	if !resp.HasCommand || resp.Command == nil || resp.Command.ID != "fresh" {
		t.Fatalf("应下发仍有效的指令 fresh，实际 %+v", resp.Command)
	}
	result, acked := bus.acked["stale"]
	if !acked || result.Success {
		t.Fatalf("过期指令应被确认为失败，实际 %+v", result)
	}
	if repo.row.Status != command.StatusTimeout {
		t.Errorf("过期指令的历史状态 = %q, want %q", repo.row.Status, command.StatusTimeout)
	}
	if repo.row.ErrorMessage == "" {
		t.Error("过期指令的历史应写明原因")
	}
}

func TestHandleCommands_OnlyExpiredCommands(t *testing.T) {
	now := time.Now()
	bus := &fakeBus{queue: []*command.Command{
		{ID: "a", Action: command.ActionScale, CreatedAt: now.Add(-10 * time.Minute)},
		{ID: "b", Action: command.ActionScale, CreatedAt: now.Add(-9 * time.Minute)},
	}}
	s := &Server{bus: bus, timeout: time.Second}

	if resp := pollCommands(t, s); resp.HasCommand {
		t.Fatalf("全部过期时不应下发任何指令，实际 %+v", resp.Command)
	}
	if len(bus.acked) != 2 {
		t.Errorf("两条过期指令都应被确认，实际 %d 条", len(bus.acked))
	}
}

func TestHandleCommands_FreshCommandIsDelivered(t *testing.T) {
	bus := &fakeBus{queue: []*command.Command{{ID: "c", Action: command.ActionScale, CreatedAt: time.Now()}}}
	s := &Server{bus: bus, timeout: time.Second}

	resp := pollCommands(t, s)
	if !resp.HasCommand || resp.Command.ID != "c" {
		t.Fatalf("有效指令应正常下发，实际 %+v", resp.Command)
	}
	if len(bus.acked) != 0 {
		t.Errorf("有效指令不应被提前确认")
	}
}
