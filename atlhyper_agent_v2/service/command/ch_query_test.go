package command

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/testutil/mock"
	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/command"
	"AtlHyper/model_v3/log"
	"AtlHyper/model_v3/metrics"
)

// newTestService 创建包含 ClickHouse mock 的测试服务
func newTestService(
	traceRepo repository.TraceQueryRepository,
	logRepo repository.LogQueryRepository,
	metricsRepo repository.MetricsQueryRepository,
) *commandService {
	return &commandService{
		podRepo:          &mock.PodRepository{},
		genericRepo:      &mock.GenericRepository{},
		traceQueryRepo:   traceRepo,
		logQueryRepo:     logRepo,
		metricsQueryRepo: metricsRepo,
	}
}

// =============================================================================
// TestExecute_QueryTraces
// =============================================================================

func TestExecute_QueryTraces_ListTraces(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		ListTracesFn: func(ctx context.Context, service, operation string, minDurationMs float64, limit int, since time.Duration, sort string, startTime, endTime string, statusCode, method string) ([]apm.TraceSummary, error) {
			return []apm.TraceSummary{
				{TraceId: "abc123", RootService: "api", SpanCount: 5},
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-1",
		Action: command.ActionQueryTraces,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "list_traces", "service": "api"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "abc123") {
		t.Errorf("expected output to contain trace ID, got: %s", result.Output)
	}
}

func TestExecute_QueryTraces_ListServices(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		ListServicesFn: func(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.APMService, error) {
			return []apm.APMService{{Name: "frontend", RPS: 10.5}}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-2",
		Action: command.ActionQueryTraces,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "list_services"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "frontend") {
		t.Errorf("expected output to contain service name, got: %s", result.Output)
	}
}

func TestExecute_QueryTraces_NilRepo(t *testing.T) {
	svc := newTestService(nil, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-nil",
		Action: command.ActionQueryTraces,
		Source: command.SourceWeb,
		Params: map[string]any{},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure when repo is nil")
	}
	if !strings.Contains(result.Error, "ClickHouse not configured") {
		t.Errorf("expected 'ClickHouse not configured', got: %s", result.Error)
	}
}

// =============================================================================
// TestExecute_QueryTraceDetail
// =============================================================================

func TestExecute_QueryTraceDetail_Success(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		GetTraceDetailFn: func(ctx context.Context, traceID string) (*apm.TraceDetail, error) {
			return &apm.TraceDetail{
				TraceId:    traceID,
				DurationMs: 42.5,
				SpanCount:  3,
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-trace-detail-1",
		Action: command.ActionQueryTraceDetail,
		Source: command.SourceWeb,
		Params: map[string]any{"trace_id": "abc123"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "abc123") {
		t.Errorf("expected output to contain trace ID, got: %s", result.Output)
	}
}

func TestExecute_QueryTraceDetail_MissingID(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{}
	svc := newTestService(traceRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-trace-detail-2",
		Action: command.ActionQueryTraceDetail,
		Source: command.SourceWeb,
		Params: map[string]any{},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing trace_id")
	}
	if !strings.Contains(result.Error, "trace_id") {
		t.Errorf("错误应指出 trace_id，got: %s", result.Error)
	}
}

// =============================================================================
// TestExecute_QueryLogs
// =============================================================================

func TestExecute_QueryLogs_Success(t *testing.T) {
	logRepo := &mock.LogQueryRepository{
		QueryLogsFn: func(ctx context.Context, opts repository.LogQueryOptions) (*log.QueryResult, error) {
			if opts.Service != "api" {
				t.Errorf("expected service 'api', got '%s'", opts.Service)
			}
			if opts.Level != "ERROR" {
				t.Errorf("expected level 'ERROR', got '%s'", opts.Level)
			}
			return &log.QueryResult{
				Logs:  []log.Entry{{Body: "something failed", ServiceName: "api"}},
				Total: 1,
			}, nil
		},
	}

	svc := newTestService(nil, logRepo, nil)
	cmd := &command.Command{
		ID:     "cmd-logs-ch-1",
		Action: command.ActionQueryLogs,
		Source: command.SourceWeb,
		Params: map[string]any{
			"service": "api",
			"level":   "ERROR",
			"limit":   float64(10),
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "something failed") {
		t.Errorf("expected output to contain log body, got: %s", result.Output)
	}
}

// =============================================================================
// TestExecute_QueryMetrics
// =============================================================================

func TestExecute_QueryMetrics_ListAll(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{
		ListAllNodeMetricsFn: func(ctx context.Context) ([]metrics.NodeMetrics, error) {
			return []metrics.NodeMetrics{
				{NodeName: "node-1", CPU: metrics.NodeCPU{UsagePct: 45.2}},
			}, nil
		},
	}

	svc := newTestService(nil, nil, metricsRepo)
	cmd := &command.Command{
		ID:     "cmd-metrics-1",
		Action: command.ActionQueryMetrics,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "list_all"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "node-1") {
		t.Errorf("expected output to contain node name, got: %s", result.Output)
	}
}

func TestExecute_QueryMetrics_GetNode(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{
		GetNodeMetricsFn: func(ctx context.Context, nodeName string) (*metrics.NodeMetrics, error) {
			if nodeName != "worker-1" {
				return nil, fmt.Errorf("unexpected node: %s", nodeName)
			}
			return &metrics.NodeMetrics{NodeName: "worker-1"}, nil
		},
	}

	svc := newTestService(nil, nil, metricsRepo)
	cmd := &command.Command{
		ID:     "cmd-metrics-2",
		Action: command.ActionQueryMetrics,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "get_node", "node_name": "worker-1"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestExecute_QueryMetrics_GetSeries(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{
		GetNodeMetricsSeriesFn: func(ctx context.Context, nodeName string, metric string, since time.Duration) ([]metrics.Point, error) {
			return []metrics.Point{{Value: 42.0}}, nil
		},
	}

	svc := newTestService(nil, nil, metricsRepo)
	cmd := &command.Command{
		ID:     "cmd-metrics-3",
		Action: command.ActionQueryMetrics,
		Source: command.SourceWeb,
		Params: map[string]any{
			"sub_action": "get_series",
			"node_name":  "worker-1",
			"metric":     "node_load1",
			"since":      "30m",
		},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
}

func TestExecute_QueryMetrics_MissingNodeName(t *testing.T) {
	metricsRepo := &mock.MetricsQueryRepository{}
	svc := newTestService(nil, nil, metricsRepo)
	cmd := &command.Command{
		ID:     "cmd-metrics-4",
		Action: command.ActionQueryMetrics,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "get_node"},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing node_name")
	}
}

func TestExecute_QueryTraces_HTTPStats(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		GetHTTPStatsFn: func(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.HTTPStats, error) {
			if service != "api" {
				t.Errorf("expected service 'api', got '%s'", service)
			}
			return []apm.HTTPStats{
				{StatusCode: 200, Method: "GET", Count: 100},
				{StatusCode: 404, Method: "GET", Count: 5},
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-http-stats",
		Action: command.ActionQueryTraces,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "http_stats", "service": "api"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "200") {
		t.Errorf("expected output to contain status code 200, got: %s", result.Output)
	}
}

func TestExecute_QueryTraces_HTTPStats_MissingService(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{}
	svc := newTestService(traceRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-http-stats-2",
		Action: command.ActionQueryTraces,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "http_stats"},
	}

	result := svc.Execute(context.Background(), cmd)

	if result.Success {
		t.Fatal("expected failure for missing service")
	}
	if !strings.Contains(result.Error, "service") {
		t.Errorf("错误应指出 service，got: %s", result.Error)
	}
}

func TestExecute_QueryTraces_DBStats(t *testing.T) {
	traceRepo := &mock.TraceQueryRepository{
		GetDBStatsFn: func(ctx context.Context, service string, since time.Duration, startTime, endTime string) ([]apm.DBOperationStats, error) {
			return []apm.DBOperationStats{
				{DBSystem: "mysql", DBName: "mydb", Operation: "SELECT", CallCount: 50},
			}, nil
		},
	}

	svc := newTestService(traceRepo, nil, nil)
	cmd := &command.Command{
		ID:     "cmd-traces-db-stats",
		Action: command.ActionQueryTraces,
		Source: command.SourceWeb,
		Params: map[string]any{"sub_action": "db_stats", "service": "api"},
	}

	result := svc.Execute(context.Background(), cmd)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Output, "mysql") {
		t.Errorf("expected output to contain 'mysql', got: %s", result.Output)
	}
}

// 参数类型不对、子动作未知、缺必填：执行前就失败并说明原因，⛔ 不按零值或默认值继续查
func TestExecute_QueryParamsRejected(t *testing.T) {
	called := false
	traceRepo := &mock.TraceQueryRepository{
		ListTracesFn: func(context.Context, string, string, float64, int, time.Duration, string, string, string, string, string) ([]apm.TraceSummary, error) {
			called = true
			return nil, nil
		},
	}
	svc := newTestService(traceRepo, &mock.LogQueryRepository{}, &mock.MetricsQueryRepository{})
	cases := []struct {
		name   string
		action string
		params map[string]any
		want   string
	}{
		{"limit 是字符串", command.ActionQueryTraces, map[string]any{"limit": "50"}, "参数格式不对"},
		{"未知链路子动作", command.ActionQueryTraces, map[string]any{"sub_action": "drop"}, "未知的链路查询"},
		{"since 不是时长", command.ActionQueryTraces, map[string]any{"since": "15"}, "since"},
		{"日志 limit 超上限", command.ActionQueryLogs, map[string]any{"limit": 100000}, "limit"},
		{"指标缺节点名", command.ActionQueryMetrics, map[string]any{"sub_action": "get_history"}, "node_name"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := svc.Execute(context.Background(), &command.Command{ID: "x", Action: c.action, Source: command.SourceWeb, Params: c.params})
			if result.Success || !strings.Contains(result.Error, c.want) {
				t.Errorf("应失败且提到 %q，实际 success=%v error=%q", c.want, result.Success, result.Error)
			}
		})
	}
	if called {
		t.Error("参数不合法时不应调用查询")
	}
}

// =============================================================================
// 失败类别：Master 按 Result.Code 决定状态码与界面提示，⛔ 不比较 Error 文字
// =============================================================================

func TestExecute_QueryFailureCodes(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "clickhouse.example.internal", IsNotFound: true}
	cases := []struct {
		name     string
		repo     repository.TraceQueryRepository
		params   map[string]any
		wantCode string
	}{
		{"未接入 ClickHouse", nil, map[string]any{"sub_action": "list_services"}, command.CodeNotConfigured},
		{"参数不合法", &mock.TraceQueryRepository{}, map[string]any{"sub_action": "no_such_action"}, command.CodeInvalidParams},
		{"上游拒绝连接", traceRepoFailing(fmt.Errorf("dial: %w", syscall.ECONNREFUSED)), map[string]any{"sub_action": "list_services"}, command.CodeUnavailable},
		{"上游域名解析失败", traceRepoFailing(fmt.Errorf("dial: %w", dnsErr)), map[string]any{"sub_action": "list_services"}, command.CodeUnavailable},
		{"执行超时", traceRepoFailing(fmt.Errorf("query: %w", context.DeadlineExceeded)), map[string]any{"sub_action": "list_services"}, command.CodeTimeout},
		{"其他失败", traceRepoFailing(errors.New("code: 60, table does not exist")), map[string]any{"sub_action": "list_services"}, command.CodeFailed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newTestService(c.repo, nil, nil)
			result := svc.Execute(context.Background(), &command.Command{
				ID: "cmd-code", Action: command.ActionQueryTraces, Source: command.SourceWeb, Params: c.params,
			})
			if result.Success {
				t.Fatal("应失败")
			}
			if result.Code != c.wantCode {
				t.Errorf("Code = %q, want %q（Error: %s）", result.Code, c.wantCode, result.Error)
			}
		})
	}
}

// 执行前的守门（来源、集群、目标）拒绝时同样带类别
func TestExecute_RejectedHasCode(t *testing.T) {
	svc := newTestService(&mock.TraceQueryRepository{}, nil, nil)
	result := svc.Execute(context.Background(), &command.Command{
		ID: "cmd-rejected", Action: command.ActionQueryTraces, Source: "no-such-source", Params: map[string]any{"sub_action": "list_services"},
	})
	if result.Success || result.Code != command.CodeRejected {
		t.Fatalf("= (success %v, code %q), want rejected", result.Success, result.Code)
	}
}

func traceRepoFailing(err error) *mock.TraceQueryRepository {
	return &mock.TraceQueryRepository{
		ListServicesFn: func(ctx context.Context, since time.Duration, startTime, endTime string) ([]apm.APMService, error) {
			return nil, err
		},
	}
}
