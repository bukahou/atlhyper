package incident

import (
	"context"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// fakeIncidentRepo 只实现 Store 用到的方法；实体按 (incident, key) 覆盖写，与表的主键一致
type fakeIncidentRepo struct {
	database.AIOpsIncidentRepository
	entities map[string]*database.AIOpsIncidentEntity
	timeline []*database.AIOpsIncidentTimeline
}

func newFakeRepo() *fakeIncidentRepo {
	return &fakeIncidentRepo{entities: map[string]*database.AIOpsIncidentEntity{}}
}

func (f *fakeIncidentRepo) CreateIncident(context.Context, *database.AIOpsIncident) error { return nil }
func (f *fakeIncidentRepo) UpdateState(context.Context, string, string) error             { return nil }
func (f *fakeIncidentRepo) RaiseSeverity(context.Context, string, string, float64) error  { return nil }
func (f *fakeIncidentRepo) MarkEscalated(context.Context, string, time.Time) error        { return nil }
func (f *fakeIncidentRepo) UpdatePeakRisk(context.Context, string, float64) error         { return nil }
func (f *fakeIncidentRepo) UpdateRootCause(context.Context, string, string) error         { return nil }
func (f *fakeIncidentRepo) IncrementRecurrence(context.Context, string) error             { return nil }
func (f *fakeIncidentRepo) SetAnomalyDuration(context.Context, string, time.Time) error   { return nil }
func (f *fakeIncidentRepo) Resolve(context.Context, string, time.Time, string) error      { return nil }
func (f *fakeIncidentRepo) AddEntity(_ context.Context, e *database.AIOpsIncidentEntity) error {
	cp := *e
	f.entities[e.IncidentID+"|"+e.EntityKey] = &cp
	return nil
}
func (f *fakeIncidentRepo) GetEntities(_ context.Context, id string) ([]*database.AIOpsIncidentEntity, error) {
	var out []*database.AIOpsIncidentEntity
	for k, e := range f.entities {
		if strings.HasPrefix(k, id+"|") {
			cp := *e
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (f *fakeIncidentRepo) UpsertSignal(context.Context, *database.AIOpsIncidentSignal) error {
	return nil
}
func (f *fakeIncidentRepo) AddTimeline(_ context.Context, t *database.AIOpsIncidentTimeline) error {
	f.timeline = append(f.timeline, t)
	return nil
}

// selfSubject 主体就是实体自身时的一轮风险
func selfSubject(key string, local, final float64, signals ...aiops.RiskSignal) *aiops.SubjectRisk {
	lead := &aiops.EntityRisk{EntityKey: key, RLocal: local, RFinal: final, Signals: signals}
	sr := &aiops.SubjectRisk{SubjectKey: key, RFinal: final, Lead: lead}
	if final > 0 {
		sr.Members = []*aiops.EntityRisk{lead}
	}
	return sr
}

// apply 把一条变化交给 Store；事件号固定为 inc-1
func apply(s *Store, kind aiops.TransitionKind, risk *aiops.SubjectRisk) {
	s.ApplyTransition(context.Background(), aiops.IncidentTransition{
		Kind: kind, IncidentID: "inc-1", ClusterID: "c1", SubjectKey: risk.SubjectKey,
		RootCause: risk.Lead.EntityKey, Severity: aiops.SeverityFromRisk(risk.RFinal),
		RFinal: risk.RFinal, PeakRisk: risk.RFinal, At: time.Now(),
	}, risk)
}

// A8：时间线文字的风险值与界面同一刻度（0-100），曾写成「R_final=0.882」
func TestTimelineRiskUsesDisplayScale(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	apply(s, aiops.TransitionCreated, selfSubject("ns/pod/a", 0.9, 0.7))
	apply(s, aiops.TransitionEscalated, selfSubject("ns/pod/a", 0.97, 0.882))
	apply(s, aiops.TransitionRecurred, selfSubject("ns/pod/a", 0.9, 0.615))

	var details []string
	for _, tl := range repo.timeline {
		details = append(details, tl.Detail)
	}
	joined := strings.Join(details, " | ")
	for _, want := range []string{"风险 88", "风险 62"} {
		if !strings.Contains(joined, want) {
			t.Errorf("时间线应含 %q：%s", want, joined)
		}
	}
	if strings.Contains(joined, "0.882") || strings.Contains(joined, "0.615") {
		t.Errorf("不得出现 [0,1] 刻度：%s", joined)
	}
}

// 创建 / 升级 / 复发时追加一条 anomaly_detected，写明领头实体的触发信号；恢复用 recovery_started
func TestStore_AppendsSignalTimeline(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	signals := []aiops.RiskSignal{
		{Metric: "critical_event", Score: 0.85, Detail: "K8s 事件 Unhealthy: probe"},
		{Metric: "container_anomaly", Score: 0.6, Detail: "容器 app: NotReady"},
	}

	apply(s, aiops.TransitionCreated, selfSubject("ns/pod/a", 0.6, 0.48, signals...))
	apply(s, aiops.TransitionEscalated, selfSubject("ns/pod/a", 0.7, 0.56, signals...))
	apply(s, aiops.TransitionRecoveryStarted, selfSubject("ns/pod/a", 0, 0))
	apply(s, aiops.TransitionRecurred, selfSubject("ns/pod/a", 0.6, 0.48, signals...))

	var types []string
	for _, tl := range repo.timeline {
		types = append(types, tl.EventType)
	}
	want := []string{
		aiops.TimelineStateChange, aiops.TimelineAnomalyDetected, // 创建
		aiops.TimelineStateChange, aiops.TimelineAnomalyDetected, // 升级
		aiops.TimelineRecoveryStarted,                           // 恢复：无信号行
		aiops.TimelineRecurrence, aiops.TimelineAnomalyDetected, // 复发
	}
	if strings.Join(types, ",") != strings.Join(want, ",") {
		t.Fatalf("时间线类型\n got  %v\n want %v", types, want)
	}
	sig := repo.timeline[1].Detail
	if sig != "K8s 事件 Unhealthy: probe；容器 app: NotReady" {
		t.Errorf("信号行 = %q", sig)
	}
	if d := repo.timeline[4].Detail; !strings.HasPrefix(d, "状态变更 → recovery") {
		t.Errorf("恢复行文案保持不变（旧数据兼容依赖此前缀），got %q", d)
	}
	for _, tl := range repo.timeline {
		if tl.EntityKey != "ns/pod/a" {
			t.Errorf("时间线条目应挂在领头实体上，实际 %q", tl.EntityKey)
		}
	}
}

// 本地无异常、风险全部来自依赖传播 → 明说，而不是空着
func TestStore_SignalTimelinePropagatedOnly(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	apply(s, aiops.TransitionCreated, selfSubject("ns/service/x", 0, 0.3))
	if len(repo.timeline) != 2 || repo.timeline[1].Detail != "无本地异常信号（风险由依赖传播）" {
		t.Errorf("got %+v", repo.timeline)
	}
}

// 没有事件号的变化不写任何东西
func TestApplyTransition_WithoutIncidentIDWritesNothing(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	s.ApplyTransition(context.Background(), aiops.IncidentTransition{Kind: aiops.TransitionClosedStale, At: time.Now()}, nil)
	if len(repo.timeline) != 0 || s.PendingWrites() != 0 {
		t.Errorf("没有事件号时不应写库: %+v", repo.timeline)
	}
}

func (f *fakeIncidentRepo) GetTimeline(context.Context, string) ([]*database.AIOpsIncidentTimeline, error) {
	return f.timeline, nil
}

// 重启后要从时间线取回「进入恢复观察期的时刻」：最后一次复发之后的那条恢复记录。
func TestRecoveryStartedAt(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entry := func(offset time.Duration, eventType string) *database.AIOpsIncidentTimeline {
		return &database.AIOpsIncidentTimeline{IncidentID: "inc-1", Timestamp: base.Add(offset), EventType: eventType}
	}

	tests := []struct {
		name     string
		timeline []*database.AIOpsIncidentTimeline
		want     time.Time
		wantOK   bool
	}{
		{"没有恢复记录", []*database.AIOpsIncidentTimeline{entry(0, aiops.TimelineStateChange)}, time.Time{}, false},
		{"恢复一次", []*database.AIOpsIncidentTimeline{
			entry(0, aiops.TimelineStateChange),
			entry(time.Hour, aiops.TimelineRecoveryStarted),
		}, base.Add(time.Hour), true},
		{"复发后再次恢复，取后一次", []*database.AIOpsIncidentTimeline{
			entry(time.Hour, aiops.TimelineRecoveryStarted),
			entry(2*time.Hour, aiops.TimelineRecurrence),
			entry(3*time.Hour, aiops.TimelineRecoveryStarted),
		}, base.Add(3 * time.Hour), true},
		{"复发后尚未恢复", []*database.AIOpsIncidentTimeline{
			entry(time.Hour, aiops.TimelineRecoveryStarted),
			entry(2*time.Hour, aiops.TimelineRecurrence),
		}, time.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.timeline = tt.timeline
			got, ok := NewStore(repo).RecoveryStartedAt(context.Background(), "inc-1")
			if ok != tt.wantOK || !got.Equal(tt.want) {
				t.Errorf("RecoveryStartedAt() = %v, %v; want %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
