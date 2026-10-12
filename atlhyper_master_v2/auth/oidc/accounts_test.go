package oidc

import (
	"context"
	"errors"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"

	"github.com/bukahou/akasha/pkg/oidcrp"
)

const testIssuer = "https://issuer.example.com"

// memAccounts 内存版账号存储：模拟唯一约束（身份唯一、用户名唯一）
type memAccounts struct {
	users      map[int64]*database.User
	identities map[string]int64 // issuer|subject → user id
	nextID     int64
	// raceOnce 第一次建号时模拟「另一个并发请求抢先建好了同一身份」
	raceOnce bool
	updates  int
}

func newMemAccounts() *memAccounts {
	return &memAccounts{users: map[int64]*database.User{}, identities: map[string]int64{}, nextID: 1}
}

func (m *memAccounts) FindUserIDByIdentity(_ context.Context, issuer, subject string) (int64, bool, error) {
	id, ok := m.identities[issuer+"|"+subject]
	return id, ok, nil
}

func (m *memAccounts) GetUserByID(_ context.Context, id int64) (*database.User, error) {
	if u, ok := m.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, nil
}

func (m *memAccounts) insert(u *database.User, issuer, subject string) {
	u.ID = m.nextID
	m.nextID++
	cp := *u
	m.users[u.ID] = &cp
	m.identities[issuer+"|"+subject] = u.ID
}

func (m *memAccounts) CreateFederatedUser(_ context.Context, u *database.User, issuer, subject string) error {
	if m.raceOnce {
		m.raceOnce = false
		m.insert(&database.User{Username: "racer", Role: 0, Status: 1}, issuer, subject)
		return database.ErrIdentityExists
	}
	if _, ok := m.identities[issuer+"|"+subject]; ok {
		return database.ErrIdentityExists
	}
	for _, existing := range m.users {
		if existing.Username == u.Username {
			return database.ErrUsernameTaken
		}
	}
	m.insert(u, issuer, subject)
	return nil
}

func (m *memAccounts) UpdateUser(_ context.Context, u *database.User) error {
	m.updates++
	cp := *u
	m.users[u.ID] = &cp
	return nil
}

func (m *memAccounts) UpdateUserLastLogin(context.Context, int64, string) error { return nil }

type denyAll struct{}

func (denyAll) Admit(oidcrp.Identity) bool { return false }

func resolver(acc *memAccounts) *accountResolver {
	return &accountResolver{accounts: acc, admission: AdmitAll{}, issuer: testIssuer, newUserRole: 0}
}

func TestResolve_FirstLoginCreatesMember(t *testing.T) {
	acc := newMemAccounts()
	u, err := resolver(acc).resolve(context.Background(), oidcrp.Identity{
		Subject: "sub-1", Email: "alice@example.com", Name: "Alice", PreferredUsername: "Alice",
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != 0 || u.Status != 1 || u.PasswordHash != "" {
		t.Errorf("首登应为 0 级、启用、无本地密码，得到 %+v", u)
	}
	if u.Username != "alice" || u.DisplayName != "Alice" || u.Email != "alice@example.com" {
		t.Errorf("资料不对：%+v", u)
	}
}

func TestResolve_ReturningUserGetsSameAccount(t *testing.T) {
	acc := newMemAccounts()
	r := resolver(acc)
	first, _ := r.resolve(context.Background(), oidcrp.Identity{Subject: "sub-1", PreferredUsername: "alice"})
	acc.users[first.ID].Role = 2 // 管理员已提升

	again, err := r.resolve(context.Background(), oidcrp.Identity{Subject: "sub-1", PreferredUsername: "alice"})
	if err != nil || again.ID != first.ID || again.Role != 2 {
		t.Errorf("再登应回到同一账号且保留角色：%+v %v", again, err)
	}
	if len(acc.users) != 1 {
		t.Errorf("不得建第二个账号，现有 %d 个", len(acc.users))
	}
}

// 展示资料以上游为准，变了才写库
func TestResolve_SyncsProfileOnlyWhenChanged(t *testing.T) {
	acc := newMemAccounts()
	r := resolver(acc)
	id := oidcrp.Identity{Subject: "sub-1", Name: "Alice", Email: "a@example.com"}
	if _, err := r.resolve(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := r.resolve(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if acc.updates != 0 {
		t.Errorf("资料未变不应写库，写了 %d 次", acc.updates)
	}
	id.Name = "Alice B."
	u, _ := r.resolve(context.Background(), id)
	if acc.updates != 1 || u.DisplayName != "Alice B." {
		t.Errorf("资料变化应同步：updates=%d user=%+v", acc.updates, u)
	}
}

func TestResolve_DisabledUserRejected(t *testing.T) {
	acc := newMemAccounts()
	r := resolver(acc)
	u, _ := r.resolve(context.Background(), oidcrp.Identity{Subject: "sub-1"})
	acc.users[u.ID].Status = 0

	if _, err := r.resolve(context.Background(), oidcrp.Identity{Subject: "sub-1"}); !errors.Is(err, errDisabled) {
		t.Errorf("err = %v，want errDisabled", err)
	}
}

func TestResolve_AdmissionRejectsNewcomerOnly(t *testing.T) {
	acc := newMemAccounts()
	r := resolver(acc)
	existing, _ := r.resolve(context.Background(), oidcrp.Identity{Subject: "sub-1"})

	r.admission = denyAll{}
	if _, err := r.resolve(context.Background(), oidcrp.Identity{Subject: "sub-2"}); !errors.Is(err, errRejected) {
		t.Errorf("新身份应被准入策略拒绝：%v", err)
	}
	if u, err := r.resolve(context.Background(), oidcrp.Identity{Subject: "sub-1"}); err != nil || u.ID != existing.ID {
		t.Errorf("已有账号不受准入策略影响：%+v %v", u, err)
	}
}

func TestResolve_UsernameConflictGetsSuffix(t *testing.T) {
	acc := newMemAccounts()
	acc.users[99] = &database.User{ID: 99, Username: "alice", Role: 3, Status: 1}

	u, err := resolver(acc).resolve(context.Background(), oidcrp.Identity{Subject: "sub-1", PreferredUsername: "alice"})
	if err != nil || u.Username != "alice-2" {
		t.Errorf("用户名冲突应加后缀：%+v %v", u, err)
	}
}

// 并发首登：建号时发现身份已被另一请求建好 → 回到那个账号，不报错、不建第二个
func TestResolve_ConcurrentFirstLogin(t *testing.T) {
	acc := newMemAccounts()
	acc.raceOnce = true
	u, err := resolver(acc).resolve(context.Background(), oidcrp.Identity{Subject: "sub-1", PreferredUsername: "alice"})
	if err != nil || u.Username != "racer" {
		t.Errorf("应回到抢先建好的账号：%+v %v", u, err)
	}
	if len(acc.users) != 1 {
		t.Errorf("只应有一个账号，现有 %d 个", len(acc.users))
	}
}
