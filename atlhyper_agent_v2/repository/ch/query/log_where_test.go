package query

import (
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
)

// 起止时间写错时报错：不能变成「不加时间条件」的全表扫描
func TestLogBuildWhere_InvalidAbsoluteRangeIsAnError(t *testing.T) {
	r := &logRepository{}
	_, _, err := r.buildWhere(300, repository.LogQueryOptions{StartTime: "yesterday", EndTime: "2026-11-19T00:00:00Z"})
	if err == nil {
		t.Fatal("起止时间解析失败应返回错误")
	}
	where, args, err := r.buildWhere(300, repository.LogQueryOptions{StartTime: "2026-11-18T00:00:00Z", EndTime: "2026-11-19T00:00:00Z"})
	if err != nil || !strings.Contains(where, "Timestamp >= ?") || !strings.Contains(where, "Timestamp <= ?") || len(args) != 2 {
		t.Fatalf("合法起止时间：where=%q args=%v err=%v", where, args, err)
	}
}

// 相对时间有上限（执行方自己守门，不只靠 Master）
func TestSinceSeconds_Capped(t *testing.T) {
	if got := sinceSeconds(365 * 24 * time.Hour); got != int64(maxQueryWindow.Seconds()) {
		t.Errorf("一年 → %d 秒，want 上限 %d", got, int64(maxQueryWindow.Seconds()))
	}
	if got := sinceSeconds(0); got != 300 {
		t.Errorf("没给 → %d，want 默认 300", got)
	}
}
