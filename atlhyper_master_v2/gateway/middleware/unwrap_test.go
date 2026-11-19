package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type nopAudit struct{}

func (nopAudit) Create(context.Context, *database.AuditLog) error { return nil }

// 流式响应要在处理器里逐块续期写超时：经过日志、审计两层包装后，http.ResponseController 仍要能拿到底层连接
func TestResponseWrappers_SupportWriteDeadline(t *testing.T) {
	result := make(chan error, 1)
	inner := func(w http.ResponseWriter, _ *http.Request) {
		result <- http.NewResponseController(w).SetWriteDeadline(time.Now().Add(time.Minute))
	}
	h := Logging(Audit(nopAudit{}, AuditConfig{Action: "x", Resource: "y"})(inner))
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if err := <-result; err != nil {
		t.Fatalf("经过包装后设置写超时失败: %v", err)
	}
}
