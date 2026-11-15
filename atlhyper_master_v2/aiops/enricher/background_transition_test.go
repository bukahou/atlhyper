package enricher

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// 后台分析订阅事件变化：只有建事件与升级会触发，触发名与此前保持一致（报告的 trigger 字段、前端文案都认它）
func TestOnIncidentTransition_SubmitsBackgroundTrigger(t *testing.T) {
	tests := []struct {
		kind        aiops.TransitionKind
		wantTrigger string // 空 = 不触发
	}{
		{aiops.TransitionCreated, "incident_created"},
		{aiops.TransitionEscalated, "state_escalated"},
		{aiops.TransitionSeverityRaised, ""},
		{aiops.TransitionRecoveryStarted, ""},
		{aiops.TransitionRecurred, ""},
		{aiops.TransitionClosed, ""},
		{aiops.TransitionClosedStale, ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			// 只有队列、没有消费协程：看放进队列的是什么
			e := &Enricher{bgTrigger: &backgroundTrigger{queue: make(chan triggerEvent, 1)}}

			e.OnIncidentTransition(aiops.IncidentTransition{
				Kind: tt.kind, IncidentID: "inc-1", Severity: "high", RFinal: 0.3,
			})

			select {
			case got := <-e.bgTrigger.queue:
				if tt.wantTrigger == "" {
					t.Fatalf("%s 不应触发后台分析，实际提交了 %+v", tt.kind, got)
				}
				want := triggerEvent{IncidentID: "inc-1", Severity: "high", Trigger: tt.wantTrigger}
				if got != want {
					t.Errorf("提交的触发 = %+v, want %+v（严重度用事件的峰值严重度）", got, want)
				}
			default:
				if tt.wantTrigger != "" {
					t.Fatalf("%s 应触发后台分析 %s", tt.kind, tt.wantTrigger)
				}
			}
		})
	}
}

// 没启用后台触发器时，收到事件变化什么也不做
func TestOnIncidentTransition_WithoutBackgroundTrigger(t *testing.T) {
	e := &Enricher{}
	e.OnIncidentTransition(aiops.IncidentTransition{Kind: aiops.TransitionCreated, IncidentID: "inc-1"})
}

// 订阅者在快照处理主路径上被同步调用：队列满了直接丢弃，不阻塞
func TestOnIncidentTransition_FullQueueDoesNotBlock(t *testing.T) {
	e := &Enricher{bgTrigger: &backgroundTrigger{queue: make(chan triggerEvent, 1)}}
	for i := 0; i < 3; i++ {
		e.OnIncidentTransition(aiops.IncidentTransition{Kind: aiops.TransitionCreated, IncidentID: "inc-1"})
	}
	if len(e.bgTrigger.queue) != 1 {
		t.Errorf("队列容量 1，应只留下 1 条，实际 %d", len(e.bgTrigger.queue))
	}
}
