package aiops

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/aiops/enricher"
	"AtlHyper/atlhyper_master_v2/service"
)

type stubSummarizeSvc struct {
	service.Query
	err error
}

func (s *stubSummarizeSvc) SummarizeIncident(context.Context, string) (*enricher.SummarizeResponse, error) {
	return nil, s.err
}

// B1：失败时给前端稳定的错误码（前端据此显示可操作的 i18n 文案）；⛔ 不把上游原始错误文本透传给页面
func TestSummarize_ErrorCodes(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"角色未分配", fmt.Errorf("LLM 调用失败: AI 配置错误: %w", ai.ErrRoleNotAssigned), http.StatusConflict, `"code":"ai_role_unassigned"`},
		{"预算耗尽", fmt.Errorf("x: %w", ai.ErrBudgetExhausted), http.StatusConflict, `"code":"ai_budget_exhausted"`},
		{"上游失败", errors.New("AccessDeniedException: Invalid API Key format RequestID: 8c8e"), http.StatusBadGateway, `"code":"ai_upstream"`},
	}
	for _, c := range cases {
		h := NewAIOpsAIHandler(&stubSummarizeSvc{err: c.err})
		rec := httptest.NewRecorder()
		h.Summarize(rec, httptest.NewRequest(http.MethodPost, "/api/v2/aiops/ai/summarize", strings.NewReader(`{"incidentId":"inc-1"}`)))
		if rec.Code != c.wantCode || !strings.Contains(rec.Body.String(), c.wantBody) {
			t.Errorf("%s: %d %s", c.name, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "RequestID") || strings.Contains(rec.Body.String(), "AccessDenied") {
			t.Errorf("%s: 不得透传上游原始错误：%s", c.name, rec.Body.String())
		}
	}
}
