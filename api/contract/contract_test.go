package contract

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// 规格本身能解析、且通过 OpenAPI 3.0 校验
func TestSpec_Valid(t *testing.T) {
	if _, err := Spec(); err != nil {
		t.Fatal(err)
	}
}

// 阳性对照：校验装置真的会拒绝不合规格的响应
func TestAssertResponse_RejectsMismatches(t *testing.T) {
	cases := map[string]string{
		"多出未声明的字段":  `{"message":"ok","data":[],"extra":1}`,
		"缺必填字段":     `{"data":[]}`,
		"类型不对":      `{"message":1,"data":[]}`,
		"数组给了 null": `{"message":"ok","data":null}`,
	}
	schema, err := ResponseSchema("GET", "/api/v2/observe/metrics/nodes", 200)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if err := visit(schema, body); err == nil {
				t.Errorf("应被拒绝: %s", body)
			}
		})
	}
}

// 可空的对象引用（nullable + allOf）：null 合规，错误类型仍被拒绝
func TestNullableRef(t *testing.T) {
	schema, err := ResponseSchema("GET", "/api/v2/observe/metrics/hardware", 200)
	if err != nil {
		t.Fatal(err)
	}
	_ = schema
	doc, _ := Spec()
	row := doc.Components.Schemas["HardwareRow"].Value
	var nullableRefs []string
	for name, p := range row.Properties {
		if p.Value != nil && p.Value.Nullable && len(p.Value.AllOf) > 0 {
			nullableRefs = append(nullableRefs, name)
		}
	}
	if len(nullableRefs) == 0 {
		t.Skip("HardwareRow 没有可空的对象引用")
	}
	prop := row.Properties[nullableRefs[0]].Value
	if err := prop.VisitJSON(nil); err != nil {
		t.Errorf("%s = null 应合规: %v", nullableRefs[0], err)
	}
	if err := prop.VisitJSON("oops"); err == nil {
		t.Errorf("%s = 字符串应被拒绝", nullableRefs[0])
	}
}

func visit(s *openapi3.Schema, body string) error {
	var v any
	if err := jsonUnmarshal([]byte(body), &v); err != nil {
		return err
	}
	return s.VisitJSON(v, openapi3.MultiErrors())
}

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

type fillProbe struct {
	Name   string
	Level  probeLevel
	Count  int
	Ratio  float64
	OK     bool
	At     time.Time
	Ptr    *int
	Items  []struct{ A string }
	ByName map[string]int
	hidden string
}

type probeLevel string

func TestFill_EveryExportedFieldNonZero(t *testing.T) {
	var p fillProbe
	Fill(&p, probeLevel("good"))
	if p.Name != "x" || p.Level != "good" || p.Count != 1 || p.Ratio == 0 || !p.OK || !p.At.Equal(FixedTime) ||
		p.Ptr == nil || *p.Ptr != 1 || len(p.Items) != 1 || p.Items[0].A != "x" || len(p.ByName) != 1 || p.hidden != "" {
		t.Fatalf("没有填满: %+v", p)
	}
}

func TestFill_NamedStringWithoutSamplePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("缺枚举样例时应 panic")
		}
	}()
	var p fillProbe
	Fill(&p)
}
