package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/truncate"
	"AtlHyper/model_v3/command"
	slomodel "AtlHyper/model_v3/slo"
)

// 观测查询返回给模型的上限（工具说明从这里渲染，⛔ 不在提示词里另写一份）
const (
	maxTraces       = 10
	maxLogs         = 20
	maxLogBodyRunes = 200
	maxSLOServices  = 50
	defaultSince    = "1h"
)

// sincePattern 时间范围：正整数 + m / h（Agent 按 Go 的时长解析）
const sincePattern = `^[1-9][0-9]*(m|h)$`

type queryTracesParams struct {
	Service       string  `json:"service,omitempty" jsonschema:"服务名"`
	Operation     string  `json:"operation,omitempty" jsonschema:"操作名，如 GET /api/v1/users"`
	MinDurationMs float64 `json:"min_duration_ms,omitempty" jsonschema:"最小耗时（毫秒），找慢请求用"`
	StatusCode    string  `json:"status_code,omitempty" jsonschema:"HTTP 状态码，如 500"`
	Since         string  `json:"since,omitempty" jsonschema:"时间范围，如 5m、1h、24h（不填为 1h）"`
}

type queryLogsParams struct {
	Query   string `json:"query,omitempty" jsonschema:"全文搜索关键词（模糊匹配日志正文）"`
	Service string `json:"service,omitempty" jsonschema:"服务名"`
	Level   string `json:"level,omitempty" jsonschema:"日志级别"`
	TraceID string `json:"trace_id,omitempty" jsonschema:"按 TraceId 过滤（与链路关联）"`
	Since   string `json:"since,omitempty" jsonschema:"时间范围，如 15m、1h、24h（不填为 1h）"`
}

type querySLOParams struct {
	Domain string `json:"domain,omitempty" jsonschema:"入口的域名或服务标识（namespace/service），不填返回全部"`
	Window string `json:"window,omitempty" jsonschema:"时间窗口（不填为默认窗口）"`
}

func observeTools(d Deps) []*toolkit.Tool {
	since := func(s *jsonschema.Schema) { s.Properties["since"].Pattern = sincePattern }
	return []*toolkit.Tool{
		toolkit.MustDefine(toolkit.Spec[queryTracesParams]{
			Name: "query_traces",
			Description: fmt.Sprintf("查询 APM 分布式链路：可按服务、操作、耗时、状态码过滤，返回链路摘要（耗时、Span 数、错误信息），最多 %d 条。"+
				"服务之间的调用、慢请求、报错的请求用它查。", maxTraces),
			Permission: toolkit.PermTracesRead,
			Refine:     since,
			Handler: func(ctx context.Context, clusterID string, p queryTracesParams) (string, error) {
				params := map[string]any{"sub_action": "list_traces", "limit": maxTraces, "since": orDefaultString(p.Since, defaultSince)}
				setIf(params, "service", p.Service)
				setIf(params, "operation", p.Operation)
				setIf(params, "status_code", p.StatusCode)
				if p.MinDurationMs > 0 {
					params["min_duration_ms"] = p.MinDurationMs
				}
				out, err := runCommand(ctx, d, &model.CreateCommandRequest{
					ClusterID: clusterID, Action: command.ActionQueryTraces, Source: command.SourceAI, Params: params,
				})
				if err != nil {
					return "", err
				}
				return clipTraces(out), nil
			},
		}),
		toolkit.MustDefine(toolkit.Spec[queryLogsParams]{
			Name: "query_logs",
			Description: fmt.Sprintf("查询 OpenTelemetry 结构化日志：全文搜索，按服务 / 级别 / TraceId 过滤，最多 %d 条，日志正文截到 %d 个字符。",
				maxLogs, maxLogBodyRunes),
			Permission: toolkit.PermLogsRead,
			Refine: func(s *jsonschema.Schema) {
				since(s)
				s.Properties["level"].Enum = []any{"DEBUG", "INFO", "WARN", "ERROR"}
			},
			Handler: func(ctx context.Context, clusterID string, p queryLogsParams) (string, error) {
				params := map[string]any{"limit": maxLogs, "since": orDefaultString(p.Since, defaultSince)}
				setIf(params, "query", p.Query)
				setIf(params, "service", p.Service)
				setIf(params, "level", p.Level)
				setIf(params, "trace_id", p.TraceID)
				out, err := runCommand(ctx, d, &model.CreateCommandRequest{
					ClusterID: clusterID, Action: command.ActionQueryLogs, Source: command.SourceAI, Params: params,
				})
				if err != nil {
					return "", err
				}
				return clipLogs(out), nil
			},
		}),
		toolkit.MustDefine(toolkit.Spec[querySLOParams]{
			Name: "query_slo",
			Description: "查询入口（ingress）的 SLO：可用性、错误率（百分数）、延迟分位数（P50 / P90 / P99）、RPS。" +
				"看对外服务质量时用；服务之间的调用质量用 query_traces。",
			Permission: toolkit.PermSLORead,
			Refine: func(s *jsonschema.Schema) {
				s.Properties["window"].Enum = anySlice(slomodel.Windows)
			},
			Handler: func(_ context.Context, clusterID string, p querySLOParams) (string, error) {
				return querySLO(d, clusterID, p)
			},
		}),
	}
}

