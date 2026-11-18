package clustertoken

import (
	"strings"
	"testing"
)

func TestGenerate_FormatAndUniqueness(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(first.Plaintext, "atlk_") {
		t.Errorf("令牌应以 atlk_ 开头，实际 %q", first.Plaintext[:8])
	}
	// 32 字节随机数的 base64url（无填充）是 43 个字符
	if got := len(first.Plaintext) - len("atlk_"); got != 43 {
		t.Errorf("随机部分应为 43 个字符，实际 %d", got)
	}
	if first.Plaintext == second.Plaintext || first.Hash == second.Hash {
		t.Error("两次生成的令牌相同")
	}
	if first.Hash != Hash(first.Plaintext) {
		t.Error("返回的哈希与对明文再算一遍的结果不一致")
	}
	if !strings.HasPrefix(first.Plaintext, first.Prefix) || len(first.Prefix) != 8 {
		t.Errorf("前缀应为明文的前 8 个字符，实际 %q", first.Prefix)
	}
	if strings.Contains(first.Hash, first.Plaintext[5:]) {
		t.Error("哈希里不应出现明文")
	}
}

func TestMatches(t *testing.T) {
	token, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		presented  string
		storedHash string
		want       bool
	}{
		{name: "正确的令牌", presented: token.Plaintext, storedHash: token.Hash, want: true},
		{name: "错误的令牌", presented: token.Plaintext + "x", storedHash: token.Hash, want: false},
		{name: "空令牌", presented: "", storedHash: token.Hash, want: false},
		{name: "库里没有签发过令牌", presented: token.Plaintext, storedHash: "", want: false},
		{name: "两边都为空也不算匹配", presented: "", storedHash: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(tt.presented, tt.storedHash); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromBearerHeader(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{header: "Bearer atlk_abc", want: "atlk_abc"},
		{header: "bearer atlk_abc", want: "atlk_abc"},
		{header: "Bearer   atlk_abc  ", want: "atlk_abc"},
		{header: "", want: ""},
		{header: "Basic dXNlcjpwYXNz", want: ""},
		{header: "atlk_abc", want: ""},
		{header: "Bearer", want: ""},
	}
	for _, tt := range tests {
		if got := FromBearerHeader(tt.header); got != tt.want {
			t.Errorf("FromBearerHeader(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}
