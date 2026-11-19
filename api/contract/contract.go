// Package contract 对照 OpenAPI 规格校验接口响应，供各处理函数的测试使用。
//
// 规格是前后端的单一来源：前端类型由它生成，Go 的响应在测试里经这里校验，
// 字段名、类型、必填、可空、枚举任何一处与规格不一致即失败。
package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"AtlHyper/api"
)

var (
	loadOnce sync.Once
	loaded   *openapi3.T
	loadErr  error
)

// Spec 解析并校验内嵌的规格（只做一次）
func Spec() (*openapi3.T, error) {
	loadOnce.Do(func() {
		loader := openapi3.NewLoader()
		doc, err := loader.LoadFromData(api.OpenAPI)
		if err != nil {
			loadErr = fmt.Errorf("解析 openapi.yaml: %w", err)
			return
		}
		if err := doc.Validate(context.Background()); err != nil {
			loadErr = fmt.Errorf("openapi.yaml 不合规: %w", err)
			return
		}
		loaded = doc
	})
	return loaded, loadErr
}

// ResponseSchema 取规格里某个接口某个状态码的 JSON 响应 schema
func ResponseSchema(method, path string, status int) (*openapi3.Schema, error) {
	doc, err := Spec()
	if err != nil {
		return nil, err
	}
	item := doc.Paths.Value(path)
	if item == nil {
		return nil, fmt.Errorf("规格里没有路径 %s", path)
	}
	op := item.GetOperation(strings.ToUpper(method))
	if op == nil {
		return nil, fmt.Errorf("规格里没有 %s %s", method, path)
	}
	resp := op.Responses.Status(status)
	if resp == nil || resp.Value == nil {
		return nil, fmt.Errorf("规格里 %s %s 没有声明 %d 响应", method, path, status)
	}
	media := resp.Value.Content.Get("application/json")
	if media == nil || media.Schema == nil || media.Schema.Value == nil {
		return nil, fmt.Errorf("规格里 %s %s 的 %d 响应没有 JSON schema", method, path, status)
	}
	return media.Schema.Value, nil
}

// AssertValid 只校验响应体符合规格（不要求每个属性都出现），用于稀疏数据：
// 例如代理什么都没采到时，数组仍须是数组（⛔ null）、等级仍须在枚举里。
func AssertValid(t testing.TB, method, path string, status int, body []byte) {
	t.Helper()
	schema, err := ResponseSchema(method, path, status)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("响应不是 JSON: %v\n%s", err, body)
	}
	if err := schema.VisitJSON(value, openapi3.MultiErrors()); err != nil {
		t.Fatalf("%s %s 的 %d 响应不符合规格:\n%v", method, path, status, err)
	}
}

// AssertResponse 校验响应体符合规格。
//
// 另外核对「规格里声明的每个属性在响应里至少出现过一次」：测试数据填满了，
// Go 一侧改名或删字段才会被发现（没出现的字段无从校验）。允许缺席的属性用 absentOK 列出（规格路径，如 "data[].disks[].mountpoint"）。
func AssertResponse(t testing.TB, method, path string, status int, body []byte, absentOK ...string) {
	t.Helper()
	AssertValid(t, method, path, status, body)
	schema, err := ResponseSchema(method, path, status)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	_ = json.Unmarshal(body, &value)
	unseen := map[string]bool{}
	collectDeclared(schema, "", unseen, map[*openapi3.Schema]bool{})
	markSeen(schema, value, "", unseen, map[*openapi3.Schema]int{})
	for _, p := range absentOK {
		delete(unseen, p)
	}
	if len(unseen) > 0 {
		missing := make([]string, 0, len(unseen))
		for p := range unseen {
			missing = append(missing, p)
		}
		sort.Strings(missing)
		t.Fatalf("%s %s：规格声明了、但测试响应里一次都没出现的属性（补全测试数据，或确认后列入 absentOK）:\n  %s",
			method, path, strings.Join(missing, "\n  "))
	}
}

// 展开 allOf（规格里用 allOf 包一层来给引用加说明或可空）
func members(s *openapi3.Schema) []*openapi3.Schema {
	out := []*openapi3.Schema{s}
	for _, ref := range s.AllOf {
		if ref.Value != nil {
			out = append(out, members(ref.Value)...)
		}
	}
	return out
}

func collectDeclared(s *openapi3.Schema, prefix string, out map[string]bool, visiting map[*openapi3.Schema]bool) {
	if visiting[s] {
		return
	}
	visiting[s] = true
	defer delete(visiting, s)
	for _, m := range members(s) {
		for name, prop := range m.Properties {
			p := join(prefix, name)
			out[p] = true
			if prop.Value != nil {
				collectDeclared(prop.Value, p, out, visiting)
			}
		}
		if m.Items != nil && m.Items.Value != nil {
			collectDeclared(m.Items.Value, prefix+"[]", out, visiting)
		}
		if m.AdditionalProperties.Schema != nil && m.AdditionalProperties.Schema.Value != nil {
			collectDeclared(m.AdditionalProperties.Schema.Value, prefix+"{}", out, visiting)
		}
	}
}

func markSeen(s *openapi3.Schema, value any, prefix string, unseen map[string]bool, depth map[*openapi3.Schema]int) {
	if depth[s] > 8 {
		return
	}
	depth[s]++
	defer func() { depth[s]-- }()
	for _, m := range members(s) {
		switch v := value.(type) {
		case map[string]any:
			for name, prop := range m.Properties {
				child, ok := v[name]
				if !ok {
					continue
				}
				p := join(prefix, name)
				delete(unseen, p)
				if prop.Value != nil && child != nil {
					markSeen(prop.Value, child, p, unseen, depth)
				}
			}
			if m.AdditionalProperties.Schema != nil && m.AdditionalProperties.Schema.Value != nil {
				for _, child := range v {
					if child != nil {
						markSeen(m.AdditionalProperties.Schema.Value, child, prefix+"{}", unseen, depth)
					}
				}
			}
		case []any:
			if m.Items != nil && m.Items.Value != nil {
				for _, child := range v {
					if child != nil {
						markSeen(m.Items.Value, child, prefix+"[]", unseen, depth)
					}
				}
			}
		}
	}
}

func join(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
