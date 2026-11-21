package observe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/model_v3/command"
)

// blockingOps 记录调用次数；release 关闭前一直阻塞
type blockingOps struct {
	calls   atomic.Int32
	release chan struct{}
	result  *command.Result
	err     error
}

func (b *blockingOps) ExecuteCommandSync(ctx context.Context, _ *model.CreateCommandRequest, _ time.Duration) (*command.Result, error) {
	b.calls.Add(1)
	if b.release != nil {
		<-b.release
	}
	return b.result, b.err
}

// 同时到达的相同查询只发一条指令
func TestCommandQuery_ConcurrentIdenticalQueriesShareOneCommand(t *testing.T) {
	ops := &blockingOps{release: make(chan struct{}), result: &command.Result{Success: true, Output: `{"ok":true}`}}
	q := newCommandQuery(ops)
	params := command.QueryTracesParams{SubAction: command.TraceListServices}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := q.run(context.Background(), "c1", command.ActionQueryTraces, params, time.Minute); err != nil {
				t.Error(err)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond) // 让五个请求都进到等待里
	close(ops.release)
	wg.Wait()

	if n := ops.calls.Load(); n != 1 {
		t.Errorf("发出了 %d 条指令，want 1", n)
	}
	// 缓存期内再查不发指令
	if _, err := q.run(context.Background(), "c1", command.ActionQueryTraces, params, time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := ops.calls.Load(); n != 1 {
		t.Errorf("缓存期内又发了指令（共 %d 条）", n)
	}
}

// 某一个请求断开不影响共享这次查询的其他请求
func TestCommandQuery_CallerCancelDoesNotCancelSharedQuery(t *testing.T) {
	var seen context.Context
	ops := &ctxRecordingOps{result: &command.Result{Success: true, Output: `{}`}, onCall: func(ctx context.Context) { seen = ctx }}
	q := newCommandQuery(ops)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := q.run(ctx, "c1", command.ActionQueryLogs, command.QueryLogsParams{}, 0); err != nil {
		t.Fatal(err)
	}
	if seen.Err() != nil {
		t.Error("下发给指令通道的 context 不应随发起请求取消")
	}
}

func TestCommandQuery_Errors(t *testing.T) {
	agentFail := newCommandQuery(&blockingOps{result: &command.Result{Success: false, Error: "clickhouse: timeout"}})
	_, err := agentFail.run(context.Background(), "c1", command.ActionQueryLogs, command.QueryLogsParams{}, time.Minute)
	rec := httptest.NewRecorder()
	writeQueryError(rec, err)
	if rec.Code != http.StatusBadGateway {
		t.Errorf("Agent 查询失败 → %d, want 502", rec.Code)
	}

	noResult := newCommandQuery(&blockingOps{err: mq.ErrWaitTimeout})
	_, err = noResult.run(context.Background(), "c1", command.ActionQueryLogs, command.QueryLogsParams{}, time.Minute)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("等不到结果应是超时错误，实际 %v", err)
	}
	rec = httptest.NewRecorder()
	writeQueryError(rec, err)
	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("超时 → %d, want 504", rec.Code)
	}

	// 失败不进缓存：下次还会真的去查
	ops := &blockingOps{result: &command.Result{Success: false, Error: "x"}}
	q := newCommandQuery(ops)
	_, _ = q.run(context.Background(), "c1", command.ActionQueryLogs, command.QueryLogsParams{}, time.Minute)
	_, _ = q.run(context.Background(), "c1", command.ActionQueryLogs, command.QueryLogsParams{}, time.Minute)
	if ops.calls.Load() != 2 {
		t.Errorf("失败结果不应被缓存（调用 %d 次）", ops.calls.Load())
	}
}

// Agent 查询失败：状态码按 Result.Code 定；响应只给错误码与通用说明，⛔ 不带 Agent 的原始错误文字
func TestWriteQueryError_AgentFailureByCode(t *testing.T) {
	const raw = "dial tcp 10.0.0.9:9000: connect: connection refused; SELECT * FROM otel_logs"
	cases := []struct {
		code       string
		wantStatus int
		wantCode   string
	}{
		{command.CodeNotConfigured, http.StatusConflict, command.CodeNotConfigured},
		{command.CodeInvalidParams, http.StatusBadRequest, command.CodeInvalidParams},
		{command.CodeTimeout, http.StatusGatewayTimeout, command.CodeTimeout},
		{command.CodeUnavailable, http.StatusServiceUnavailable, command.CodeUnavailable},
		{command.CodeRejected, http.StatusBadGateway, command.CodeRejected},
		{command.CodeFailed, http.StatusBadGateway, command.CodeFailed},
		{"", http.StatusBadGateway, command.CodeFailed},             // 不带类别的旧版 Agent
		{"no_such_code", http.StatusBadGateway, command.CodeFailed}, // 不认识的类别
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			q := newCommandQuery(&blockingOps{result: &command.Result{Success: false, Code: c.code, Error: raw}})
			_, err := q.run(context.Background(), "c1", command.ActionQueryLogs, command.QueryLogsParams{}, time.Minute)
			rec := httptest.NewRecorder()
			writeQueryError(rec, err)
			if rec.Code != c.wantStatus {
				t.Errorf("状态码 %d, want %d", rec.Code, c.wantStatus)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("响应不是 JSON: %v", err)
			}
			if body["code"] != c.wantCode {
				t.Errorf("code = %q, want %q", body["code"], c.wantCode)
			}
			if body["error"] == "" {
				t.Error("应带一句通用说明")
			}
			if strings.Contains(rec.Body.String(), "10.0.0.9") || strings.Contains(rec.Body.String(), "otel_logs") {
				t.Errorf("响应里出现了 Agent 的原始错误: %s", rec.Body.String())
			}
		})
	}
}

type ctxRecordingOps struct {
	result *command.Result
	onCall func(context.Context)
}

func (c *ctxRecordingOps) ExecuteCommandSync(ctx context.Context, _ *model.CreateCommandRequest, _ time.Duration) (*command.Result, error) {
	c.onCall(ctx)
	return c.result, nil
}

// 不合法的查询在下发前就拒绝：返回 400，不发指令
func TestCommandQuery_InvalidParamsRejectedBeforeDispatch(t *testing.T) {
	ops := &blockingOps{result: &command.Result{Success: true, Output: `{}`}}
	q := newCommandQuery(ops)
	_, err := q.run(context.Background(), "c1", command.ActionQueryTraces, command.QueryTracesParams{SubAction: command.TraceHTTPStats}, 0)
	if err == nil {
		t.Fatal("缺 service 的 http_stats 应被拒绝")
	}
	if n := ops.calls.Load(); n != 0 {
		t.Errorf("不合法的查询不应下发，实际发了 %d 条", n)
	}
	rec := httptest.NewRecorder()
	writeQueryError(rec, err)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
