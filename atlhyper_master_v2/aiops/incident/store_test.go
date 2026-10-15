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
	peak     float64
}

func newFakeRepo() *fakeIncidentRepo {
	return &fakeIncidentRepo{entities: map[string]*database.AIOpsIncidentEntity{}}
}

func (f *fakeIncidentRepo) CreateIncident(context.Context, *database.AIOpsIncident) error { return nil }
func (f *fakeIncidentRepo) UpdateState(context.Context, string, string, string) error     { return nil }
func (f *fakeIncidentRepo) IncrementRecurrence(context.Context, string) error             { return nil }
func (f *fakeIncidentRepo) UpdatePeakRisk(_ context.Context, _ string, v float64) error {
	if v > f.peak {
		f.peak = v
	}
	return nil
}
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
func (f *fakeIncidentRepo) AddTimeline(_ context.Context, t *database.AIOpsIncidentTimeline) error {
	f.timeline = append(f.timeline, t)
	return nil
}

func risk(key string, local, final float64) *aiops.EntityRisk {
	return &aiops.EntityRisk{EntityKey: key, RLocal: local, RFinal: final}
}

// A9：事件升级时实体的风险值要跟着更新（曾停在创建时的 76，而事件已升级到 88）；角色不变
func TestUpdateState_RefreshesEntityRisk(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	now := time.Now()
	id := s.Create(context.Background(), "c1", "ns/pod/a", risk("ns/pod/a", 0.95, 0.76), now)

	s.UpdateState(context.Background(), id, aiops.StateIncident, risk("ns/pod/a", 0.97, 0.882), now.Add(5*time.Minute))

	e := repo.entities[id+"|ns/pod/a"]
	if e == nil || e.RFinal != 0.882 || e.RLocal != 0.97 {
		t.Fatalf("实体风险应更新为 0.97 / 0.882，得到 %+v", e)
	}
	if e.Role != "root_cause" || e.EntityType == "" {
		t.Errorf("更新风险不得丢角色 / 类型：%+v", e)
	}
}

// 状态机传来的实体不在该事件的实体表里 → 不凭空新增（不替它编角色）
func TestUpdateState_UnknownEntityNotAdded(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	id := s.Create(context.Background(), "c1", "ns/pod/a", risk("ns/pod/a", 0.9, 0.7), time.Now())
	s.UpdateState(context.Background(), id, aiops.StateIncident, risk("ns/pod/zzz", 0.9, 0.9), time.Now())
	if _, ok := repo.entities[id+"|ns/pod/zzz"]; ok {
		t.Error("不应新增未知实体")
	}
}

// A8：时间线文字的风险值与界面同一刻度（0-100），曾写成「R_final=0.882」
func TestTimelineRiskUsesDisplayScale(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	id := s.Create(context.Background(), "c1", "ns/pod/a", risk("ns/pod/a", 0.9, 0.7), time.Now())
	s.UpdateState(context.Background(), id, aiops.StateIncident, risk("ns/pod/a", 0.97, 0.882), time.Now())
	s.IncrementRecurrence(context.Background(), id, risk("ns/pod/a", 0.9, 0.615), time.Now())

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

// 功能二：创建 / 升级 / 复发时追加一条 anomaly_detected，写明触发信号；恢复用 recovery_started
func TestStore_AppendsSignalTimeline(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	now := time.Now()
	withSignals := func(local, final float64) *aiops.EntityRisk {
		r := risk("ns/pod/a", local, final)
		r.Signals = []aiops.RiskSignal{
			{Metric: "critical_event", Score: 0.85, Detail: "K8s 事件 Unhealthy: probe"},
			{Metric: "container_anomaly", Score: 0.6, Detail: "容器 app: NotReady"},
		}
		return r
	}

	id := s.Create(context.Background(), "c1", "ns/pod/a", withSignals(0.6, 0.48), now)
	s.UpdateState(context.Background(), id, aiops.StateIncident, withSignals(0.7, 0.56), now)
	s.UpdateState(context.Background(), id, aiops.StateRecovery, risk("ns/pod/a", 0, 0), now)
	s.IncrementRecurrence(context.Background(), id, withSignals(0.6, 0.48), now)

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
}

// 本地无异常、风险全部来自依赖传播 → 明说，而不是空着
func TestStore_SignalTimelinePropagatedOnly(t *testing.T) {
	repo := newFakeRepo()
	s := NewStore(repo)
	s.Create(context.Background(), "c1", "ns/service/x", &aiops.EntityRisk{EntityKey: "ns/service/x", RLocal: 0, RFinal: 0.3}, time.Now())
	if len(repo.timeline) != 2 || repo.timeline[1].Detail != "无本地异常信号（风险由依赖传播）" {
		t.Errorf("got %+v", repo.timeline)
	}
}
