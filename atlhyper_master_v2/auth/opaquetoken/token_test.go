package opaquetoken

import (
	"strings"
	"testing"
)

func TestGenerate_FormatAndUniqueness(t *testing.T) {
	for _, prefix := range []string{"atlk_", "atlp_"} {
		t.Run(prefix, func(t *testing.T) {
			first, err := Generate(prefix)
			if err != nil {
				t.Fatal(err)
			}
			second, err := Generate(prefix)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(first.Plaintext, prefix) {
				t.Errorf("令牌应以 %s 开头", prefix)
			}
			// 32 字节随机数的 base64url（无填充）是 43 个字符
			if got := len(first.Plaintext) - len(prefix); got != 43 {
				t.Errorf("随机部分应为 43 个字符，实际 %d", got)
			}
			if first.Plaintext == second.Plaintext || first.Hash == second.Hash {
				t.Error("两次生成的令牌相同")
			}
			if first.Hash != Hash(first.Plaintext) {
				t.Error("返回的哈希与对明文再算一遍的结果不一致")
			}
			// 显示用前缀 = 固定开头 + 随机部分的前 3 个字符
			if first.Prefix != first.Plaintext[:len(prefix)+3] {
				t.Errorf("显示用前缀不对：%q", first.Prefix)
			}
			if strings.Contains(first.Hash, first.Plaintext[len(prefix):]) {
				t.Error("哈希里不应出现明文")
			}
		})
	}
}

func TestHash_IsSHA256Hex(t *testing.T) {
	// 空串的 SHA-256 是公开的固定值：入库格式（十六进制小写）一旦变了，已签发的令牌全部失效
	const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := Hash(""); got != emptySHA256 {
		t.Errorf("Hash(\"\") = %s", got)
	}
}

func TestMatches(t *testing.T) {
	token, err := Generate("atlp_")
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
		{name: "库里没有哈希", presented: token.Plaintext, storedHash: "", want: false},
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
		{header: "Bearer atlp_abc", want: "atlp_abc"},
		{header: "bearer atlp_abc", want: "atlp_abc"},
		{header: "Bearer   atlp_abc  ", want: "atlp_abc"},
		{header: "", want: ""},
		{header: "Basic dXNlcjpwYXNz", want: ""},
		{header: "atlp_abc", want: ""},
		{header: "Bearer", want: ""},
	}
	for _, tt := range tests {
		if got := FromBearerHeader(tt.header); got != tt.want {
			t.Errorf("FromBearerHeader(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}
