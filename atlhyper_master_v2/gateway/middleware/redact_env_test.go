package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 模拟各资源详情的嵌套形态：Pod 的 containers[].envs，工作负载的 template.containers[].envs
const detailWithEnvs = `{"message":"ok","data":{
  "containers":[{"name":"app","envs":[
    {"name":"DB_PASSWORD","value":"hunter2"},
    {"name":"API_TOKEN","valueFrom":"secret:app-secret"}]}],
  "template":{"containers":[{"name":"side","envs":[{"name":"LEVEL","value":"debug"}]}]}
}}`

func serveDetail(t *testing.T, p Principal, body string, status int) *httptest.ResponseRecorder {
	t.Helper()
	inner := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v2/pods/x", nil)
	RedactEnvValues(PermWorkloadEnvRead, inner)(rec, req.WithContext(WithPrincipal(context.Background(), p)))
	return rec
}

func TestRedactEnvValues_GuestSeesNamesNotValues(t *testing.T) {
	rec := serveDetail(t, Principal{Level: RoleMember}, detailWithEnvs, http.StatusOK)
	out := rec.Body.String()
	if strings.Contains(out, "hunter2") || strings.Contains(out, "debug") {
		t.Fatalf("字面值必须被剥掉：%s", out)
	}
	for _, keep := range []string{"DB_PASSWORD", "LEVEL", "secret:app-secret"} {
		if !strings.Contains(out, keep) {
			t.Errorf("变量名与引用来源应保留：缺 %s", keep)
		}
	}
	var parsed struct {
		Data struct {
			Containers []struct {
				Envs []map[string]any `json:"envs"`
			} `json:"containers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	envs := parsed.Data.Containers[0].Envs
	if envs[0]["valueHidden"] != true {
		t.Errorf("被剥掉的字面值应标记 valueHidden：%v", envs[0])
	}
	if _, ok := envs[1]["valueHidden"]; ok {
		t.Errorf("本来就没有字面值的不应标记：%v", envs[1])
	}
}

func TestRedactEnvValues_ViewerUnchanged(t *testing.T) {
	rec := serveDetail(t, Principal{UserID: 2, Level: RoleViewer}, detailWithEnvs, http.StatusOK)
	if rec.Body.String() != detailWithEnvs {
		t.Errorf("有权限时应原样透传")
	}
}

// 非 200 / 非 JSON 原样透传（错误响应里没有 env）
func TestRedactEnvValues_ErrorPassthrough(t *testing.T) {
	rec := serveDetail(t, Principal{Level: RoleMember}, `{"error":"not found"}`, http.StatusNotFound)
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "not found") {
		t.Errorf("错误响应应透传，得到 %d %s", rec.Code, rec.Body.String())
	}
}

// 200 但解析不了：失败即关，不吐出原文
func TestRedactEnvValues_UnparseableFailsClosed(t *testing.T) {
	rec := serveDetail(t, Principal{Level: RoleMember}, `{"envs":[{"value":"hunter2"`, http.StatusOK)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "hunter2") {
		t.Errorf("应 500 且不含原文，得到 %d %s", rec.Code, rec.Body.String())
	}
}
