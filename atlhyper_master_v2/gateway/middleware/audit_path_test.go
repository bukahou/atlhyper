package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// recordingAuditRepo 把写入的审计记录送进 channel（中间件是异步写的）
type recordingAuditRepo struct {
	records chan *database.AuditLog
}

func (r *recordingAuditRepo) Create(_ context.Context, record *database.AuditLog) error {
	r.records <- record
	return nil
}

// 被操作的对象常常只出现在路径里（如 /admin/clusters/{id}/token）。
// 审计必须记下路径，否则看不出操作的是哪一个；查询串可能带凭据，不记。
func TestAudit_RecordsRequestPathWithoutQuery(t *testing.T) {
	repo := &recordingAuditRepo{records: make(chan *database.AuditLog, 1)}
	handler := Audit(repo, AuditConfig{Action: "execute", Resource: "cluster"})(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/admin/clusters/c1/token?access_token=should-not-be-stored", nil)
	handler(httptest.NewRecorder(), req)

	select {
	case record := <-repo.records:
		if record.Path != "/api/v2/admin/clusters/c1/token" {
			t.Errorf("审计记录的路径 = %q，期望不含查询串的请求路径", record.Path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有写入审计记录")
	}
}
