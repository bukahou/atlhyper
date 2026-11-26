package query

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

var serviceAccountTestNow = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

type fakeServiceAccountReader struct {
	accounts []*database.User
	err      error
}

func (f *fakeServiceAccountReader) List(context.Context) ([]*database.User, error) {
	return f.accounts, f.err
}

func (f *fakeServiceAccountReader) GetByID(_ context.Context, id int64) (*database.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, account := range f.accounts {
		if account.ID == id {
			return account, nil
		}
	}
	return nil, nil
}

type fakeAPITokenReader struct {
	tokens []*database.APIToken
	err    error
}

func (f *fakeAPITokenReader) List(context.Context) ([]*database.APIToken, error) {
	return f.tokens, f.err
}

func (f *fakeAPITokenReader) ListByUser(_ context.Context, userID int64) ([]*database.APIToken, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*database.APIToken
	for _, token := range f.tokens {
		if token.UserID == userID {
			out = append(out, token)
		}
	}
	return out, nil
}

func (f *fakeAPITokenReader) GetByHash(_ context.Context, tokenHash string) (*database.APIToken, error) {
	for _, token := range f.tokens {
		if token.TokenHash == tokenHash {
			return token, nil
		}
	}
	return nil, f.err
}

func newServiceAccountQueryForTest(accounts *fakeServiceAccountReader, tokens *fakeAPITokenReader) *ServiceAccountQuery {
	return NewServiceAccountQuery(accounts, tokens, func() time.Time { return serviceAccountTestNow })
}

func serviceAccount(id int64, name string) *database.User {
	return &database.User{ID: id, Username: name, Status: database.UserStatusActive, Kind: database.UserKindService}
}

// tokensFor n 枚令牌，其中前 revoked 枚已吊销
func tokensFor(userID int64, n, revoked int) []*database.APIToken {
	tokens := make([]*database.APIToken, 0, n)
	for i := 0; i < n; i++ {
		token := &database.APIToken{ID: userID*100 + int64(i), UserID: userID, Name: fmt.Sprintf("t%d", i), TokenHash: fmt.Sprintf("h-%d-%d", userID, i)}
		if i < revoked {
			at := serviceAccountTestNow.Add(-time.Hour)
			token.RevokedAt = &at
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func TestListServiceAccounts_GroupsTokensAndJudgesLimit(t *testing.T) {
	limit := model.MaxActiveAPITokensPerAccount
	var tokens []*database.APIToken
	tokens = append(tokens, tokensFor(1, limit, 0)...)   // 满了
	tokens = append(tokens, tokensFor(2, limit+2, 3)...) // 吊销掉 3 枚，未吊销的 limit-1 枚
	q := newServiceAccountQueryForTest(
		&fakeServiceAccountReader{accounts: []*database.User{serviceAccount(1, "full"), serviceAccount(2, "room"), serviceAccount(3, "empty")}},
		&fakeAPITokenReader{tokens: tokens},
	)

	views, err := q.ListServiceAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 3 || views[0].Name != "full" || views[1].Name != "room" || views[2].Name != "empty" {
		t.Fatalf("顺序应与仓库一致：%+v", views)
	}
	if len(views[0].Tokens) != limit || views[0].CanIssueToken {
		t.Errorf("未吊销的已达上限：tokens=%d canIssue=%v", len(views[0].Tokens), views[0].CanIssueToken)
	}
	if len(views[1].Tokens) != limit+2 || !views[1].CanIssueToken {
		t.Errorf("吊销的不计入上限：tokens=%d canIssue=%v", len(views[1].Tokens), views[1].CanIssueToken)
	}
	if views[1].Tokens[0].Status != model.APITokenStatusRevoked || views[1].Tokens[limit+1].Status != model.APITokenStatusActive {
		t.Errorf("令牌状态应由后端判定：%+v", views[1].Tokens)
	}
	if views[2].Tokens == nil || len(views[2].Tokens) != 0 || !views[2].CanIssueToken {
		t.Errorf("没有令牌的账号：%+v", views[2])
	}
}

func TestListServiceAccounts_Empty(t *testing.T) {
	q := newServiceAccountQueryForTest(&fakeServiceAccountReader{}, &fakeAPITokenReader{})
	views, err := q.ListServiceAccounts(context.Background())
	if err != nil || views == nil || len(views) != 0 {
		t.Errorf("没有服务账号时应返回空切片：%#v, %v", views, err)
	}
}

func TestListServiceAccounts_ReadFailures(t *testing.T) {
	down := errors.New("database is locked")
	cases := map[string]*ServiceAccountQuery{
		"账号": newServiceAccountQueryForTest(&fakeServiceAccountReader{err: down}, &fakeAPITokenReader{}),
		"令牌": newServiceAccountQueryForTest(&fakeServiceAccountReader{accounts: []*database.User{serviceAccount(1, "a")}}, &fakeAPITokenReader{err: down}),
	}
	for name, q := range cases {
		if _, err := q.ListServiceAccounts(context.Background()); !errors.Is(err, down) {
			t.Errorf("%s读取失败：err = %v", name, err)
		}
	}
}

func TestGetServiceAccount(t *testing.T) {
	q := newServiceAccountQueryForTest(
		&fakeServiceAccountReader{accounts: []*database.User{serviceAccount(1, "haro"), serviceAccount(2, "other")}},
		&fakeAPITokenReader{tokens: append(tokensFor(1, 2, 0), tokensFor(2, 1, 0)...)},
	)
	view, err := q.GetServiceAccount(context.Background(), 1)
	if err != nil || view == nil {
		t.Fatalf("GetServiceAccount = %v, %v", view, err)
	}
	if view.Name != "haro" || len(view.Tokens) != 2 || !view.CanIssueToken {
		t.Errorf("view = %+v", view)
	}
	if missing, err := q.GetServiceAccount(context.Background(), 999); err != nil || missing != nil {
		t.Errorf("不存在应返回 (nil, nil)：%+v, %v", missing, err)
	}
}

func TestGetAPITokenByHash(t *testing.T) {
	q := newServiceAccountQueryForTest(&fakeServiceAccountReader{}, &fakeAPITokenReader{tokens: tokensFor(1, 1, 0)})
	token, err := q.GetAPITokenByHash(context.Background(), "h-1-0")
	if err != nil || token == nil || token.ID != 100 {
		t.Errorf("GetAPITokenByHash = %+v, %v", token, err)
	}
}
