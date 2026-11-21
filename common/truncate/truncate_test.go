package truncate

import (
	"testing"
	"unicode/utf8"
)

func TestRunes(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello…"},
		{"节点就绪率过低", 4, "节点就绪…"},
		{"", 3, ""},
		{"abc", 0, "…"},
	}
	for _, c := range cases {
		got := Runes(c.in, c.max, "…")
		if got != c.want {
			t.Errorf("Runes(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
		if !utf8.ValidString(got) {
			t.Errorf("Runes(%q, %d) 切坏了多字节字符: %q", c.in, c.max, got)
		}
	}
}
