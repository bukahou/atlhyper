package source

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

var sourceNow = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

// fakeSubmitter 记录提交给管道的内容
type fakeSubmitter struct {
	alerts   []alert.Alert
	deferred []func(*database.NotifyPolicy) (alert.Alert, bool)
	refuse   bool // 模拟队列已满
}

func (s *fakeSubmitter) Submit(a alert.Alert) bool {
	if s.refuse {
		return false
	}
	s.alerts = append(s.alerts, a)
	return true
}

func (s *fakeSubmitter) SubmitDeferred(build func(*database.NotifyPolicy) (alert.Alert, bool)) bool {
	if s.refuse {
		return false
	}
	s.deferred = append(s.deferred, build)
	return true
}

// fakeFiringAlerts 库里持续中的告警
type fakeFiringAlerts struct {
	rows []*database.NotifyAlert
	err  error
}

func (f *fakeFiringAlerts) ListFiring(context.Context) ([]*database.NotifyAlert, error) {
	return f.rows, f.err
}

// fakePolicy 当前策略
type fakePolicy struct{ policy *database.NotifyPolicy }

func (f *fakePolicy) Get(context.Context) (*database.NotifyPolicy, error) { return f.policy, nil }
