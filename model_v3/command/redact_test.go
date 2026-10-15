package command

import (
	"encoding/json"
	"testing"
)

func TestResultMetadata_DropsOutputKeepsOutcome(t *testing.T) {
	r := Result{CommandID: "c1", Success: false, Output: "PASSWORD=hunter2", Error: "boom"}
	m := r.Metadata()
	if m.Output != "" {
		t.Errorf("Output 必须被去掉，得到 %q", m.Output)
	}
	if m.CommandID != "c1" || m.Success || m.Error != "boom" {
		t.Errorf("执行结论字段必须保留，得到 %+v", m)
	}
	if r.Output == "" {
		t.Error("Metadata 不得修改原值（bus 内存里的结果还要交给同步等待方）")
	}
}

// 早期版本写入的行可能含 output：查询接口返回前统一剥离。
func TestRedactStoredResult(t *testing.T) {
	old, _ := json.Marshal(Result{CommandID: "c1", Success: true, Output: `{"password":"hunter2"}`})
	cases := []struct {
		name, stored string
		wantEmpty    bool
	}{
		{"旧行带 output", string(old), false},
		{"空串", "", true},
		{"非 JSON：宁丢不漏", "password=hunter2", true},
	}
	for _, c := range cases {
		got := RedactStoredResult(c.stored)
		if c.wantEmpty {
			if got != "" {
				t.Errorf("%s: 期望空串，得到 %q", c.name, got)
			}
			continue
		}
		var back map[string]any
		if err := json.Unmarshal([]byte(got), &back); err != nil {
			t.Fatalf("%s: 输出不是 JSON: %v", c.name, err)
		}
		if _, ok := back["output"]; ok {
			t.Errorf("%s: output 字段必须消失，得到 %s", c.name, got)
		}
		if back["success"] != true || back["commandId"] != "c1" {
			t.Errorf("%s: 执行结论必须保留，得到 %s", c.name, got)
		}
	}
}
