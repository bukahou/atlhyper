package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/prompts"
)

// allowOnly 只拥有给定权限的判断函数
func allowOnly(perms ...string) PermissionCheck {
	set := map[string]bool{}
	for _, p := range perms {
		set[p] = true
	}
	return func(p string) bool { return set[p] }
}

// viewerPerms 1 级（Viewer）拥有的、与 AI 工具相关的权限
var viewerPerms = []string{
	"cluster.read", "events.read", "slo.read", "aiops.read",
	"workload.env.read", "logs.read", "traces.read", "deploy.history.read", "ai.chat",
}

func TestToolPermission_QueryClusterByActionAndKind(t *testing.T) {
	cases := []struct {
		action, kind, want string
	}{
		{"get_logs", "Pod", "pod.logs.read"},
		{"get_configmap", "ConfigMap", "secret.read"},
		{"get", "ConfigMap", "secret.read"}, // ConfigMap 对象本身就是内容
		{"list", "ConfigMap", "secret.read"},
		{"describe", "ConfigMap", "secret.read"},
		{"describe", "Pod", "workload.env.read"}, // 原始对象含 env 字面值
		{"list", "Deployment", "workload.env.read"},
		{"get_events", "", "events.read"},
	}
	for _, c := range cases {
		got, ok := toolPermission("query_cluster", map[string]interface{}{"action": c.action, "kind": c.kind})
		if !ok || got != c.want {
			t.Errorf("query_cluster %s %s = %q (ok=%v), want %q", c.action, c.kind, got, ok, c.want)
		}
	}
}

// 新增工具必须先声明所需权限；没声明的一律拒绝
func TestToolPermission_EveryDefinedToolDeclared(t *testing.T) {
	for _, def := range prompts.GetToolDefinitions() {
		if _, ok := toolPermission(def.Name, map[string]interface{}{}); !ok {
			t.Errorf("工具 %s 没有声明所需权限", def.Name)
		}
	}
	if _, ok := toolPermission("not_a_tool", nil); ok {
		t.Error("未登记的工具不应有权限")
	}
}

func TestToolDefsFor_FiltersByPermission(t *testing.T) {
	names := map[string]bool{}
	for _, d := range toolDefsFor(allowOnly(viewerPerms...)) {
		names[d.Name] = true
	}
	if names["analyze_incident"] {
		t.Error("Viewer 没有 ai.analyze，不应拿到 analyze_incident")
	}
	for _, want := range []string{"query_cluster", "query_logs", "query_traces", "get_cluster_risk", "get_event_history"} {
		if !names[want] {
			t.Errorf("Viewer 应能用 %s", want)
		}
	}
	if len(toolDefsFor(allowOnly())) != countFreeTools() {
		t.Error("没有任何权限时只剩不需要额外权限的工具")
	}
}

func countFreeTools() int {
	n := 0
	for _, d := range prompts.GetToolDefinitions() {
		if p, ok := toolPermission(d.Name, map[string]interface{}{}); ok && p == "" && d.Name != "query_cluster" {
			n++
		}
	}
	return n
}

// 权限不足时在执行之前拒绝：自定义工具不被调用，集群查询不下发指令（ops 为 nil，走到即 panic）
func TestExecutor_DeniesBeforeExecuting(t *testing.T) {
	e := newToolExecutor(nil, nil, time.Second)
	called := false
	e.RegisterTool("query_logs", func(context.Context, string, map[string]interface{}) (string, error) {
		called = true
		return "logs", nil
	})

	_, err := e.Execute(context.Background(), "c1", allowOnly(viewerPerms[:4]...), &llm.ToolCall{Name: "query_logs", Params: `{}`})
	if err == nil || !strings.Contains(err.Error(), "权限不足") || called {
		t.Errorf("缺 logs.read 时应拒绝：err=%v called=%v", err, called)
	}

	_, err = e.Execute(context.Background(), "c1", allowOnly(viewerPerms...), &llm.ToolCall{
		Name: "query_cluster", Params: `{"action":"get_logs","kind":"Pod","namespace":"ns","name":"p"}`,
	})
	if err == nil || !strings.Contains(err.Error(), "pod.logs.read") {
		t.Errorf("Viewer 读容器日志应被拒绝：err=%v", err)
	}

	_, err = e.Execute(context.Background(), "c1", allowOnly(viewerPerms...), &llm.ToolCall{Name: "not_a_tool", Params: `{}`})
	if err == nil {
		t.Error("未登记的工具应被拒绝")
	}

	out, err := e.Execute(context.Background(), "c1", allowOnly(viewerPerms...), &llm.ToolCall{Name: "query_logs", Params: `{}`})
	if err != nil || out != "logs" || !called {
		t.Errorf("有权限时应正常执行：out=%q err=%v", out, err)
	}
}
