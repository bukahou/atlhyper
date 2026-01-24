// atlhyper_master_v2/ai/tool.go
// Tool 执行器
// 将 LLM 的 ToolCall 映射为 Command，通过 MQ 下发并等待结果
package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service/operations"
)

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
// 1. 解析参数 → 2. Blacklist 校验 → 3. 创建指令 → 4. 等待结果
func (e *toolExecutor) Execute(clusterID string, tc *llm.ToolCall) (string, error) {
	// 1. 映射 Tool → Command
	req, err := e.mapToolToCommand(clusterID, tc)
	if err != nil {
		return "", fmt.Errorf("映射 Tool 失败: %w", err)
	}

	// 2. Blacklist 校验
	if err := BlacklistCheck(req.Action, req.TargetNamespace, req.TargetKind); err != nil {
		return "", err
	}

	// 3. 创建指令
	resp, err := e.ops.CreateCommand(req)
	if err != nil {
		return "", fmt.Errorf("创建指令失败: %w", err)
	}

	log.Printf("[AI-Tool] 指令已下发: tool=%s, cmdID=%s", tc.Name, resp.CommandID)

	// 4. 等待结果
	result, err := e.bus.WaitCommandResult(resp.CommandID, e.timeout)
	if err != nil {
		return "", fmt.Errorf("等待指令结果超时: %w", err)
	}

	if !result.Success {
		return fmt.Sprintf("执行失败: %s", result.Error), nil
	}
	return result.Output, nil
}

// mapToolToCommand 将 ToolCall 映射为 CreateCommandRequest
func (e *toolExecutor) mapToolToCommand(clusterID string, tc *llm.ToolCall) (*operations.CreateCommandRequest, error) {
	var params map[string]interface{}
	if tc.Params != "" {
		if err := json.Unmarshal([]byte(tc.Params), &params); err != nil {
			return nil, fmt.Errorf("解析参数 JSON 失败: %w", err)
		}
	}

	req := &operations.CreateCommandRequest{
		ClusterID: clusterID,
		Source:    "ai",
	}

	switch tc.Name {
	case "get_pod_logs":
		req.Action = "get_logs"
		req.TargetKind = "Pod"
		req.TargetNamespace = getString(params, "namespace")
		req.TargetName = getString(params, "pod_name")
		req.Params = map[string]interface{}{
			"container":  getString(params, "container"),
			"tail_lines": getInt(params, "tail_lines", 100),
		}

	case "get_pod_describe":
		req.Action = "dynamic"
		req.TargetKind = "Pod"
		req.TargetNamespace = getString(params, "namespace")
		req.TargetName = getString(params, "pod_name")
		req.Params = map[string]interface{}{
			"command": "describe",
			"kind":    "pod",
		}

	case "get_deployment_status":
		req.Action = "dynamic"
		req.TargetKind = "Deployment"
		req.TargetNamespace = getString(params, "namespace")
		req.TargetName = getString(params, "deployment_name")
		req.Params = map[string]interface{}{
			"command": "describe",
			"kind":    "deployment",
		}

	case "get_events":
		req.Action = "dynamic"
		req.TargetKind = "Event"
		req.TargetNamespace = getString(params, "namespace")
		req.Params = map[string]interface{}{
			"command":       "get_events",
			"involved_kind": getString(params, "involved_kind"),
			"involved_name": getString(params, "involved_name"),
		}

	case "get_configmap":
		req.Action = "get_configmap"
		req.TargetKind = "ConfigMap"
		req.TargetNamespace = getString(params, "namespace")
		req.TargetName = getString(params, "configmap_name")

	case "get_node_status":
		req.Action = "dynamic"
		req.TargetKind = "Node"
		req.TargetName = getString(params, "node_name")
		req.Params = map[string]interface{}{
			"command": "describe",
			"kind":    "node",
		}

	case "list_pods":
		req.Action = "dynamic"
		req.TargetKind = "Pod"
		req.TargetNamespace = getString(params, "namespace")
		req.Params = map[string]interface{}{
			"command":        "list",
			"kind":           "pod",
			"label_selector": getString(params, "label_selector"),
		}

	default:
		return nil, fmt.Errorf("未知的 Tool: %s", tc.Name)
	}

	return req, nil
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
