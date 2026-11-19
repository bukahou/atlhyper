package enricher

import (
	"testing"
	"time"
)

// 定时窗口是人填的墙上时间（HH:MM），按运维时区解读，与进程所在时区无关
func TestIsWithinSchedule(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Skip("缺少时区数据")
	}
	// UTC 01:30 = 东京 10:30
	now := time.Date(2026, 11, 19, 1, 30, 0, 0, time.UTC).In(tokyo)
	cases := []struct {
		start, end string
		want       bool
	}{
		{"09:00", "18:00", true}, // 东京 10:30 在窗口内（按 UTC 的 01:30 会判成窗口外）
		{"11:00", "18:00", false},
		{"22:00", "11:00", true},  // 跨午夜
		{"22:00", "06:00", false}, // 跨午夜，10:30 不在
		{"", "", true},            // 没设窗口 = 始终有效
	}
	for _, c := range cases {
		if got := isWithinSchedule(c.start, c.end, now); got != c.want {
			t.Errorf("%s~%s @ 东京 10:30 = %v, want %v", c.start, c.end, got, c.want)
		}
	}
}
