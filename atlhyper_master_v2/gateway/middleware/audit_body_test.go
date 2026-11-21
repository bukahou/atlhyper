package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"AtlHyper/atlhyper_master_v2/database"
)

func waitRecord(t *testing.T, repo *recordingAuditRepo) *database.AuditLog {
	t.Helper()
	select {
	case record := <-repo.records:
		return record
	case <-time.After(2 * time.Second):
		t.Fatal("没有写入审计记录")
		return nil
	}
}

// 权限检查没通过时处理函数不读请求体：审计也不读、不记（不替未鉴权的请求把整个请求体读进内存）
func TestAudit_DeniedRequestBodyIsNotRead(t *testing.T) {
	repo := &recordingAuditRepo{records: make(chan *database.AuditLog, 1)}
	read := false
	body := &trackingReader{Reader: strings.NewReader(`{"name":"x"}`), onRead: func() { read = true }}
	h := Audit(repo, AuditConfig{Action: "update", Resource: "user"})(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/user/update-role", io.NopCloser(body))
	h(httptest.NewRecorder(), req)

	record := waitRecord(t, repo)
	if read {
		t.Error("被拒绝的请求，审计不应读取请求体")
	}
	if record.RequestBody != "" {
		t.Errorf("RequestBody = %q, want 空", record.RequestBody)
	}
	if record.Success {
		t.Error("403 应记为失败")
	}
}

// 处理函数读了多少就记多少（脱敏后）
func TestAudit_RecordsBodyTheHandlerRead(t *testing.T) {
	repo := &recordingAuditRepo{records: make(chan *database.AuditLog, 1)}
	h := Audit(repo, AuditConfig{Action: "update", Resource: "user"})(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(`{"userId":3,"password":"p"}`))
	h(httptest.NewRecorder(), req)

	record := waitRecord(t, repo)
	if !strings.Contains(record.RequestBody, `"userId":3`) || strings.Contains(record.RequestBody, `"p"`) {
		t.Errorf("RequestBody = %q", record.RequestBody)
	}
}

// 读进来的请求体超过记录上限：不记内容（截断的 JSON 没法可靠脱敏）
func TestAudit_OversizedBodyNotRecorded(t *testing.T) {
	repo := &recordingAuditRepo{records: make(chan *database.AuditLog, 1)}
	h := Audit(repo, AuditConfig{Action: "update", Resource: "x"})(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	big := `{"data":"` + strings.Repeat("a", maxAuditCaptureBytes) + `"}`
	h(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(big)))

	record := waitRecord(t, repo)
	if strings.Contains(record.RequestBody, "aaaa") {
		t.Errorf("超大请求体不应记录内容，实际 %q", record.RequestBody[:40])
	}
}

func TestAudit_UsesConfiguredClientIPHeader(t *testing.T) {
	repo := &recordingAuditRepo{records: make(chan *database.AuditLog, 1)}
	h := AuditWithClientIP(repo, AuditConfig{Action: "login", Resource: "user"}, "CF-Connecting-IP")(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	req.Header.Set("X-Forwarded-For", "6.6.6.6")
	req.Header.Set("CF-Connecting-IP", "203.0.113.9")
	h(httptest.NewRecorder(), req)

	if record := waitRecord(t, repo); record.IP != "203.0.113.9" {
		t.Errorf("IP = %q, want 203.0.113.9（X-Forwarded-For 不可信）", record.IP)
	}
}

func TestSanitizeRequestBody_RedactsManifests(t *testing.T) {
	got := sanitizeRequestBody(`{"path":"apps/web","manifests":"kind: Secret\nstringData:\n  password: hunter2"}`)
	if strings.Contains(got, "hunter2") || !strings.Contains(got, `"apps/web"`) {
		t.Errorf("sanitize = %q", got)
	}
}

// 截断按字符：不会把一个多字节字符切成两半
func TestSanitizeRequestBody_TruncatesOnRuneBoundary(t *testing.T) {
	got := sanitizeRequestBody(`{"note":"` + strings.Repeat("审", maxAuditBodyLen) + `"}`)
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("应被截断，实际长度 %d", len(got))
	}
	if !utf8Valid(got) {
		t.Error("截断结果不是合法的 UTF-8")
	}
}

func TestLimitBody(t *testing.T) {
	var readErr error
	h := LimitBody(16)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("x", 17))))
	var maxErr *http.MaxBytesError
	if readErr == nil || !errorsAs(readErr, &maxErr) {
		t.Fatalf("超过上限的请求体读取应返回 MaxBytesError，实际 %v", readErr)
	}

	readErr = nil
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("x", 16))))
	if readErr != nil {
		t.Fatalf("不超过上限时应正常读取，实际 %v", readErr)
	}
}

type trackingReader struct {
	io.Reader
	onRead func()
}

func (r *trackingReader) Read(p []byte) (int, error) {
	r.onRead()
	return r.Reader.Read(p)
}

func utf8Valid(s string) bool { return utf8.ValidString(s) }

func errorsAs(err error, target any) bool { return errors.As(err, target) }
