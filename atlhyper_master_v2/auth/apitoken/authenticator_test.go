package apitoken

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// 固定的「现在」：令牌的到期、最近使用都相对它构造
var testNow = time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)

type touchCall struct {
	id     int64
	usedAt time.Time
}

// fakeTokenStore 按哈希存令牌；err 非空时每次查询都返回它
type fakeTokenStore struct {
	byHash   map[string]*database.APIToken
	err      error
	touchErr error
	lookups  int
	touched  []touchCall
}

func (s *fakeTokenStore) GetAPITokenByHash(_ context.Context, tokenHash string) (*database.APIToken, error) {
	s.lookups++
	if s.err != nil {
		return nil, s.err
	}
	token, ok := s.byHash[tokenHash]
	if !ok {
		return nil, nil
	}
	copied := *token
	return &copied, nil
}

func (s *fakeTokenStore) TouchAPIToken(_ context.Context, id int64, usedAt time.Time) error {
	s.touched = append(s.touched, touchCall{id: id, usedAt: usedAt})
	return s.touchErr
}

type fakeUserLoader struct {
	users map[int64]*database.User
	err   error
}

func (l *fakeUserLoader) GetUserByID(_ context.Context, id int64) (*database.User, error) {
	if l.err != nil {
		return nil, l.err
	}
	user, ok := l.users[id]
	if !ok {
		return nil, nil
	}
	copied := *user
	return &copied, nil
}

func timePtr(t time.Time) *time.Time { return &t }

// fixture 一枚有效令牌（属于启用中的服务账号 7）与它的明文
func fixture(t *testing.T, edit func(token *database.APIToken, account *database.User)) (*fakeTokenStore, *fakeUserLoader, string) {
	t.Helper()
	issued, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	token := &database.APIToken{
		ID: 42, UserID: 7, Name: "haro-prod", TokenHash: issued.Hash, TokenPrefix: issued.Prefix,
		CreatedBy: "admin", CreatedAt: testNow.Add(-24 * time.Hour),
	}
	account := &database.User{
		ID: 7, Username: "haro", Role: 0, Status: 1, Kind: database.UserKindService,
	}
	if edit != nil {
		edit(token, account)
	}
	store := &fakeTokenStore{byHash: map[string]*database.APIToken{issued.Hash: token}}
	users := &fakeUserLoader{users: map[int64]*database.User{account.ID: account}}
	return store, users, issued.Plaintext
}

func authenticate(store *fakeTokenStore, users *fakeUserLoader, plaintext string) (Caller, error) {
	return NewAuthenticator(store, users, func() time.Time { return testNow }).Authenticate(context.Background(), plaintext)
}

func TestGenerate_UsesAccessTokenPrefix(t *testing.T) {
	issued, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(issued.Plaintext, "atlp_") || !strings.HasPrefix(issued.Prefix, "atlp_") {
		t.Errorf("访问令牌应以 atlp_ 开头：%q", issued.Prefix)
	}
	if issued.Hash != Hash(issued.Plaintext) {
		t.Error("Hash 与签发时入库的哈希不一致")
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	expiresAt := testNow.Add(30 * 24 * time.Hour)
	store, users, plaintext := fixture(t, func(token *database.APIToken, account *database.User) {
		token.ExpiresAt = timePtr(expiresAt)
		account.Role = 2
	})

	caller, err := authenticate(store, users, plaintext)
	if err != nil {
		t.Fatalf("有效令牌应通过：%v", err)
	}
	want := Caller{UserID: 7, Username: "haro", Level: 2, TokenID: 42, TokenName: "haro-prod", ExpiresAt: expiresAt}
	if caller != want {
		t.Errorf("Caller = %+v\nwant     %+v", caller, want)
	}
}

func TestAuthenticate_NoExpiryMeansZeroExpiresAt(t *testing.T) {
	store, users, plaintext := fixture(t, nil)
	caller, err := authenticate(store, users, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if !caller.ExpiresAt.IsZero() {
		t.Errorf("不过期的令牌 ExpiresAt 应为零值，实际 %v", caller.ExpiresAt)
	}
}

// 等级每次从库里读：账号的等级改了，同一枚令牌下一次校验就是新等级
func TestAuthenticate_LevelComesFromCurrentAccount(t *testing.T) {
	store, users, plaintext := fixture(t, nil)
	if caller, _ := authenticate(store, users, plaintext); caller.Level != 0 {
		t.Fatalf("Level = %d, want 0", caller.Level)
	}
	users.users[7].Role = 1
	if caller, _ := authenticate(store, users, plaintext); caller.Level != 1 {
		t.Errorf("改等级后 Level = %d, want 1", caller.Level)
	}
}

func TestAuthenticate_InvalidTokens(t *testing.T) {
	cases := []struct {
		name string
		// edit 改令牌或账号；plaintext 非空时出示它而不是 fixture 的明文
		edit      func(token *database.APIToken, account *database.User)
		plaintext string
	}{
		{name: "空串", plaintext: " "},
		{name: "集群令牌（前缀不对）", plaintext: "atlk_placeholder-not-a-real-token"},
		{name: "没有前缀", plaintext: "placeholder-not-a-real-token"},
		{name: "查不到", plaintext: "atlp_placeholder-not-a-real-token"},
		{name: "已吊销", edit: func(token *database.APIToken, _ *database.User) {
			token.RevokedAt = timePtr(testNow.Add(-time.Minute))
		}},
		{name: "已过期", edit: func(token *database.APIToken, _ *database.User) {
			token.ExpiresAt = timePtr(testNow.Add(-time.Second))
		}},
		{name: "到期的那一刻就失效", edit: func(token *database.APIToken, _ *database.User) {
			token.ExpiresAt = timePtr(testNow)
		}},
		{name: "账号不存在", edit: func(token *database.APIToken, _ *database.User) {
			token.UserID = 999
		}},
		{name: "账号是人不是服务账号", edit: func(_ *database.APIToken, account *database.User) {
			account.Kind = database.UserKindHuman
		}},
		{name: "账号种类为空（按人处理）", edit: func(_ *database.APIToken, account *database.User) {
			account.Kind = ""
		}},
		{name: "账号已停用", edit: func(_ *database.APIToken, account *database.User) {
			account.Status = 0
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store, users, plaintext := fixture(t, c.edit)
			if c.plaintext != "" {
				plaintext = strings.TrimSpace(c.plaintext)
			}

			caller, err := authenticate(store, users, plaintext)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("err = %v, want ErrInvalidToken", err)
			}
			if caller != (Caller{}) {
				t.Errorf("无效时不应返回调用者：%+v", caller)
			}
			if len(store.touched) != 0 {
				t.Error("无效的令牌不应记最近使用时间")
			}
		})
	}
}

// 格式就不对的令牌不去查库
func TestAuthenticate_MalformedTokenSkipsLookup(t *testing.T) {
	for _, plaintext := range []string{"", "atlk_placeholder-not-a-real-token", "Bearer atlp_x"} {
		store, users, _ := fixture(t, nil)
		if _, err := authenticate(store, users, plaintext); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%q: err = %v", plaintext, err)
		}
		if store.lookups != 0 {
			t.Errorf("%q: 不应查库", plaintext)
		}
	}
}

