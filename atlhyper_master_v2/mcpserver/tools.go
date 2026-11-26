package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/truncate"
)

// exposedTools 经 MCP 开放的工具：允许名单，只此一份。
// 只放只读工具（启动时核对）；会调大模型、写报告的 analyze_incident 与读代码的 github_* 不在其中。
var exposedTools = []string{
	"list_clusters",
	"get_cluster_overview",
	"get_cluster_risk",
	"get_recent_incidents",
	"get_entity_detail",
	"query_slo",
	"get_event_history",
	"query_cluster",
	"query_traces",
	"query_logs",
}

// clusterArgument 针对集群的工具在 MCP 上多出的参数名（工具自己的参数里不能有它）
const clusterArgument = "cluster_id"

// 工具结果元数据（_meta）里的键
const (
	MetaErrorCode = "atlhyper.io/errorCode" // 失败类别，取值见下
	MetaTruncated = "atlhyper.io/truncated" // 结果超过上限被截断时为 true
)

// 工具调用失败的类别（调用方按它判断，⛔ 不看说明文字）
const (
	CodePermissionDenied = "permission_denied"
	CodeInvalidParams    = "invalid_params"
	CodeClusterRequired  = "cluster_required"  // 有多个集群，必须用 cluster_id 指定
	CodeClusterNotFound  = "cluster_not_found" // 指定的集群没有在上报
	CodeNoCluster        = "no_cluster"        // 当前没有任何集群在上报
	CodeTimeout          = "timeout"
	CodeUnavailable      = "unavailable" // 依赖暂时不可用（集群离线、列不出集群）
	CodeFailed           = "failed"
)

// maxErrorRunes 交给调用方的失败说明的长度上限
const maxErrorRunes = 500

// callError 入口自己判定的失败（带类别）
type callError struct {
	code    string
	message string
}

func (e *callError) Error() string { return e.message }

// exposedTool 一个对外开放的工具及其 MCP 定义
type exposedTool struct {
	tool   *toolkit.Tool
	schema json.RawMessage
}

// resolveExposedTools 把开放名单解析成工具：名单里的工具必须已注册、必须只读、参数里不能自带 cluster_id
func resolveExposedTools(registry *toolkit.Registry, names []string) ([]exposedTool, error) {
	tools := make([]exposedTool, 0, len(names))
	for _, name := range names {
		tool, ok := registry.Lookup(name)
		if !ok {
			return nil, fmt.Errorf("开放名单里的工具 %s 没有注册", name)
		}
		if tool.Effect() != toolkit.EffectRead {
			return nil, fmt.Errorf("工具 %s 不是只读（%s），不能经 MCP 开放", name, tool.Effect())
		}
		schema, err := inputSchemaWithCluster(tool)
		if err != nil {
			return nil, err
		}
		tools = append(tools, exposedTool{tool: tool, schema: schema})
	}
	return tools, nil
}

// inputSchemaWithCluster 工具自己的参数 schema；针对集群的工具再加一个可选的 cluster_id
func inputSchemaWithCluster(tool *toolkit.Tool) (json.RawMessage, error) {
	if !tool.NeedsCluster() {
		return tool.InputSchema(), nil
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema(), &schema); err != nil {
		return nil, fmt.Errorf("工具 %s 的参数 schema 无法解析: %w", tool.Name(), err)
	}
	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		properties = map[string]any{}
		schema["properties"] = properties
	}
	if _, clash := properties[clusterArgument]; clash {
		return nil, fmt.Errorf("工具 %s 自己的参数里已有 %s", tool.Name(), clusterArgument)
	}
	properties[clusterArgument] = map[string]any{
		"type":        "string",
		"description": "集群 ID。只有一个集群在上报时可以不填；有多个时必须指定（用 list_clusters 查看）",
	}
	return json.Marshal(schema)
}

// definition 工具的 MCP 定义（每个服务端实例各拿一份，登记后不再修改）
func (t exposedTool) definition() *mcp.Tool {
	notDestructive, closedWorld := false, false
	return &mcp.Tool{
		Name:        t.tool.Name(),
		Description: t.tool.Description(),
		InputSchema: t.schema,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			DestructiveHint: &notDestructive,
			IdempotentHint:  true,
			OpenWorldHint:   &closedWorld, // 只查本平台管理的集群，不接触外部世界
		},
	}
}

