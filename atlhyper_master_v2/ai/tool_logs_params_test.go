package ai

import (
	"testing"

	"AtlHyper/model_v3/command"
)

// AI 请求的日志行数必须按 Agent 能解析的参数传过去。
func TestMapToInternalAction_GetLogsCarriesTailLines(t *testing.T) {
	action, params := mapToInternalAction("get_logs", map[string]interface{}{
		"container":  "app",
		"tail_lines": float64(150), // LLM 的工具参数经 JSON 解码，数字是 float64
	})
	if action != command.ActionGetLogs {
		t.Fatalf("action = %q, want %q", action, command.ActionGetLogs)
	}

	var received command.GetLogsParams
	if err := command.ParseParams(params, &received); err != nil {
		t.Fatal(err)
	}
	if received.TailLines != 150 {
		t.Errorf("Agent 解析到的行数 = %d, want 150", received.TailLines)
	}
	if received.Container != "app" {
		t.Errorf("Agent 解析到的容器 = %q, want app", received.Container)
	}
}
