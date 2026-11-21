package tools

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/command"
)

type recordingRunner struct {
	reqs   []*model.CreateCommandRequest
	result *command.Result
}

func (r *recordingRunner) ExecuteCommandSync(_ context.Context, req *model.CreateCommandRequest, _ time.Duration) (*command.Result, error) {
	r.reqs = append(r.reqs, req)
	if r.result != nil {
		return r.result, nil
	}
	return &command.Result{Success: true, Output: "ok"}, nil
}

func clusterRegistry(runner *recordingRunner) *toolkit.Registry {
	reg := toolkit.NewRegistry()
	reg.Register(queryClusterTool(Deps{Commands: runner, ToolTimeout: time.Second}))
	return reg
}

func allowAll(string) bool { return true }

func call(reg *toolkit.Registry, allow toolkit.PermissionCheck, params string) (string, error) {
	return reg.Execute(context.Background(), "c1", allow, &llm.ToolCall{Name: "query_cluster", Params: params})
}

// query_cluster 用允许名单：动作与资源类型都在 schema 的枚举里，服务端按同一份 schema 拦下名单外的
func TestQueryCluster_AllowList(t *testing.T) {
	runner := &recordingRunner{}
	reg := clusterRegistry(runner)
	for name, params := range map[string]string{
		"Secret 不在名单里":   `{"action":"get","kind":"Secret","namespace":"ns","name":"s"}`,
		"小写的 secret 也不行": `{"action":"list","kind":"secrets"}`,
		"写动作":            `{"action":"delete","kind":"Pod","namespace":"ns","name":"p"}`,
		"缩写的资源类型":        `{"action":"list","kind":"PVC"}`,
		"多余的参数":          `{"action":"list","kind":"Pod","patch":"x"}`,
	} {
		if _, err := call(reg, allowAll, params); !errors.Is(err, toolkit.ErrInvalidParams) {
			t.Errorf("%s: err = %v, want ErrInvalidParams", name, err)
		}
	}
	if len(runner.reqs) != 0 {
		t.Errorf("名单外的调用不应下发指令，实际 %d 条", len(runner.reqs))
	}
}

// 按动作细分权限：日志要日志权限、ConfigMap 要凭据权限、事件要事件权限
func TestQueryCluster_PermissionByAction(t *testing.T) {
	runner := &recordingRunner{}
	reg := clusterRegistry(runner)
	onlyWorkload := func(p string) bool { return p == toolkit.PermWorkloadEnvRead }
	denied := []string{
		`{"action":"get_logs","kind":"Pod","namespace":"ns","name":"p"}`,
		`{"action":"get_configmap","kind":"ConfigMap","namespace":"ns","name":"c"}`,
		`{"action":"get","kind":"ConfigMap","namespace":"ns","name":"c"}`,
		`{"action":"get_events","kind":"Event"}`,
	}
	for _, params := range denied {
		if _, err := call(reg, onlyWorkload, params); !errors.Is(err, toolkit.ErrPermissionDenied) {
			t.Errorf("%s: err = %v, want ErrPermissionDenied", params, err)
		}
	}
	if _, err := call(reg, onlyWorkload, `{"action":"get","kind":"Pod","namespace":"ns","name":"p"}`); err != nil {
		t.Errorf("读工作负载对象应通过: %v", err)
	}
}

// 系统命名空间拒绝；get 缺 name 拒绝；日志行数不填为默认值、按 Agent 能解析的参数传
func TestQueryCluster_CommandShape(t *testing.T) {
	runner := &recordingRunner{}
	reg := clusterRegistry(runner)

	if _, err := call(reg, allowAll, `{"action":"list","kind":"Pod","namespace":"kube-system"}`); err == nil {
		t.Error("系统命名空间应拒绝")
	}
	if _, err := call(reg, allowAll, `{"action":"get","kind":"Pod","namespace":"ns"}`); err == nil {
		t.Error("get 缺 name 应拒绝")
	}

	if _, err := call(reg, allowAll, `{"action":"get_logs","kind":"Pod","namespace":"ns","name":"p","container":"app"}`); err != nil {
		t.Fatal(err)
	}
	req := runner.reqs[len(runner.reqs)-1]
	var logs command.GetLogsParams
	if err := command.ParseParams(req.Params, &logs); err != nil {
		t.Fatal(err)
	}
	if req.Action != command.ActionGetLogs || logs.TailLines != defaultTailLines || logs.Container != "app" || req.Source != command.SourceAI {
		t.Errorf("日志指令 = %+v / %+v", req, logs)
	}
}

// Agent 报告失败：返回错误（对话里显示为失败），⛔ 不当作正常结果
func TestQueryCluster_AgentFailureIsAnError(t *testing.T) {
	runner := &recordingRunner{result: &command.Result{Success: false, Error: "forbidden"}}
	reg := clusterRegistry(runner)
	if _, err := call(reg, allowAll, `{"action":"list","kind":"Pod"}`); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("err = %v, want 带 Agent 的失败原因", err)
	}
}

// 可选依赖缺失的工具不注册；全部工具的定义都是合法的 JSON Schema、都有说明
func TestRegister_OptionalToolsAndDefinitions(t *testing.T) {
	reg := toolkit.NewRegistry()
	Register(reg, Deps{})
	names := reg.Names()
	for _, optional := range []string{"github_read_file", "github_search_code", "github_recent_commits"} {
		if slices.Contains(names, optional) {
			t.Errorf("没有配置时不应注册 %s", optional)
		}
	}
	for _, want := range []string{"query_cluster", "get_event_history", "query_slo", "analyze_incident"} {
		if !slices.Contains(names, want) {
			t.Errorf("缺 %s", want)
		}
	}

	all := toolkit.NewRegistry()
	all.Register(catalog()...)
	for _, def := range all.Definitions(allowAll) {
		var schema map[string]any
		if err := json.Unmarshal(def.Parameters, &schema); err != nil || schema["type"] != "object" {
			t.Errorf("%s 的参数 schema 不合法: %s", def.Name, def.Parameters)
		}
		if def.Description == "" {
			t.Errorf("%s 没有说明", def.Name)
		}
	}
}
