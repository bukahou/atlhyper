package operations

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/auth/apitoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

var serviceAccountOpsNow = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

// fakeServiceAccountWriter 内存里的服务账号；err 非空时写操作返回它
type fakeServiceAccountWriter struct {
	accounts map[int64]*database.User
	created  []*database.User
	updated  []*database.User
	deleted  []int64
	err      error
}

func (f *fakeServiceAccountWriter) Create(_ context.Context, account *database.User) error {
	if f.err != nil {
		return f.err
	}
	copied := *account
	f.created = append(f.created, &copied)
	account.ID = 41
	return nil
}

func (f *fakeServiceAccountWriter) GetByID(_ context.Context, id int64) (*database.User, error) {
	return f.accounts[id], nil
}

func (f *fakeServiceAccountWriter) Update(_ context.Context, account *database.User) error {
	if f.err != nil {
		return f.err
	}
	copied := *account
	f.updated = append(f.updated, &copied)
	return nil
}

func (f *fakeServiceAccountWriter) Delete(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

type revokeCall struct {
	userID, tokenID int64
	at              time.Time
}

type fakeAPITokenWriter struct {
	created []*database.APIToken
	limits  []int
	revoked []revokeCall
	touched []int64
	err     error
}

func (f *fakeAPITokenWriter) CreateWithinLimit(_ context.Context, token *database.APIToken, limit int) error {
	if f.err != nil {
		return f.err
	}
	copied := *token
	f.created = append(f.created, &copied)
	f.limits = append(f.limits, limit)
	token.ID = 9
	return nil
}

func (f *fakeAPITokenWriter) Revoke(_ context.Context, userID, tokenID int64, at time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.revoked = append(f.revoked, revokeCall{userID: userID, tokenID: tokenID, at: at})
	return nil
}

func (f *fakeAPITokenWriter) Touch(_ context.Context, id int64, _ time.Time) error {
	f.touched = append(f.touched, id)
	return f.err
}

func newServiceAccountServiceForTest() (*ServiceAccountService, *fakeServiceAccountWriter, *fakeAPITokenWriter) {
	accounts := &fakeServiceAccountWriter{accounts: map[int64]*database.User{
		7: {ID: 7, Username: "haro", Kind: database.UserKindService, Status: database.UserStatusActive},
	}}
	tokens := &fakeAPITokenWriter{}
	svc := NewServiceAccountService(accounts, tokens)
	svc.now = func() time.Time { return serviceAccountOpsNow }
	return svc, accounts, tokens
}

func role(v int) *int { return &v }

func TestCreateServiceAccount(t *testing.T) {
	svc, accounts, _ := newServiceAccountServiceForTest()
	id, err := svc.CreateServiceAccount(context.Background(), &model.CreateServiceAccountRequest{
		Name: "haro-prod", DisplayName: "  Haro 语音助手  ", Role: role(0),
	})
	if err != nil || id != 41 {
		t.Fatalf("id = %d, err = %v", id, err)
	}
	if len(accounts.created) != 1 {
		t.Fatal("应写入一个账号")
	}
	got := accounts.created[0]
	if got.Username != "haro-prod" || got.DisplayName != "Haro 语音助手" || got.Role != 0 || got.Status != database.UserStatusActive {
		t.Errorf("写入的账号不对（显示名应去掉首尾空白，新建即启用）：%+v", got)
	}
}

func TestCreateServiceAccount_Rejected(t *testing.T) {
	cases := []struct {
		name string
		req  model.CreateServiceAccountRequest
	}{
		{"等级 3 一律拒绝", model.CreateServiceAccountRequest{Name: "haro", Role: role(3)}},
		{"没给等级", model.CreateServiceAccountRequest{Name: "haro"}},
		{"账号名太短", model.CreateServiceAccountRequest{Name: "ha", Role: role(0)}},
		{"账号名有大写", model.CreateServiceAccountRequest{Name: "Haro", Role: role(0)}},
		{"账号名数字开头", model.CreateServiceAccountRequest{Name: "9haro", Role: role(0)}},
		{"显示名太长", model.CreateServiceAccountRequest{Name: "haro", DisplayName: strings.Repeat("语", 65), Role: role(0)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, accounts, _ := newServiceAccountServiceForTest()
			if _, err := svc.CreateServiceAccount(context.Background(), &c.req); !errors.Is(err, model.ErrInvalidServiceAccountInput) {
				t.Errorf("err = %v, want ErrInvalidServiceAccountInput", err)
			}
			if len(accounts.created) != 0 {
				t.Error("不合规时不应写库")
			}
		})
	}
}

// 重名由仓库判定（与人的用户名同一命名空间），错误原样上报
func TestCreateServiceAccount_NameTaken(t *testing.T) {
	svc, accounts, _ := newServiceAccountServiceForTest()
	accounts.err = database.ErrUsernameTaken
	_, err := svc.CreateServiceAccount(context.Background(), &model.CreateServiceAccountRequest{Name: "admin", Role: role(0)})
	if !errors.Is(err, database.ErrUsernameTaken) {
		t.Errorf("err = %v, want ErrUsernameTaken", err)
	}
}

func TestUpdateServiceAccount(t *testing.T) {
	svc, accounts, _ := newServiceAccountServiceForTest()
	displayName, enabled := " Haro ", false
	err := svc.UpdateServiceAccount(context.Background(), 7, &model.UpdateServiceAccountRequest{
		DisplayName: &displayName, Role: role(2), Enabled: &enabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := accounts.updated[0]
	if got.ID != 7 || got.DisplayName != "Haro" || got.Role != 2 || got.Status != database.UserStatusDisabled {
		t.Errorf("写入的修改不对：%+v", got)
	}

	tooHigh := 3
	err = svc.UpdateServiceAccount(context.Background(), 7, &model.UpdateServiceAccountRequest{
		DisplayName: &displayName, Role: &tooHigh, Enabled: &enabled,
	})
	if !errors.Is(err, model.ErrInvalidServiceAccountInput) || len(accounts.updated) != 1 {
		t.Errorf("等级 3：err = %v，updated = %d", err, len(accounts.updated))
	}

	accounts.err = database.ErrServiceAccountNotFound
	err = svc.UpdateServiceAccount(context.Background(), 8, &model.UpdateServiceAccountRequest{
		DisplayName: &displayName, Role: role(0), Enabled: &enabled,
	})
	if !errors.Is(err, database.ErrServiceAccountNotFound) {
		t.Errorf("不存在：err = %v", err)
	}
}

func TestDeleteServiceAccount(t *testing.T) {
	svc, accounts, _ := newServiceAccountServiceForTest()
	if err := svc.DeleteServiceAccount(context.Background(), 7); err != nil || len(accounts.deleted) != 1 || accounts.deleted[0] != 7 {
		t.Errorf("err = %v deleted = %v", err, accounts.deleted)
	}
}

func TestIssueAPIToken(t *testing.T) {
	svc, _, tokens := newServiceAccountServiceForTest()
	days := 90
	issued, err := svc.IssueAPIToken(context.Background(), 7, &model.IssueAPITokenRequest{Name: " prod ", ExpiresInDays: &days}, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(issued.Token, "atlp_") || !strings.HasPrefix(issued.Token, issued.TokenPrefix) {
		t.Errorf("明文与前缀不对：prefix = %q", issued.TokenPrefix)
	}
	if issued.ExpiresAt != "2027-02-24T10:00:00Z" {
		t.Errorf("expiresAt = %q, want 90 天后", issued.ExpiresAt)
	}
	if len(tokens.created) != 1 || tokens.limits[0] != model.MaxActiveAPITokensPerAccount {
		t.Fatalf("应按上限 %d 写入一枚：%+v %v", model.MaxActiveAPITokensPerAccount, tokens.created, tokens.limits)
	}
	stored := tokens.created[0]
	if stored.UserID != 7 || stored.Name != "prod" || stored.CreatedBy != "admin" || !stored.CreatedAt.Equal(serviceAccountOpsNow) ||
		stored.TokenPrefix != issued.TokenPrefix || stored.ExpiresAt == nil || !stored.ExpiresAt.Equal(serviceAccountOpsNow.Add(90*24*time.Hour)) {
		t.Errorf("入库的令牌不对：%+v", stored)
	}
	// 库里只有哈希
	if stored.TokenHash != apitoken.Hash(issued.Token) || strings.Contains(stored.TokenHash, issued.Token[len("atlp_"):]) {
		t.Error("入库的应是明文的哈希")
	}
}

func TestIssueAPIToken_NoExpiry(t *testing.T) {
	svc, _, tokens := newServiceAccountServiceForTest()
	issued, err := svc.IssueAPIToken(context.Background(), 7, &model.IssueAPITokenRequest{Name: "prod"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if issued.ExpiresAt != "" || tokens.created[0].ExpiresAt != nil {
		t.Errorf("不给有效期 = 不过期：%q %v", issued.ExpiresAt, tokens.created[0].ExpiresAt)
	}
}

func TestIssueAPIToken_Failures(t *testing.T) {
	cases := []struct {
		name      string
		accountID int64
		req       model.IssueAPITokenRequest
		writeErr  error
		want      error
	}{
		{"账号不存在（或不是服务账号）", 999, model.IssueAPITokenRequest{Name: "prod"}, nil, database.ErrServiceAccountNotFound},
		{"名称为空", 7, model.IssueAPITokenRequest{Name: "   "}, nil, model.ErrInvalidServiceAccountInput},
		{"有效期超上限", 7, model.IssueAPITokenRequest{Name: "prod", ExpiresInDays: role(model.MaxAPITokenExpiresInDays + 1)}, nil, model.ErrInvalidServiceAccountInput},
		{"未吊销的已达上限", 7, model.IssueAPITokenRequest{Name: "prod"}, database.ErrAPITokenLimitReached, database.ErrAPITokenLimitReached},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, _, tokens := newServiceAccountServiceForTest()
			tokens.err = c.writeErr
			issued, err := svc.IssueAPIToken(context.Background(), c.accountID, &c.req, "admin")
			if !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
			if issued != nil {
				t.Error("失败时不应返回令牌")
			}
		})
	}
}

func TestRevokeAndTouchAPIToken(t *testing.T) {
	svc, _, tokens := newServiceAccountServiceForTest()
	if err := svc.RevokeAPIToken(context.Background(), 7, 9); err != nil {
		t.Fatal(err)
	}
	if len(tokens.revoked) != 1 || tokens.revoked[0] != (revokeCall{userID: 7, tokenID: 9, at: serviceAccountOpsNow}) {
		t.Errorf("revoked = %+v", tokens.revoked)
	}
	if err := svc.TouchAPIToken(context.Background(), 9, serviceAccountOpsNow); err != nil || len(tokens.touched) != 1 {
		t.Errorf("touch: err = %v touched = %v", err, tokens.touched)
	}

	tokens.err = database.ErrAPITokenNotFound
	if err := svc.RevokeAPIToken(context.Background(), 7, 999); !errors.Is(err, database.ErrAPITokenNotFound) {
		t.Errorf("不存在：err = %v", err)
	}
}
