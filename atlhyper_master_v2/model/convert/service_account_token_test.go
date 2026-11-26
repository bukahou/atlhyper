package convert

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

func TestAPITokenView_Status(t *testing.T) {
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	past, future := now.Add(-time.Hour), now.Add(time.Hour)

	cases := []struct {
		name      string
		expiresAt *time.Time
		revokedAt *time.Time
		want      string
	}{
		{"不过期", nil, nil, model.APITokenStatusActive},
		{"未到期", &future, nil, model.APITokenStatusActive},
		{"到期的那一刻", &now, nil, model.APITokenStatusExpired},
		{"已过期", &past, nil, model.APITokenStatusExpired},
		{"已吊销", nil, &past, model.APITokenStatusRevoked},
		{"过期后又吊销：吊销优先", &past, &now, model.APITokenStatusRevoked},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			token := &database.APIToken{ID: 1, ExpiresAt: c.expiresAt, RevokedAt: c.revokedAt}
			if got := APITokenView(token, now).Status; got != c.want {
				t.Errorf("status = %s, want %s", got, c.want)
			}
		})
	}
}

func TestServiceAccountView(t *testing.T) {
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	// 带时区的时间：输出必须统一换成 UTC
	tokyo := time.FixedZone("JST", 9*3600)
	created := time.Date(2026, 11, 20, 9, 0, 0, 0, tokyo)
	expires := time.Date(2027, 2, 18, 9, 0, 0, 0, tokyo)
	usedEarlier := time.Date(2026, 11, 25, 9, 0, 0, 0, tokyo)
	usedLater := time.Date(2026, 11, 26, 18, 30, 0, 0, tokyo)
	revoked := time.Date(2026, 11, 26, 8, 0, 0, 0, time.UTC)

	account := &database.User{
		ID: 7, Username: "haro", DisplayName: "Haro", Role: 1, Status: database.UserStatusActive,
		Kind: database.UserKindService, CreatedAt: created,
	}
	tokens := []*database.APIToken{
		{ID: 2, UserID: 7, Name: "new", TokenHash: "not-a-real-hash", TokenPrefix: "atlp_Ab3", ExpiresAt: &expires,
			LastUsedAt: &usedEarlier, CreatedBy: "admin", CreatedAt: created},
		{ID: 1, UserID: 7, Name: "old", TokenHash: "not-a-real-hash", TokenPrefix: "atlp_Cd4", LastUsedAt: &usedLater,
			CreatedBy: "admin", CreatedAt: created, RevokedAt: &revoked},
	}

	got := ServiceAccountView(account, tokens, true, now)
	want := model.ServiceAccountView{
		ID: 7, Name: "haro", DisplayName: "Haro", Role: 1, Enabled: true,
		CreatedAt: "2026-11-20T00:00:00Z", LastUsedAt: "2026-11-26T09:30:00Z", CanIssueToken: true,
		Tokens: []model.APITokenView{
			{ID: 2, Name: "new", TokenPrefix: "atlp_Ab3", Status: model.APITokenStatusActive, CreatedBy: "admin",
				CreatedAt: "2026-11-20T00:00:00Z", ExpiresAt: "2027-02-18T00:00:00Z", LastUsedAt: "2026-11-25T00:00:00Z"},
			{ID: 1, Name: "old", TokenPrefix: "atlp_Cd4", Status: model.APITokenStatusRevoked, CreatedBy: "admin",
				CreatedAt: "2026-11-20T00:00:00Z", LastUsedAt: "2026-11-26T09:30:00Z", RevokedAt: "2026-11-26T08:00:00Z"},
		},
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("\n got %s\nwant %s", gotJSON, wantJSON)
	}
	// 哈希不出库
	if strings.Contains(string(gotJSON), "not-a-real-hash") {
		t.Error("结果里不应有令牌哈希")
	}
}

// 没有令牌：tokens 是空数组不是 null；从未使用；停用的账号 enabled 为假
func TestServiceAccountView_NoTokens(t *testing.T) {
	account := &database.User{ID: 8, Username: "idle", Status: database.UserStatusDisabled, Kind: database.UserKindService}
	got := ServiceAccountView(account, nil, false, time.Now())
	if got.Tokens == nil || len(got.Tokens) != 0 {
		t.Errorf("tokens = %#v, want 空数组", got.Tokens)
	}
	if got.LastUsedAt != "" || got.Enabled || got.CanIssueToken {
		t.Errorf("view = %+v", got)
	}
	body, _ := json.Marshal(got)
	if !strings.Contains(string(body), `"tokens":[]`) {
		t.Errorf("JSON 里 tokens 应为 []：%s", body)
	}
}
