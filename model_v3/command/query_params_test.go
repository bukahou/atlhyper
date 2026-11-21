package command

import (
	"reflect"
	"testing"
	"time"
)

// Master 生成、Agent 解析：同一份结构体往返不丢字段（键名只在结构体标签里）
func TestQueryParams_RoundTrip(t *testing.T) {
	in := QueryTracesParams{
		SubAction: TraceHTTPStats, Service: "geass-gateway", Operation: "GET /x", MinDurationMs: 12.5,
		Limit: 100, Sort: TraceSortDurationDesc, StatusCode: "500", Method: "GET", Mode: "latency",
		TimeWindow: TimeWindow{StartTime: "2026-01-02T03:04:05Z", EndTime: "2026-01-02T04:04:05Z"},
	}
	var out QueryTracesParams
	if err := ParseParams(in.ToParams(), &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("往返后不一致:\n in %+v\nout %+v", in, out)
	}
	// 线上键名不变（与改造前的 map 写法一致）
	params := in.ToParams()
	for _, key := range []string{"sub_action", "service", "operation", "min_duration_ms", "limit", "sort", "status_code", "method", "mode", "start_time", "end_time"} {
		if _, ok := params[key]; !ok {
			t.Errorf("缺键 %s: %v", key, params)
		}
	}
}

// 类型不对时解析失败，⛔ 不按零值继续
func TestParseParams_WrongTypeFails(t *testing.T) {
	var p QueryLogsParams
	if err := ParseParams(map[string]any{"limit": "50"}, &p); err == nil {
		t.Fatal("limit 给了字符串应解析失败")
	}
}

func TestQueryTracesParams_Validate(t *testing.T) {
	cases := []struct {
		name string
		p    QueryTracesParams
		ok   bool
	}{
		{"默认子动作", QueryTracesParams{}, true},
		{"未知子动作", QueryTracesParams{SubAction: "drop_table"}, false},
		{"http_stats 缺 service", QueryTracesParams{SubAction: TraceHTTPStats}, false},
		{"service_series 有 service", QueryTracesParams{SubAction: TraceServiceSeries, Service: "a"}, true},
		{"limit 超上限", QueryTracesParams{Limit: MaxTraceLimit + 1}, false},
		{"limit 为负", QueryTracesParams{Limit: -1}, false},
		{"未知 mode", QueryTracesParams{SubAction: TraceCorrelations, Mode: "x"}, false},
		{"未知 sort", QueryTracesParams{Sort: "random"}, false},
		{"since 不是时长", QueryTracesParams{TimeWindow: TimeWindow{Since: "15"}}, false},
		{"since 为负", QueryTracesParams{TimeWindow: TimeWindow{Since: "-5m"}}, false},
		{"只给 start_time", QueryTracesParams{TimeWindow: TimeWindow{StartTime: "2026-01-02T03:04:05Z"}}, false},
		{"start 晚于 end", QueryTracesParams{TimeWindow: TimeWindow{StartTime: "2026-01-02T05:00:00Z", EndTime: "2026-01-02T04:00:00Z"}}, false},
		{"时间格式错", QueryTracesParams{TimeWindow: TimeWindow{StartTime: "yesterday", EndTime: "today"}}, false},
	}
	for _, c := range cases {
		if err := c.p.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: Validate() = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestQueryLogsParams_Validate(t *testing.T) {
	cases := []struct {
		name string
		p    QueryLogsParams
		ok   bool
	}{
		{"列表", QueryLogsParams{Limit: 50, TimeWindow: TimeWindow{Since: "1h"}}, true},
		{"直方图", QueryLogsParams{SubAction: LogsHistogram}, true},
		{"未知子动作", QueryLogsParams{SubAction: "tail"}, false},
		{"limit 超上限", QueryLogsParams{Limit: MaxLogLimit + 1}, false},
		{"offset 为负", QueryLogsParams{Offset: -1}, false},
	}
	for _, c := range cases {
		if err := c.p.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: Validate() = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestQueryMetricsParams_Validate(t *testing.T) {
	cases := []struct {
		name string
		p    QueryMetricsParams
		ok   bool
	}{
		{"默认汇总", QueryMetricsParams{}, true},
		{"get_node 缺节点", QueryMetricsParams{SubAction: MetricsGetNode}, false},
		{"get_series 缺指标", QueryMetricsParams{SubAction: MetricsGetSeries, NodeName: "n1"}, false},
		{"get_history", QueryMetricsParams{SubAction: MetricsGetHistory, NodeName: "n1", Since: "24h"}, true},
		{"未知子动作", QueryMetricsParams{SubAction: "delete"}, false},
	}
	for _, c := range cases {
		if err := c.p.Validate(); (err == nil) != c.ok {
			t.Errorf("%s: Validate() = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestTimeWindow_SinceOr(t *testing.T) {
	if got := (TimeWindow{}).SinceOr(15 * time.Minute); got != 15*time.Minute {
		t.Errorf("没给 since 时取默认，得到 %v", got)
	}
	if got := (TimeWindow{Since: "2h"}).SinceOr(15 * time.Minute); got != 2*time.Hour {
		t.Errorf("since=2h 得到 %v", got)
	}
}

func TestQueryTraceDetailParams_Validate(t *testing.T) {
	if (QueryTraceDetailParams{}).Validate() == nil {
		t.Error("trace_id 为空应拒绝")
	}
	if (QueryTraceDetailParams{TraceID: "4bf92f3577b34da6a3ce929d0e0e4736"}).Validate() != nil {
		t.Error("合法 trace_id 被拒")
	}
}
