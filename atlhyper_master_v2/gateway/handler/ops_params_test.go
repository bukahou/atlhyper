package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/model_v3/command"
)

// recordingOps 只实现 CreateCommand，记录被创建的指令；其余方法不该被调到
type recordingOps struct {
	service.Ops
	created []*model.CreateCommandRequest
}

func (r *recordingOps) CreateCommand(_ context.Context, req *model.CreateCommandRequest) (*model.CreateCommandResponse, error) {
	r.created = append(r.created, req)
	return &model.CreateCommandResponse{CommandID: "cmd-1", Status: command.StatusPending}, nil
}

func postOps(t *testing.T, h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	return rec
}

func TestDeploymentScale_Params(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"缺副本数", `{"clusterId":"c1","namespace":"default","name":"web"}`, http.StatusBadRequest},
		{"负数", `{"clusterId":"c1","namespace":"default","name":"web","replicas":-1}`, http.StatusBadRequest},
		{"超过上限", `{"clusterId":"c1","namespace":"default","name":"web","replicas":1000}`, http.StatusBadRequest},
		{"明确缩到 0", `{"clusterId":"c1","namespace":"default","name":"web","replicas":0}`, http.StatusOK},
		{"正常", `{"clusterId":"c1","namespace":"default","name":"web","replicas":3}`, http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ops := &recordingOps{}
			rec := postOps(t, NewOpsHandler(ops).DeploymentScale, c.body)
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d（%s）", rec.Code, c.wantStatus, rec.Body.String())
			}
			if (len(ops.created) == 1) != (c.wantStatus == http.StatusOK) {
				t.Errorf("创建了 %d 条指令", len(ops.created))
			}
		})
	}
}

func TestDeploymentImage_RejectsInvalidReference(t *testing.T) {
	ops := &recordingOps{}
	body := `{"clusterId":"c1","namespace":"default","name":"web","container":"app","image":"nginx\",\"securityContext\":{\"privileged\":true}"}`
	rec := postOps(t, NewOpsHandler(ops).DeploymentImage, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if len(ops.created) != 0 {
		t.Error("不合法的镜像引用不应创建指令")
	}

	ok := postOps(t, NewOpsHandler(ops).DeploymentImage, `{"clusterId":"c1","namespace":"default","name":"web","container":"app","image":"nginx:1.27"}`)
	if ok.Code != http.StatusOK || len(ops.created) != 1 {
		t.Fatalf("合法请求 status = %d，创建 %d 条", ok.Code, len(ops.created))
	}
}
