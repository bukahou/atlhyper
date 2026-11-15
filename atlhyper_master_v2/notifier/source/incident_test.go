package source

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

func transition(kind aiops.TransitionKind, change func(*aiops.IncidentTransition)) aiops.IncidentTransition {
	t := aiops.IncidentTransition{
		Kind: kind, IncidentID: "inc-123", ClusterID: "prod", SubjectKey: "shop/deployment/api",
		RootCause: "shop/pod/api-7d9f", Members: []string{"shop/pod/api-7d9f", "shop/pod/api-x2k4"},
		FromState: aiops.StateWarning, ToState: aiops.StateIncident,
		Severity: "high", RFinal: 0.66, PeakRisk: 0.72, Recurrence: 0, Occurrences24h: 1, At: sourceNow,
	}
	if change != nil {
		change(&t)
	}
	return t
}

// 设计文档「事件 → 告警」表：每一行一个用例
func TestIncidentAlert_TransitionTable(t *testing.T) {
	policy := database.DefaultNotifyPolicy() // 事件告警开启，反复阈值 3

	cases := []struct {
		name       string
		transition aiops.IncidentTransition
		wantAlert  bool
		wantStatus alert.Status
		wantKind   alert.Kind
		wantQuiet  bool
	}{
		{"escalated：进入事件中 → 触发",
			transition(aiops.TransitionEscalated, func(t *aiops.IncidentTransition) { t.ReachedIncident = true }),
			true, alert.StatusFiring, alert.KindFiring, false},
		{"severity_raised 且事件处于事件中 → 升级",
			transition(aiops.TransitionSeverityRaised, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState, t.Severity, t.ReachedIncident = aiops.StateIncident, aiops.StateIncident, "critical", true
			}),
			true, alert.StatusFiring, alert.KindEscalated, false},
		{"severity_raised 但事件只在警告 → 不产生告警",
			transition(aiops.TransitionSeverityRaised, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState = aiops.StateWarning, aiops.StateWarning
			}),
			false, "", "", false},
		{"recovery_started（升级过的事件）→ 结束，发恢复通知",
			transition(aiops.TransitionRecoveryStarted, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState, t.ReachedIncident = aiops.StateIncident, aiops.StateRecovery, true
			}),
			true, alert.StatusResolved, alert.KindResolved, false},
		{"recurred，事件升级过 → 复发",
			transition(aiops.TransitionRecurred, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState, t.ReachedIncident, t.Recurrence, t.Occurrences24h = aiops.StateRecovery, aiops.StateWarning, true, 2, 3
			}),
			true, alert.StatusFiring, alert.KindRecurred, false},
		{"recurred，没升级过，24 小时内次数达到阈值 → 反复",
			transition(aiops.TransitionRecurred, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState, t.Recurrence, t.Occurrences24h = aiops.StateRecovery, aiops.StateWarning, 2, 3
			}),
			true, alert.StatusFiring, alert.KindFlapping, false},
		{"recurred，没升级过，次数不到阈值 → 不产生告警",
			transition(aiops.TransitionRecurred, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState, t.Recurrence, t.Occurrences24h = aiops.StateRecovery, aiops.StateWarning, 1, 2
			}),
			false, "", "", false},
		{"created → 不产生告警",
			transition(aiops.TransitionCreated, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState = aiops.StateHealthy, aiops.StateWarning
			}),
			false, "", "", false},
		{"closed → 结束仍在持续的（反复类）告警，发恢复通知",
			transition(aiops.TransitionClosed, func(t *aiops.IncidentTransition) {
				t.FromState, t.ToState, t.Members = aiops.StateRecovery, aiops.StateStable, nil
			}),
			true, alert.StatusResolved, alert.KindResolved, false},
		{"closed_stale → 结束，不发恢复通知",
			transition(aiops.TransitionClosedStale, func(t *aiops.IncidentTransition) {
				t.ToState, t.Members, t.ReachedIncident = aiops.StateStable, nil, true
			}),
			true, alert.StatusResolved, alert.KindResolved, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, produced := IncidentAlert(c.transition, policy)
			if produced != c.wantAlert {
				t.Fatalf("是否产生告警 = %v，期望 %v", produced, c.wantAlert)
			}
			if !produced {
				return
			}
			if got.Status != c.wantStatus || got.Kind != c.wantKind || got.Quiet != c.wantQuiet {
				t.Errorf("状态 / 种类 / 安静 = %s / %s / %v，期望 %s / %s / %v",
					got.Status, got.Kind, got.Quiet, c.wantStatus, c.wantKind, c.wantQuiet)
			}
			if got.Fingerprint != "incident:inc-123" || got.Source != alert.SourceIncident || got.ClusterID != "prod" {
				t.Errorf("指纹 / 来源 / 集群不对: %+v", got)
			}
		})
	}
}

