// atlhyper_master_v2/auth/oidc/username.go
// 外部身份建号时的用户名生成
package oidc

import (
	"strconv"
	"strings"

	"github.com/bukahou/akasha/pkg/oidcrp"
)

const maxUsernameLen = 32

// reservedUsernames 与系统账号或硬编码保护名易混淆，不直接给外部账号使用
var reservedUsernames = map[string]bool{"admin": true, "administrator": true, "root": true, "system": true}

// baseUsername 依次取 preferred_username → 邮箱本地部分 → "user"，规整为 [a-z0-9._-]
func baseUsername(id oidcrp.Identity) string {
	local := id.Email
	if i := strings.IndexByte(local, '@'); i >= 0 {
		local = local[:i]
	}
	for _, raw := range []string{id.PreferredUsername, local} {
		if s := sanitizeUsername(raw); s != "" {
			return s
		}
	}
	return "user"
}

// usernameCandidate 第 n 个候选：base、base-2、base-3…；保留名从 base-2 起步
func usernameCandidate(base string, n int) string {
	if reservedUsernames[base] {
		n++
	}
	if n == 0 {
		return base
	}
	return base + "-" + strconv.Itoa(n+1)
}

func sanitizeUsername(raw string) string {
	var b strings.Builder
	lastSep := true // 开头不留分隔符
	for _, r := range strings.ToLower(raw) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			lastSep = false
		case r == '.' || r == '-':
			if !lastSep {
				b.WriteRune(r)
				lastSep = true
			}
		default: // 其余字符视为分隔
			if !lastSep {
				b.WriteRune('-')
				lastSep = true
			}
		}
	}
	s := strings.Trim(b.String(), ".-_")
	if len(s) > maxUsernameLen {
		s = strings.Trim(s[:maxUsernameLen], ".-_")
	}
	return s
}
