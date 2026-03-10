// atlhyper_master_v2/ai/tool.go
// Tool 执行器 + 黑名单校验
// 将 LLM 的 ToolCall 解析为 Command，通过 MQ 下发并等待结果
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/common/logger"
)

var toolLog = logger.Module("AI-Tool")

// ToolHandler 自定义 Tool 处理函数
type ToolHandler func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error)

// toolExecutor 工具执行器
type toolExecutor struct {
	ops         *operations.CommandService
	bus         mq.Producer
	timeout     time.Duration              // 指令执行超时（默认 30s）
	customTools map[string]ToolHandler      // 自定义 Tool 注册表
}

// newToolExecutor 创建工具执行器
func newToolExecutor(ops *operations.CommandService, bus mq.Producer, timeout time.Duration) *toolExecutor {
	return &toolExecutor{ops: ops, bus: bus, timeout: timeout}
}

// RegisterTool 注册自定义 Tool（开闭原则，不修改 Execute 主逻辑）
func (e *toolExecutor) RegisterTool(name string, handler ToolHandler) {
	if e.customTools == nil {
		e.customTools = make(map[string]ToolHandler)
	}
	e.customTools[name] = handler
	toolLog.Debug("自定义 Tool 已注册", "name", name)
}

// Execute 执行 Tool Call
// 1. 解析参数 → 2. 映射 action → 3. Blacklist 校验 → 4. 创建指令 → 5. 等待结果
func (e *toolExecutor) Execute(ctx context.Context, clusterID string, tc *llm.ToolCall) (string, error) {
	// 0. 先查自定义 Tool
	if handler, ok := e.customTools[tc.Name]; ok {
		var params map[string]interface{}
		if tc.Params != "" {
			if err := json.Unmarshal([]byte(tc.Params), &params); err != nil {
				return "", fmt.Errorf("解析参数 JSON 失败: %w", err)
			}
		}
		return handler(ctx, clusterID, params)
	}

	// 1. 解析参数
	var params map[string]interface{}
	if tc.Params != "" {
		if err := json.Unmarshal([]byte(tc.Params), &params); err != nil {
			return "", fmt.Errorf("解析参数 JSON 失败: %w", err)
		}
	}

	action := getString(params, "action")
	kind := getString(params, "kind")
	namespace := getString(params, "namespace")
	name := getString(params, "name")

	// 2. Blacklist 校验
	if err := BlacklistCheck(action, namespace, kind); err != nil {
		return "", err
	}

	// 3. 映射为内部 action + 构建 Command 参数
	internalAction, cmdParams := mapToInternalAction(action, params)

	req := &model.CreateCommandRequest{
		ClusterID:       clusterID,
		Action:          internalAction,
		TargetKind:      kind,
		TargetNamespace: namespace,
		TargetName:      name,
		Source:          "ai",
		Params:          cmdParams,
	}

	// 4. 创建指令
	resp, err := e.ops.CreateCommand(req)
	if err != nil {
		return "", fmt.Errorf("创建指令失败: %w", err)
	}

	toolLog.Debug("指令已下发", "action", action, "kind", kind, "ns", namespace, "name", name, "cmd", resp.CommandID)

	// 5. 等待结果（支持 ctx 取消）
	result, err := e.bus.WaitCommandResult(ctx, resp.CommandID, e.timeout)
	if err != nil {
		return "", fmt.Errorf("等待指令结果失败: %w", err)
	}
	if result == nil {
		return "", fmt.Errorf("指令执行超时: 未收到 Agent 响应 (cmdID=%s)", resp.CommandID)
	}

	if !result.Success {
		return fmt.Sprintf("执行失败: %s", result.Error), nil
	}
	return result.Output, nil
}

// mapToInternalAction 将 AI 的 action 映射为系统内部 action
// get_logs / get_configmap 有专用 action，其余统一走 dynamic
func mapToInternalAction(action string, params map[string]interface{}) (string, map[string]interface{}) {
	cmdParams := map[string]interface{}{}

	switch action {
	case "get_logs":
		// 直接使用 get_logs action
		cmdParams["container"] = getString(params, "container")
		cmdParams["tail_lines"] = getInt(params, "tail_lines", 100)
		return "get_logs", cmdParams

	case "get_configmap":
		// 直接使用 get_configmap action
		return "get_configmap", cmdParams

	default:
		// get / list / describe / get_events 等统一走 dynamic
		cmdParams["command"] = action
		cmdParams["kind"] = getString(params, "kind")
		// 传递可选过滤参数
		if v := getString(params, "label_selector"); v != "" {
			cmdParams["label_selector"] = v
		}
		if v := getString(params, "involved_kind"); v != "" {
			cmdParams["involved_kind"] = v
		}
		if v := getString(params, "involved_name"); v != "" {
			cmdParams["involved_name"] = v
		}
		return "dynamic", cmdParams
	}
}