// 没升级过的事件进入恢复观察：反复类告警不在这里结束（它要等事件关闭），否则每抖一次就是「恢复 + 反复」两条
func TestIncidentAlert_RecoveryOfNeverEscalatedIncidentProducesNothing(t *testing.T) {
	recovery := transition(aiops.TransitionRecoveryStarted, func(t *aiops.IncidentTransition) {
		t.FromState, t.ToState, t.ReachedIncident = aiops.StateWarning, aiops.StateRecovery, false
	})
	if _, produced := IncidentAlert(recovery, database.DefaultNotifyPolicy()); produced {
		t.Error("没升级过的事件进入恢复观察不应结束告警")
	}
}

func TestIncidentAlert_Content(t *testing.T) {
	escalated := transition(aiops.TransitionEscalated, func(t *aiops.IncidentTransition) {
		t.ReachedIncident, t.Recurrence, t.Occurrences24h = true, 2, 3
	})
	got, _ := IncidentAlert(escalated, database.DefaultNotifyPolicy())

	if got.Severity != alert.SeverityHigh || !got.StartsAt.Equal(sourceNow) || got.Title == "" || got.Summary == "" {
		t.Errorf("基本字段不对: %+v", got)
	}
	wantLabels := map[string]string{alert.LabelNamespace: "shop", alert.LabelSubject: "shop/deployment/api", alert.LabelSubjectType: "deployment"}
	for key, want := range wantLabels {
		if got.Labels[key] != want {
			t.Errorf("标签 %s = %q，期望 %q", key, got.Labels[key], want)
		}
	}
	wantDetails := map[string]string{
		alert.DetailIncidentID: "inc-123", alert.DetailRootCause: "shop/pod/api-7d9f", alert.DetailMemberCount: "2",
		alert.DetailMembers: "shop/pod/api-7d9f, shop/pod/api-x2k4", alert.DetailPeakRisk: "72",
		alert.DetailRecurrence: "2", alert.DetailOccurrences24h: "3",
	}
	for key, want := range wantDetails {
		if got.Details[key] != want {
			t.Errorf("详情 %s = %q，期望 %q", key, got.Details[key], want)
		}
	}
}

// 关闭类的变化不带成员：结束告警时不能把成员数、根因这些已有的详情冲成空
// 告警的开始时间是这一轮异常开始的时刻（建事件或复发），不是升级那一刻：通知里的持续时间按它算
func TestIncidentAlert_StartsAtEpisodeStart(t *testing.T) {
	episodeStart := sourceNow.Add(-6 * time.Minute)
	escalated := transition(aiops.TransitionEscalated, func(t *aiops.IncidentTransition) {
		t.ReachedIncident, t.EpisodeStartedAt = true, episodeStart
	})
	got, ok := IncidentAlert(escalated, database.DefaultNotifyPolicy())
	if !ok || !got.StartsAt.Equal(episodeStart) {
		t.Errorf("开始时间应是这一轮异常的起点 %v，实际 %v", episodeStart, got.StartsAt)
	}

	// 没有起点（旧数据等）：退回到这条变化的时刻
	noStart := transition(aiops.TransitionEscalated, func(t *aiops.IncidentTransition) { t.ReachedIncident = true })
	if got, _ := IncidentAlert(noStart, database.DefaultNotifyPolicy()); !got.StartsAt.Equal(sourceNow) {
		t.Errorf("没有起点时应用变化的时刻，实际 %v", got.StartsAt)
	}
}

func TestIncidentAlert_ClosingTransitionOmitsEmptyDetails(t *testing.T) {
	closed := transition(aiops.TransitionClosed, func(t *aiops.IncidentTransition) { t.Members, t.RootCause = nil, "" })
	got, _ := IncidentAlert(closed, database.DefaultNotifyPolicy())
	for _, key := range []string{alert.DetailMemberCount, alert.DetailMembers, alert.DetailRootCause} {
		if _, present := got.Details[key]; present {
			t.Errorf("没有值的详情 %s 不应出现（会覆盖告警里已有的值）", key)
		}
	}
	if got.EndsAt == nil || !got.EndsAt.Equal(sourceNow) {
		t.Errorf("结束时间应为变化发生的时刻: %v", got.EndsAt)
	}
}

