package toolkit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

type probeParams struct {
	Action string `json:"action" jsonschema:"操作"`
	Limit  int    `json:"limit,omitempty" jsonschema:"条数"`
}

func probeTool(t *testing.T) (*Tool, *probeParams) {
	t.Helper()
	got := &probeParams{}
	tool, err := Define(Spec[probeParams]{
		Name:        "probe",
		Description: "测试用",
		Permission:  PermWorkloadEnvRead,
		PermissionFor: func(p probeParams) string {
			if p.Action == "logs" {
				return PermPodLogsRead
			}
			return PermWorkloadEnvRead
		},
		Permissions: []string{PermWorkloadEnvRead, PermPodLogsRead},
		Refine: func(s *jsonschema.Schema) {
			s.Properties["action"].Enum = []any{"get", "logs"}
			max := 50.0
			s.Properties["limit"].Maximum = &max
		},
		Handler: func(_ context.Context, _ string, p probeParams) (string, error) {
			*got = p
			return "ok", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tool, got
}

func allowOnly(perms ...string) PermissionCheck {
	return func(p string) bool {
		for _, x := range perms {
			if x == p {
				return true
			}
		}
		return false
	}
}

// 发给模型的 schema 与服务端校验是同一份：必填、枚举、上限、未知字段都在服务端拦下
func TestExecute_ValidatesAgainstTheSameSchema(t *testing.T) {
	tool, got := probeTool(t)
	reg := NewRegistry()
	reg.Register(tool)
	all := allowOnly(PermWorkloadEnvRead, PermPodLogsRead)

	cases := []struct {
		name   string
		params string
	}{
		{"缺必填", `{}`},
		{"不在枚举里", `{"action":"delete"}`},
		{"超上限", `{"action":"get","limit":500}`},
		{"未知字段", `{"action":"get","extra":1}`},
		{"类型不对", `{"action":"get","limit":"ten"}`},
		{"不是 JSON", `{"action":`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := reg.Execute(context.Background(), "c1", all, &llm.ToolCall{Name: "probe", Params: c.params})
			if !errors.Is(err, ErrInvalidParams) {
				t.Errorf("err = %v, want ErrInvalidParams", err)
			}
		})
	}

	out, err := reg.Execute(context.Background(), "c1", all, &llm.ToolCall{Name: "probe", Params: `{"action":"get","limit":5}`})
	if err != nil || out != "ok" || got.Action != "get" || got.Limit != 5 {
		t.Errorf("合法参数: out=%q err=%v got=%+v", out, err, got)
	}

	// 发给模型的定义里有同样的约束
	var schema map[string]any
	_ = json.Unmarshal(tool.Definition().Parameters, &schema)
	props := schema["properties"].(map[string]any)
	if props["action"].(map[string]any)["enum"] == nil || schema["required"] == nil {
		t.Errorf("发给模型的 schema 缺枚举或必填: %s", tool.Definition().Parameters)
	}
}

// 权限：按参数细分；没有权限时拒绝；allow 为 nil 一律拒绝；未注册的工具单独分类
func TestExecute_Permissions(t *testing.T) {
	tool, _ := probeTool(t)
	reg := NewRegistry()
	reg.Register(tool)

	readOnly := allowOnly(PermWorkloadEnvRead)
	if _, err := reg.Execute(context.Background(), "c1", readOnly, &llm.ToolCall{Name: "probe", Params: `{"action":"logs"}`}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("看日志需要日志权限: err = %v", err)
	}
	if _, err := reg.Execute(context.Background(), "c1", readOnly, &llm.ToolCall{Name: "probe", Params: `{"action":"get"}`}); err != nil {
		t.Errorf("有权限时应通过: %v", err)
	}
	if _, err := reg.Execute(context.Background(), "c1", nil, &llm.ToolCall{Name: "probe", Params: `{"action":"get"}`}); !errors.Is(err, ErrPermissionDenied) {
		t.Errorf("allow 为 nil 应拒绝: %v", err)
	}
	if _, err := reg.Execute(context.Background(), "c1", readOnly, &llm.ToolCall{Name: "nope"}); !errors.Is(err, ErrUnknownTool) {
		t.Errorf("未注册的工具: %v", err)
	}
}

// 清单 = 已注册 ∩ 有权限；全部权限可列出（给网关核对）
func TestRegistry_DefinitionsAndPermissions(t *testing.T) {
	tool, _ := probeTool(t)
	reg := NewRegistry()
	reg.Register(tool)

	if defs := reg.Definitions(allowOnly()); len(defs) != 0 {
		t.Errorf("没有权限时清单应为空，实际 %d", len(defs))
	}
	if defs := reg.Definitions(allowOnly(PermWorkloadEnvRead)); len(defs) != 1 || defs[0].Name != "probe" {
		t.Errorf("清单 = %+v", defs)
	}
	if defs := reg.Definitions(nil); defs != nil {
		t.Error("allow 为 nil 时不给任何工具")
	}
	perms := reg.Permissions()
	if len(perms) != 2 || perms[0] != PermPodLogsRead || perms[1] != PermWorkloadEnvRead {
		t.Errorf("Permissions = %v", perms)
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	tool, _ := probeTool(t)
	reg := NewRegistry()
	reg.Register(tool)
	defer func() {
		if recover() == nil {
			t.Error("重名注册应 panic")
		}
	}()
	reg.Register(tool)
}

// 工具的效果只在定义上标一处：不标 = 只读；对外入口（MCP 的注解、确认闸）都从这里取
func TestTool_Effect(t *testing.T) {
	read, _ := probeTool(t)
	if read.Effect() != EffectRead {
		t.Errorf("没标效果的工具应是只读，得到 %q", read.Effect())
	}
	write := MustDefine(Spec[probeParams]{
		Name: "writer", Description: "测试用", Permission: PermNone, Effect: EffectDestructive,
		Handler: func(context.Context, string, probeParams) (string, error) { return "", nil },
	})
	if write.Effect() != EffectDestructive {
		t.Errorf("Effect = %q, want %q", write.Effect(), EffectDestructive)
	}
}

// 不针对某个集群的工具（如列出集群）要标出来：调用方据此不要求、也不解析集群
func TestTool_ClusterScope(t *testing.T) {
	scoped, _ := probeTool(t)
	if !scoped.NeedsCluster() {
		t.Error("默认的工具针对一个集群")
	}
	global := MustDefine(Spec[probeParams]{
		Name: "global", Description: "测试用", Permission: PermNone, NoCluster: true,
		Handler: func(context.Context, string, probeParams) (string, error) { return "", nil },
	})
	if global.NeedsCluster() {
		t.Error("NoCluster 的工具不需要集群")
	}
}

// 清单的可见性与 Definitions 同一条规则（基础权限），供别的入口逐个工具判断
func TestTool_VisibleTo(t *testing.T) {
	tool, _ := probeTool(t)
	cases := []struct {
		name  string
		allow PermissionCheck
		want  bool
	}{
		{"有基础权限", allowOnly(PermWorkloadEnvRead), true},
		{"只有细分权限、没有基础权限", allowOnly(PermPodLogsRead), false},
		{"没给权限判断", nil, false},
	}
	for _, c := range cases {
		if got := tool.VisibleTo(c.allow); got != c.want {
			t.Errorf("%s: VisibleTo = %v, want %v", c.name, got, c.want)
		}
	}
	open := MustDefine(Spec[probeParams]{
		Name: "open", Description: "测试用", Permission: PermNone,
		Handler: func(context.Context, string, probeParams) (string, error) { return "", nil },
	})
	if !open.VisibleTo(allowOnly()) {
		t.Error("不需要权限的工具对任何已识别的调用者可见")
	}
}

func TestRegistry_ToolsAndLookup(t *testing.T) {
	reg := NewRegistry()
	tool, _ := probeTool(t)
	reg.Register(tool)
	if got := reg.Tools(); len(got) != 1 || got[0].Name() != "probe" {
		t.Fatalf("Tools() = %v", got)
	}
	if found, ok := reg.Lookup("probe"); !ok || found != tool {
		t.Errorf("Lookup(probe) = (%v, %v)", found, ok)
	}
	if _, ok := reg.Lookup("missing"); ok {
		t.Error("没注册的工具不应查到")
	}
	if tool.Description() != "测试用" {
		t.Errorf("Description = %q", tool.Description())
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.InputSchema(), &schema); err != nil || schema["type"] != "object" {
		t.Errorf("InputSchema 应是对象 schema 的 JSON: %v %v", schema["type"], err)
	}
}
