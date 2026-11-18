package source

import (
	"errors"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	agentmodel "AtlHyper/model_v3/agent"
)

type fakeAgents struct {
	agents []agentmodel.AgentInfo
	err    error
}

func (f *fakeAgents) ListAgents() ([]agentmodel.AgentInfo, error) { return f.agents, f.err }

func heartbeatRow(clusterID string) *database.NotifyAlert {
	return &database.NotifyAlert{Fingerprint: alert.HeartbeatFingerprint(clusterID), Source: string(alert.SourceHeartbeat),
		ClusterID: clusterID, Status: "firing"}
}

func newTestHeartbeatSource(agents *fakeAgents, firing *fakeFiringAlerts, submitter *fakeSubmitter) *HeartbeatSource {
	return NewHeartbeatSource(agents, firing, submitter, HeartbeatConfig{
		CheckInterval: 30 * time.Second, OfflineAfter: time.Minute, Now: func() time.Time { return sourceNow },
	})
}

func TestHeartbeatSource_Check(t *testing.T) {
	online := agentmodel.AgentInfo{ClusterID: "prod", Status: agentmodel.StatusOnline, LastHeartbeat: sourceNow.Add(-10 * time.Second)}
	stale := agentmodel.AgentInfo{ClusterID: "prod", Status: agentmodel.StatusOnline, LastHeartbeat: sourceNow.Add(-2 * time.Minute)}
	markedOffline := agentmodel.AgentInfo{ClusterID: "prod", Status: agentmodel.StatusOffline, LastHeartbeat: sourceNow.Add(-10 * time.Second)}
	neverSeen := agentmodel.AgentInfo{ClusterID: "prod", Status: agentmodel.StatusOnline}

	cases := []struct {
		name       string
		agent      agentmodel.AgentInfo
		firing     bool
		wantStatus alert.Status // 空 = 不提交
	}{
		{"在线、没有告警：不提交", online, false, ""},
		{"心跳超时、没有告警：触发", stale, false, alert.StatusFiring},
		{"状态已标为离线：触发", markedOffline, false, alert.StatusFiring},
		{"从未有过心跳：触发", neverSeen, false, alert.StatusFiring},
		{"仍离线、告警已在持续：不重复提交", stale, true, ""},
		{"恢复上报、告警在持续：结束", online, true, alert.StatusResolved},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			firing := &fakeFiringAlerts{}
			if c.firing {
				firing.rows = []*database.NotifyAlert{heartbeatRow("prod")}
			}
			submitter := &fakeSubmitter{}
			newTestHeartbeatSource(&fakeAgents{agents: []agentmodel.AgentInfo{c.agent}}, firing, submitter).check()

			if c.wantStatus == "" {
				if len(submitter.alerts) != 0 {
					t.Fatalf("不应提交，实际 %+v", submitter.alerts)
				}
				return
			}
			if len(submitter.alerts) != 1 {
				t.Fatalf("应提交 1 条，实际 %d", len(submitter.alerts))
			}
			got := submitter.alerts[0]
			if got.Status != c.wantStatus || got.Fingerprint != "heartbeat:prod" || got.Source != alert.SourceHeartbeat ||
				got.ClusterID != "prod" || got.Severity != alert.SeverityCritical || got.Quiet {
				t.Errorf("提交的告警不对: %+v", got)
			}
		})
	}
}

func TestHeartbeatSource_FiringAlertContent(t *testing.T) {
	stale := agentmodel.AgentInfo{ClusterID: "prod", Status: agentmodel.StatusOnline, LastHeartbeat: sourceNow.Add(-2 * time.Minute)}
	submitter := &fakeSubmitter{}
	newTestHeartbeatSource(&fakeAgents{agents: []agentmodel.AgentInfo{stale}}, &fakeFiringAlerts{}, submitter).check()

	got := submitter.alerts[0]
	if got.Kind != alert.KindFiring || got.Title != "Agent 离线" || got.Summary == "" || !got.StartsAt.Equal(sourceNow) ||
		got.Details[alert.DetailLastHeartbeat] != "2026-11-15T09:58:00Z" {
		t.Errorf("离线告警的内容不对: %+v", got)
	}
}

// 只看心跳告警：别的来源的告警在持续，不影响判断
func TestHeartbeatSource_IgnoresOtherSources(t *testing.T) {
	stale := agentmodel.AgentInfo{ClusterID: "prod", Status: agentmodel.StatusOnline, LastHeartbeat: sourceNow.Add(-2 * time.Minute)}
	firing := &fakeFiringAlerts{rows: []*database.NotifyAlert{
		{Fingerprint: "incident:inc-1", Source: string(alert.SourceIncident), ClusterID: "prod", Status: "firing"},
	}}
	submitter := &fakeSubmitter{}
	newTestHeartbeatSource(&fakeAgents{agents: []agentmodel.AgentInfo{stale}}, firing, submitter).check()
	if len(submitter.alerts) != 1 || submitter.alerts[0].Status != alert.StatusFiring {
		t.Errorf("同集群有事件告警时，离线告警仍应触发: %+v", submitter.alerts)
	}
}

// 读不到现有告警时这一轮不做判断：宁可晚一轮，也不凭空重复触发或误结束
func TestHeartbeatSource_SkipsRoundWhenStateUnavailable(t *testing.T) {
	stale := agentmodel.AgentInfo{ClusterID: "prod", Status: agentmodel.StatusOnline, LastHeartbeat: sourceNow.Add(-2 * time.Minute)}
	submitter := &fakeSubmitter{}
	newTestHeartbeatSource(&fakeAgents{agents: []agentmodel.AgentInfo{stale}}, &fakeFiringAlerts{err: errors.New("db busy")}, submitter).check()
	newTestHeartbeatSource(&fakeAgents{err: errors.New("store down")}, &fakeFiringAlerts{}, submitter).check()
	if len(submitter.alerts) != 0 {
		t.Errorf("状态读不到时不应提交: %+v", submitter.alerts)
	}
}

func TestHeartbeatSource_StartStop(t *testing.T) {
	src := newTestHeartbeatSource(&fakeAgents{}, &fakeFiringAlerts{}, &fakeSubmitter{})
	if err := src.Start(); err != nil {
		t.Fatal(err)
	}
	if err := src.Start(); err != nil { // 重复启动无效果
		t.Fatal(err)
	}
	if err := src.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := src.Stop(); err != nil { // 重复停止无效果
		t.Fatal(err)
	}
}
