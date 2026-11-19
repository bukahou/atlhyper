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