func querySLO(d Deps, clusterID string, p querySLOParams) (string, error) {
	window := orDefaultString(p.Window, slomodel.DefaultWindow)
	snapshot, err := d.Snapshots.GetSnapshot(clusterID)
	if err != nil {
		return "", fmt.Errorf("读取快照失败: %w", err)
	}
	if snapshot == nil || snapshot.OTel == nil {
		return "当前没有 OTel 数据", nil
	}
	otel := snapshot.OTel

	// 窗口数据还没算好（如 Agent 刚启动）时用实时数据，并如实标成 realtime
	source := otel.SLOIngress
	label := "realtime（" + window + " 窗口的数据还没有算好）"
	if windowData, ok := otel.SLOWindows[window]; ok && windowData != nil {
		source, label = windowData.Current, window
	}

	services := make([]map[string]any, 0, len(source))
	for i := range source {
		s := &source[i]
		if p.Domain != "" && !s.MatchesHost(p.Domain) {
			continue
		}
		services = append(services, map[string]any{
			"service":     s.ServiceKey,
			"domains":     s.DisplayName,
			"rps":         s.RPS,
			"successRate": s.SuccessRate, // 百分数
			"errorRate":   s.ErrorRate,   // 百分数
			"p50Ms":       s.P50Ms,
			"p90Ms":       s.P90Ms,
			"p99Ms":       s.P99Ms,
		})
		if len(services) == maxSLOServices {
			break
		}
	}
	return marshal(map[string]any{"window": label, "services": services, "count": len(services)})
}

// clipTraces 链路最多留 maxTraces 条
func clipTraces(raw string) string {
	var traces []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &traces); err != nil {
		return raw
	}
	if len(traces) > maxTraces {
		traces = traces[:maxTraces]
	}
	out, err := marshal(map[string]any{"traces": traces, "showing": len(traces)})
	if err != nil {
		return raw
	}
	return out
}

// clipLogs 日志最多留 maxLogs 条，正文按字符截断
func clipLogs(raw string) string {
	var qr struct {
		Logs  []json.RawMessage `json:"logs"`
		Total int64             `json:"total"`
	}
	if err := json.Unmarshal([]byte(raw), &qr); err != nil {
		return raw
	}
	logs := make([]json.RawMessage, 0, min(len(qr.Logs), maxLogs))
	for _, entryRaw := range qr.Logs {
		if len(logs) == maxLogs {
			break
		}
		var entry map[string]any
		if err := json.Unmarshal(entryRaw, &entry); err != nil {
			logs = append(logs, entryRaw)
			continue
		}
		if body, ok := entry["body"].(string); ok {
			entry["body"] = truncate.Runes(body, maxLogBodyRunes, "(truncated)")
		}
		clipped, err := json.Marshal(entry)
		if err != nil {
			logs = append(logs, entryRaw)
			continue
		}
		logs = append(logs, clipped)
	}
	out, err := marshal(map[string]any{
		"logs": logs, "total": qr.Total, "showing": len(logs),
		"hint": fmt.Sprintf("共 %d 条日志，显示最近 %d 条", qr.Total, len(logs)),
	})
	if err != nil {
		return raw
	}
	return out
}

func setIf(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

func orDefaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
