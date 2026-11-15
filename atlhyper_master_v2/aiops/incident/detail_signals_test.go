package incident

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

func crashingPod(key string, rFinal float64) *aiops.EntityRisk {
	risk := podRisk(key, rFinal)
	risk.Signals = []aiops.RiskSignal{
		{Metric: "container_anomaly", Score: rFinal, Detail: "容器 app: CrashLoopBackOff"},
		{Metric: "deployment_impact", Score: 0.7, Detail: "Deployment shop/api 不可用 67%"},
	}
	return risk
}

// 建事件时把全部成员的信号记进证据；之后每轮评估由 RecordSignals 继续累计；详情里带出证据、结论与时间线展示形态
func TestDetailView_SignalsFromTransitionsAndRecording(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()
	risk := workloadRisk(crashingPod(podA, 0.6), crashingPod(podB, 0.5))
	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, risk), risk)

	later := transitionEpoch.Add(3 * time.Minute)
	s.RecordSignals(ctx, "inc-1", []*aiops.EntityRisk{crashingPod(podA, 0.9), crashingPod(podB, 0.8), crashingPod(podC, 0.8)}, later)

	detail := loadDetail(t, s)
	if len(detail.Evidence) != 2 {
		t.Fatalf("应有 2 类证据，实际 %+v", detail.Evidence)
	}
	crash := detail.Evidence[0]
	if crash.Signal != aiops.SignalContainerCrash || crash.AffectedMembers != 3 || crash.Hits != 5 || crash.MaxScore != 0.9 ||
		!crash.FirstSeen.Equal(transitionEpoch) || !crash.LastSeen.Equal(later) || crash.Reason != "CrashLoopBackOff" {
		t.Errorf("容器崩溃证据不对: %+v", crash)
	}
	if detail.Headline == nil || detail.Headline.Signal != aiops.SignalContainerCrash || detail.Headline.TotalMembers != len(detail.Entities) {
		t.Errorf("结论应取分数最高的证据，成员总数取实体表: %+v（实体 %d）", detail.Headline, len(detail.Entities))
	}
	if len(detail.TimelineView) == 0 || detail.TimelineView[0].Kind != aiops.TimelineItemCreated {
		t.Errorf("时间线展示形态应以开事件开头: %+v", detail.TimelineView)
	}
	if detail.Members == nil || detail.Evidence == nil {
		t.Error("成员与证据在 JSON 里应是数组，不是 null")
	}
}

// 没有本地信号的成员（风险来自依赖传播）不记证据
func TestRecordSignals_SkipsMembersWithoutLocalSignals(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()
	risk := workloadRisk(podRisk(podA, 0.6))
	s.ApplyTransition(ctx, change(aiops.TransitionCreated, transitionEpoch, risk), risk)
	s.RecordSignals(ctx, "inc-1", []*aiops.EntityRisk{{EntityKey: podB, RFinal: 0.4}}, transitionEpoch.Add(time.Minute))

	detail := loadDetail(t, s)
	if len(detail.Evidence) != 0 || detail.Headline != nil {
		t.Errorf("没有信号时证据为空、结论为 nil: %+v %+v", detail.Evidence, detail.Headline)
	}
}