// getString 从 map 中安全获取字符串
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getInt 从 map 中安全获取整数（带默认值）
func getInt(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

// ==================== 黑名単校验 ====================

// 禁止的写操作（包含所有可能的写动词）
var forbiddenActions = map[string]bool{
	"create":       true,
	"update":       true,
	"patch":        true,
	"delete":       true,
	"delete_pod":   true,
	"scale":        true,
	"restart":      true,
	"exec":         true,
	"cordon":       true,
	"uncordon":     true,
	"drain":        true,
	"update_image": true,
	"apply":        true,
	"edit":         true,
	"replace":      true,
}

// 禁止的命名空间
var forbiddenNamespaces = map[string]bool{
	"kube-system":     true,
	"kube-public":     true,
	"kube-node-lease": true,
}

// 禁止的资源类型
var forbiddenResources = map[string]bool{
	"Secret": true,
}

// ==================== AI Tool 辅助函数 ====================

// TruncateToolResult 精简 AI Tool 返回数据，避免 LLM 上下文爆炸
func TruncateToolResult(raw string, dataType string) string {
	switch dataType {
	case "traces":
		var traces []json.RawMessage
		if err := json.Unmarshal([]byte(raw), &traces); err != nil {
			return raw
		}
		if len(traces) > 10 {
			traces = traces[:10]
		}
		data, _ := json.Marshal(map[string]interface{}{
			"traces":  traces,
			"showing": len(traces),
		})
		return string(data)

	case "logs":
		var qr struct {
			Logs  []json.RawMessage `json:"logs"`
			Total int64             `json:"total"`
		}
		if err := json.Unmarshal([]byte(raw), &qr); err != nil {
			return raw
		}
		truncated := truncateLogBodies(qr.Logs, 200)
		if len(truncated) > 20 {
			truncated = truncated[:20]
		}
		data, _ := json.Marshal(map[string]interface{}{
			"logs":    truncated,
			"total":   qr.Total,
			"showing": len(truncated),
			"hint":    fmt.Sprintf("共 %d 条日志，显示最近 %d 条", qr.Total, len(truncated)),
		})
		return string(data)
	}
	return raw
}

// truncateLogBodies 截断每条日志的 Body 字段
func truncateLogBodies(logs []json.RawMessage, maxLen int) []json.RawMessage {
	result := make([]json.RawMessage, 0, len(logs))
	for _, raw := range logs {
		var entry map[string]interface{}
		if err := json.Unmarshal(raw, &entry); err != nil {
			result = append(result, raw)
			continue
		}
		if body, ok := entry["body"].(string); ok && len(body) > maxLen {
			entry["body"] = body[:maxLen] + "(truncated)"
		}
		data, _ := json.Marshal(entry)
		result = append(result, data)
	}
	return result
}

// BuildEntityKey 构造 AIOps Engine 内部的 entityKey 格式
// 格式: "namespace/entityType/name"（与 aiops.EntityKey() 一致）
// 示例: "default/pod/api-1", "_cluster/node/worker-3", "default/service/api"
func BuildEntityKey(entityType, namespace, name string) string {
	if namespace == "" {
		switch entityType {
		case "node":
			namespace = "_cluster"
		default:
			namespace = "default"
		}
	}
	return namespace + "/" + entityType + "/" + name
}

// SimplifyEntityDetail 精简 EntityRiskDetail 用于 AI 返回
// 只保留异常指标，减少 LLM 上下文消耗
func SimplifyEntityDetail(detail *aiops.EntityRiskDetail) map[string]interface{} {
	var anomalyMetrics []map[string]interface{}
	for _, m := range detail.Metrics {
		if m.IsAnomaly {
			anomalyMetrics = append(anomalyMetrics, map[string]interface{}{
				"metricName":   m.MetricName,
				"currentValue": m.CurrentValue,
				"baseline":     m.Baseline,
				"deviation":    m.Deviation,
			})
		}
	}

	return map[string]interface{}{
		"entityKey":   detail.EntityKey,
		"entityType":  detail.EntityType,
		"namespace":   detail.Namespace,
		"name":        detail.Name,
		"rFinal":      detail.RFinal,
		"riskLevel":   detail.RiskLevel,
		"metrics":     anomalyMetrics,
		"causalTree":  detail.CausalTree,
		"propagation": detail.Propagation,
	}
}

// BlacklistCheck 黑名単校验
// 返回 nil 表示通过，返回 error 表示被拒绝
func BlacklistCheck(action, namespace, targetKind string) error {
	// 1. 校验 Action
	if forbiddenActions[action] {
		return fmt.Errorf("操作被禁止: %s 为写操作，AI 不允许执行", action)
	}

	// 2. 校验命名空间
	if forbiddenNamespaces[namespace] {
		return fmt.Errorf("命名空间被禁止: %s 为系统命名空間，AI 不允许访问", namespace)
	}

	// 3. 校验资源类型
	if forbiddenResources[targetKind] {
		return fmt.Errorf("资源类型被禁止: %s 为敏感资源，AI 不允许访问", targetKind)
	}

	return nil
}
