package model

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateClusterID(t *testing.T) {
	cases := []struct {
		name      string
		clusterID string
		valid     bool
	}{
		{"单个字母", "a", true},
		{"单个数字", "7", true},
		{"常见写法", "prod-tokyo-1", true},
		{"带点", "prod.tokyo", true},
		{"集群 UID 形式", "0f8fad5b-d9cb-469f-a165-70867728950e", true},
		{"恰好 63 个字符", strings.Repeat("a", 63), true},
		{"空串", "", false},
		{"64 个字符", strings.Repeat("a", 64), false},
		{"大写字母", "Prod", false},
		{"下划线", "prod_1", false},
		{"空格", "prod 1", false},
		{"斜杠", "prod/1", false},
		{"以连字符开头", "-prod", false},
		{"以连字符结尾", "prod-", false},
		{"以点开头", ".prod", false},
		{"以点结尾", "prod.", false},
		{"单个连字符", "-", false},
		{"首尾空白不自动去掉", " prod ", false},
		{"非 ASCII 字符", "生产", false},
		{"换行", "prod\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateClusterID(c.clusterID)
			if c.valid && err != nil {
				t.Errorf("%q 应当合法，实际 %v", c.clusterID, err)
			}
			if !c.valid && !errors.Is(err, ErrInvalidClusterInput) {
				t.Errorf("%q 应当返回 ErrInvalidClusterInput，实际 %v", c.clusterID, err)
			}
		})
	}
}

func TestValidateClusterProfile(t *testing.T) {
	cases := []struct {
		name        string
		displayName string
		description string
		valid       bool
	}{
		{"正常", "生产", "东京机房", true},
		{"描述可以为空", "生产", "", true},
		{"显示名恰好 64 个字", strings.Repeat("生", 64), "", true},
		{"描述恰好 256 个字", "生产", strings.Repeat("述", 256), true},
		{"显示名为空", "", "东京", false},
		{"显示名超长", strings.Repeat("生", 65), "", false},
		{"描述超长", "生产", strings.Repeat("述", 257), false},
		{"显示名带控制字符", "生\n产", "", false},
		{"描述带控制字符", "生产", "东\x00京", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateClusterProfile(c.displayName, c.description)
			if c.valid && err != nil {
				t.Errorf("应当合法，实际 %v", err)
			}
			if !c.valid && !errors.Is(err, ErrInvalidClusterInput) {
				t.Errorf("应当返回 ErrInvalidClusterInput，实际 %v", err)
			}
		})
	}
}
