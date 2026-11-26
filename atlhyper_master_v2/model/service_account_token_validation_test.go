package model

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateServiceAccountName(t *testing.T) {
	cases := []struct {
		name  string
		input string
		valid bool
	}{
		{"常见写法", "haro", true},
		{"带数字与连字符", "haro-prod-2", true},
		{"恰好 3 个字符", "abc", true},
		{"恰好 32 个字符", "a" + strings.Repeat("b", 31), true},
		{"2 个字符", "ab", false},
		{"33 个字符", "a" + strings.Repeat("b", 32), false},
		{"空串", "", false},
		{"数字开头", "1haro", false},
		{"连字符开头", "-haro", false},
		{"大写字母", "Haro", false},
		{"下划线", "haro_prod", false},
		{"点", "haro.prod", false},
		{"空格", "haro prod", false},
		{"前后空白不替用户去掉", " haro", false},
		{"中文", "语音助手", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateServiceAccountName(c.input)
			if c.valid && err != nil {
				t.Errorf("%q 应合规：%v", c.input, err)
			}
			if !c.valid && !errors.Is(err, ErrInvalidServiceAccountInput) {
				t.Errorf("%q 应不合规，err = %v", c.input, err)
			}
		})
	}
}

func TestValidateServiceAccountProfile(t *testing.T) {
	cases := []struct {
		name        string
		displayName string
		role        *int
		valid       bool
	}{
		{"0 级", "Haro", intPtr(0), true},
		{"2 级（上限）", "Haro", intPtr(2), true},
		{"显示名可空", "", intPtr(1), true},
		{"显示名 64 个字", strings.Repeat("语", 64), intPtr(0), true},
		{"3 级一律拒绝", "Haro", intPtr(3), false},
		{"负数", "Haro", intPtr(-1), false},
		{"没给等级（⛔ 不当作 0）", "Haro", nil, false},
		{"显示名 65 个字", strings.Repeat("语", 65), intPtr(0), false},
		{"显示名含换行", "Ha\nro", intPtr(0), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateServiceAccountProfile(c.displayName, c.role)
			if c.valid && err != nil {
				t.Errorf("应合规：%v", err)
			}
			if !c.valid && !errors.Is(err, ErrInvalidServiceAccountInput) {
				t.Errorf("应不合规，err = %v", err)
			}
		})
	}
}

func TestValidateAPITokenRequest(t *testing.T) {
	cases := []struct {
		name          string
		tokenName     string
		expiresInDays *int
		valid         bool
	}{
		{"不过期", "prod", nil, true},
		{"1 天", "prod", intPtr(1), true},
		{"上限 3650 天", "prod", intPtr(MaxAPITokenExpiresInDays), true},
		{"名称 64 个字", strings.Repeat("令", 64), nil, true},
		{"0 天", "prod", intPtr(0), false},
		{"负数天", "prod", intPtr(-7), false},
		{"超过 3650 天", "prod", intPtr(MaxAPITokenExpiresInDays + 1), false},
		{"名称为空", "", nil, false},
		{"名称 65 个字", strings.Repeat("令", 65), nil, false},
		{"名称含控制字符", "pr\u0000od", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateAPITokenRequest(c.tokenName, c.expiresInDays)
			if c.valid && err != nil {
				t.Errorf("应合规：%v", err)
			}
			if !c.valid && !errors.Is(err, ErrInvalidServiceAccountInput) {
				t.Errorf("应不合规，err = %v", err)
			}
		})
	}
}

// 修改是整体替换：三个字段都必须给出，缺哪个都拒绝（⛔ 落成零值去执行）
func TestValidateUpdateServiceAccountRequest(t *testing.T) {
	full := UpdateServiceAccountRequest{DisplayName: strPtr("Haro"), Role: intPtr(1), Enabled: boolPtr(true)}
	if err := ValidateUpdateServiceAccountRequest(&full); err != nil {
		t.Fatalf("完整的请求应合规：%v", err)
	}
	missing := map[string]UpdateServiceAccountRequest{
		"displayName": {Role: intPtr(1), Enabled: boolPtr(true)},
		"role":        {DisplayName: strPtr("Haro"), Enabled: boolPtr(true)},
		"enabled":     {DisplayName: strPtr("Haro"), Role: intPtr(1)},
	}
	for field, req := range missing {
		if err := ValidateUpdateServiceAccountRequest(&req); !errors.Is(err, ErrInvalidServiceAccountInput) {
			t.Errorf("缺 %s：err = %v", field, err)
		}
	}
	tooHigh := UpdateServiceAccountRequest{DisplayName: strPtr("Haro"), Role: intPtr(3), Enabled: boolPtr(true)}
	if err := ValidateUpdateServiceAccountRequest(&tooHigh); !errors.Is(err, ErrInvalidServiceAccountInput) {
		t.Errorf("3 级：err = %v", err)
	}
}

// 下发给前端的业务参数与校验用的是同一组常量
func TestServiceAccountLimits(t *testing.T) {
	limits := CurrentServiceAccountLimits()
	if len(limits.AssignableRoles) != ServiceAccountMaxRole+1 || limits.AssignableRoles[0] != 0 ||
		limits.AssignableRoles[len(limits.AssignableRoles)-1] != ServiceAccountMaxRole {
		t.Errorf("可选等级应为 0..%d：%v", ServiceAccountMaxRole, limits.AssignableRoles)
	}
	for _, role := range limits.AssignableRoles {
		if err := ValidateServiceAccountProfile("", intPtr(role)); err != nil {
			t.Errorf("下发的等级 %d 校验却不通过：%v", role, err)
		}
	}
	if limits.MaxActiveTokens != MaxActiveAPITokensPerAccount || limits.MaxExpiresInDays != MaxAPITokenExpiresInDays {
		t.Errorf("limits = %+v", limits)
	}
}