// 查失败与没查到是两回事：存储出错原样上报，⛔ 不算令牌无效（上层据此回 5xx 而不是 401）
func TestAuthenticate_LookupFailureIsNotInvalidToken(t *testing.T) {
	storeDown := errors.New("database is locked")
	store, users, plaintext := fixture(t, nil)
	store.err = storeDown
	if _, err := authenticate(store, users, plaintext); !errors.Is(err, storeDown) || errors.Is(err, ErrInvalidToken) {
		t.Errorf("令牌查询出错：err = %v，应能认出原错误且不是 ErrInvalidToken", err)
	}

	usersDown := errors.New("disk I/O error")
	store, users, plaintext = fixture(t, nil)
	users.err = usersDown
	if _, err := authenticate(store, users, plaintext); !errors.Is(err, usersDown) || errors.Is(err, ErrInvalidToken) {
		t.Errorf("账号查询出错：err = %v，应能认出原错误且不是 ErrInvalidToken", err)
	}
}

// 最近使用时间按库里那一行节流：距上次记录超过 1 分钟才更新
func TestAuthenticate_LastUsedThrottle(t *testing.T) {
	cases := []struct {
		name       string
		lastUsedAt *time.Time
		wantTouch  bool
	}{
		{"从未使用", nil, true},
		{"30 秒前用过", timePtr(testNow.Add(-30 * time.Second)), false},
		{"恰好 1 分钟前", timePtr(testNow.Add(-time.Minute)), false},
		{"61 秒前", timePtr(testNow.Add(-61 * time.Second)), true},
		{"一天前", timePtr(testNow.Add(-24 * time.Hour)), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store, users, plaintext := fixture(t, func(token *database.APIToken, _ *database.User) {
				token.LastUsedAt = c.lastUsedAt
			})
			if _, err := authenticate(store, users, plaintext); err != nil {
				t.Fatal(err)
			}
			if !c.wantTouch {
				if len(store.touched) != 0 {
					t.Errorf("不应更新最近使用时间：%+v", store.touched)
				}
				return
			}
			if len(store.touched) != 1 || store.touched[0] != (touchCall{id: 42, usedAt: testNow}) {
				t.Errorf("touched = %+v, want 令牌 42 于 %v", store.touched, testNow)
			}
		})
	}
}

// 记最近使用时间失败只记告警，校验照常通过
func TestAuthenticate_TouchFailureDoesNotFailAuthentication(t *testing.T) {
	store, users, plaintext := fixture(t, nil)
	store.touchErr = errors.New("database is locked")
	caller, err := authenticate(store, users, plaintext)
	if err != nil || caller.TokenID != 42 {
		t.Errorf("caller = %+v, err = %v；记录失败不应让校验失败", caller, err)
	}
}

// 错误信息里既没有令牌明文也没有哈希（错误可能进日志）
func TestAuthenticate_ErrorsDoNotLeakToken(t *testing.T) {
	cases := map[string]func(token *database.APIToken, account *database.User){
		"已吊销":   func(token *database.APIToken, _ *database.User) { token.RevokedAt = timePtr(testNow) },
		"账号已停用": func(_ *database.APIToken, account *database.User) { account.Status = 0 },
	}
	for name, edit := range cases {
		store, users, plaintext := fixture(t, edit)
		_, err := authenticate(store, users, plaintext)
		if err == nil {
			t.Fatalf("%s：应失败", name)
		}
		if strings.Contains(err.Error(), plaintext) || strings.Contains(err.Error(), Hash(plaintext)) ||
			strings.Contains(err.Error(), plaintext[len("atlp_"):]) {
			t.Errorf("%s：错误信息泄露了令牌：%v", name, err)
		}
	}

	// 查不到的令牌同样不回显
	store, users, _ := fixture(t, nil)
	const presented = "atlp_placeholder-not-a-real-token"
	_, err := authenticate(store, users, presented)
	if err == nil || strings.Contains(err.Error(), presented) || strings.Contains(err.Error(), Hash(presented)) {
		t.Errorf("错误信息泄露了令牌：%v", err)
	}
}
