package atlhyper_master_v2

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 每类数据按自己的保留期清理；没配保留期的不动；一类出错不影响其他类
func TestPurgeOnce(t *testing.T) {
	now := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	got := map[string]time.Time{}
	job := func(name string, keep time.Duration, err error) retentionJob {
		return retentionJob{name: name, keep: keep, purge: func(_ context.Context, before time.Time) (int64, error) {
			got[name] = before
			return 1, err
		}}
	}
	purgeOnce(context.Background(), []retentionJob{
		job("commands", 90*24*time.Hour, nil),
		job("audit", 180*24*time.Hour, errors.New("database is locked")),
		job("reports", 0, nil),
		job("incidents", 180*24*time.Hour, nil),
	}, now)

	if !got["commands"].Equal(now.Add(-90 * 24 * time.Hour)) {
		t.Errorf("commands 截止 = %v", got["commands"])
	}
	if _, ran := got["reports"]; ran {
		t.Error("没配保留期的不应清理")
	}
	if _, ran := got["incidents"]; !ran {
		t.Error("前一类出错不应影响后面的清理")
	}
}
