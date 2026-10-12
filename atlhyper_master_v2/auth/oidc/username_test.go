package oidc

import (
	"testing"

	"github.com/bukahou/akasha/pkg/oidcrp"
)

func TestBaseUsername(t *testing.T) {
	cases := []struct {
		name string
		id   oidcrp.Identity
		want string
	}{
		{"优先 preferred_username，转小写", oidcrp.Identity{PreferredUsername: "Alice", Email: "x@example.com"}, "alice"},
		{"其次邮箱本地部分", oidcrp.Identity{Email: "Bob.Smith@example.com"}, "bob.smith"},
		{"非法字符替换并收拢", oidcrp.Identity{PreferredUsername: "李 雷 / ops!!"}, "ops"},
		{"全是非法字符 → 退到邮箱", oidcrp.Identity{PreferredUsername: "李雷", Email: "lilei@example.com"}, "lilei"},
		{"都没有 → user", oidcrp.Identity{}, "user"},
		{"超长截断", oidcrp.Identity{PreferredUsername: "abcdefghijklmnopqrstuvwxyz0123456789"}, "abcdefghijklmnopqrstuvwxyz012345"},
		{"首尾符号去掉", oidcrp.Identity{PreferredUsername: "--.al_ice._"}, "al_ice"},
	}
	for _, c := range cases {
		if got := baseUsername(c.id); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// 保留名（与系统账号、硬编码保护名易混淆）从第二个候选起步
func TestUsernameCandidates_Reserved(t *testing.T) {
	if got := usernameCandidate("admin", 0); got != "admin-2" {
		t.Errorf("保留名不得直接使用，得到 %q", got)
	}
	if got := usernameCandidate("alice", 0); got != "alice" {
		t.Errorf("got %q", got)
	}
	if got := usernameCandidate("alice", 2); got != "alice-3" {
		t.Errorf("got %q", got)
	}
}
