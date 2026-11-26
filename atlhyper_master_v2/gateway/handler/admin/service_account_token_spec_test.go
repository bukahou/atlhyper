package admin

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"AtlHyper/api/contract"
	"AtlHyper/atlhyper_master_v2/model"
)

// specAccepts 规格里某个请求 schema 是否接受这个请求体
func specAccepts(t *testing.T, schemaName, body string) bool {
	t.Helper()
	doc, err := contract.Spec()
	if err != nil {
		t.Fatal(err)
	}
	ref, ok := doc.Components.Schemas[schemaName]
	if !ok || ref.Value == nil {
		t.Fatalf("规格里没有 %s", schemaName)
	}
	var value any
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("样例不是 JSON：%s", body)
	}
	return ref.Value.VisitJSON(value) == nil
}

// 规格里写的取值范围必须有运行时执行者：同一组请求体，规格与后端校验的结论逐条一致
// （规格与 model 的业务参数各写了一份，任何一边改了另一边没改，这里就会失败）
func TestServiceAccountRequestRules_AgreeWithSpec(t *testing.T) {
	longName := "a" + strings.Repeat("b", 31)
	createCases := []struct {
		name, displayName string
		role              int
	}{
		{"haro", "", 0},
		{"h-1", "Haro", 2},
		{longName, strings.Repeat("语", 64), 1},
		{longName + "c", "", 0},
		{"ab", "", 0},
		{"1haro", "", 0},
		{"Haro", "", 0},
		{"haro_x", "", 0},
		{"haro", strings.Repeat("语", 65), 0},
		{"haro", "", 3},
		{"haro", "", -1},
	}
	for _, c := range createCases {
		body := fmt.Sprintf(`{"name":%q,"displayName":%q,"role":%d}`, c.name, c.displayName, c.role)
		role := c.role
		goAccepts := model.ValidateServiceAccountName(c.name) == nil && model.ValidateServiceAccountProfile(c.displayName, &role) == nil
		if spec := specAccepts(t, "CreateServiceAccountRequest", body); spec != goAccepts {
			t.Errorf("新建 %s：规格接受 = %v，后端接受 = %v", body, spec, goAccepts)
		}
	}

	for _, role := range []int{-1, 0, model.ServiceAccountMaxRole, model.ServiceAccountMaxRole + 1} {
		body := fmt.Sprintf(`{"displayName":"","role":%d,"enabled":true}`, role)
		displayName, enabled, r := "", true, role
		goAccepts := model.ValidateUpdateServiceAccountRequest(&model.UpdateServiceAccountRequest{DisplayName: &displayName, Role: &r, Enabled: &enabled}) == nil
		if spec := specAccepts(t, "UpdateServiceAccountRequest", body); spec != goAccepts {
			t.Errorf("修改 %s：规格接受 = %v，后端接受 = %v", body, spec, goAccepts)
		}
	}

	tokenCases := []struct {
		name string
		days *int
	}{
		{"prod", nil},
		{"", nil},
		{strings.Repeat("令", 64), nil},
		{strings.Repeat("令", 65), nil},
	}
	for _, days := range []int{0, 1, model.MaxAPITokenExpiresInDays, model.MaxAPITokenExpiresInDays + 1} {
		d := days
		tokenCases = append(tokenCases, struct {
			name string
			days *int
		}{"prod", &d})
	}
	for _, c := range tokenCases {
		body := fmt.Sprintf(`{"name":%q}`, c.name)
		if c.days != nil {
			body = fmt.Sprintf(`{"name":%q,"expiresInDays":%d}`, c.name, *c.days)
		}
		goAccepts := model.ValidateAPITokenRequest(c.name, c.days) == nil
		if spec := specAccepts(t, "IssueAPITokenRequest", body); spec != goAccepts {
			t.Errorf("签发 %s：规格接受 = %v，后端接受 = %v", body, spec, goAccepts)
		}
	}
}

// 阳性对照：比对装置真的会区分接受与拒绝
func TestSpecAccepts_PositiveControl(t *testing.T) {
	if !specAccepts(t, "IssueAPITokenRequest", `{"name":"prod"}`) {
		t.Error("合规的请求体应被规格接受")
	}
	if specAccepts(t, "IssueAPITokenRequest", `{"name":"prod","expiresInDays":0}`) {
		t.Error("越界的请求体应被规格拒绝")
	}
}
