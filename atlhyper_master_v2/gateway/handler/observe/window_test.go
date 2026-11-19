package observe

import (
	"net/http/httptest"
	"testing"
)

func TestParseTimeRangeMinutes(t *testing.T) {
	cases := []struct {
		in     string
		want   int
		wantOK bool
	}{
		{"15m", 15, true},
		{"24h", 1440, true},
		{"720h", maxWindowMinutes, true},
		{"8760h", maxWindowMinutes, true}, // 一年：按上限查
		{"0m", 0, false},
		{"-1h", 0, false},
		{"abc", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := parseTimeRangeMinutes(c.in)
		if got != c.want || ok != c.wantOK {
			t.Errorf("parseTimeRangeMinutes(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestWindowMinutesParam(t *testing.T) {
	cases := []struct {
		query string
		want  int
	}{
		{"", 60},
		{"minutes=30", 30},
		{"minutes=999999999", maxWindowMinutes},
		{"minutes=0", 60},
		{"minutes=-5", 60},
		{"minutes=x", 60},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/x?"+c.query, nil)
		if got := windowMinutesParam(r, "minutes", 60); got != c.want {
			t.Errorf("windowMinutesParam(%q) = %d, want %d", c.query, got, c.want)
		}
	}
}

// 小时数参数换成分钟后也受上限约束，且大数不会溢出
func TestWindowHoursParam(t *testing.T) {
	cases := []struct {
		query string
		want  int
	}{
		{"", 24 * 60},
		{"hours=6", 360},
		{"hours=168", 168 * 60},
		{"hours=99999999999", maxWindowMinutes},
		{"hours=0", 24 * 60},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/x?"+c.query, nil)
		if got := windowHoursParamMinutes(r, "hours", 24); got != c.want {
			t.Errorf("windowHoursParamMinutes(%q) = %d, want %d", c.query, got, c.want)
		}
	}
}