// splitClusterArgument 把 cluster_id 从调用参数里拆出来，其余原样交给工具（工具按自己的 schema 校验）
func splitClusterArgument(raw json.RawMessage, needsCluster bool) (toolArgs []byte, clusterID string, err error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = json.RawMessage("{}")
	}
	if !needsCluster {
		return raw, "", nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, "", &callError{code: CodeInvalidParams, message: "参数必须是一个 JSON 对象"}
	}
	if value, given := fields[clusterArgument]; given {
		if err := json.Unmarshal(value, &clusterID); err != nil {
			return nil, "", &callError{code: CodeInvalidParams, message: clusterArgument + " 必须是字符串"}
		}
		delete(fields, clusterArgument)
	}
	toolArgs, err = json.Marshal(fields)
	if err != nil {
		return nil, "", err
	}
	return toolArgs, clusterID, nil
}

// resolveCluster 确定这次调用针对哪个集群：给了就核对它在上报；没给时只有一个集群才用它，⛔ 不猜
func (s *Server) resolveCluster(ctx context.Context, requested string) (string, error) {
	clusters, err := s.cfg.Clusters.ListClusters(ctx)
	if err != nil {
		log.Error("列出集群失败", "err", err)
		return "", &callError{code: CodeUnavailable, message: "暂时无法取得集群列表，请稍后重试"}
	}
	ids := make([]string, 0, len(clusters))
	for _, c := range clusters {
		ids = append(ids, c.ClusterID)
	}
	slices.Sort(ids)

	if requested != "" {
		if slices.Contains(ids, requested) {
			return requested, nil
		}
		return "", &callError{code: CodeClusterNotFound,
			message: fmt.Sprintf("集群 %q 没有在上报；正在上报的集群：%s", requested, listOrNone(ids))}
	}
	switch len(ids) {
	case 0:
		return "", &callError{code: CodeNoCluster, message: "当前没有集群在上报"}
	case 1:
		return ids[0], nil
	default:
		return "", &callError{code: CodeClusterRequired,
			message: fmt.Sprintf("有多个集群在上报，请用 %s 指定其中一个：%s", clusterArgument, strings.Join(ids, "、"))}
	}
}

func listOrNone(ids []string) string {
	if len(ids) == 0 {
		return "（无）"
	}
	return strings.Join(ids, "、")
}

// classifyFailure 失败 → 类别与交给调用方的说明。按错误的类型判断，⛔ 不看文字
func classifyFailure(err error) (code, message string) {
	var entryErr *callError
	switch {
	case errors.As(err, &entryErr):
		return entryErr.code, entryErr.message
	case errors.Is(err, toolkit.ErrPermissionDenied):
		return CodePermissionDenied, err.Error()
	case errors.Is(err, toolkit.ErrInvalidParams):
		return CodeInvalidParams, err.Error()
	case errors.Is(err, context.DeadlineExceeded):
		return CodeTimeout, "工具执行超时，请缩小查询范围后重试"
	case errors.Is(err, model.ErrClusterOffline):
		return CodeUnavailable, "集群不在线：没有可接收指令的 Agent"
	}
	return CodeFailed, truncate.Runes(err.Error(), maxErrorRunes, "…")
}

// successResult 工具结果：JSON 文本；超过上限时截断并在元数据里标出
func (s *Server) successResult(output string) *mcp.CallToolResult {
	result := &mcp.CallToolResult{}
	if len([]rune(output)) > s.cfg.MaxResultRunes {
		output = truncate.Runes(output, s.cfg.MaxResultRunes, "…(已截断)")
		result.Meta = mcp.Meta{MetaTruncated: true}
	}
	result.Content = []mcp.Content{&mcp.TextContent{Text: output}}
	return result
}

func failureResult(code, message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		Meta:    mcp.Meta{MetaErrorCode: code},
	}
}
