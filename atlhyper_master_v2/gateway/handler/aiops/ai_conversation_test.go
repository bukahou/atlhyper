package aiops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	aiPkg "AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
)

// fakeConvAI 只实现会话读删；记录调用方传入的用户，模拟「会话属于用户 1」
type fakeConvAI struct {
	aiPkg.AIService
	gotUser int64
}

func (f *fakeConvAI) GetMessages(_ context.Context, userID, _ int64) ([]*aiPkg.Message, error) {
	f.gotUser = userID
	if userID != 1 {
		return nil, aiPkg.ErrConversationNotFound
	}
	return []*aiPkg.Message{}, nil
}

func (f *fakeConvAI) DeleteConversation(_ context.Context, userID, _ int64) error {
	f.gotUser = userID
	if userID != 1 {
		return aiPkg.ErrConversationNotFound
	}
	return nil
}

func serveConv(ai *fakeConvAI, method, path string, p middleware.Principal) int {
	req := httptest.NewRequest(method, path, nil)
	req = req.WithContext(middleware.WithPrincipal(req.Context(), p))
	rec := httptest.NewRecorder()
	NewAIHandler(ai).ConversationByID(rec, req)
	return rec.Code
}

// 别人的会话与不存在的会话同样 404；handler 把当前用户交给 service 判断归属
func TestConversationByID_ForeignConversationIs404(t *testing.T) {
	owner := middleware.Principal{UserID: 1, Level: middleware.RoleOperator}
	other := middleware.Principal{UserID: 2, Level: middleware.RoleOperator}
	cases := []struct {
		method, path string
		p            middleware.Principal
		want         int
	}{
		{http.MethodGet, "/api/v2/ai/conversations/10/messages", owner, http.StatusOK},
		{http.MethodGet, "/api/v2/ai/conversations/10/messages", other, http.StatusNotFound},
		{http.MethodDelete, "/api/v2/ai/conversations/10", owner, http.StatusOK},
		{http.MethodDelete, "/api/v2/ai/conversations/10", other, http.StatusNotFound},
	}
	for _, c := range cases {
		ai := &fakeConvAI{}
		if got := serveConv(ai, c.method, c.path, c.p); got != c.want {
			t.Errorf("user %d %s %s = %d, want %d", c.p.UserID, c.method, c.path, got, c.want)
		}
		if ai.gotUser != c.p.UserID {
			t.Errorf("service 收到的用户 = %d, want %d", ai.gotUser, c.p.UserID)
		}
	}
}