func TestIncidentAlert_ClusterScopedSubjectHasNoNamespace(t *testing.T) {
	node := transition(aiops.TransitionEscalated, func(t *aiops.IncidentTransition) { t.SubjectKey = "_cluster/node/worker-1" })
	got, _ := IncidentAlert(node, database.DefaultNotifyPolicy())
	if got.Labels[alert.LabelNamespace] != "" || got.Labels[alert.LabelSubjectType] != "node" || got.Labels[alert.LabelSubject] != "_cluster/node/worker-1" {
		t.Errorf("集群级主体不应有命名空间标签: %v", got.Labels)
	}
}

func TestIncidentAlert_UnknownSeverityFallsBackToLow(t *testing.T) {
	odd := transition(aiops.TransitionEscalated, func(t *aiops.IncidentTransition) { t.Severity = "" })
	got, _ := IncidentAlert(odd, database.DefaultNotifyPolicy())
	if got.Severity != alert.SeverityLow {
		t.Errorf("认不出的严重度按 low 处理，实际 %q", got.Severity)
	}
}

// 警告级反复的阈值来自策略；0 = 不通知
func TestIncidentAlert_FlapThresholdFromPolicy(t *testing.T) {
	flapping := func(occurrences int) aiops.IncidentTransition {
		return transition(aiops.TransitionRecurred, func(t *aiops.IncidentTransition) {
			t.FromState, t.ToState, t.Occurrences24h = aiops.StateRecovery, aiops.StateWarning, occurrences
		})
	}
	cases := []struct {
		threshold, occurrences int
		want                   bool
	}{
		{3, 2, false}, {3, 3, true}, {3, 9, true}, {5, 4, false}, {5, 5, true}, {0, 99, false},
	}
	for _, c := range cases {
		policy := database.DefaultNotifyPolicy()
		policy.WarningFlapThreshold = c.threshold
		if _, produced := IncidentAlert(flapping(c.occurrences), policy); produced != c.want {
			t.Errorf("阈值 %d、发生 %d 次：是否通知 = %v，期望 %v", c.threshold, c.occurrences, produced, c.want)
		}
	}
}

// 关掉事件告警：不再产生新的告警；已有告警的结束仍然放行，否则它们永远停在「持续中」
func TestIncidentAlert_DisabledByPolicy(t *testing.T) {
	policy := database.DefaultNotifyPolicy()
	policy.IncidentAlertsEnabled = false

	opening := []aiops.IncidentTransition{
		transition(aiops.TransitionEscalated, nil),
		transition(aiops.TransitionSeverityRaised, func(t *aiops.IncidentTransition) { t.FromState = aiops.StateIncident }),
		transition(aiops.TransitionRecurred, func(t *aiops.IncidentTransition) { t.ReachedIncident = true }),
		transition(aiops.TransitionRecurred, func(t *aiops.IncidentTransition) { t.Occurrences24h = 9 }),
	}
	for _, tr := range opening {
		if _, produced := IncidentAlert(tr, policy); produced {
			t.Errorf("事件告警关闭时 %s 不应产生告警", tr.Kind)
		}
	}
	closing := transition(aiops.TransitionRecoveryStarted, func(t *aiops.IncidentTransition) { t.ReachedIncident = true })
	if got, produced := IncidentAlert(closing, policy); !produced || got.Status != alert.StatusResolved {
		t.Error("事件告警关闭时，已有告警的结束仍应放行")
	}
}

// 订阅回调只入队：转换推迟到管道协程里、拿到当前策略之后才做
func TestIncidentSource_OnTransitionOnlyEnqueues(t *testing.T) {
	submitter := &fakeSubmitter{}
	src := NewIncidentSource(submitter)

	members := []string{"shop/pod/api-7d9f"}
	src.OnTransition(transition(aiops.TransitionEscalated, func(t *aiops.IncidentTransition) { t.Members = members }))
	src.OnTransition(transition(aiops.TransitionCreated, nil)) // 建事件永远不产生告警：不占队列

	if len(submitter.alerts) != 0 || len(submitter.deferred) != 1 {
		t.Fatalf("应只入队一个延迟构造，实际 alerts=%d deferred=%d", len(submitter.alerts), len(submitter.deferred))
	}
	members[0] = "被调用方改写" // 引擎之后复用了这块内存：入队的那份不受影响
	got, produced := submitter.deferred[0](database.DefaultNotifyPolicy())
	if !produced || got.Kind != alert.KindFiring || got.Details[alert.DetailMembers] != "shop/pod/api-7d9f" {
		t.Errorf("延迟构造的结果不对: %+v", got)
	}

	// 队列满：丢弃，不阻塞、不 panic
	submitter.refuse = true
	start := time.Now()
	src.OnTransition(transition(aiops.TransitionEscalated, nil))
	if time.Since(start) > time.Second {
		t.Error("队列满时回调应立即返回")
	}
}
