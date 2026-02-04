// atlhyper_master_v2/ai/tool.go
// Tool 执行器
// 将 LLM 的 ToolCall 解析为 Command，通过 MQ 下发并等待结果
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/common/logger"
)

var toolLog = logger.Module("AI-Tool")

// toolExecutor 工具执行器
type toolExecutor struct {
	ops     *operations.CommandService
	bus     mq.Producer
	timeout time.Duration // 指令执行超时（默认 30s）
}

// newToolExecutor 创建工具执行器
func newToolExecutor(ops *operations.CommandService, bus mq.Producer, timeout time.Duration) *toolExecutor {
	return &toolExecutor{ops: ops, bus: bus, timeout: timeout}
}

// Execute 执行 Tool Call
// 1. 解析参数 → 2. 映射 action → 3. Blacklist 校验 → 4. 创建指令 → 5. 等待结果
func (e *toolExecutor) Execute(ctx context.Context, clusterID string, tc *llm.ToolCall) (string, error) {
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

	req := &operations.CreateCommandRequest{
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
